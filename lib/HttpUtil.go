package lib

import (
	"net/http"
	"strings"
	"time"
)

func GetIp(r *http.Request) string { // {{{
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" || ip == "127.0.0.1" {
		ip = r.Header.Get("X-Real-IP")
		if ip == "" {
			ip = r.Header.Get("Host")
			if ip == "" {
				ip = r.RemoteAddr
			}
		}
	} else {
		//X-Forwarded-For 的格式 client1, proxy1, proxy2
		ips := strings.Split(ip, ",")
		if len(ips) > 0 {
			ip = ips[0]
		}
	}

	//去除端口号
	ips := strings.Split(ip, ":")
	if len(ips) > 0 {
		ip = ips[0]
	}

	return ip
} // }}}

func GetCookie(r *http.Request, key string) string { // {{{
	cookie, err := r.Cookie(key)
	if err == nil {
		return cookie.Value
	}

	return ""
} // }}}

//lifetime<0时删除cookie
//options: domain,secure,httponly,path
func SetCookie(rw http.ResponseWriter, key, val string, lifetime int, options ...interface{}) { // {{{
	domain := ""
	secure := false
	httponly := false
	path := "/"

	if len(options) > 0 {
		domain = AsString(options[0])
	}

	if len(options) > 1 {
		secure = AsBool(options[1])
	}

	if len(options) > 2 {
		httponly = AsBool(options[2])
	}

	if len(options) > 3 {
		path = AsString(options[3])
	}

	cookie := &http.Cookie{
		Name:   key,
		Value:  val,
		Path:   path,
		Domain: domain,
		Secure: secure,
		//SameSite: http.SameSiteNoneMode,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: httponly,
		MaxAge:   lifetime,
		Expires:  time.Now().Add(time.Second * time.Duration(lifetime)),
	}
	http.SetCookie(rw, cookie)
} // }}}
