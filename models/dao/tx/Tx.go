package tx

import (
	"github.com/mlaoji/ygo/lib"
)

//opts: confName, [isReadOnly], 最后一个参数如果为bool值，则表示是否开启只读事务
func TransBegin(opts ...interface{}) *lib.MysqlClient {
	conf_name := "mysql_master"
	is_readonly := false

	l := len(opts)
	if l > 0 {
		if v, ok := opts[0].(string); ok {
			conf_name = v
		}

		if v, ok := opts[l-1].(bool); ok {
			is_readonly = v
		}
	}

	tx := lib.Mysql.Get(conf_name)
	return tx.Begin(is_readonly)
}
