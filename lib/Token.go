package lib

import (
	"encoding/binary"
	"strconv"
	"time"
)

var TokenSecret = ""

//用户系统中token.go的代码片段,只提供校验方法
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
