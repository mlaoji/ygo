package lib

import (
	"bytes"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"strconv"
	"sync"
	"time"
)

var Mysql = &MysqlProxy{c: map[string]*MysqlClient{}}

type MysqlProxy struct {
	mutex sync.RWMutex
	c     map[string]*MysqlClient
}

func (this *MysqlProxy) Add(conf_name string) { // {{{
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.c[conf_name] == nil {
		conf := Conf.GetAll(conf_name)
		if 0 == len(conf) {
			panic("Mysql 资源不存在:" + conf_name)
		}

		moc, _ := strconv.Atoi(conf["max_open_conns"])
		mic, _ := strconv.Atoi(conf["max_idle_conns"])
		dbg, _ := strconv.ParseBool(conf["debug"])
		my := &MysqlClient{
			Host:         conf["host"],
			User:         conf["user"],
			Password:     conf["password"],
			Database:     conf["database"],
			Charset:      conf["charset"],
			MaxOpenConns: moc,
			MaxIdleConns: mic,
			Debug:        dbg,
		}

		my.Init()

		this.c[conf_name] = my
		fmt.Println("add mysql :" + conf_name + "[" + conf["host"] + "]")
	}
} // }}}

func (this *MysqlProxy) Get(conf_name string) *MysqlClient { // {{{
	this.mutex.RLock()
	if this.c[conf_name] == nil {
		this.mutex.RUnlock()
		this.Add(conf_name)
	} else {
		this.mutex.RUnlock()
	}

	return this.c[conf_name]
} // }}}

type Executor interface {
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

type dbExecutor struct {
	*sql.DB
}
type txExecutor struct {
	*sql.Tx
}

type MysqlClient struct {
	Host         string
	User         string
	Password     string
	Database     string
	Charset      string
	MaxOpenConns int
	MaxIdleConns int
	Debug        bool
	db           *sql.DB
	intx         bool
	tx           *sql.Tx
	executor     Executor
}

//Init {{{
func (this *MysqlClient) Init() {
	var err error
	if "" == this.Charset {
		this.Charset = "utf8mb4"
	}

	this.db, err = sql.Open("mysql", this.User+":"+this.Password+"@tcp("+this.Host+")/"+this.Database+"?charset="+this.Charset)
	if err != nil {
		panic(fmt.Sprintf("mysql connect error:%v", err))
	}

	if this.MaxOpenConns > 0 {
		this.db.SetMaxOpenConns(this.MaxOpenConns)
	}

	if this.MaxIdleConns > 0 {
		this.db.SetMaxIdleConns(this.MaxIdleConns)
	}

	this.executor = &dbExecutor{this.db}
	//defer this.db.Close()
} // }}}

func (this *MysqlClient) SetDebug(open bool) { //{{{
	this.Debug = open
} //}}}

func (this *MysqlClient) Begin() *MysqlClient { // {{{
	tx, err := this.db.Begin()
	if err != nil {
		panic(fmt.Sprintf("mysql trans error:%v", err))
	}
	return &MysqlClient{
		db:       this.db,
		executor: &txExecutor{tx},
		tx:       tx,
		intx:     true,
		Debug:    this.Debug,
	}
} // }}}

func (this *MysqlClient) Rollback() { // {{{
	if this.intx && nil != this.tx {
		this.intx = false
		err := this.tx.Rollback()
		if err != nil {
			panic(fmt.Sprintf("mysql trans rollback error:%v", err))
		}
	}
} // }}}

func (this *MysqlClient) Commit() { // {{{
	if this.intx && nil != this.tx {
		this.intx = false
		err := this.tx.Commit()
		if err != nil {
			panic(fmt.Sprintf("mysql trans commit error:%v", err))
		}
	}
} // }}}

//GetOne {{{
func (this *MysqlClient) GetOne(_sql string, val ...interface{}) interface{} {
	var name string
	var err error

	var start_time time.Time
	if this.Debug {
		start_time = time.Now()
	}

	err = this.executor.QueryRow(_sql, val...).Scan(&name)
	if this.Debug {
		Logger.Debug(map[string]interface{}{"tx": this.intx, "consume": time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000, "sql": _sql, "val": val})
	}

	if err != nil {
		if err == sql.ErrNoRows {
			//fmt.Println("no data")
			// there were no rows, but otherwise no error occurred
		} else {
			panic(err)
		}
	}

	return name
} // }}}

//insert {{{
func (this *MysqlClient) insert(table string, vals map[string]interface{}, isreplace bool) int {
	buf := bytes.NewBufferString("")

	if isreplace {
		buf.WriteString("replace into ")
	} else {
		buf.WriteString("insert into ")
	}

	buf.WriteString(table)
	buf.WriteString(" set ")

	var value []interface{}
	i := 0
	for col, val := range vals {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(col)
		buf.WriteString("=")
		buf.WriteString("?")
		value = append(value, val)
		i++
	}
	_sql := buf.String()
	result := this.execute(_sql, value...)
	lastid, _ := result.LastInsertId()

	return int(lastid)
} // }}}

//Insert{{{
func (this *MysqlClient) Insert(table string, vals map[string]interface{}) int {
	return this.insert(table, vals, false)
} // }}}

//Replace {{{
func (this *MysqlClient) Replace(table string, vals map[string]interface{}) int {
	return this.insert(table, vals, true)
} // }}}

//Update{{{
func (this *MysqlClient) Update(table string, vals map[string]interface{}, where string, val ...interface{}) int {
	buf := bytes.NewBufferString("update ")

	buf.WriteString(table)
	buf.WriteString(" set ")

	var value []interface{}
	i := 0
	for col, v := range vals {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(col)
		buf.WriteString("=")
		buf.WriteString("?")
		value = append(value, v)
		i++
	}

	buf.WriteString(" where ")
	buf.WriteString(where)
	_sql := buf.String()

	value = append(value, val...)
	result := this.execute(_sql, value...)
	affect, _ := result.RowsAffected()

	return int(affect)
} // }}}

//Execute {{{
func (this *MysqlClient) Execute(_sql string, val ...interface{}) int {
	result := this.execute(_sql, val...)
	affect, _ := result.RowsAffected()

	return int(affect)
} // }}}

//execute {{{
func (this *MysqlClient) execute(_sql string, val ...interface{}) (result sql.Result) {
	var start_time time.Time
	if this.Debug {
		start_time = time.Now()
	}

	result, err := this.executor.Exec(_sql, val...)

	if this.Debug {
		Logger.Debug(map[string]interface{}{"tx": this.intx, "consume": time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000, "sql": _sql, "val": val})
	}

	if err != nil {
		panic(err)
	}

	return result
} // }}}

//GetRow {{{
func (this *MysqlClient) GetRow(_sql string, val ...interface{}) map[string]interface{} {
	list := this.GetAll(_sql, val...)
	if len(list) > 0 {
		return list[0]
	}

	return make(map[string]interface{}, 0)
} // }}}

//GetAll {{{
func (this *MysqlClient) GetAll(_sql string, val ...interface{}) []map[string]interface{} {
	var start_time time.Time
	if this.Debug {
		start_time = time.Now()
	}

	var rows *sql.Rows
	rows, err := this.executor.Query(_sql, val...)

	if this.Debug {
		Logger.Debug(map[string]interface{}{"tx": this.intx, "consume": time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000, "sql": _sql, "val": val})
	}

	if err != nil {
		panic(err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		panic(err)
	}

	// Make a slice for the values
	values := make([]sql.RawBytes, len(cols))

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	var data []map[string]interface{}
	// Fetch rows
	var j = 0
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(scanArgs...)
		if err != nil {
			panic(err.Error())
		}

		row := map[string]interface{}{}
		var value interface{}
		for i, col := range values {
			// Here we can check if the value is nil (NULL value)
			if col == nil {
				value = ""
			} else {
				value = string(col)
			}

			row[cols[i]] = value

		}

		data = append(data, row)
		j++
	}

	if err = rows.Err(); err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	return data
} // }}}

/*
func main() {
	_db := &Mysql{
		Host:     "101.201.121.93:4389",
		User:     "testdb",
		Password: "testserverfortt",
		Database: "yb_live",
	}
	_db.Init()

	_sql := "select count(0) as title from topic where tid = ?"

	v := _db.GetOne(_sql, "1")

	fmt.Printf("%#v", v)
	s1, _ := strconv.Atoi(v)
	fmt.Println(s1)

	row := _db.GetRow(_sql, "1")

	fmt.Printf("%#v", row)
	for k, v := range row {
		fmt.Printf("%#v", k)
		fmt.Printf("%#v", v)
	}

	fmt.Println("...")
	_sql = "select title from topic where tid = ?"
	s := _db.GetAll(_sql, "1")

	fmt.Printf("%+v", s)
	for k, v := range s {
		fmt.Sprintln(k, v)

		for k1, v1 := range v {
			fmt.Sprintln(k1, v1)
		}
	}

	fmt.Println("...")
	id := _db.Update("topic", map[string]interface{}{"title": "testaaa"}, "tid=?", 28)
	fmt.Printf("%#v", id)

	id1 := _db.Replace("topic", map[string]interface{}{"title": "testaaa", "tid": 28})
	fmt.Printf("%#v", id1)

	id2 := _db.Update("topic", map[string]interface{}{"title": "testbbb"}, "tid=?", 28)
	fmt.Printf("%#v", id2)
}

*/
