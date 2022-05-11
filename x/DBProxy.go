package x

import (
	"fmt"
	"github.com/mlaoji/ygo/x/db"
	"strings"
	"sync"
)

type DBProxy struct {
	mutex sync.RWMutex
	c     map[string]db.DBClient
}

func (this *DBProxy) add(conf_name string) { // {{{
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.c[conf_name] == nil {
		conf := Conf.GetMap(conf_name)
		if 0 == len(conf) {
			panic("db资源不存在:" + conf_name)
		}

		dbt := strings.ToLower(conf["type"])
		var dbClient db.DBClient
		var err error

		switch dbt {
		case "mysql":
			dbClient, err = db.NewMysqlClient(conf["host"], conf["user"], conf["password"], conf["database"], conf["charset"], AsInt(conf["max_open_conns"]), AsInt(conf["max_idle_conns"]))

			if err != nil {
				panic(fmt.Sprintf("mysql connect error: %v", err))
			}
		default:
			panic("不支持的db类型:" + dbt)
		}

		dbClient.SetDebug(AsBool(conf["debug"]))

		this.c[conf_name] = dbClient
		fmt.Println("add db: ", conf_name, " type:", dbt, " ["+conf["host"]+"] #ID:"+dbClient.ID())
	}
} // }}}

func (this *DBProxy) Get(conf_name string) db.DBClient { // {{{
	this.mutex.RLock()
	if this.c[conf_name] == nil {
		this.mutex.RUnlock()
		this.add(conf_name)
	} else {
		this.mutex.RUnlock()
	}

	return this.c[conf_name]
} // }}}
