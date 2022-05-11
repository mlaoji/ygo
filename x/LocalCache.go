package x

import (
	"github.com/mlaoji/ygo/x/cache"
	"time"
)

const (
	DEFAULT_TTL    = 5 * time.Minute
	CLEAN_INTERVAL = 30 * time.Minute
)

func NewLocalCache() *cache.Cache { // {{{
	//若配置了redis, 则开启广播，用于远程更新缓存
	redis_conf := Conf.GetMap("redis_localcache")
	if len(redis_conf) > 0 {
		cache.RedisConf = redis_conf
		cache.PubSubOpen = true

		channel_conf := Conf.Get("localcache_pubsubchannel")
		if channel_conf != "" {
			cache.PubSubChannel = channel_conf
		}
	}

	return cache.NewCache(DEFAULT_TTL, CLEAN_INTERVAL)
} // }}}
