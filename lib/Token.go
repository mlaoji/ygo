package lib

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var TokenSecret = ""

const (
	TOKEN_SECRET = "U4rnhBc9yruM"
	TOKEN_TTL    = 604800 //86400 * 7
	TOKEN_VER    = "1"    //固定1位, token版本，更新secret时修改
)

//国定1位，token权限范围，目前没实际意义,留为扩展
var TOKEN_SCOPE = map[string]string{
	"*":   "0",
	"app": "1",
	"web": "2",
}

//[18-26]位header + 8位随机 + 4位签名(最大39位，最小31位)
//ttl 表示生成token 的时间戳，过期时间在校验时计算
//token 和 sign 是成对的，token 用于身份授权认证，sign 级别稍低，仅用于身份证明，不用于授权(比如多个终端间用户状态流转，可以颁发sign, 不用担心中间过程中token被(不信任的终端)恶意保存)
//token 和 sign 生成算法一致，主要区分在于 signflag参数， [0-4] 则为token, [5-9] 则为sign, 且同一对token和sign的signflag参数之和等于9
func MakeToken(uid int, options ...string) (token string, tsign string) { //{{{
	saltstr := ""
	scopestr := ""

	l := len(options)
	if l > 0 { //盐
		saltstr = options[0]
	}
	if l > 1 { //scope
		scopestr = options[1]
	}

	if "" == scopestr || "" == TOKEN_SCOPE[scopestr] {
		scopestr = "*"
	}

	ttl := int(time.Now().Unix())
	salt := getSalt(saltstr) // 8位
	scope := TOKEN_SCOPE[scopestr]
	random := string(Rand(8, RAND_KIND_ALL)) // 8位

	signflag := rand.Intn(5)

	token = makeToken(uid, salt, ttl, scope, random, signflag)
	tsign = makeToken(uid, salt, ttl, scope, random, 9-signflag)
	return
} //}}}

//issign:Y/N
func makeToken(uid int, salt string, ttl int, scope, random string, signflag int) string { //{{{
	userid := make([]byte, 4)
	expire := make([]byte, 4)

	binary.BigEndian.PutUint32(userid, uint32(uid))
	binary.BigEndian.PutUint32(expire, uint32(ttl))

	header := fmt.Sprintf("%x%x%s%s%s%d", userid, expire, salt, scope, TOKEN_VER, signflag)

	/*
		fmt.Printf("header:%s\n", header)
		fmt.Printf("uid:%x\n", string(userid))
		fmt.Printf("expire:%x\n", string(expire))
		fmt.Printf("scope:%x\n", string(scope))
		fmt.Printf("salt:%s\n", salt)
		fmt.Printf("r:%s\n", random)
	*/

	sign := getSign(Concat(ToString(uid), ToString(ttl), salt, scope, TOKEN_VER, ToString(signflag), random))
	token := Concat(header, random, sign)

	return strings.TrimLeft(token, "0")
}

// }}}

func getTokenInfo(token string) (ret map[string]interface{}, err bool) { //{{{
	ret = map[string]interface{}{}
	err = false

	header := ""
	random_str := ""
	sign := ""

	if hlen := len(token); hlen > 12 {
		header = string([]byte(token)[0 : hlen-12]) //8位随机 4位签名
		random_str = string([]byte(token)[hlen-12 : hlen-4])
		sign = string([]byte(token)[hlen-4:])
	} else {
		err = true
		return
	}

	hlen := len(header)
	if hlen < 19 {
		err = true
		return
	}

	signflag := string([]byte(header)[hlen-1:])
	ver := string([]byte(header)[hlen-2 : hlen-1])
	scope := string([]byte(header)[hlen-3 : hlen-2])
	salt := string([]byte(header)[hlen-11 : hlen-3])

	_expire, _ := strconv.ParseUint(string([]byte(header)[hlen-19:hlen-11]), 16, 32)
	_userid, _ := strconv.ParseUint(string([]byte(header)[0:hlen-19]), 16, 32)

	userid := make([]byte, 4)
	expire := make([]byte, 4)

	binary.BigEndian.PutUint32(userid, uint32(_userid))
	binary.BigEndian.PutUint32(expire, uint32(_expire))

	/*
		fmt.Printf("header:%s\n", header)
		fmt.Printf("uid:%x\n", string(userid))
		fmt.Printf("expire:%x\n", string(expire))
		fmt.Printf("scope:%x\n", string(scope))
		fmt.Printf("salt:%s\n", salt)
		fmt.Printf("ver:%s\n", ver)
		fmt.Printf("r:%s\n", random_str)
	*/

	checksign := getSign(Concat(AsString(_userid), AsString(_expire), salt, scope, ver, signflag, random_str))

	/*
		fmt.Printf("%#v", checksign)
		fmt.Printf("%#v", sign)
	*/

	if checksign != sign || _userid == 0 || ver == "" || signflag == "" || scope == "" || random_str == "" || salt == "" {
		err = true
		return
	}

	ret["userid"] = int(_userid)
	ret["time"] = int(_expire)
	ret["salt"] = salt
	ret["random"] = random_str
	ret["ver"] = ver
	ret["signflag"] = ToInt(signflag)

	for k, v := range TOKEN_SCOPE {
		if v == scope {
			ret["scope"] = k
			break
		}
	}

	return
} // }}}

func GetTokenInfo(token string, ttls ...int) (ret map[string]interface{}, err bool) { //{{{
	ret, err = getTokenInfo(token)
	if err {
		return
	}

	if ret["signflag"].(int) > 4 {
		err = true
		return
	}

	ttl := TOKEN_TTL
	if len(ttls) > 0 && ttls[0] > 0 {
		ttl = ttls[0]
	}

	if ret["time"].(int)+ttl < int(time.Now().Unix()) {
		err = true
		return
	}

	return
} // }}}

func GetSignInfo(sign string, ttls ...int) (ret map[string]interface{}, err bool) { //{{{
	ret, err = getTokenInfo(sign)
	if err {
		return
	}

	if ret["signflag"].(int) < 5 {
		err = true
		return
	}

	ttl := TOKEN_TTL
	if len(ttls) > 0 && ttls[0] > 0 {
		ttl = ttls[0]
	}

	if ret["time"].(int)+ttl < int(time.Now().Unix()) {
		err = true
		return
	}

	return
} // }}}

//加强校验时，需要传salt
func CheckToken(token string, userid int, salt string, ttls ...int) bool { //{{{
	tokeninfo, err := GetTokenInfo(token)
	if err {
		return false
	}

	ttl := TOKEN_TTL
	if len(ttls) > 0 && ttls[0] > 0 {
		ttl = ttls[0]
	}

	if tokeninfo["time"].(int)+ttl < int(time.Now().Unix()) {
		return false
	}

	if tokeninfo["userid"].(int) != userid || userid == 0 {
		return false
	}

	if "" != salt && tokeninfo["salt"].(string) != getSalt(salt) {
		return false
	}

	return true
} // }}}

func CheckSign(sign string, userid int, ttls ...int) bool { //{{{
	tokeninfo, err := GetSignInfo(sign)
	if err {
		return false
	}

	ttl := TOKEN_TTL
	if len(ttls) > 0 && ttls[0] > 0 {
		ttl = ttls[0]
	}

	if tokeninfo["time"].(int)+ttl < int(time.Now().Unix()) {
		return false
	}

	if tokeninfo["userid"].(int) != userid {
		return false
	}

	return true
} // }}}

//GetTokenBySign{{{
func GetTokenBySign(sign string) (token string, err bool) {
	tokeninfo, _err := GetSignInfo(sign)
	if _err {
		err = true
		return
	}

	uid := tokeninfo["userid"].(int)
	salt := tokeninfo["salt"].(string)
	random := tokeninfo["random"].(string)
	ttl := tokeninfo["time"].(int)
	signflag := tokeninfo["signflag"].(int)
	scope := TOKEN_SCOPE[tokeninfo["scope"].(string)]

	token = makeToken(uid, salt, ttl, scope, random, 9-signflag)
	return
} // }}}

func getSign(str string) string { // {{{
	secret := TokenSecret
	if "" == secret {
		secret = TOKEN_SECRET
	}

	sign := MD5(str + secret)
	return string([]byte(sign)[5:9])
} // }}}

func getSalt(str string) string { // {{{
	return string([]byte(Sha1(str))[0:8])
} // }}}
