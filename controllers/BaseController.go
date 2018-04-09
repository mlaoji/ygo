package controllers

import (
	"bytes"
	"fmt"
	"github.com/mlaoji/ygo/lib"
	"html/template"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type iRequest struct {
	Form url.Values
}

func (this *iRequest) FormValue(key string) string {
	if vs := this.Form[key]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}

var (
	DEBUG_OPEN = false
)

const (
	HTTP_MODE = iota
	RPC_MODE
	CLI_MODE
)

type BaseController struct {
	RW         http.ResponseWriter
	R          *http.Request
	RBody      []byte
	IR         *iRequest
	UserId     int
	Guid       string
	startTime  time.Time
	mode       int
	rpcContent string
	uri        string
	Controller string
	Action     string
	Debug      bool
}

func (this *BaseController) Prepare(rw http.ResponseWriter, r *http.Request, requestUri string) { // {{{
	this.RW = rw
	this.R = r

	this.RBody, _ = this.getRequestBody(r)

	r.ParseMultipartForm(32 << 20) //32M

	this.prepare(r.Form, HTTP_MODE, requestUri)

	//校验token是否有效
	//优先使用cookie中的token 和 userid, 不存在则取Form参数

	userId := this.GetInt("userid")
	ck_userId := this.GetCookie("userid")
	if len(ck_userId) > 0 {
		userId = lib.ToInt(ck_userId)
	}

	auth_conf := lib.Conf.GetAll("auth_conf")
	token_ttl := lib.ToInt(auth_conf["token_ttl"])
	token_secret := auth_conf["token_secret"]
	token_key := auth_conf["token_key"]
	access_redirect := auth_conf["access_redirect"] //校验token失败时跳转

	if 0 == len(token_key) {
		token_key = "token"
	}

	token := this.GetString(token_key)
	ck_token := this.GetCookie(token_key)
	if len(ck_token) > 0 {
		token = ck_token
	}

	if len(token_secret) > 0 {
		lib.TokenSecret = token_secret
	}

	logined := lib.CheckToken(token, userId, "", token_ttl)

	if logined {
		this.UserId = userId
	}

	uri := this.uri
	//需要校验token的接口在配置中定义
	if auth_conf["check_token"] == "1" && -1 == strings.Index(uri, "monitor/") { //默认关闭
		api_need_check_token_except := auth_conf["token_api_except"]
		if "" == api_need_check_token_except || !strings.Contains(","+strings.ToLower(api_need_check_token_except)+",", ","+uri+",") {
			api_need_check_token := "," + strings.ToLower(auth_conf["token_api"]) + ","
			if api_need_check_token == ",," || strings.Contains(api_need_check_token, ",*,") || strings.Contains(api_need_check_token, ","+uri+",") {
				if len(access_redirect) > 0 && !logined {
					if strings.Contains(access_redirect, "?") {
						access_redirect += "&"
					} else {
						access_redirect += "?"
					}

					access_redirect += "http_referer=" + url.QueryEscape(this.GetHeader("Referer"))
					access_redirect += "&request_controller=" + this.Controller
					access_redirect += "&request_action=" + this.Action
					access_redirect += "&request_uri=" + url.QueryEscape(this.R.URL.String())
					this.Redirect(access_redirect) //如果设置跳转URL，则直接跳转
				}

				lib.Interceptor(logined, lib.ERR_TOKEN, userId)
			}
		}
	}
	//需要校验sign的接口在配置中定义
	if auth_conf["check_sign"] == "1" && -1 == strings.Index(uri, "monitor/") { //默认关闭
		api_need_check_sign_except := auth_conf["sign_api_except"]
		if "" == api_need_check_sign_except || !strings.Contains(","+strings.ToLower(api_need_check_sign_except)+",", ","+uri+",") {
			api_need_check_sign := "," + strings.ToLower(auth_conf["sign_api"]) + ","
			if api_need_check_sign == ",," || strings.Contains(api_need_check_sign, ",*,") || strings.Contains(api_need_check_sign, ","+uri+",") {
				sign := this.GetCookie("sign")
				ttl := lib.ToInt(auth_conf["sign_ttl"])

				token_secret := auth_conf["token_secret"]
				if len(token_secret) > 0 {
					lib.TokenSecret = token_secret
				}

				lib.Interceptor(lib.CheckSign(sign, userId, ttl), lib.ERR_SIGN, userId)
			}
		}
	}

	secret := auth_conf["guid_secret"]
	//if auth_conf["check_guid"] != "0" && -1 == strings.Index(uri, "monitor/") {
	if auth_conf["check_guid"] == "1" && -1 == strings.Index(uri, "monitor/") {
		api_need_check_guid_except := auth_conf["guid_api_except"]
		if "" == api_need_check_guid_except || !strings.Contains(","+strings.ToLower(api_need_check_guid_except)+",", ","+uri+",") {
			api_need_check_guid := "," + strings.ToLower(auth_conf["guid_api"]) + ","
			if api_need_check_guid == ",," || strings.Contains(api_need_check_guid, ",*,") || strings.Contains(api_need_check_guid, ","+uri+",") {
				guid := this.GetString("guid")
				field := lib.Conf.GetSlice("guid_field", ",", "auth_conf")

				lib.Interceptor(lib.CheckGuid(r.Form, secret, field), lib.ERR_GUID, guid)
				lib.Interceptor(lib.CheckFlood(guid), lib.ERR_FLOOD, guid)
			}
		}
	}

	freq_conf := lib.Conf.GetAll("api_freq_conf")
	if freq_conf["check_freq"] == "1" { //默认关闭
		mtd := this.uri
		mtd_cnf := lib.Conf.GetSlice(mtd, ",", "api_freq_conf")
		for _, freq_rule := range mtd_cnf {
			whitelist := lib.Conf.GetSlice(freq_rule+"_whitelist", ",", "api_freq_conf")
			blacklist := lib.Conf.GetSlice(freq_rule+"_blacklist", ",", "api_freq_conf")
			freq_rule_conf := lib.Conf.GetSlice(freq_rule, ",", "api_freq_conf")
			if len(freq_rule_conf) > 2 {
				freq := lib.NewRestrict(freq_rule, lib.ToInt(freq_rule_conf[0]), lib.ToInt(freq_rule_conf[1])) //规则,频率,周期(秒)
				if len(whitelist) > 0 {
					freq.AddWhitelist(whitelist)
				}
				if len(blacklist) > 0 {
					freq.AddBlacklist(blacklist)
				}
				check_key := freq_rule_conf[2]
				check_val := ""
				if "ip" == check_key {
					check_val = this.GetIp()
				} else {
					check_val = this.GetString(check_key)
				}

				if "" != freq_conf["use_redis_conf"] {
					freq.UseRedis(freq_conf["use_redis_conf"])
				}

				lib.Interceptor(!freq.Add(check_val), lib.ERR_FREQ, check_key)
			}
		}
	}
} // }}}

func (this *BaseController) getRequestBody(r *http.Request) ([]byte, error) { // {{{
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body = ioutil.NopCloser(bytes.NewReader(buf))
	return buf, nil
} // }}}

func (this *BaseController) PrepareRpc(r url.Values, requestUri string) { // {{{
	this.prepare(r, RPC_MODE, requestUri)

	appid := this.GetString("appid")
	secret := this.GetString("secret")
	lib.Interceptor(lib.CheckRpcAuth(appid, secret), lib.ERR_RPCAUTH, appid)
} // }}}

func (this *BaseController) PrepareCli(r url.Values, requestUri string) { // {{{
	this.prepare(r, CLI_MODE, requestUri)
} // }}}

func (this *BaseController) prepare(r url.Values, mode int, requestUri string) { // {{{
	this.startTime = time.Now()
	this.Debug = DEBUG_OPEN
	this.IR = &iRequest{r}
	this.mode = mode
	this.uri = strings.ToLower(requestUri)
	uris := strings.Split(this.uri, "/")
	this.Controller = uris[0]
	this.Action = uris[1]
} // }}}

//以下 GetX 方法用于获取参数
func (this *BaseController) GetCookie(key string) string { // {{{
	cookie, err := this.R.Cookie(key)
	if err == nil {
		return cookie.Value
	}

	return ""
} // }}}

func (this *BaseController) GetHeader(key string) string { // {{{
	return this.R.Header.Get(key)
} // }}}

func (this *BaseController) _getFormValue(key string) string { // {{{
	val := this.IR.FormValue(key)
	return strings.Trim(val, " \r\t\v")
} // }}}

func (this *BaseController) GetString(key string, defaultValues ...string) string { // {{{
	defaultValue := ""

	if len(defaultValues) > 0 {
		defaultValue = defaultValues[0]
	}

	ret := this._getFormValue(key)
	if ret == "" {
		ret = defaultValue
	}
	return ret
} // }}}

func (this *BaseController) GetSlice(key string, separators ...string) []string { //{{{
	separator := ","
	if len(separators) > 0 {
		separator = separators[0]
	}

	value := this.GetString(key)
	if "" == value {
		return nil
	}

	slice := []string{}
	for _, part := range strings.Split(value, separator) {
		slice = append(slice, strings.Trim(part, " \r\t\v"))
	}
	return slice
} // }}}

func (this *BaseController) GetSliceInt(key string, separators ...string) []int { //{{{
	slice := this.GetSlice(key, separators...)

	if nil == slice {
		return nil
	}

	sliceint := []int{}
	for _, val := range slice {
		if val, err := strconv.Atoi(val); nil == err {
			sliceint = append(sliceint, val)
		}
	}

	return sliceint
} // }}}

func (this *BaseController) GetParams() map[string]string { // {{{
	if this.IR.Form == nil {
		return nil
	}

	params := map[string]string{}
	for k, v := range this.IR.Form {
		if len(v) > 0 {
			params[k] = strings.Trim(v[0], " \r\t\v")
		}
	}

	return params
} // }}}

func (this *BaseController) GetArray(key string) []string { // {{{
	if this.IR.Form == nil {
		return nil
	}
	vs := this.IR.Form[key]
	return vs
} // }}}

func (this *BaseController) GetInt(key string, defaultValues ...int) int { // {{{
	defaultValue := 0

	if len(defaultValues) > 0 {
		defaultValue = defaultValues[0]
	}

	ret, err := strconv.Atoi(this._getFormValue(key))
	if err != nil {
		ret = defaultValue
	}
	return ret
} // }}}

func (this *BaseController) GetInt64(key string, defaultValues ...int64) int64 { // {{{
	defaultValue := int64(0)

	if len(defaultValues) > 0 {
		defaultValue = defaultValues[0]
	}

	ret, err := strconv.ParseInt(this._getFormValue(key), 10, 64)
	if err != nil {
		ret = defaultValue
	}
	return ret
} // }}}

func (this *BaseController) GetBool(key string, defaultValues ...bool) bool { // {{{
	defaultValue := false

	if len(defaultValues) > 0 {
		defaultValue = defaultValues[0]
	}

	ret, err := strconv.ParseBool(this._getFormValue(key))
	if err != nil {
		ret = defaultValue
	}
	return ret
} // }}}

func (this *BaseController) GetFromJson(key string) interface{} { // {{{
	ret := this._getFormValue(key)
	if ret != "" {
		return lib.JsonDecode(ret)
	}
	return ret
} // }}}

func (this *BaseController) GetFile(key string) (multipart.File, *multipart.FileHeader, error) { // {{{
	return this.R.FormFile(key)
} // }}}

func (this *BaseController) GetIp() string { // {{{
	r := this.R

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

func (this *BaseController) SetCookie(key, val string, lifetime int) { // {{{
	cookie := &http.Cookie{
		Name:     key,
		Value:    val,
		Path:     "/",
		HttpOnly: false,
		MaxAge:   lifetime,
		Expires:  time.Now().Add(time.Second * time.Duration(lifetime)),
	}
	http.SetCookie(this.RW, cookie)
} // }}}

func (this *BaseController) UnsetCookie(key string) { // {{{
	cookie := &http.Cookie{
		Name:     key,
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		MaxAge:   0,
		Expires:  time.Now().AddDate(-1, 0, 0),
	}
	http.SetCookie(this.RW, cookie)
} // }}}

func (this *BaseController) SetHeader(key, val string) { // {{{
	this.RW.Header().Set(key, val)
} // }}}

func (this *BaseController) SetHeaders(headers http.Header) { // {{{
	this_header := this.RW.Header()
	for k, v := range headers {
		this_header.Set(k, v[0])
	}
} // }}}

//接口正常输出
func (this *BaseController) Render(data ...interface{}) { // {{{
	var retdata interface{}
	if len(data) > 0 {
		retdata = data[0]
	} else {
		retdata = make(map[string]interface{})
	}

	ret := map[string]interface{}{
		"code":    lib.ERR_SUC.GetCode(),
		"msg":     lib.ERR_SUC.GetMessage(),
		"time":    time.Now().Unix(),
		"consume": this.Cost(),
		"data":    retdata,
	}

	res := lib.JsonEncode(ret)
	lib.Logger.Access(this.genLog(), res)

	this.renderJson(res)
} // }}}

//接口异常输出，在HttpApiServer中回调
func (this *BaseController) RenderError(err interface{}, data ...interface{}) { // {{{
	var errno int
	var errmsg string
	var isbizerr bool

	switch errinfo := err.(type) {
	case string:
		errno = lib.ERR_SYSTEM.GetCode()
		errmsg = errinfo
	case *lib.Error:
		errno = errinfo.GetCode()
		errmsg = errinfo.GetMessage()
		isbizerr = true
	case *lib.Errorf:
		lang := this.GetString("lang")
		errno = errinfo.GetCode()
		errmsg = errinfo.GetMessage(lang)
		isbizerr = true
	case error:
		errno = lib.ERR_SYSTEM.GetCode()
		errmsg = errinfo.Error()
	default:
		errno = lib.ERR_SYSTEM.GetCode()
		errmsg = fmt.Sprint(errinfo)
	}

	if !isbizerr {
		debug_trace := debug.Stack()
		lib.Logger.Error(this.genLog(), errmsg, string(debug_trace))

		fmt.Println(errmsg)
		os.Stderr.Write(debug_trace)

		if 1 != lib.Conf.GetInt("dev_mode") {
			errmsg = lib.ERR_SYSTEM.GetMessage()
		}
	}

	var retdata interface{}
	if len(data) > 0 {
		retdata = data[0]
	} else {
		retdata = make(map[string]interface{})
	}

	ret := map[string]interface{}{
		"code":    errno,
		"msg":     errmsg,
		"time":    time.Now().Unix(),
		"consume": this.Cost(),
		"data":    retdata, //错误时，也可附带一些数据
	}

	res := lib.JsonEncode(ret)
	lib.Logger.Warn(this.genLog(), res)

	this.renderJson(res)
} // }}}

//输出字符串
func (this *BaseController) RenderString(data string) { // {{{
	this.writeToWriter([]byte(data))
	lib.Logger.Access(this.genLog(), data)
} // }}}

//输出HTTP状态码
func (this *BaseController) RenderStatus(code int) { // {{{
	this.RW.WriteHeader(code)
} // }}}

//重定向URL
func (this *BaseController) Redirect(url string, codes ...int) { // {{{
	code := http.StatusFound //302
	if len(codes) > 0 {
		code = codes[0]
	}
	http.Redirect(this.RW, this.R, url, code)
} // }}}

func (this *BaseController) renderJson(data string) { // {{{
	if this.mode == RPC_MODE {
		this.rpcContent = data
	} else if this.mode == CLI_MODE {
		fmt.Println(data)
	} else {
		this.RW.Header().Set("Content-Type", "application/json;charset=UTF-8")
		this.writeToWriter([]byte(data))
	}
} // }}}

//获取日志内容
func (this *BaseController) genLog() map[string]interface{} { // {{{
	ret := make(map[string]interface{})

	if HTTP_MODE == this.mode && nil != this.R {
		//访问ip
		ret["ip"] = this.GetIp()
		//请求路径
		ret["uri"] = this.R.URL

		if this.R.Method == "POST" {
			ret["post"] = this.R.PostForm
		}

		ret["ua"] = this.R.UserAgent()
	}

	if RPC_MODE == this.mode && nil != this.IR {
		delete(this.IR.Form, "secret")
		ret["uri"] = this.uri
		ret["post"] = this.IR.Form
	}

	return ret
} // }}}

func (this *BaseController) Cost() int64 {
	return time.Now().Sub(this.startTime).Nanoseconds() / 1000 / 1000
}

func (this *BaseController) writeToWriter(rb []byte) {
	this.RW.Write(rb)
}

func (this *BaseController) GetRpcContent() string {
	return this.rpcContent
}

func (this *BaseController) RenderHtml(file string) {
	t, err := template.New(file).ParseFiles("../static/views/" + file)
	if err != nil {
		this.Render(err.Error())
		return
	}
	values := map[string]template.HTML{"html": template.HTML("<br/>")}

	t.Execute(this.RW, values)
}
