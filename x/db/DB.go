package db

import (
	"database/sql"
)

type Executor interface {
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

type DbExecutor struct {
	*sql.DB
}

type TxExecutor struct {
	*sql.Tx
}

type DBClient interface {
	Init() error
	ID() string
	SetDebug(open bool)
	Begin(is_readonly bool) DBClient
	Rollback()
	Commit()
	GetOne(_sql string, val ...interface{}) interface{}
	Insert(table string, vals map[string]interface{}) int
	Replace(table string, vals map[string]interface{}) int
	Update(table string, vals map[string]interface{}, where string, val ...interface{}) int
	Execute(_sql string, val ...interface{}) int
	GetRow(_sql string, val ...interface{}) map[string]interface{}
	GetAll(_sql string, val ...interface{}) []map[string]interface{}
}
