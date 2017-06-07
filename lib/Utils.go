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
	"strconv"
	"strings"
	"time"
)

func Interceptor(guard bool, errmsg *Error, fmts ...interface{}) {
	if !guard {
		panic(&Error{errmsg.Code, fmt.Sprintf(errmsg.Msg, fmts...)})
	}
}

/*
*获取本机ip
 */
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
		return int(val)
	default:
		return r
	}
} // }}}

func InArray(val interface{}, arr interface{}) bool { // {{{
	switch vals := arr.(type) {
	case map[string]string:
		for _, v := range vals {
			if v == val.(string) {
				return true
			}
		}
	case map[int]int:
		for _, v := range vals {
			if v == val.(int) {
				return true
			}
		}
	case map[int]interface{}:
		for _, v := range vals {
			if v == val.(int) {
				return true
			}
		}
	case map[string]interface{}:
		for _, v := range vals {
			if v == val.(string) {
				return true
			}
		}
	case map[interface{}]interface{}:
		for _, v := range vals {
			if v == val {
				return true
			}
		}
	case map[int]string:
		for _, v := range vals {
			if v == val.(string) {
				return true
			}
		}
	case map[string]int:
		for _, v := range vals {
			if v == val.(int) {
				return true
			}
		}
	case []byte:
		for _, v := range vals {
			if v == val.(byte) {
				return true
			}
		}
	case []int:
		for _, v := range vals {
			if v == val.(int) {
				return true
			}
		}
	case []string:
		for _, v := range vals {
			if v == val.(string) {
				return true
			}
		}
	case []interface{}:
		for _, v := range vals {
			if v == val {
				return true
			}
		}
	default:
		return false
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

func Concat(str ...string) string { // {{{
	buf := bytes.NewBufferString("")
	for _, val := range str {
		buf.WriteString(val)
	}

	return buf.String()
} // }}}

func Now() int {
	return int(time.Now().Unix())
}

func Date(times ...int) string { // {{{
	if len(times) > 0 && times[0] > 0 {
		return time.Unix(int64(times[0]), 0).Format("2006-01-02")
	}

	return time.Now().Format("2006-01-02")
} // }}}

func DateTime(times ...int) string { // {{{
	if len(times) > 0 && times[0] > 0 {
		return time.Unix(int64(times[0]), 0).Format("2006-01-02 15:04:05")
	}

	return time.Now().Format("2006-01-02 15:04:05")
} // }}}

func StrToTime(datetime string) int {
	t, _ := time.ParseInLocation("2006-01-02 15:04:05", datetime, time.Local)
	return int(t.Unix())
}

//参数：小时,分,秒,月,日,年
func MkTime(t ...int) int { // {{{
	var M time.Month
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
		tn := time.Now()
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

	td := time.Date(y, M, d, h, m, s, 0, time.Local)
	return int(td.Unix())
} // }}}

func Cost(start_time time.Time) int { //start_time=time.Now()
	return int(time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000)
}

func AsInt(num interface{}) int { // {{{
	if val, ok := num.(int); ok {
		return val
	}

	val, ok := num.(string)
	if !ok {
		val = fmt.Sprint(num)
	}

	return Toint(val)
} // }}}

func Toint(num string) int {
	numint, err := strconv.Atoi(num)
	if nil != err {
		return 0
	}
	return numint
}

func Tostring(num int) string {
	return strconv.Itoa(num)
}

func TointArray(nums []string) []int { // {{{
	intnums := []int{}

	for _, v := range nums {
		intnums = append(intnums, Toint(v))
	}

	return intnums
} // }}}

func TostringArray(nums []int) []string { // {{{
	strnums := []string{}

	for _, v := range nums {
		strnums = append(strnums, strconv.Itoa(v))
	}

	return strnums
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

		v1 := Toint(vs1[i])
		v2 := Toint(vs2[i])
		if v1 > v2 {
			return true
		} else if v1 < v2 {
			return false
		}
	}

	return false
} // }}}
