package lib

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"time"
)

var LocalCache = &localCache{}

const (
	DEFAULT_TTL    = 5 * time.Minute
	CLEAN_INTERVAL = 30 * time.Second
)

type localCache struct {
	c *cache.Cache
}

func (this *localCache) Init() {
	this.c = cache.New(DEFAULT_TTL, CLEAN_INTERVAL)
	fmt.Println("LocalCache Init")
}

//Add {{{
func (this *localCache) Add(key string, val interface{}, ttls ...time.Duration) error {
	ttl := cache.DefaultExpiration

	if len(ttls) > 0 {
		ttl = ttls[0]
	}

	return this.c.Add(key, val, ttl)
} // }}}

//Set {{{
func (this *localCache) Set(key string, val interface{}, ttls ...time.Duration) {
	ttl := cache.DefaultExpiration

	if len(ttls) > 0 {
		ttl = ttls[0]
	}

	this.c.Set(key, val, ttl)
} // }}}

//Get {{{
func (this *localCache) Get(key string) (val interface{}, found bool) {
	val, found = this.c.Get(key)
	return
} // }}}

//Increment {{{
func (this *localCache) Increment(key string, n ...int) error {
	step := 1

	if len(n) > 0 {
		step = n[0]
	}

	return this.c.Increment(key, int64(step))
} // }}}

//Del {{{
func (this *localCache) Del(key string) {
	this.c.Delete(key)
} // }}}
