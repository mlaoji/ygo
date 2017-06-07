package lib

import (
	"time"
)

var (
	SIGN_SECRET = "8p43Re$%9aB72@0Y*1y"
	SIGN_EXPIRE = 3600
	SIGN_PARAMS = []string{"deviceid", "platform", "rand", "time", "userid", "version"}
)

func CheckGuid(params map[string][]string, sign_secret string, sign_field []string) bool {
	guid := ""
	if vs := params["guid"]; len(vs) > 0 {
		guid = vs[0]
	} else {
		return false
	}

	if len(sign_field) == 0 {
		sign_field = SIGN_PARAMS
	}

	str := ""
	for _, v := range sign_field {
		if vs := params[v]; len(vs) > 0 {
			str = str + v + "=" + vs[0]
		}
	}

	if sign_secret == "" {
		sign_secret = SIGN_SECRET
	}

	return guid == MD5(str+sign_secret)
}

func CheckRpcAuth(appid, secret string) bool {
	conf := Conf.Get("app_"+appid, "rpc_auth")
	return secret == conf
}

func CheckFlood(str string) bool {
	key := "fd_" + str

	if _, found := LocalCache.Get(key); !found {
		LocalCache.Set(key, 1, time.Duration(SIGN_EXPIRE)*time.Second)
		return true
	}

	return false
}
