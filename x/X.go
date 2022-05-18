package x

import (
	"errors"
	"github.com/mlaoji/ygo/x/cache"
	"github.com/mlaoji/ygo/x/db"
	"github.com/mlaoji/ygo/x/log"
	"github.com/mlaoji/ygo/x/redis"
	"github.com/mlaoji/ygo/x/yaml"
)

//应用程序运行路径
var AppRoot string

//在项目代码中指定时区
var TIME_ZONE = "Local" // Asia/Shanghai, UTC

//全局共用日志
var Logger = &log.Logger{}

//全局共用配置
var Conf *Config

//本地缓存
var LocalCache *cache.Cache

//全局共用db代理
var DB = NewDBProxy()

//全局共用redis代理
var Redis = NewRedisProxy()

//通过配置文件名字使用redis
func NewRedis(conf_name string) (*redis.RedisClient, error) { // {{{
	config := Conf.GetMap(conf_name)
	if 0 == len(config) {
		return nil, errors.New("Redis 资源不存在:" + conf_name)
	}
	return Redis.Get(config)
} // }}}

//方便直接从x引用
type YamlTree = yaml.YamlTree
type YamlNode = yaml.YamlNode
type YamlMap = yaml.YamlMap
type YamlList = yaml.YamlList
type YamlScalar = yaml.YamlScalar
type DBClient = db.DBClient

//简化业务代码
//使用MAP替代map[string]interface{}
type MAP = map[string]interface{}

//使用MAPS替代map[string]string
type MAPS = map[string]string

//使用MAPI替代map[string]int
type MAPI = map[string]int
