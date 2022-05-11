package db

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"strings"
	"time"
)

func NewMysqlClient(host, user, password, database, charset string, max_open_conns, max_idle_conns int) (*MysqlClient, error) { // {{{
	c := &MysqlClient{
		Host:         host,
		User:         user,
		Password:     password,
		Database:     database,
		Charset:      charset,
		MaxOpenConns: max_open_conns,
		MaxIdleConns: max_idle_conns,
	}

	err := c.Init()

	return c, err
} // }}}

type MysqlClient struct {
	Host         string
	User         string
	Password     string
	Database     string
	Charset      string
	MaxOpenConns int
	MaxIdleConns int
	Debug        bool
	id           string
	db           *sql.DB
	intx         bool
	tx           *sql.Tx
	executor     Executor
	p            *MysqlClient //实际上没什么用，只在事务中打印调式信息时使用(因为在事务中执行explain语句会出现'busy buffer'的错误)
}

//Init {{{
func (this *MysqlClient) Init() error {
	var err error
	if "" == this.Charset {
		this.Charset = "utf8mb4"
	}

	dsn := this.User + ":" + this.Password + "@tcp(" + this.Host + ")/" + this.Database + "?charset=" + this.Charset
	this.db, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	if this.MaxOpenConns > 0 {
		this.db.SetMaxOpenConns(this.MaxOpenConns)
	}

	if this.MaxIdleConns > 0 {
		this.db.SetMaxIdleConns(this.MaxIdleConns)
	}

	this.executor = &DbExecutor{this.db}
	//defer this.db.Close()

	h := md5.New()
	h.Write([]byte(dsn))
	md5 := hex.EncodeToString(h.Sum(nil))

	this.id = string([]byte(md5)[1:8])

	return nil
} // }}}

func (this *MysqlClient) SetDebug(open bool) { //{{{
	this.Debug = open
} //}}}

func (this *MysqlClient) ID() string { //{{{
	return this.id
} //}}}

func (this *MysqlClient) Begin(is_readonly bool) DBClient { // {{{
	//tx, err := this.db.Begin()
	tx, err := this.db.BeginTx(context.Background(), &sql.TxOptions{
		ReadOnly: is_readonly,
	})

	if err != nil {
		errorHandle(fmt.Sprintf("mysql trans error:%v", err))
	}

	if this.Debug {
		if is_readonly {
			fmt.Println("Begin readonly transaction on #ID:", this.id)
		} else {
			fmt.Println("Begin transaction on #ID:", this.id)
		}
	}

	return &MysqlClient{
		id:       this.id,
		db:       this.db,
		executor: &TxExecutor{tx},
		tx:       tx,
		intx:     true,
		Debug:    this.Debug,
		p:        this,
	}
} // }}}

func (this *MysqlClient) Rollback() { // {{{
	if this.intx && nil != this.tx {
		this.intx = false
		err := this.tx.Rollback()
		if err != nil {
			errorHandle(fmt.Sprintf("mysql trans rollback error:%v", err))
		}

		if this.Debug {
			fmt.Println("Rollback transaction on #ID:", this.id)
		}

	}
} // }}}

func (this *MysqlClient) Commit() { // {{{
	if this.intx && nil != this.tx {
		this.intx = false
		err := this.tx.Commit()
		if err != nil {
			errorHandle(fmt.Sprintf("mysql trans commit error:%v", err))
		}

		if this.Debug {
			fmt.Println("Commit transaction on #ID:", this.id)
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
		fmt.Println(map[string]interface{}{"tx": this.intx, "consume": time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000, "sql": _sql, "val": val, "#ID": this.id})
	}

	if err != nil {
		if err == sql.ErrNoRows {
			// there were no rows, but otherwise no error occurred
		} else {
			errorHandle(err)
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

		if fval := this.getFuncParam(val); fval != "" {
			buf.WriteString(fval)
		} else {
			buf.WriteString("?")
			value = append(value, val)
		}

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
	for col, val := range vals {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(col)
		buf.WriteString("=")

		if fval := this.getFuncParam(val); fval != "" {
			buf.WriteString(fval)
		} else {
			buf.WriteString("?")
			value = append(value, val)
		}

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

func (this *MysqlClient) getFuncParam(param interface{}) string { // {{{
	val := fmt.Sprint(param)
	if strings.HasPrefix(val, "#:F:#") {
		return string([]byte(val)[6:])
	}

	return ""
} // }}}

//拼装参数时，作为可执行字符，而不是字符串值
func DBFuncParam(param interface{}) string { // {{{
	val := fmt.Sprint(param)
	if "" != val {
		return "#:F:#" + val
	}

	return ""
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
		fmt.Println(map[string]interface{}{"tx": this.intx, "consume": time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000, "sql": _sql, "val": val, "#ID": this.id})
	}

	if err != nil {
		errorHandle(err)
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

func (this *MysqlClient) GetAll(_sql string, val ...interface{}) []map[string]interface{} { //{{{
	//分析sql,如果使用了select SQL_CALC_FOUND_ROWS, 分析语句会干扰结果，所以放在真正查询的前面
	if this.Debug {
		if strings.HasPrefix(_sql, "select") {
			expl_results := []map[string]interface{}{}
			if this.intx {
				expl_results = this.p.GetAll("explain "+_sql, val...)
			} else {
				expl_results = this.GetAll("explain "+_sql, val...)
			}
			expl := &MysqlExplain{expl_results}
			expl.DrawConsole()
		}

		fmt.Println("")
	}

	var start_time time.Time
	if this.Debug {
		start_time = time.Now()
	}

	var rows *sql.Rows
	rows, err := this.executor.Query(_sql, val...)

	if this.Debug {
		fmt.Println(map[string]interface{}{"tx": this.intx, "consume": time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000, "sql": _sql, "val": val, "#ID": this.id})
	}

	if err != nil {
		errorHandle(err)
	}

	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		errorHandle(err)
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
			errorHandle(err.Error())
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
		errorHandle(err.Error())
	}

	return data
} // }}}

func errorHandle(err interface{}) { //{{{
	fmt.Println(err)
	panic(err)
} // }}}
