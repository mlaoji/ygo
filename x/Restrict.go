package x

import (
	"fmt"
	"github.com/mlaoji/ygo/x/redis"
	"time"
)

func NewRestrict(rule string, freq, interval int) *Restrict {
	rds, _ := NewRedis(DefaultRestrictRedis)

	return &Restrict{
		Rule:     rule,
		Freq:     freq,
		Interval: interval,
		redis:    rds,
	}
}

var (
	//redis配置文件名, 若找不到此配置则使用本地缓存
	DefaultRestrictRedis = "freq_redis"
)

type Restrict struct {
	//规则名
	Rule string
	//频率
	Freq int
	//周期
	Interval int
	//白名单
	Whitelist []string
	//黑名单
	Blacklist []string

	redis *redis.RedisClient
}

func (this *Restrict) AddWhitelist(uniqid []string) { // {{{
	if len(this.Whitelist) == 0 {
		this.Whitelist = []string{}
	}

	this.Whitelist = append(this.Whitelist, uniqid...)
} // }}}

func (this *Restrict) AddBlacklist(uniqid []string) { // {{{
	if len(this.Blacklist) == 0 {
		this.Blacklist = []string{}
	}

	this.Blacklist = append(this.Blacklist, uniqid...)
} // }}}

func (this *Restrict) Add(uniqid string) bool {
	return this.Check(uniqid, true)
}

//封禁检查
//uniqid: 封禁检查值
//update: 是否检查同时更新次数值，默认只检查是否触犯封禁次数
//返回是否达到封禁次数
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
	return fmt.Sprint(this.Rule+uniqid, Now()/this.Interval)
} //}}}

func (this *Restrict) getRecord(uniqid string) (int, error) { //{{{
	key := this.getKey(uniqid)

	if this.redis != nil {
		cache, err := this.redis.Get(key)
		if nil != err {
			return 0, err
		}

		return ToInt(cache), nil
	} else {
		times, found := LocalCache.Get(key)
		if !found {
			return 0, fmt.Errorf("not found")
		}

		return times.(int), nil
	}
} // }}}

func (this *Restrict) incrRecord(uniqid string) (times int) { //{{{
	var err error
	key := this.getKey(uniqid)

	if this.redis != nil {
		times, err = this.redis.Incr(key)
		if nil != err {
			return 0
		}

		this.redis.Expire(key, this.Interval)
	} else {
		ts, err := LocalCache.Incr(key, 1)
		if nil != err {
			LocalCache.Add(key, 1, time.Duration(this.Interval)*time.Second)
			return 1
		}

		times = ts.(int)
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

	if this.redis != nil {
		this.redis.Del(key)
	} else {
		LocalCache.Del(key)
	}
} // }}}
