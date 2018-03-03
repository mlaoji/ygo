package lib

import (
	"errors"
	"fmt"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
	"strconv"
	"sync"
	"time"
)

var Redis = &RedisProxy{c: map[string]*RedisClient{}}

//通过配置文件名字实例化
func NewRedis(conf_name string) (*RedisClient, error) { // {{{
	config := Conf.GetAll(conf_name)
	if 0 == len(config) {
		return nil, errors.New("Redis 资源不存在:" + conf_name)
	}
	return Redis.get(config)
} // }}}

//通过配置信息实例化
func NewRedisClient(config map[string]string) (*RedisClient, error) { // {{{
	return Redis.get(config)
} // }}}

type RedisProxy struct {
	mutex sync.RWMutex
	c     map[string]*RedisClient
}

func (this *RedisProxy) get(config map[string]string) (*RedisClient, error) { //{{{
	host := config["host"]
	this.mutex.RLock()
	if this.c[host] == nil {
		this.mutex.RUnlock()
		this.mutex.Lock()
		defer this.mutex.Unlock()

		if this.c[host] == nil {
			timeout, _ := strconv.Atoi(config["timeout"])
			poolsize, _ := strconv.Atoi(config["poolsize"])
			rc := &RedisClient{
				Host:     host,
				Password: config["password"],
				Timeout:  timeout,
				Poolsize: poolsize,
			}

			err := rc.Init()
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

type RedisClient struct {
	Host     string
	Password string
	Timeout  int
	Poolsize int
	network  string
	_pool    *pool.Pool
}

// {{{ Init
func (this *RedisClient) Init() error {
	var err error
	if "" == this.network {
		this.network = "tcp"
	}

	if 0 == this.Poolsize {
		this.Poolsize = 10
	}

	if 0 == this.Timeout {
		this.Timeout = 3
	}

	_dialfuc := func(network, addr string) (*redis.Client, error) {
		client, err := redis.DialTimeout(network, addr, time.Second*time.Duration(this.Timeout))
		if err != nil {
			return nil, err
		}

		if "" != this.Password {
			if err = client.Cmd("AUTH", this.Password).Err; err != nil {
				client.Close()
				return nil, err
			}
		}

		return client, nil
	}

	this._pool, err = pool.NewCustom(this.network, this.Host, this.Poolsize, _dialfuc)

	if err != nil {
		return err
	}

	return nil
}

// }}}

// {{{ Set
func (this *RedisClient) Set(key string, val interface{}) error {
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("SET", key, val).Err
	this._pool.Put(c)
	return err
}

// }}}

// {{{ Setex
func (this *RedisClient) Setex(key string, secs int, val interface{}) error {
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("SETEX", key, secs, val).Err
	this._pool.Put(c)
	return err
}

// }}}

// {{{ Expire
func (this *RedisClient) Expire(key string, expire int) error {
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("EXPIRE", key, expire).Err
	this._pool.Put(c)
	return err
}

// }}}

// {{{ Exists
func (this *RedisClient) Exists(key string) (bool, error) {
	c, err := this._pool.Get()
	if err != nil {
		return false, err
	}
	val := 0
	val, err = c.Cmd("Exists", key).Int()
	this._pool.Put(c)
	return val == 1, nil
} // }}}

// {{{ incr
func (this *RedisClient) Incr(key string) (val int, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("INCR", key).Int()
	this._pool.Put(c)
	return
} //}}}

// {{{ incrby
func (this *RedisClient) Incrby(key string, increment int) (val int, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("INCRBY", key, increment).Int()
	this._pool.Put(c)
	return
} //}}}

// {{{ incrbyfloat
func (this *RedisClient) IncrbyFloat(key string, increment interface{}) (val float64, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("INCRBYFLOAT", key, increment).Float64()
	this._pool.Put(c)
	return
} //}}}

// {{{ decr
func (this *RedisClient) Decr(key string) (val int, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("DECR", key).Int()
	this._pool.Put(c)
	return
} //}}}

// {{{ decrby
func (this *RedisClient) Decrby(key string, increment int) (val int, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("DECRBY", key, increment).Int()
	this._pool.Put(c)
	return
} //}}}

// {{{ Get
func (this *RedisClient) Get(key string) (val string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("GET", key).Str()
	this._pool.Put(c)
	return
}

// }}}

// {{{ Del
func (this *RedisClient) Del(key string) (err error) {
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("DEL", key).Err
	this._pool.Put(c)
	return err
} // }}}

// {{{ DelAll
func (this *RedisClient) DelAll(keys []string) (err error) {
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("DEL", keys).Err
	this._pool.Put(c)
	return err
} // }}}

// {{{ ExpireAt
func (this *RedisClient) ExpireAt(key string, timestamp int) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	_ = c.Cmd("EXPIREAT", key, timestamp).Err
	this._pool.Put(c)
} // }}}

// {{{ Keys
func (this *RedisClient) Keys(key string) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}
	val, err = c.Cmd("KEYS", key).List()
	this._pool.Put(c)
	return
} // }}}

// {{{ Scan
func (this *RedisClient) Scan(cursor, pattern, count string) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}
	val, err = c.Cmd("SCAN", cursor, "MATCH", pattern, "COUNT", count).List()
	this._pool.Put(c)
	return
} // }}}

//list
// {{{ Rpush
func (this *RedisClient) Rpush(key string, val interface{}) error {
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("Rpush", key, val).Err
	this._pool.Put(c)
	return err
}

// }}}

// {{{ Lpush
func (this *RedisClient) Lpush(key string, val interface{}) error {
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("Lpush", key, val).Err
	this._pool.Put(c)
	return err
}

// }}}

// {{{ Rpop
func (this *RedisClient) Rpop(key string) (val string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("Rpop", key).Str()
	this._pool.Put(c)
	return
}

// }}}

// {{{ Lpop
func (this *RedisClient) Lpop(key string) (val string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("Lpop", key).Str()
	this._pool.Put(c)
	return
}

// }}}

// {{{ Brpop
func (this *RedisClient) Brpop(key string, timeout int) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("BRpop", key, timeout).List()
	this._pool.Put(c)
	return
}

// }}}

// {{{ Blpop
func (this *RedisClient) Blpop(key string, timeout int) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("BLpop", key, timeout).List()
	this._pool.Put(c)
	return
}

// }}}

// {{{ Llen
func (this *RedisClient) Llen(key string) (val int, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("Llen", key).Int()
	this._pool.Put(c)
	return
}

// }}}

// {{{ Lrange key start stop
func (this *RedisClient) Lrange(key string, start, stop int) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}

	val, err = c.Cmd("LRANGE", key, start, stop).List()

	this._pool.Put(c)
	return
} // }}}

// {{{ Mget
func (this *RedisClient) Mget(keys []string) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}

	r := c.Cmd("MGET", keys)
	if r.Err != nil {
		return nil, r.Err
	}

	val, err = r.List()
	this._pool.Put(c)
	return
}

// }}}

//hash
// {{{ Hset
func (this *RedisClient) Hset(key string, field interface{}, val interface{}) error {
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("HSET", key, field, val).Err
	this._pool.Put(c)
	return err
}

// }}}

// {{{ Hsetnx
func (this *RedisClient) Hsetnx(key string, field interface{}, val interface{}) (err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	err = c.Cmd("HSETNX", key, field, val).Err
	this._pool.Put(c)
	return
}

// }}}

// {{{ Hmset
func (this *RedisClient) Hmset(key string, val interface{}) (err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	err = c.Cmd("HMSET", key, val).Err
	this._pool.Put(c)
	return
} // }}}

// {{{ Hget
func (this *RedisClient) Hget(key string, field interface{}) (val string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("HGET", key, field).Str()
	this._pool.Put(c)
	return
} // }}}

// {{{ Hmget
func (this *RedisClient) Hmget(key string, fields interface{}) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("HMGET", key, fields).List()
	this._pool.Put(c)
	return
} // }}}

// {{{ HgetAll
func (this *RedisClient) HgetAll(key string) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("HGETALL", key).List()
	this._pool.Put(c)
	return
} // }}}

// {{{ Hkeys
func (this *RedisClient) Hkeys(key string) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("HKEYS", key).List()
	this._pool.Put(c)
	return
} // }}}

// {{{ Hdel
func (this *RedisClient) Hdel(key, field interface{}) error {
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("HDEL", key, field).Err
	this._pool.Put(c)
	return err
} // }}}

// {{{ HdelAll
func (this *RedisClient) HdelAll(key string) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	_ = c.Cmd("DEL", key).Err
	this._pool.Put(c)
} // }}}

// {{{ Hscan
func (this *RedisClient) Hscan(key string, cursor, pattern, count interface{}) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}
	val, err = c.Cmd("HSCAN", key, cursor, "MATCH", pattern, "COUNT", count).List()
	this._pool.Put(c)
	return
} // }}}

// {{{ Hexists
func (this *RedisClient) Hexists(key string) (bool, error) {
	c, err := this._pool.Get()
	if err != nil {
		return false, err
	}
	val := 0
	val, err = c.Cmd("HExists", key).Int()
	this._pool.Put(c)
	return val == 1, nil
} // }}}

// {{{ Hincrby
func (this *RedisClient) Hincrby(key string, field interface{}, increment int) (val int, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("HINCRBY", key, field, increment).Int()
	this._pool.Put(c)
	return
} // }}}

// {{{ HincrbyFloat
func (this *RedisClient) HincrbyFloat(key string, field interface{}, increment interface{}) (val float64, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("HINCRBYFLOAT", key, field, increment).Float64()
	this._pool.Put(c)
	return
} // }}}

//zset
// {{{ Zadd
func (this *RedisClient) Zadd(key string, score int, val interface{}) error {
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("ZADD", key, score, val).Err
	this._pool.Put(c)
	return err
} // }}}

// {{{ Zincrby
func (this *RedisClient) Zincrby(key string, increment int, val interface{}) error {
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("ZINCRBY", key, increment, val).Err
	this._pool.Put(c)
	return err
} // }}}

// {{{ Zcard
func (this *RedisClient) Zcard(key string) (val int, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("Zcard", key).Int()
	this._pool.Put(c)
	return
} // }}}

// {{{ Zrank
func (this *RedisClient) Zrank(key string, member interface{}) (val int, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("Zrank", key, member).Int()
	this._pool.Put(c)
	return
} // }}}

// {{{ Zrevrank
func (this *RedisClient) Zrevrank(key string, member interface{}) (val int, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("Zrevrank", key, member).Int()
	this._pool.Put(c)
	return
} // }}}

// {{{ Zscore key member
func (this *RedisClient) Zscore(key string, member interface{}) (val int, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return 0, err
	}

	val, err = c.Cmd("ZSCORE", key, member).Int()

	this._pool.Put(c)
	return
} // }}}

// {{{ Zrange key start stop [WITHSCORES]
func (this *RedisClient) Zrange(key string, start, stop int, withscores bool) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}
	if withscores {
		val, err = c.Cmd("ZRANGE", key, start, stop, "WITHSCORES").List()
	} else {
		val, err = c.Cmd("ZRANGE", key, start, stop).List()
	}
	this._pool.Put(c)
	return
} // }}}

// {{{ Zrevrange key start stop [WITHSCORES]
func (this *RedisClient) Zrevrange(key string, start, stop int, withscores bool) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}
	if withscores {
		val, err = c.Cmd("ZREVRANGE", key, start, stop, "WITHSCORES").List()
	} else {
		val, err = c.Cmd("ZREVRANGE", key, start, stop).List()
	}
	this._pool.Put(c)
	return
} // }}}

// {{{ ZrevrangeByScore key max min [WITHSCORES] [LIMIT offset count]
func (this *RedisClient) ZrevrangeByScore(key string, start, step int, withscores bool) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}

	if withscores {
		val, err = c.Cmd("ZREVRANGEBYSCORE", key, "+inf", "-inf", "WITHSCORES", "LIMIT", start, step).List()
	} else {
		val, err = c.Cmd("ZREVRANGEBYSCORE", key, "+inf", "-inf", "LIMIT", start, step).List()
	}

	this._pool.Put(c)
	return
} // }}}

// {{{ ZrangeByScore key min max [WITHSCORES] [LIMIT offset count]
// 返回有序集 key 中，所有 score 值介于 min 和 max 之间(包括等于 min 或 max )的成员。按 score 值递增(从小到大)次序排列。
func (this *RedisClient) ZrangeByScore(key string, min, max, start, step int, withscores bool) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}
	if withscores {
		val, err = c.Cmd("ZRANGEBYSCORE", key, min, max, "WITHSCORES", "LIMIT", start, step).List()
	} else {
		val, err = c.Cmd("ZRANGEBYSCORE", key, min, max, "LIMIT", start, step).List()
	}
	this._pool.Put(c)
	return
} // }}}

// {{{ ZremrangeByScore
func (this *RedisClient) ZremrangeByScore(key string, min, max int) (err error) {
	c, err := this._pool.Get()
	if err != nil {
		return err
	}

	err = c.Cmd("ZREMRANGEBYSCORE", key, min, max).Err

	this._pool.Put(c)
	return err
} // }}}

// {{{ ZrangeBytes key start stop [WITHSCORES]
func (this *RedisClient) ZrangeBytes(key string, start, stop int, withscores bool) (val [][]byte, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}
	if withscores {
		val, err = c.Cmd("ZRANGE", key, start, stop, "WITHSCORES").ListBytes()
	} else {
		val, err = c.Cmd("ZRANGE", key, start, stop).ListBytes()
	}
	this._pool.Put(c)
	return
} // }}}

// {{{ ZrevrangeBytes key start stop [WITHSCORES]
func (this *RedisClient) ZrevrangeBytes(key string, start, stop int, withscores bool) (val [][]byte, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}
	if withscores {
		val, err = c.Cmd("ZREVRANGE", key, start, stop, "WITHSCORES").ListBytes()
	} else {
		val, err = c.Cmd("ZREVRANGE", key, start, stop).ListBytes()
	}
	this._pool.Put(c)
	return
} // }}}

// {{{ Zrem
func (this *RedisClient) Zrem(key string, member interface{}) (val int, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return 0, err
	}

	val, err = c.Cmd("ZREM", key, member).Int()

	this._pool.Put(c)
	return
} // }}}

//sets
// {{{ Sadd
func (this *RedisClient) Sadd(key string, val interface{}) error {
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("SADD", key, val).Err
	this._pool.Put(c)
	return err
} // }}}

// {{{ SisMember
func (this *RedisClient) SisMember(key string, member interface{}) (bool, error) {
	c, err := this._pool.Get()
	if err != nil {
		return false, err
	}
	val := 0
	val, err = c.Cmd("SISMEMBER", key, member).Int()
	this._pool.Put(c)
	return val == 1, nil
} // }}}

// {{{ Srem
func (this *RedisClient) Srem(key string, member interface{}) (val int, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return 0, err
	}

	val, err = c.Cmd("SREM", key, member).Int()

	this._pool.Put(c)
	return
} // }}}

// {{{ Spop
func (this *RedisClient) Spop(key, count int) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}

	val, err = c.Cmd("SPOP", key, count).List()

	this._pool.Put(c)
	return
} // }}}

// {{{ SrandMember
func (this *RedisClient) SrandMember(key, count int) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}

	val, err = c.Cmd("SRANDMEMBER", key, count).List()

	this._pool.Put(c)
	return
} // }}}

// {{{ Smembers
func (this *RedisClient) Smembers(key string) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}

	val, err = c.Cmd("SMEMBERS", key).List()

	this._pool.Put(c)
	return
} // }}}

// {{{ Scard
func (this *RedisClient) Scard(key string) (val int, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("Scard", key).Int()
	this._pool.Put(c)
	return
} // }}}

// {{{ Sscan
func (this *RedisClient) Sscan(key string, cursor, pattern, count interface{}) (val []string, err error) {
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}
	val, err = c.Cmd("SSCAN", key, cursor, "MATCH", pattern, "COUNT", count).List()
	this._pool.Put(c)
	return
} // }}}
