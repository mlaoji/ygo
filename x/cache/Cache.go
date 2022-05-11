package cache

import (
	"context"
	"fmt"
	"github.com/mediocregopher/radix.v2/pubsub"
	"github.com/mediocregopher/radix.v2/redis"
	"log"
	"runtime"
	"sync"
	"time"
)

var (
	//默认关闭广播，开启需要正确配置redisConf
	PubSubOpen = false
	//redis 广播订阅频道,需要保持唯一
	PubSubChannel = "ygo-cache"
	//redis 配置
	RedisConf = map[string]string{
		"host":     "127.0.0.1:6379",
		"password": "",
	}
)

type Cache = wrapper

//加个壳，避免goroutine内存溢出
//https://golang.org/pkg/runtime/#SetFinalizer
type wrapper struct {
	*cache
}

//缓存数据基本单元
type Item struct {
	Object     interface{}
	Expiration int64
}

//是否过期
func (item Item) IsExpired() bool { // {{{
	if item.Expiration == 0 {
		return false
	}

	return time.Now().UnixNano() > item.Expiration
} // }}}

type cache struct {
	defaultExpiration time.Duration
	items             map[string]Item
	mu                sync.RWMutex
	janitor           *janitor
	ctx               context.Context
	cancel            context.CancelFunc
}

//初始化缓存实例，设置默认缓存过期时间, 缓存清理间隔时间
func NewCache(defaultExpiration, cleanupInterval time.Duration) *Cache { // {{{
	ctx, cancel := context.WithCancel(context.Background())
	c := &cache{
		defaultExpiration: defaultExpiration,
		ctx:               ctx,
		cancel:            cancel,
		items:             map[string]Item{},
	}

	C := &Cache{c}
	runJanitor(c, cleanupInterval)
	runtime.SetFinalizer(C, stopJanitor)

	return C
} // }}}

func runJanitor(c *cache, ci time.Duration) { // {{{
	j := &janitor{
		Interval: ci,
		stop:     make(chan bool),
	}
	c.janitor = j

	if ci > 0 {
		go j.runCleaner(c)
	}

	if PubSubOpen {
		go j.runBroadcast(c)

		//监控广播redis, 有异常则重启
		go func() {
			for {
				select {
				case <-c.ctx.Done():
					log.Println("restart broadcast")

					c.ctx, c.cancel = context.WithCancel(context.Background())
					j.runBroadcast(c)
				}

				time.Sleep(1 * time.Second)
			}
		}()
	}
} // }}}

func stopJanitor(c *Cache) { // {{{
	c.cancel()
} // }}}

//设置缓存，如果存在则替换，不存在则新增
//过期时间= -1 则永不过期
func (c *cache) Set(k string, v interface{}, td ...time.Duration) { // {{{
	c.mu.Lock()
	c.set(k, v, td...)
	c.mu.Unlock()
} // }}}

//新增缓存，存在则返回错误
func (c *cache) Add(k string, v interface{}, td ...time.Duration) error { // {{{
	c.mu.Lock()
	_, _, found := c.get(k)
	if found {
		c.mu.Unlock()
		return fmt.Errorf("Item %s already exists", k)
	}

	c.set(k, v, td...)
	c.mu.Unlock()

	return nil
} // }}}

//重置缓存，不存在则返回错误
func (c *cache) Replace(k string, v interface{}, td ...time.Duration) error { // {{{
	c.mu.Lock()
	_, _, found := c.get(k)
	if !found {
		c.mu.Unlock()
		return fmt.Errorf("Item %s doesn't exist", k)
	}

	c.set(k, v, td...)
	c.mu.Unlock()

	return nil
} // }}}

// 不设置过期时间，则使用默认过期时间设置，0 则不过期
func (c *cache) set(k string, v interface{}, td ...time.Duration) { // {{{
	var t time.Duration
	if len(td) > 0 {
		t = td[0]
	} else {
		t = c.defaultExpiration
	}

	var e int64
	if t > 0 {
		e = time.Now().Add(t).UnixNano()
	}

	c.items[k] = Item{
		Object:     v,
		Expiration: e,
	}
} // }}}

//获取缓存、是否存在
func (c *cache) Get(k string) (interface{}, bool) { // {{{
	c.mu.RLock()
	v, _, found := c.get(k)
	c.mu.RUnlock()

	return v, found
} // }}}

//获取剩余时间，单位秒, 不存在：-2， 未设置有效期：-1
func (c *cache) Ttl(k string) int64 { // {{{
	c.mu.RLock()
	_, t, found := c.get(k)
	c.mu.RUnlock()

	if !found {
		return -2
	}

	if t.IsZero() {
		return -1
	}

	return t.Unix() - time.Now().Unix()
} // }}}

//获取缓存、过期时间及是否存在
func (c *cache) get(k string) (interface{}, time.Time, bool) { // {{{
	item, found := c.items[k]
	if !found {
		return nil, time.Time{}, false
	}

	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			return nil, time.Time{}, false
		}

		return item.Object, time.Unix(0, item.Expiration), true
	}

	return item.Object, time.Time{}, true
} // }}}

//递增, ns 可选步长
func (c *cache) Incr(k string, ns ...interface{}) (interface{}, error) { // {{{
	var n interface{} = 1

	if len(ns) > 0 {
		n = ns[0]
	}

	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.IsExpired() {
		c.mu.Unlock()
		return nil, fmt.Errorf("Item %s not found", k)
	}
	var nv interface{}
	switch v.Object.(type) {
	case int:
		nv = v.Object.(int) + n.(int)
		v.Object = nv.(int)
	case int8:
		nv = v.Object.(int8) + n.(int8)
		v.Object = nv.(int8)
	case int16:
		nv = v.Object.(int16) + n.(int16)
		v.Object = nv.(int16)
	case int32:
		nv = v.Object.(int32) + n.(int32)
		v.Object = nv.(int32)
	case int64:
		nv = v.Object.(int64) + n.(int64)
		v.Object = nv.(int64)
	case uint:
		nv = v.Object.(uint) + n.(uint)
		v.Object = nv.(uint)
	case uintptr:
		nv = v.Object.(uintptr) + n.(uintptr)
		v.Object = nv.(uintptr)
	case uint8:
		nv = v.Object.(uint8) + n.(uint8)
		v.Object = nv.(uint8)
	case uint16:
		nv = v.Object.(uint16) + n.(uint16)
		v.Object = nv.(uint16)
	case uint32:
		nv = v.Object.(uint32) + n.(uint32)
		v.Object = nv.(uint32)
	case uint64:
		nv = v.Object.(uint64) + n.(uint64)
		v.Object = nv.(uint64)
	case float32:
		nv = v.Object.(float32) + n.(float32)
		v.Object = nv.(float32)
	case float64:
		nv = v.Object.(float64) + n.(float64)
		v.Object = nv.(float64)
	default:
		c.mu.Unlock()
		return nil, fmt.Errorf("The value for %s is not an integer", k)
	}
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
} // }}}

//递减, ns 可选步长
func (c *cache) Decr(k string, ns ...interface{}) (interface{}, error) { // {{{
	var n interface{} = 1

	if len(ns) > 0 {
		n = ns[0]
	}

	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.IsExpired() {
		c.mu.Unlock()
		return nil, fmt.Errorf("Item %s not found", k)
	}
	var nv interface{}
	switch v.Object.(type) {
	case int:
		nv = v.Object.(int) - n.(int)
		v.Object = nv.(int)
	case int8:
		nv = v.Object.(int8) - n.(int8)
		v.Object = nv.(int8)
	case int16:
		nv = v.Object.(int16) - n.(int16)
		v.Object = nv.(int16)
	case int32:
		nv = v.Object.(int32) - n.(int32)
		v.Object = nv.(int32)
	case int64:
		nv = v.Object.(int64) - n.(int64)
		v.Object = nv.(int64)
	case uint:
		nv = v.Object.(uint) - n.(uint)
		v.Object = nv.(uint)
	case uintptr:
		nv = v.Object.(uintptr) - n.(uintptr)
		v.Object = nv.(uintptr)
	case uint8:
		nv = v.Object.(uint8) - n.(uint8)
		v.Object = nv.(uint8)
	case uint16:
		nv = v.Object.(uint16) - n.(uint16)
		v.Object = nv.(uint16)
	case uint32:
		nv = v.Object.(uint32) - n.(uint32)
		v.Object = nv.(uint32)
	case uint64:
		nv = v.Object.(uint64) - n.(uint64)
		v.Object = nv.(uint64)
	case float32:
		nv = v.Object.(float32) - n.(float32)
		v.Object = nv.(float32)
	case float64:
		nv = v.Object.(float64) - n.(float64)
		v.Object = nv.(float64)
	default:
		c.mu.Unlock()
		return nil, fmt.Errorf("The value for %s is not an integer", k)
	}
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
} // }}}

//删除缓存
func (c *cache) Del(k string) { // {{{
	c.mu.Lock()
	_, found := c.items[k]
	if found {
		delete(c.items, k)
	}
	c.mu.Unlock()
} // }}}

//删除过期缓存
func (c *cache) DelExpired() { // {{{
	now := time.Now().UnixNano()
	c.mu.Lock()
	for k, v := range c.items {
		if v.Expiration > 0 && now > v.Expiration {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
} // }}}

//返回所有缓存列表
func (c *cache) Items() map[string]Item { // {{{
	c.mu.RLock()
	defer c.mu.RUnlock()
	m := make(map[string]Item, len(c.items))
	now := time.Now().UnixNano()
	for k, v := range c.items {
		if v.Expiration > 0 {
			if now > v.Expiration {
				continue
			}
		}
		m[k] = v
	}
	return m
} // }}}

//清除所有缓存
func (c *cache) Flush() { // {{{
	c.mu.Lock()
	c.items = map[string]Item{}
	c.mu.Unlock()
} // }}}

//分布式更新缓存, 发送一条广播消息, 订阅的客户端收到消息后更新
func (c *cache) FlushCache(key string) error { // {{{
	//init redis
	pub, err := c.janitor.getPubClient()
	if nil != err {
		return err
	}

	return pub.Cmd("PUBLISH", PubSubChannel, key).Err
} // }}}

type janitor struct { // {{{
	Interval time.Duration
	stop     chan bool
} // }}}

func (j *janitor) runCleaner(c *cache) { // {{{
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			c.DelExpired()
		case <-c.ctx.Done():
			ticker.Stop()
			return
		}
	}
} // }}}

func (j *janitor) runBroadcast(c *cache) { // {{{
	sub, err := j.getSubClient()
	if nil != err {
		log.Printf("getSubClient err: %#v, context canceled\n", err)
		c.cancel()
		return
	}

	log.Println("go-cache start broadcast")

	sub.Subscribe(PubSubChannel)

	subChan := make(chan *pubsub.SubResp)

	go func() {
		for {
			time.Sleep(1 * time.Second)
			resv := sub.Receive()

			if nil != resv.Err {
				if resv.Timeout() {
					continue
				}

				log.Printf("receive  err: %#v, context canceled\n", resv.Err)
				c.cancel()
				return
			}

			subChan <- resv

			ping := sub.Ping()
			if nil != ping.Err {
				log.Printf("ping err: %#v, context canceled\n", ping.Err)
				c.cancel()
				return
			}
		}
	}()

	for {
		select {
		case msg, ok := <-subChan:
			if !ok {
				log.Println("subChan closed")
				c.cancel()
			}

			if msg.Type == pubsub.Message {
				key := msg.Message

				//删除cache
				c.Del(key)
			}
		case <-c.ctx.Done():
			return
		}
	}
} // }}}

func (j *janitor) getPubClient() (*redis.Client, error) { // {{{
	return j.getRedis()
} // }}}

func (j *janitor) getSubClient() (*pubsub.SubClient, error) { // {{{
	sub, err := j.getRedis()

	if nil != err {
		return nil, err
	}

	return pubsub.NewSubClient(sub), nil
} // }}}

func (j *janitor) getRedis() (*redis.Client, error) { // {{{
	host := RedisConf["host"]
	pass := RedisConf["password"]

	r, err := redis.DialTimeout("tcp", host, time.Second*30)
	if nil != err {
		return nil, err
	}

	if "" != pass {
		if err = r.Cmd("AUTH", pass).Err; err != nil {
			r.Close()
			return nil, err
		}
	}

	return r, nil
} // }}}
