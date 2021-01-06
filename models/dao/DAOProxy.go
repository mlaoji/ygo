package dao

import (
	"bytes"
	"fmt"
	"github.com/mlaoji/ygo/lib"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type DAOProxy struct {
	DBWriter, DBReader *lib.MysqlClient
	table              string
	primary            string
	defaultFields      string //默认字段,逗号分隔
	fields             string //通过setFields方法指定的字段,逗号分隔,只能通过getFields使用一次
	countField         string //getCount方法使用的字段
	index              string //查询使用的索引
	forceMaster        bool   //强制使用主库读，只能通过useMaster 使用一次
	bind               interface{}
}

func (this *DAOProxy) Init(conf ...string) { //{{{
	master_conf := "mysql_master"
	slave_conf := "mysql_slave"

	if len(conf) > 0 {
		master_conf = conf[0]
		slave_conf = conf[0]
	}

	if len(conf) > 1 {
		slave_conf = conf[1]
	}

	slave_confs := lib.Conf.GetSlice(slave_conf, "slaves")
	if len(slave_confs) > 0 {
		rand.Seed(int64(time.Now().Nanosecond()))
		idx := rand.Intn(len(slave_confs))
		slave_conf = slave_confs[idx]
	}

	this.defaultFields = "*"
	this.DBWriter = lib.Mysql.Get(master_conf)
	this.DBReader = lib.Mysql.Get(slave_conf)
} // }}}

func (this *DAOProxy) InitTx(tx *lib.MysqlClient) { //使用事务{{{
	this.defaultFields = "*"
	this.DBWriter = tx
	this.DBReader = tx
} // }}}

func (this *DAOProxy) SetTable(table string) {
	this.table = table
}

func (this *DAOProxy) GetTable() string {
	return this.table
}

func (this *DAOProxy) SetPrimary(field string) {
	this.primary = field
}

func (this *DAOProxy) GetPrimary() string {
	return this.primary
}

func (this *DAOProxy) SetCountField(field string) *DAOProxy {
	this.countField = field
	return this
}

func (this *DAOProxy) GetCountField() string {
	field := "1"
	if "" != this.countField {
		field = this.countField
		this.countField = ""
	}

	return field
}

func (this *DAOProxy) SetDefaultFields(fields string) *DAOProxy {
	this.defaultFields = fields
	return this
}

//可在读方法前使用，且仅对本次查询起作用，如: NewDAOUser().SetFields("uid").GetRecord(uid)
func (this *DAOProxy) SetFields(fields string) *DAOProxy {
	this.fields = fields
	return this
}

func (this *DAOProxy) GetFields() string {
	fields := this.defaultFields
	if "" != this.fields {
		fields = this.fields
		this.fields = ""
	}

	return fields
}

func (this *DAOProxy) UseIndex(idx string) *DAOProxy {
	this.index = idx
	return this
}

func (this *DAOProxy) UseMaster(flag ...bool) *DAOProxy {
	use := true
	if len(flag) > 0 {
		use = flag[0]
	}

	this.forceMaster = use
	return this
}

func (this *DAOProxy) GetDBReader() *lib.MysqlClient {
	if this.forceMaster {
		this.forceMaster = false

		return this.DBWriter
	}

	return this.DBReader
}

//必须为指针 单条记录指向 struct , 多条记录指向[]struct 或 []*struct
//使用: NewDAOUser().bind([]struct).GetRecords(...
func (this *DAOProxy) Bind(objPtr interface{}) *DAOProxy { // {{{
	if reflect.ValueOf(objPtr).Kind() != reflect.Ptr {
		panic("needs a pointer to a slice or a struct")
	}

	this.bind = objPtr
	this.parseFields(objPtr)
	return this
} // }}}

//parseFields {{{
func (this *DAOProxy) parseFields(objPtr interface{}) {
	objVal := reflect.Indirect(reflect.ValueOf(objPtr))
	data := map[string]interface{}{}
	var valPtr interface{}

	if objVal.Kind() == reflect.Slice {
		sliceElementType := objVal.Type().Elem()

		if sliceElementType.Kind() == reflect.Ptr {
			if sliceElementType.Elem().Kind() == reflect.Struct {
				valPtr = reflect.New(sliceElementType.Elem()).Interface()
			} else {
				panic("need type []*Struct")
			}
		} else if sliceElementType.Kind() == reflect.Struct {
			valPtr = reflect.New(sliceElementType).Interface()
		} else {
			panic("need type []Struct or []*Struct")
		}
	} else if objVal.Kind() == reflect.Struct {
		valPtr = objVal.Interface()
	} else {
		panic("needs a pointer to a slice or a struct")
	}

	data = this.preParams(valPtr)
	buf := bytes.NewBufferString("")
	i := 0
	for k, _ := range data {
		if i > 0 {
			buf.WriteString(",")
		}

		buf.WriteString(k)
		i++
	}

	this.fields = buf.String()
} // }}}

//parseRecord{{{
func (this *DAOProxy) parseRecord(data map[string]interface{}) {
	this.Fillin(this.bind, data)
} // }}}

//parseRecords{{{
func (this *DAOProxy) parseRecords(data []map[string]interface{}) {
	rowsSlicePtr := this.bind
	sliceValue := reflect.Indirect(reflect.ValueOf(rowsSlicePtr))
	if sliceValue.Kind() != reflect.Slice {
		panic("needs a pointer to a slice")
	}

	sliceElementType := sliceValue.Type().Elem()

	isptr := false
	if sliceElementType.Kind() == reflect.Ptr {
		isptr = true
	}

	for _, v := range data {
		if isptr {
			newValue := reflect.New(sliceElementType.Elem())
			this.Fillin(newValue.Interface(), v)
			sliceValue.Set(reflect.Append(sliceValue, reflect.ValueOf(newValue.Interface())))
		} else {
			newValue := reflect.New(sliceElementType)
			this.Fillin(newValue.Interface(), v)
			sliceValue.Set(reflect.Append(sliceValue, reflect.Indirect(reflect.ValueOf(newValue.Interface()))))
		}
	}
} // }}}

//map to struct
func (this *DAOProxy) Fillin(obj interface{}, data map[string]interface{}) { // {{{
	dataStruct := reflect.Indirect(reflect.ValueOf(obj))
	typ := dataStruct.Type()

	numField := typ.NumField()
	for i := 0; i < numField; i++ {
		typField := typ.Field(i)
		valField := dataStruct.Field(i)

		if !valField.CanSet() {
			continue
		}

		tag := typField.Tag.Get("db")
		if tag == "-" || tag == "nil" {
			continue
		}

		if tag == "" {
			tag = strings.ToLower(typField.Name)
		}

		value, ok := data[tag]
		if !ok {
			continue
		}

		kind := typField.Type.Kind()

		switch kind {
		case reflect.Bool:
			b, err := strconv.ParseBool(value.(string))
			if err != nil {
				panic(err)
			}
			valField.SetBool(b)
		case reflect.String:
			valField.SetString(value.(string))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i, err := strconv.ParseInt(value.(string), 0, 64)
			if err != nil {
				panic(err)
			}
			valField.SetInt(i)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			u, err := strconv.ParseUint(value.(string), 0, 64)
			if err != nil {
				panic(err)
			}
			valField.SetUint(u)
		case reflect.Float32, reflect.Float64:
			f, err := strconv.ParseFloat(value.(string), 64)
			if err != nil {
				panic(err)
			}
			valField.SetFloat(f)
		default:
			panic(fmt.Sprintf("unsupported type: %s", kind.String()))
		}
	}
} // }}}

//preParams {{{
//struct2Map
func (this *DAOProxy) preParams(obj interface{}) map[string]interface{} {
	objVal := reflect.ValueOf(obj)

	if objVal.Kind() == reflect.Ptr {
		objVal = objVal.Elem()
	}

	t := objVal.Type()

	if objVal.Kind() == reflect.Map { //map[string]interface{}
		return objVal.Interface().(map[string]interface{})
	}

	if objVal.Kind() != reflect.Struct {
		panic("need a [map or Struct ] or a pointer to [map or Struct ]")
	}

	var data = make(map[string]interface{})

	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("db")
		if tag == "-" || tag == "nil" {
			continue
		}

		if tag == "" {
			tag = strings.ToLower(t.Field(i).Name)
		}

		data[tag] = objVal.Field(i).Interface()
	}

	return data
} // }}}

//以无引号方式代入值,谨慎使用！！！
func (this *DAOProxy) FuncParam(param interface{}) string { //{{{
	return this.DBReader.FuncParam(param)
} // }}}

func (this *DAOProxy) Execute(sql string, params ...interface{}) int { //{{{
	return this.DBWriter.Execute(sql, params...)
} // }}}

//复杂查询
func (this *DAOProxy) Query(sql string, params ...interface{}) []map[string]interface{} { //{{{
	list := this.GetDBReader().GetAll(sql, params...)

	if len(list) > 0 && nil != this.bind {
		this.parseRecords(list)
	}

	return list
} // }}}

//AddRecord、SetRecord、ResetRecord 支持传入map[string]interface{} 和 struct 两种类型参数
//AddRecord {{{
func (this *DAOProxy) AddRecord(vals interface{}) int {
	return this.DBWriter.Insert(this.table, this.preParams(vals))
} // }}}

//SetRecord {{{
func (this *DAOProxy) SetRecord(vals interface{}, id int) int {
	return this.DBWriter.Update(this.table, this.preParams(vals), this.primary+"=?", id)
} // }}}

//SetRecordBy {{{
func (this *DAOProxy) SetRecordBy(vals interface{}, where string, params ...interface{}) int {
	return this.DBWriter.Update(this.table, this.preParams(vals), where, params...)
} // }}}

//ResetRecord {{{
func (this *DAOProxy) ResetRecord(vals interface{}) int {
	return this.DBWriter.Replace(this.table, this.preParams(vals))
} // }}}

//GetRecord {{{
func (this *DAOProxy) GetRecord(id int) map[string]interface{} {
	row := this.GetDBReader().GetRow("select "+this.GetFields()+" from "+this.table+" where "+this.primary+"=?", id)

	if len(row) > 0 && nil != this.bind {
		this.parseRecord(row)
	}

	return row

} // }}}

//DelRecord {{{
func (this *DAOProxy) DelRecord(id int) int {
	return this.DBWriter.Execute("delete from "+this.table+" where "+this.primary+"=? limit 1", id)
} // }}}

//DelRecordBy {{{
func (this *DAOProxy) DelRecordBy(where string, params ...interface{}) int {
	return this.DBWriter.Execute("delete from "+this.table+" where "+where+" limit 1", params...)
} // }}}

//DelRecords {{{ Is Dangerous!
func (this *DAOProxy) DelRecords(where string, params ...interface{}) int {
	return this.DBWriter.Execute("delete from "+this.table+" where "+where, params...)
} // }}}

func (this *DAOProxy) GetOne(field, where string, params ...interface{}) interface{} { //{{{
	if "" == where {
		where = "1"
	}

	return this.GetDBReader().GetOne("select "+field+" from "+this.table+" where "+where, params...)
} // }}}

//GetCount {{{
func (this *DAOProxy) GetCount(where string, params ...interface{}) int {
	idx := ""
	if "" != this.index {
		idx = " force key(" + this.index + ") "
	}

	if "" == where {
		where = "1"
	}

	total, _ := strconv.Atoi(this.GetDBReader().GetOne("select count("+this.GetCountField()+") as total from "+this.table+idx+" where "+where, params...).(string))

	return total
} // }}}

//Exists {{{
func (this *DAOProxy) Exists(id int) bool {
	return this.GetCount(this.primary+"=?", id) > 0
} // }}}

//ExistsBy {{{
func (this *DAOProxy) ExistsBy(where string, params ...interface{}) bool {
	return this.GetCount(where, params...) > 0
} // }}}

//GetRecordBy {{{
func (this *DAOProxy) GetRecordBy(where string, params ...interface{}) map[string]interface{} {
	row := this.GetDBReader().GetRow("select "+this.GetFields()+" from "+this.table+" where "+where, params...)

	if len(row) > 0 && nil != this.bind {
		this.parseRecord(row)
	}

	return row
} // }}}

//GetRecords {{{
func (this *DAOProxy) GetRecords(where string, start, num int, order string, params ...interface{}) []map[string]interface{} {
	idx := ""
	if "" != this.index {
		idx = " force key(" + this.index + ") "
	}

	if "" == where {
		where = "1"
	}

	if "" != order {
		where = where + " order by " + order
	}

	if num > 0 {
		where = where + " limit " + lib.ToString(start) + "," + lib.ToString(num)
	}

	list := this.GetDBReader().GetAll("select "+this.GetFields()+" from "+this.table+idx+" where "+where, params...)

	if len(list) > 0 && nil != this.bind {
		this.parseRecords(list)
	}

	return list
} // }}}
