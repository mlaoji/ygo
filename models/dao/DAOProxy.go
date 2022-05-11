package dao

import (
	"bytes"
	"fmt"
	"github.com/mlaoji/ygo/x"
	"github.com/mlaoji/ygo/x/db"
	"github.com/mlaoji/ygo/x/yaml"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type DAOProxy struct {
	DBWriter, DBReader db.DBClient
	table              string
	primary            string
	defaultFields      string //默认字段,逗号分隔
	fields             string //通过setFields方法指定的字段,逗号分隔,只能通过getFields使用一次
	countField         string //getCount方法使用的字段
	index              string //查询使用的索引
	limit              string
	autoOrder          bool //是否自动排序(默认按自动主键倒序排序)
	order              string
	forceMaster        bool //强制使用主库读，只能通过useMaster 使用一次
	bind               interface{}
}

func (this *DAOProxy) Init(conf ...string) { //{{{
	master_conf := "db_master"
	slave_conf := "db_slave"

	if len(conf) > 0 {
		master_conf = conf[0]
		slave_conf = conf[0]
	}

	if len(conf) > 1 {
		slave_conf = conf[1]
	}

	slave_confs := x.Conf.GetNode(slave_conf)
	if nil == slave_confs {
		slave_conf = master_conf
	} else if slave_list, ok := slave_confs.(yaml.YamlList); ok {
		rand.Seed(int64(time.Now().Nanosecond()))
		idx := rand.Intn(len(slave_list))
		slave_conf = fmt.Sprintf("%s[%d]", slave_conf, idx)
	}

	this.defaultFields = "*"
	this.autoOrder = true
	this.DBWriter = x.DB.Get(master_conf)
	this.DBReader = x.DB.Get(slave_conf)
} // }}}

func (this *DAOProxy) InitTx(tx db.DBClient) { //使用事务{{{
	this.defaultFields = "*"
	this.autoOrder = true
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

func (this *DAOProxy) SetCountField(field string) *DAOProxy { // {{{
	this.countField = field
	return this
} // }}}

func (this *DAOProxy) GetCountField() string { // {{{
	field := "1"
	if "" != this.countField {
		field = this.countField
		this.countField = ""
	}

	return field
} // }}}

func (this *DAOProxy) SetDefaultFields(fields string) *DAOProxy { // {{{
	this.defaultFields = fields
	return this
} // }}}

//可在读方法前使用，且仅对本次查询起作用，如: NewDAOUser().SetFields("uid").GetRecord(uid)
func (this *DAOProxy) SetFields(fields string) *DAOProxy {
	this.fields = fields
	return this
}

func (this *DAOProxy) GetFields() string { // {{{
	fields := this.defaultFields
	if "" != this.fields {
		fields = this.fields
		this.fields = ""
	}

	return fields
} // }}}

func (this *DAOProxy) UseIndex(idx string) *DAOProxy {
	this.index = idx
	return this
}

func (this *DAOProxy) getIndex() string { // {{{
	idx := this.index
	this.index = ""
	return idx
} // }}}

func (this *DAOProxy) UseMaster(flag ...bool) *DAOProxy { // {{{
	use := true
	if len(flag) > 0 {
		use = flag[0]
	}

	this.forceMaster = use
	return this
} // }}}

func (this *DAOProxy) SetAutoOrder(flag ...bool) *DAOProxy { // {{{
	use := true
	if len(flag) > 0 {
		use = flag[0]
	}

	this.autoOrder = use
	return this
} // }}}

func (this *DAOProxy) Order(order string) *DAOProxy { // {{{
	this.order = order
	this.autoOrder = false
	return this
} // }}}

func (this *DAOProxy) getOrder() string { // {{{
	order := this.order
	if "" == order && this.autoOrder {
		order = this.GetPrimary() + " desc"
	}

	this.order = ""
	this.autoOrder = true

	return order
} // }}}

func (this *DAOProxy) Limit(limit int, limits ...int) *DAOProxy { // {{{
	this.limit = x.ToString(limit)

	if len(limits) > 0 {
		this.limit = this.limit + "," + x.ToString(limits[0])
	}

	return this
} // }}}

func (this *DAOProxy) getLimit() string { // {{{
	limit := this.limit
	this.limit = ""

	return limit
} // }}}

func (this *DAOProxy) GetDBReader() db.DBClient { // {{{
	if this.forceMaster {
		this.forceMaster = false

		return this.DBWriter
	}

	return this.DBReader
} // }}}

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

func (this *DAOProxy) parseFields(objPtr interface{}) { //{{{
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

func (this *DAOProxy) parseRecord(data map[string]interface{}) { //{{{
	this.Fillin(this.bind, data)
} // }}}

func (this *DAOProxy) parseRecords(data []map[string]interface{}) { //{{{
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

//struct2Map
func (this *DAOProxy) preParams(obj interface{}) map[string]interface{} { //{{{
	if p, ok := obj.(map[string]interface{}); ok {
		return p
	}

	objVal := reflect.ValueOf(obj)

	if objVal.Kind() == reflect.Ptr {
		objVal = objVal.Elem()
	}

	if objVal.Kind() != reflect.Struct {
		panic("need a [map or Struct ] or a pointer to [map or Struct ]")
	}

	t := objVal.Type()

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

//解析where条件
//例1:parseParams("x=? and y=?", 1, 2)
//例2:parseParams("x=? and y=?", []interface{}{1,2}) 等价于 parseParams("a=? and b=?", 1, 2)
//例3:parseParams(map[string]interface{}{"a":1,"b":2}) 等价于 parseParams("a=? and b=?", 1, 2)
//例4:parseParams(map[string]interface{}{"a":1,"b":[]interface{}{2, 3}}) 等价于 parseParams("a=? and b in ('2','3')", 1)
func (this *DAOProxy) parseParams(params ...interface{}) (string, []interface{}) { //{{{
	where := ""
	values := []interface{}{}

	l := len(params)
	if l > 0 {
		switch val := params[0].(type) {
		case string:
			where = val
			values = params[1:]
			if l == 2 {
				ptype := reflect.TypeOf(params[1]).Kind()
				if ptype == reflect.Slice || ptype == reflect.Array {
					nvals := []interface{}{}
					nval := reflect.ValueOf(params[1])
					nval = nval.Convert(nval.Type())

					for i := 0; i < nval.Len(); i++ {
						nvals = append(nvals, nval.Index(i).Interface())
					}
					values = nvals
				}
			}
		case map[string]interface{}:
			for k, v := range val {
				if where != "" {
					where = where + " and "
				}

				ptype := reflect.TypeOf(v).Kind()
				if ptype == reflect.Slice || ptype == reflect.Array {
					nval := reflect.ValueOf(v)
					nval = nval.Convert(nval.Type())

					where = where + "`" + k + "` in ("

					for i := 0; i < nval.Len(); i++ {
						if i > 0 {
							where = where + ","
						}
						where = where + "'" + strings.ReplaceAll(fmt.Sprint(nval.Index(i).Interface()), `'`, `\'`) + "'"
					}
					where = where + ")"
				} else {
					where = where + "`" + k + "`=?"
					values = append(values, v)
				}
			}
		default:
			where = fmt.Sprint(params[0])
			values = params[1:]
		}
	}

	return where, values
} //}}}

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
func (this *DAOProxy) AddRecord(vals interface{}) int { //{{{
	return this.DBWriter.Insert(this.table, this.preParams(vals))
} // }}}

func (this *DAOProxy) SetRecord(vals interface{}, id interface{}) int { //{{{
	return this.DBWriter.Update(this.table, this.preParams(vals), this.primary+"=?", id)
} // }}}

func (this *DAOProxy) SetRecordBy(vals interface{}, where string, params ...interface{}) int { //{{{
	return this.DBWriter.Update(this.table, this.preParams(vals), where, params...)
} // }}}

func (this *DAOProxy) ResetRecord(vals interface{}) int { //{{{
	return this.DBWriter.Replace(this.table, this.preParams(vals))
} // }}}

func (this *DAOProxy) GetRecord(id interface{}) map[string]interface{} { //{{{
	row := this.GetDBReader().GetRow("select "+this.GetFields()+" from "+this.table+" where "+this.primary+"=?  limit 1", id)

	if len(row) > 0 && nil != this.bind {
		this.parseRecord(row)
	}

	return row

} // }}}

func (this *DAOProxy) DelRecord(id interface{}) int { //{{{
	return this.DBWriter.Execute("delete from "+this.table+" where "+this.primary+"=? limit 1", id)
} // }}}

func (this *DAOProxy) DelRecordBy(params ...interface{}) int { //{{{
	where, values := this.parseParams(params...)
	return this.DBWriter.Execute("delete from "+this.table+" where "+where+" limit 1", values...)
} // }}}

//Is Dangerous!
func (this *DAOProxy) DelRecords(params ...interface{}) int { //{{{
	where, values := this.parseParams(params...)
	return this.DBWriter.Execute("delete from "+this.table+" where "+where, values...)
} // }}}

func (this *DAOProxy) GetOne(field string, params ...interface{}) interface{} { //{{{
	where, values := this.parseParams(params...)
	if "" != where {
		where = " where " + where
	}

	return this.GetDBReader().GetOne("select "+field+" from "+this.table+where+" limit 1", values...)
} // }}}

//alias for GetOne
func (this *DAOProxy) GetValue(field string, params ...interface{}) interface{} { //{{{
	return this.GetOne(field, params...)
} // }}}

func (this *DAOProxy) GetValues(field string, params ...interface{}) []interface{} { //{{{
	where, values := this.parseParams(params...)
	if "" != where {
		where = " where " + where
	}

	list := this.GetDBReader().GetAll("select "+field+" from "+this.table+where, values...)
	return x.ArrayColumn(list, field).([]interface{})
} // }}}

func (this *DAOProxy) GetValuesMap(keyfield, valfield string, params ...interface{}) x.MAP { //{{{
	where, values := this.parseParams(params...)
	if "" != where {
		where = " where " + where
	}

	list := this.GetDBReader().GetAll("select "+keyfield+", "+valfield+" from "+this.table+where, values...)
	return x.ArrayColumn(list, valfield, keyfield).(x.MAP)
} // }}}

func (this *DAOProxy) GetCount(params ...interface{}) int { //{{{
	where, values := this.parseParams(params...)
	if "" != where {
		where = " where " + where
	}

	fidx := ""
	idx := this.getIndex()
	if "" != idx {
		fidx = " force key(" + idx + ") "
	}

	total, _ := strconv.Atoi(this.GetDBReader().GetOne("select count("+this.GetCountField()+") as total from "+this.table+fidx+where+" limit 1", values...).(string))

	return total
} // }}}

func (this *DAOProxy) Exists(id interface{}) bool { //{{{
	return this.GetOne(this.primary, this.primary+"=?", id) != nil
} // }}}

func (this *DAOProxy) ExistsBy(params ...interface{}) bool { //{{{
	return this.GetOne(this.primary, params...) != nil
} // }}}

func (this *DAOProxy) GetRecordBy(params ...interface{}) map[string]interface{} { //{{{
	where, values := this.parseParams(params...)
	if "" != where {
		where = " where " + where
	}

	row := this.GetDBReader().GetRow("select "+this.GetFields()+" from "+this.table+where+" limit 1", values...)

	if len(row) > 0 && nil != this.bind {
		this.parseRecord(row)
	}

	return row
} // }}}

func (this *DAOProxy) GetRecords(params ...interface{}) []map[string]interface{} { //{{{
	where, values := this.parseParams(params...)

	if "" != where {
		where = " where " + where
	}

	fidx := ""
	idx := this.getIndex()
	if "" != idx {
		fidx = " force key(" + idx + ") "
	}

	order := this.getOrder()
	if "" != order {
		where = where + " order by " + order
	}

	limit := this.getLimit()
	if "" != limit {
		where = where + " limit " + limit
	}

	list := this.GetDBReader().GetAll("select "+this.GetFields()+" from "+this.table+fidx+where, values...)

	if len(list) > 0 && nil != this.bind {
		this.parseRecords(list)
	}

	return list
} // }}}

//大数据下会有性能问题，请谨慎使用
//由于底层每次查询都是从连接池中获取连接，所以开启只读事务，以保证FOUND_ROWS()的两条sql使用同一连接
func (this *DAOProxy) GetList(params ...interface{}) (int, []map[string]interface{}) { //{{{
	where, values := this.parseParams(params...)

	if "" != where {
		where = " where " + where
	}

	fidx := ""
	idx := this.getIndex()
	if "" != idx {
		fidx = " force key(" + idx + ") "
	}

	order := this.getOrder()
	if "" != order {
		where = where + " order by " + order
	}

	limit := this.getLimit()
	if "" != limit {
		where = where + " limit " + limit
	}

	reader := this.GetDBReader().Begin(true)
	defer reader.Rollback()

	list := reader.GetAll("select SQL_CALC_FOUND_ROWS "+this.GetFields()+" from "+this.table+fidx+where, values...)
	total, _ := strconv.Atoi(reader.GetOne("select FOUND_ROWS() as total").(string))

	reader.Commit()

	if len(list) > 0 && nil != this.bind {
		this.parseRecords(list)
	}

	return total, list
} // }}}
