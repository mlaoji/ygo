package lib

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var TokenSecret = ""

const (
	TOKEN_SECRET = "U4rnhBc9yruM"
	TOKEN_TTL    = 604800     //86400 * 7
	TOKEN_VER    = "1"        //固定1位, token版本，更新secret时修改
	TOKEN_MAXID  = 4294967295 //uint32 最大值
)

//国定1位，token权限范围，目前没实际意义,留为扩展
var TOKEN_SCOPE = map[string]string{
	"*":   "0",
	"app": "1",
	"web": "2",
}

//Make {{{
//26位header + 8位随机 + 4位签名(最大38位，最小30位)
//ttl 表示生成token 的时间戳，过期时间在校验时计算
func MakeToken(uid int, options ...string) (token string, tsign string) {
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

	token = makeToken(uid, salt, ttl, scope, random)
	tsign = makeToken(TOKEN_MAXID-uid, random, ttl, scope, salt) //交换salt 和 random位置
	return
} //}}}

//{{{
func makeToken(uid int, salt string, ttl int, scope, random string) string {
	userid := make([]byte, 4)
	expire := make([]byte, 4)

	binary.BigEndian.PutUint32(userid, uint32(uid))
	binary.BigEndian.PutUint32(expire, uint32(ttl))

	header := fmt.Sprintf("%x%x%s%s%s", userid, expire, scope, TOKEN_VER, salt)

	/*
		fmt.Printf("header:%s\n", header)
		fmt.Printf("uid:%x\n", string(userid))
		fmt.Printf("expire:%x\n", string(expire))
		fmt.Printf("scope:%x\n", string(scope))
		fmt.Printf("salt:%s\n", salt)
		fmt.Printf("r:%s\n", random)
	*/

	sign := getSign(Concat(string(userid), string(expire), string(scope), TOKEN_VER, salt, random))
	token := Concat(header, random, sign)

	return strings.TrimLeft(token, "0")
}

// }}}

//GetTokenInfo {{{
func GetTokenInfo(token string) (ret map[string]interface{}, err bool) {
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
	salt := string([]byte(header)[hlen-8:])
	ver := string([]byte(header)[hlen-9 : hlen-8])
	scope := string([]byte(header)[hlen-10 : hlen-9])

	_expire, _ := strconv.ParseUint(string([]byte(header)[hlen-18:hlen-10]), 16, 32)
	_userid, _ := strconv.ParseUint(string([]byte(header)[0:hlen-18]), 16, 32)

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

	checksign := getSign(Concat(string(userid), string(expire), scope, ver, salt, random_str))

	/*
		fmt.Printf("%#v", checksign)
		fmt.Printf("%#v", sign)
	*/

	if checksign != sign || _userid == 0 || ver == "" || scope == "" || random_str == "" || salt == "" {
		err = true
		return
	}

	ret["userid"] = int(_userid)
	ret["time"] = int(_expire)
	ret["salt"] = salt
	ret["random"] = random_str
	ret["ver"] = ver

	for k, v := range TOKEN_SCOPE {
		if v == scope {
			ret["scope"] = k
			break
		}
	}

	return
} // }}}

//CheckToken {{{//加强校验时，需要传salt
func CheckToken(token string, userid int, salt string, ttls ...int) bool {
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

//CheckSign{{{
func CheckSign(sign string, userid int, ttls ...int) bool {
	tokeninfo, err := GetTokenInfo(sign)
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

	if tokeninfo["userid"].(int) != TOKEN_MAXID-userid {
		return false
	}

	return true
} // }}}

//GetTokenBySign{{{
func GetTokenBySign(sign string) (token string, err bool) {
	tokeninfo, _err := GetTokenInfo(sign)
	if _err {
		err = true
		return
	}

	uid := tokeninfo["userid"].(int)
	salt := tokeninfo["random"].(string)
	random := tokeninfo["salt"].(string)
	ttl := tokeninfo["time"].(int)
	scope := TOKEN_SCOPE[tokeninfo["scope"].(string)]

	token = makeToken(TOKEN_MAXID-uid, salt, ttl, scope, random)
	return
} // }}}

func getSign(str string) string {
	secret := TokenSecret
	if "" == secret {
		secret = TOKEN_SECRET
	}

	sign := MD5(str + secret)
	return string([]byte(sign)[5:9])
}

func getSalt(str string) string {
	return string([]byte(Sha1(str))[0:8])
}
