package tx

import (
	"github.com/mlaoji/ygo/lib"
)

func TransBegin(conf ...string) *lib.MysqlClient {
	master_conf := "mysql_master"
	if len(conf) > 0 {
		master_conf = conf[0]
	}
	tx := lib.Mysql.Get(master_conf)
	return tx.Begin()
}
