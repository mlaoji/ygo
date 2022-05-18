package x

import (
	"fmt"
	"github.com/mlaoji/ygo/x/redis"
	"sync"
)

func NewRedisProxy() *RedisProxy {
	return &RedisProxy{c: map[string]*redis.RedisClient{}}
}

type RedisProxy struct {
	mutex sync.RWMutex
	c     map[string]*redis.RedisClient
}

func (this *RedisProxy) Get(config map[string]string) (*redis.RedisClient, error) { //{{{
	host := config["host"]
	this.mutex.RLock()
	if this.c[host] == nil {
		this.mutex.RUnlock()
		this.mutex.Lock()
		defer this.mutex.Unlock()

		if this.c[host] == nil {
			rc, err := redis.NewRedisClient(host, config["password"], AsInt(config["timeout"]), AsInt(config["poolsize"]))
			if nil != err {
				fmt.Println("add redis error:", "[", host, "] :", err)
				return nil, err
			}

			this.c[host] = rc
			fmt.Println("add redis :", "[", host, "]")
		}
	} else {
		this.mutex.RUnlock()
	}

	return this.c[host], nil
} // }}}
