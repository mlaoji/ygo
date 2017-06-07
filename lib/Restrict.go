package lib

import (
	"errors"
	"fmt"
	"math"
	"time"
)

func NewRestrict(rule string, freq, interval int) *Restrict {
	return &Restrict{
		Rule:     rule,
		Freq:     freq,
		Interval: interval,
	}
}

const (
	CACHE_TYPE_LOCAL = iota
	CACHE_TYPE_REDIS
)

type Restrict struct {
	Rule      string
	Freq      int //频率
	Interval  int //周期
	Whitelist []string
	Blacklist []string
	CacheType int
	RedisConf string
}

func (this *Restrict) AddWhitelist(uniqid []string) {
	if len(this.Whitelist) == 0 {
		this.Whitelist = []string{}
	}

	this.Whitelist = append(this.Whitelist, uniqid...)
}

func (this *Restrict) AddBlacklist(uniqid []string) {
	if len(this.Blacklist) == 0 {
		this.Blacklist = []string{}
	}

	this.Blacklist = append(this.Blacklist, uniqid...)
}

func (this *Restrict) UseRedis(conf_name string) {
	this.CacheType = CACHE_TYPE_REDIS
	this.RedisConf = conf_name
}

func (this *Restrict) Add(uniqid string) bool {
	return this.Check(uniqid, true)
}

/**
 * 封禁检查
 * @param $uniqid 封禁检查值
 * @param bool $update 是否检查同时更新次数值，默认只检查是否触犯封禁次数
 * @return bool true 达到封禁次数 false 未达到封禁次数
 */
func (this *Restrict) Check(uniqid string, update bool) bool { //{{{
	for _, v := range this.Whitelist {
		if uniqid == v {
			return false
		}
	}

	for _, v := range this.Blacklist {
		if uniqid == v {
			return true
		}
	}

	times, _ := this.getRecord(uniqid)

	if times >= this.Freq {
		return true
	}

	if update {
		times = this.incrRecord(uniqid)

		if times > this.Freq {
			return true
		}
	}

	return false
} // }}}

func (this *Restrict) getKey(uniqid string) string { //{{{
	return fmt.Sprint(this.Rule+uniqid, math.Ceil(float64(Now())/float64(this.Interval)))
} //}}}

func (this *Restrict) getRecord(uniqid string) (int, error) { //{{{
	key := this.getKey(uniqid)

	if this.CacheType == CACHE_TYPE_LOCAL {
		times, found := LocalCache.Get(key)
		if !found {
			return 0, errors.New("not found")
		}

		return times.(int), nil
	} else {
		redis, err := NewRedis(this.RedisConf)
		if nil != err {
			return 0, errors.New(fmt.Sprint(err))
		}

		cache, err := redis.Get(key)
		if nil != err {
			return 0, errors.New(fmt.Sprint(err))
		}

		return Toint(cache), nil
	}
} // }}}

func (this *Restrict) incrRecord(uniqid string) (times int) { //{{{
	key := this.getKey(uniqid)

	if this.CacheType == CACHE_TYPE_LOCAL {
		err := LocalCache.Increment(key, 1)
		if nil != err {
			LocalCache.Add(key, 1, time.Duration(this.Interval)*time.Second)
			return 1
		}

		times, found := LocalCache.Get(key)
		if !found {
			return 0
		}

		return times.(int)
	} else {
		redis, err := NewRedis(this.RedisConf)
		if nil != err {
			return 0
		}
		times, err = redis.Incr(key)
		if nil != err {
			return 0
		}

		redis.Expire(key, this.Interval)
	}

	return
} // }}}

func (this *Restrict) Surplus(uniqid string) int { //{{{
	key := this.getKey(uniqid)

	times, err := this.getRecord(key)
	if nil != err {
		return 0
	}

	surplus := this.Freq - times
	if surplus > 0 {
		return surplus
	}

	return 0
} // }}}

func (this *Restrict) Delete(uniqid string) { //{{{
	key := this.getKey(uniqid)

	if this.CacheType == CACHE_TYPE_LOCAL {
		LocalCache.Del(key)
	} else {
		if redis, err := NewRedis(this.RedisConf); nil == err {
			redis.Del(key)
		}
	}
} // }}}
