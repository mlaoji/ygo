package lib

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"math/rand"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//在项目代码中指定时区
var TIME_ZONE = "Local" // Asia/Shanghai, UTC

//简化业务代码,直接使用MAP替代map[string]interface{}
type MAP = map[string]interface{}

//获取MAP的某个节点
func GetMapNode(m interface{}, keys ...string) (value interface{}, find bool) { // {{{
	value = m
	if len(keys) > 0 {
		for _, v := range keys {
			if newval, ok := value.(map[string]interface{}); ok {
				value = newval[v]
			} else {
				return nil, false
			}
		}
	}

	return value, true
} // }}}

//获取本机ip
func GetLocalIp() string { // {{{
	addrs, _ := net.InterfaceAddrs()
	var ip string = ""
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return ip
} // }}}

func JsonEncode(data interface{}) string { // {{{
	content, err := json.MarshalIndent(data, "", "")
	if err != nil {
		return ""
	}

	return strings.Replace(string(content), "\n", "", -1)
} // }}}

func JsonDecode(data string) interface{} { // {{{
	var obj interface{}
	err := json.Unmarshal([]byte(data), &obj)
	if err != nil {
		return nil
	}

	return convertFloat(obj)
} // }}}

//格式化科学法表示的数字
func convertFloat(r interface{}) interface{} { // {{{
	switch val := r.(type) {
	case map[string]interface{}:
		s := map[string]interface{}{}
		for k, v := range val {
			s[k] = convertFloat(v)
		}
		return s
	case []interface{}:
		s := []interface{}{}
		for _, v := range val {
			s = append(s, convertFloat(v))
		}
		return s
	case float64:
		if float64(int(val)) == val {
			return int(val)
		}
		return val
	default:
		return r
	}
} // }}}

//同php in_array
func InArray(search interface{}, arr interface{}, stricts ...bool) bool { // {{{
	//是否严格检查类型
	strict := false
	if len(stricts) > 0 {
		strict = stricts[0]
	}

	val := reflect.ValueOf(arr)
	val = val.Convert(val.Type())

	typ := reflect.TypeOf(arr).Kind()

	switch typ {
	case reflect.Map:
		s := val.MapRange()

		for s.Next() {
			s.Value().Convert(s.Value().Type())
			if strict {
				if reflect.DeepEqual(search, s.Value().Interface()) {
					return true
				}
			} else {
				if strings.Contains(fmt.Sprint(s.Value().Interface()), fmt.Sprint(search)) {
					return true
				}
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			if strict {
				if reflect.DeepEqual(search, val.Index(i).Interface()) {
					return true
				}
			} else {
				if strings.Contains(fmt.Sprint(val.Index(i).Interface()), fmt.Sprint(search)) {
					return true
				}
			}
		}
	}

	return false
} // }}}

const (
	RAND_KIND_NUM   = 0 // 纯数字
	RAND_KIND_LOWER = 1 // 小写字母
	RAND_KIND_UPPER = 2 // 大写字母
	RAND_KIND_ALL   = 3 // 数字、大小写字母
)

// 随机字符串
func Rand(size int, kind int) []byte { // {{{
	ikind, kinds, result := kind, [][]int{[]int{10, 48}, []int{26, 97}, []int{26, 65}}, make([]byte, size)
	is_all := kind > 2 || kind < 0
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < size; i++ {
		if is_all { // random ikind
			ikind = rand.Intn(3)
		}
		scope, base := kinds[ikind][0], kinds[ikind][1]
		result[i] = uint8(base + rand.Intn(scope))
	}
	return result
} // }}}

func RandStr(size int, kind ...int) string { // {{{
	k := RAND_KIND_ALL
	if len(kind) > 0 {
		k = kind[0]
	}
	return string(Rand(size, k))
} // }}}

func MD5(str string) string { // {{{
	h := md5.New()
	h.Write([]byte(str))

	return hex.EncodeToString(h.Sum(nil))
} // }}}

func Sha1(str string) string { // {{{
	h := sha1.New()
	h.Write([]byte(str))

	return hex.EncodeToString(h.Sum(nil))
} // }}}

func Crc32(str string) int { // {{{
	ieee := crc32.NewIEEE()
	io.WriteString(ieee, str)
	return int(ieee.Sum32())
} // }}}

//拼接字符串
func Concat(str ...string) string { // {{{
	buf := bytes.NewBufferString("")
	for _, val := range str {
		buf.WriteString(val)
	}

	return buf.String()
} // }}}

//unix时间戳
func Now() int { // {{{
	return int(time.Now().Unix())
} // }}}

func getLoc() *time.Location { // {{{
	if TIME_ZONE == "Local" {
		return time.Local
	}

	if TIME_ZONE == "UTC" {
		return time.UTC
	}

	loc, err := time.LoadLocation(TIME_ZONE)
	if nil != err {
		panic(err)
	}

	return loc
} // }}}

//返回2013-01-20 格式的日期, 可以指定时间戳，默认当前时间
func Date(times ...int) string { // {{{
	var t time.Time
	if len(times) > 0 && times[0] > 0 {
		t = time.Unix(int64(times[0]), 0)
	} else {
		t = time.Now()
	}

	loc := getLoc()
	return t.In(loc).Format("2006-01-02")
} // }}}

//返回2013-01-20 10:20:00 格式的时间, 可以指定时间戳，默认当前时间
func DateTime(times ...int) string { // {{{
	var t time.Time
	if len(times) > 0 && times[0] > 0 {
		t = time.Unix(int64(times[0]), 0)
	} else {
		t = time.Now()
	}

	loc := getLoc()
	return t.In(loc).Format("2006-01-02 15:04:05")
} // }}}

//日期时间字符串转化为时间戳
func StrToTime(datetime string) int { // {{{
	loc := getLoc()
	t, _ := time.ParseInLocation("2006-01-02 15:04:05", datetime, loc)
	return int(t.Unix())
} // }}}

//生成时间戳
//参数：小时,分,秒,月,日,年
func MkTime(t ...int) int { // {{{
	var M time.Month
	loc := getLoc()
	h, m, s, d, y := 0, 0, 0, 0, 0

	l := len(t)

	if l > 0 {
		h = t[0]
	}

	if l > 1 {
		m = t[1]
	}

	if l > 2 {
		s = t[2]
	}

	if l > 3 {
		M = time.Month(t[3])
	}

	if l > 4 {
		d = t[4]
	}

	if l > 5 {
		y = t[5]
	} else {
		tn := time.Now().In(loc)
		y = tn.Year()
		if l < 5 {
			d = tn.Day()
		}
		if l < 4 {
			M = tn.Month()
		}
		if l < 3 {
			s = tn.Second()
		}
		if l < 2 {
			m = tn.Minute()
		}
		if l < 1 {
			h = tn.Hour()
		}
	}

	td := time.Date(y, M, d, h, m, s, 0, loc)
	return int(td.Unix())
} // }}}

//从start_time开始的消耗时间, 单位毫秒
func Cost(start_time time.Time) int { //start_time=time.Now()
	return int(time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000)
}

//强制转换为bool
func AsBool(b interface{}, def ...bool) bool { // {{{
	if val, ok := b.(bool); ok {
		return val
	}

	val, ok := b.(string)
	if !ok {
		val = fmt.Sprint(b)
	}

	ret, err := strconv.ParseBool(val)
	if nil != err {
		if len(def) > 0 {
			ret = def[0]
		}
	}

	return ret
} // }}}

//强制转换为int
func AsInt(num interface{}, def ...int) int { // {{{
	if val, ok := num.(int); ok {
		return val
	}

	if val, ok := num.(float64); ok {
		return int(val)
	}

	if val, ok := num.(float32); ok {
		return int(val)
	}

	val, ok := num.(string)
	if !ok {
		val = fmt.Sprint(num)
	}

	numint := ToInt(val)
	if numint == 0 && len(def) > 0 {
		numint = def[0]
	}

	return numint
} // }}}

//强制转换为string
func AsString(str interface{}, def ...string) string { // {{{
	newstr := ""
	if nil != str {
		newstr = fmt.Sprint(str)
	}

	if newstr == "" && len(def) > 0 {
		newstr = def[0]
	}

	return newstr
} // }}}

//string转换为int
func ToInt(num string) int { // {{{
	numint, err := strconv.Atoi(num)
	if nil != err {
		return 0
	}
	return numint
} // }}}

//string转换为float64
func ToFloat(num string, size ...int) float64 { // {{{
	bitsize := 64
	if len(size) > 0 && size[0] > 0 {
		bitsize = size[0]
	}

	numfloat, err := strconv.ParseFloat(num, bitsize)
	if nil != err {
		return 0
	}
	return numfloat
} // }}}

//int转换为string
func ToString(num int) string { // {{{
	return strconv.Itoa(num)
} // }}}

//string的切片转换为int切片
func ToIntArray(nums []string) []int { // {{{
	intnums := []int{}

	for _, v := range nums {
		intnums = append(intnums, ToInt(v))
	}

	return intnums
} // }}}

//int的切片转换为string切片
func ToStringArray(nums []int) []string { // {{{
	strnums := []string{}

	for _, v := range nums {
		strnums = append(strnums, strconv.Itoa(v))
	}

	return strnums
} // }}}

//[]int, []string, []interface, 连接成字符串
func Join(strs interface{}, seps ...string) string { // {{{
	sep := ","
	if len(seps) > 0 {
		sep = seps[0]
	}

	newstrs := []string{}
	switch val := strs.(type) {
	case []int:
		for _, v := range val {
			newstrs = append(newstrs, AsString(v))
		}
	case []string:
		for _, v := range val {
			newstrs = append(newstrs, AsString(v))
		}
	case []interface{}:
		for _, v := range val {
			newstrs = append(newstrs, AsString(v))
		}
	}

	return strings.Join(newstrs, sep)
} // }}}

//判断ver是否大于oldver
func VersionCompare(ver, oldver string) bool { // {{{
	vs1 := strings.Split(ver, ".")
	vs2 := strings.Split(oldver, ".")
	len1 := len(vs1)
	len2 := len(vs2)

	l := len1
	if len1 < len2 {
		l = len2
	}

	for i := 0; i < l; i++ {
		vs1 = append(vs1, "")
		vs2 = append(vs2, "")

		v1 := ToInt(vs1[i])
		v2 := ToInt(vs2[i])
		if v1 > v2 {
			return true
		} else if v1 < v2 {
			return false
		}
	}

	return false
} // }}}

//x.x.x.x格式的IP转换为数字
func Ip2long(ipstr string) (ip int) { // {{{
	r := `^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})`
	reg, err := regexp.Compile(r)
	if err != nil {
		return
	}
	ips := reg.FindStringSubmatch(ipstr)
	if ips == nil {
		return
	}

	ip1, _ := strconv.Atoi(ips[1])
	ip2, _ := strconv.Atoi(ips[2])
	ip3, _ := strconv.Atoi(ips[3])
	ip4, _ := strconv.Atoi(ips[4])

	if ip1 > 255 || ip2 > 255 || ip3 > 255 || ip4 > 255 {
		return
	}

	ip += int(ip1 * 0x1000000)
	ip += int(ip2 * 0x10000)
	ip += int(ip3 * 0x100)
	ip += int(ip4)

	return
} // }}}

//数字格式的IP转换为x.x.x.x
func Long2ip(ip int) string { // {{{
	return fmt.Sprintf("%d.%d.%d.%d", ip>>24, ip<<8>>24, ip<<16>>24, ip<<24>>24)
} // }}}
