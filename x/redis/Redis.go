package redis

import (
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
	"time"
)

var (
	DefaultPoolsize     = 10 //连接池最大连接数
	DefaultTimeout      = 3  //连接超时, 单位:秒
	DefaultReadTimeout  = 0  //读超时, 单位:秒
	DefaultWriteTimeout = 0  //写超时, 单位:秒
)

func NewRedisClient(host, password string, options ...FuncRcOption) (*RedisClient, error) { // {{{
	rc := &RedisClient{
		Host:         host,
		Password:     password,
		Poolsize:     DefaultPoolsize,
		Timeout:      DefaultTimeout,
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
	}

	for _, opt := range options {
		opt(rc)
	}

	err := rc.Init()

	return rc, err
} // }}}

type FuncRcOption func(rc *RedisClient)

//NewRedisClient 设置参数 poolsize
func WithPoolsize(poolsize int) FuncRcOption { // {{{
	return func(rc *RedisClient) {
		if poolsize > 0 {
			rc.Poolsize = poolsize
		}
	}
} // }}}

//NewRedisClient 设置参数 Timeout
func WithTimeout(timeout int) FuncRcOption { // {{{
	return func(rc *RedisClient) {
		if timeout > 0 {
			rc.Timeout = timeout
		}
	}
} // }}}

//NewRedisClient 设置参数 ReadTimeout
func WithReadTimeout(timeout int) FuncRcOption { // {{{
	return func(rc *RedisClient) {
		if timeout > 0 {
			rc.ReadTimeout = timeout
		}
	}
} // }}}

//NewRedisClient 设置参数 WriteTimeout
func WithWriteTimeout(timeout int) FuncRcOption { // {{{
	return func(rc *RedisClient) {
		if timeout > 0 {
			rc.WriteTimeout = timeout
		}
	}
} // }}}

type RedisClient struct {
	Host         string
	Password     string
	Timeout      int //DialTimeout
	ReadTimeout  int
	WriteTimeout int
	Poolsize     int
	network      string
	_pool        *pool.Pool
}

func (this *RedisClient) Init() error { // {{{
	var err error
	if "" == this.network {
		this.network = "tcp"
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

		if this.ReadTimeout > 0 {
			client.ReadTimeout = time.Second * time.Duration(this.ReadTimeout)
		}

		if this.WriteTimeout > 0 {
			client.WriteTimeout = time.Second * time.Duration(this.WriteTimeout)
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

func (this *RedisClient) Set(key string, val interface{}) error { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("SET", key, val).Err
	this._pool.Put(c)
	return err
}

// }}}

func (this *RedisClient) Setex(key string, secs int, val interface{}) error { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("SETEX", key, secs, val).Err
	this._pool.Put(c)
	return err
}

// }}}

func (this *RedisClient) Expire(key string, expire int) error { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("EXPIRE", key, expire).Err
	this._pool.Put(c)
	return err
}

// }}}

func (this *RedisClient) Exists(key string) (bool, error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return false, err
	}
	val := 0
	val, err = c.Cmd("Exists", key).Int()
	this._pool.Put(c)
	return val == 1, nil
} // }}}

func (this *RedisClient) Ttl(key string) (int, error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return 0, err
	}
	val := 0
	val, err = c.Cmd("Ttl", key).Int()
	this._pool.Put(c)
	return val, nil
} // }}}

func (this *RedisClient) Incr(key string) (val int, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("INCR", key).Int()
	this._pool.Put(c)
	return
} //}}}

func (this *RedisClient) Incrby(key string, increment int) (val int, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("INCRBY", key, increment).Int()
	this._pool.Put(c)
	return
} //}}}

func (this *RedisClient) IncrbyFloat(key string, increment interface{}) (val float64, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("INCRBYFLOAT", key, increment).Float64()
	this._pool.Put(c)
	return
} //}}}

func (this *RedisClient) Decr(key string) (val int, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("DECR", key).Int()
	this._pool.Put(c)
	return
} //}}}

func (this *RedisClient) Decrby(key string, increment int) (val int, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("DECRBY", key, increment).Int()
	this._pool.Put(c)
	return
} //}}}

func (this *RedisClient) Get(key string) (val string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("GET", key).Str()
	this._pool.Put(c)
	return
}

// }}}

func (this *RedisClient) Del(key string) (err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("DEL", key).Err
	this._pool.Put(c)
	return err
} // }}}

func (this *RedisClient) DelAll(keys []string) (err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("DEL", keys).Err
	this._pool.Put(c)
	return err
} // }}}

func (this *RedisClient) ExpireAt(key string, timestamp int) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	_ = c.Cmd("EXPIREAT", key, timestamp).Err
	this._pool.Put(c)
} // }}}

func (this *RedisClient) Keys(key string) (val []string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}
	val, err = c.Cmd("KEYS", key).List()
	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Scan(cursor, pattern, count string) (val []string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}
	val, err = c.Cmd("SCAN", cursor, "MATCH", pattern, "COUNT", count).List()
	this._pool.Put(c)
	return
} // }}}

//list
func (this *RedisClient) Rpush(key string, val interface{}) error { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("Rpush", key, val).Err
	this._pool.Put(c)
	return err
}

// }}}

func (this *RedisClient) Lpush(key string, val interface{}) error { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("Lpush", key, val).Err
	this._pool.Put(c)
	return err
}

// }}}

func (this *RedisClient) Rpop(key string) (val string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("Rpop", key).Str()
	this._pool.Put(c)
	return
}

// }}}

func (this *RedisClient) Lpop(key string) (val string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("Lpop", key).Str()
	this._pool.Put(c)
	return
}

// }}}

func (this *RedisClient) Brpop(key string, timeout int) (val []string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("BRpop", key, timeout).List()
	this._pool.Put(c)
	return
}

// }}}

func (this *RedisClient) Blpop(key string, timeout int) (val []string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("BLpop", key, timeout).List()
	this._pool.Put(c)
	return
}

// }}}

func (this *RedisClient) Llen(key string) (val int, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("Llen", key).Int()
	this._pool.Put(c)
	return
}

// }}}

func (this *RedisClient) Lrange(key string, start, stop int) (val []string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}

	val, err = c.Cmd("LRANGE", key, start, stop).List()

	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Mget(keys []string) (val []string, err error) { // {{{
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
func (this *RedisClient) Hset(key string, field interface{}, val interface{}) error { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("HSET", key, field, val).Err
	this._pool.Put(c)
	return err
}

// }}}

func (this *RedisClient) Hsetnx(key string, field interface{}, val interface{}) (err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	err = c.Cmd("HSETNX", key, field, val).Err
	this._pool.Put(c)
	return
}

// }}}

func (this *RedisClient) Hmset(key string, val interface{}) (err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	err = c.Cmd("HMSET", key, val).Err
	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Hget(key string, field interface{}) (val string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("HGET", key, field).Str()
	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Hmget(key string, fields interface{}) (val []string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("HMGET", key, fields).List()
	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) HgetAll(key string) (val map[string]string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("HGETALL", key).Map()
	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Hkeys(key string) (val []string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("HKEYS", key).List()
	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Hdel(key, field interface{}) error { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("HDEL", key, field).Err
	this._pool.Put(c)
	return err
} // }}}

func (this *RedisClient) HdelAll(key string) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	_ = c.Cmd("DEL", key).Err
	this._pool.Put(c)
} // }}}

func (this *RedisClient) Hscan(key string, cursor, pattern, count interface{}) (val []string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}
	val, err = c.Cmd("HSCAN", key, cursor, "MATCH", pattern, "COUNT", count).List()
	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Hexists(key string) (bool, error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return false, err
	}
	val := 0
	val, err = c.Cmd("HExists", key).Int()
	this._pool.Put(c)
	return val == 1, nil
} // }}}

func (this *RedisClient) Hincrby(key string, field interface{}, increment int) (val int, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("HINCRBY", key, field, increment).Int()
	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) HincrbyFloat(key string, field interface{}, increment interface{}) (val float64, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("HINCRBYFLOAT", key, field, increment).Float64()
	this._pool.Put(c)
	return
} // }}}

//zset
func (this *RedisClient) Zadd(key string, score int, val interface{}) error { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("ZADD", key, score, val).Err
	this._pool.Put(c)
	return err
} // }}}

func (this *RedisClient) Zincrby(key string, increment int, val interface{}) error { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("ZINCRBY", key, increment, val).Err
	this._pool.Put(c)
	return err
} // }}}

func (this *RedisClient) Zcard(key string) (val int, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("Zcard", key).Int()
	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Zrank(key string, member interface{}) (val int, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("Zrank", key, member).Int()
	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Zrevrank(key string, member interface{}) (val int, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("Zrevrank", key, member).Int()
	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Zscore(key string, member interface{}) (val int, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return 0, err
	}

	val, err = c.Cmd("ZSCORE", key, member).Int()

	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Zrange(key string, start, stop int, withscores bool) (val []string, err error) { // {{{
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

func (this *RedisClient) Zrevrange(key string, start, stop int, withscores bool) (val []string, err error) { // {{{
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

func (this *RedisClient) ZrevrangeByScore(key string, start, step int, withscores bool) (val []string, err error) { // {{{
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

// 返回有序集 key 中，所有 score 值介于 min 和 max 之间(包括等于 min 或 max )的成员。按 score 值递增(从小到大)次序排列。
func (this *RedisClient) ZrangeByScore(key string, min, max, start, step int, withscores bool) (val []string, err error) { // {{{
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

func (this *RedisClient) ZremrangeByScore(key string, min, max int) (err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return err
	}

	err = c.Cmd("ZREMRANGEBYSCORE", key, min, max).Err

	this._pool.Put(c)
	return err
} // }}}

func (this *RedisClient) ZrangeBytes(key string, start, stop int, withscores bool) (val [][]byte, err error) { // {{{
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

func (this *RedisClient) ZrevrangeBytes(key string, start, stop int, withscores bool) (val [][]byte, err error) { // {{{
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

func (this *RedisClient) Zrem(key string, member interface{}) (val int, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return 0, err
	}

	val, err = c.Cmd("ZREM", key, member).Int()

	this._pool.Put(c)
	return
} // }}}

//sets
func (this *RedisClient) Sadd(key string, val interface{}) error { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return err
	}
	err = c.Cmd("SADD", key, val).Err
	this._pool.Put(c)
	return err
} // }}}

func (this *RedisClient) SisMember(key string, member interface{}) (bool, error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return false, err
	}
	val := 0
	val, err = c.Cmd("SISMEMBER", key, member).Int()
	this._pool.Put(c)
	return val == 1, nil
} // }}}

func (this *RedisClient) Srem(key string, member interface{}) (val int, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return 0, err
	}

	val, err = c.Cmd("SREM", key, member).Int()

	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Spop(key, count int) (val []string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}

	val, err = c.Cmd("SPOP", key, count).List()

	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) SrandMember(key, count int) (val []string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}

	val, err = c.Cmd("SRANDMEMBER", key, count).List()

	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Smembers(key string) (val []string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}

	val, err = c.Cmd("SMEMBERS", key).List()

	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Scard(key string) (val int, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	val, err = c.Cmd("Scard", key).Int()
	this._pool.Put(c)
	return
} // }}}

func (this *RedisClient) Sscan(key string, cursor, pattern, count interface{}) (val []string, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return nil, err
	}
	val, err = c.Cmd("SSCAN", key, cursor, "MATCH", pattern, "COUNT", count).List()
	this._pool.Put(c)
	return
} // }}}

//__call 魔术方法
func (this *RedisClient) Call(cmd string, args ...interface{}) (resp *redis.Resp, err error) { // {{{
	c, err := this._pool.Get()
	if err != nil {
		return
	}
	resp = c.Cmd(cmd, args...)
	this._pool.Put(c)
	return
} // }}}
