package lib

import (
	"fmt"
	"github.com/mlaoji/go-cache"
	"time"
)

var LocalCache = &localCache{}

const (
	DEFAULT_TTL    = 5 * time.Minute
	CLEAN_INTERVAL = 30 * time.Minute
)

type localCache struct {
	c *cache.Cache
}

func (this *localCache) Init() { // {{{
	//若配置了redis, 则开启广播，用于远程更新缓存
	redis_conf := Conf.GetAll("redis_gocache")
	if len(redis_conf) > 0 {
		cache.RedisConf = redis_conf
		cache.PubSubOpen = true
	}

	this.c = cache.NewCache(DEFAULT_TTL, CLEAN_INTERVAL)
	fmt.Println("LocalCache Init")
} // }}}

func (this *localCache) Add(key string, val interface{}, ttls ...time.Duration) error { //{{{
	return this.c.Add(key, val, ttls...)
} // }}}

func (this *localCache) Set(key string, val interface{}, ttls ...time.Duration) { //{{{
	this.c.Set(key, val, ttls...)
} // }}}

func (this *localCache) Get(key string) (val interface{}, found bool) { //{{{
	val, _, found = this.c.Get(key)
	return
} // }}}

func (this *localCache) Increment(key string, n ...int) (interface{}, error) { //{{{
	step := 1

	if len(n) > 0 {
		step = n[0]
	}

	return this.c.Increment(key, int64(step))
} // }}}

func (this *localCache) Decrement(key string, n ...int) (interface{}, error) { //{{{
	step := 1

	if len(n) > 0 {
		step = n[0]
	}

	return this.c.Decrement(key, int64(step))
} // }}}

func (this *localCache) Del(key string) { //{{{
	this.c.Delete(key)
} // }}}

//通过redis广播，更新所有服务器上的同名缓存
func (this *localCache) FlushCache(key string) error { //{{{
	return this.c.FlushCache(key)
} // }}}
