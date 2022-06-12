//go:build norpc
// +build norpc

package controllers

import (
	"bytes"
	"context"
	"fmt"
	"github.com/mlaoji/ygo/x"
	//"google.golang.org/grpc"
	//"google.golang.org/grpc/metadata"
	//"google.golang.org/grpc/peer"
	"io/ioutil"
	"mime/multipart"
	//"net"
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

var (
	DEBUG      = false
	LOG_ACCESS = true
	LOG_ERROR  = true

	//默认表单上限
	defaultMaxPostSize int64 = 32 << 20 //32M
)

const (
	HTTP_MODE = iota
	RPC_MODE
	CLI_MODE
)

type BaseController struct {
	RW        http.ResponseWriter
	R         *http.Request
	RBody     []byte
	IR        *iRequest
	Ctx       context.Context
	startTime time.Time
	Mode      int
	//rpcInHeaders  metadata.MD
	rpcOutHeaders map[string]string
	rpcContent    []byte
	Controller    string
	Action        string
	Uri           string
	Debug         bool
	logParams     map[string]interface{} //需要额外记录在日志中的参数
	logOmitParams []string               //不希望记录在日志中的参数
	Tpl           *x.Template
}

//默认的初始化方法，可通过在项目中重写此方法实现公共入口方法
func (this *BaseController) Init() {}

//设置post表单大小, 单位M
func (this *BaseController) SetMaxPostSize(m int) { // {{{
	if m > 0 {
		defaultMaxPostSize = int64(m << 20)
	}
} // }}}

func (this *BaseController) Prepare(rw http.ResponseWriter, r *http.Request, controller, action string) { // {{{
	this.RW = rw
	this.R = r
	this.Tpl = x.NewTemplate()

	this.RBody, _ = this.getRequestBody(r)

	r.ParseMultipartForm(defaultMaxPostSize)

	this.prepare(r.Form, HTTP_MODE, controller, action)

	//api 接口频度控制
	check_freq := x.Conf.GetBool("check_freq")

	if check_freq { //默认关闭
		rules := x.Conf.GetSliceMap("freq_conf." + this.Uri)

		for _, freq_rule := range rules {
			whitelist := freq_rule["whitelist"]
			blacklist := freq_rule["blacklist"]
			check_key := freq_rule["key"]
			freq := x.AsInt(freq_rule["freq"])
			interval := x.AsInt(freq_rule["interval"])

			check_val := ""
			if "ip" == check_key {
				check_val = this.GetIp()
			} else {
				check_val = this.GetString(check_key)
			}

			if check_val != "" && freq > 0 && interval > 0 {
				restrict := x.NewRestrict(this.Uri+"_"+check_key+"_"+freq_rule["freq"]+"_"+freq_rule["interval"], freq, interval) //规则名,频率,周期(秒)
				if whitelist != "" {
					restrict.AddWhitelist(strings.Split(whitelist, ","))
				}
				if blacklist != "" {
					restrict.AddBlacklist(strings.Split(blacklist, ","))
				}

				x.Interceptor(!restrict.Add(check_val), x.ERR_FREQ, check_key)
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

/* run in norpc
func (this *BaseController) PrepareRpc(r url.Values, ctx context.Context, controller, action string) { // {{{
	this.prepare(r, RPC_MODE, controller, action)

	this.Ctx = ctx

	//rpc 接口鉴权
	appid := this.GetHeader("appid")
	secret := this.GetHeader("secret")

	x.Interceptor(secret == x.Conf.Get("rpc_auth."+appid), x.ERR_RPCAUTH, appid)
} // }}}
*/

func (this *BaseController) PrepareCli(r url.Values, controller, action string) { // {{{
	this.prepare(r, CLI_MODE, controller, action)
} // }}}

func (this *BaseController) prepare(r url.Values, mode int, controller, action string) { // {{{
	this.startTime = time.Now()
	this.Debug = DEBUG
	this.IR = &iRequest{r}
	this.Mode = mode
	this.Controller = strings.ToLower(controller)
	this.Action = strings.ToLower(action)
	this.Uri = this.Controller + "/" + this.Action
} // }}}

//以下 GetX 方法用于获取参数
func (this *BaseController) GetCookie(key string) string { // {{{
	return x.GetCookie(this.R, key)
} // }}}

func (this *BaseController) GetHeader(key string) string { // {{{
	if HTTP_MODE == this.Mode {
		return this.R.Header.Get(key)
	}

	/* run in norpc
	if RPC_MODE == this.Mode {
		if this.rpcInHeaders == nil {
			this.rpcInHeaders, _ = metadata.FromIncomingContext(this.Ctx)
		}

		if this.rpcInHeaders != nil {
			if v, ok := this.rpcInHeaders[key]; ok {
				return v[0]
			}
		}
	}
	*/

	return ""
} // }}}

func (this *BaseController) getFormValue(key string, trimSpace bool) string { // {{{
	if this.IR.Form == nil {
		return ""
	}

	if vs := this.IR.Form[key]; len(vs) > 0 {
		if trimSpace {
			return strings.TrimSpace(vs[0])
		} else {
			return vs[0]
		}
	}

	return ""
} // }}}

//获取参数, 默认string类型
func (this *BaseController) GetParam(key string, defaultValues ...string) string { // {{{
	return this.GetString(key, defaultValues...)
} // }}}

//获取string类型参数
func (this *BaseController) GetString(key string, defaultValues ...string) string { // {{{
	ret := this.getFormValue(key, true)
	if ret == "" {
		if len(defaultValues) > 0 {
			return defaultValues[0]
		}
	}

	return ret
} // }}}

//获取bytes类型参数
func (this *BaseController) GetBytes(key string, defaultValues ...[]byte) []byte { // {{{
	ret := this.getFormValue(key, false)
	if ret == "" {
		if len(defaultValues) > 0 {
			return defaultValues[0]
		}
	}

	return []byte(ret)
} // }}}

//获取指定字符连接的字符串并转换成[]string
func (this *BaseController) GetSlice(key string, separators ...string) []string { //{{{
	value := this.GetString(key)
	if "" == value {
		return nil
	}

	separator := ","
	if len(separators) > 0 {
		separator = separators[0]
	}

	slice := []string{}
	for _, part := range strings.Split(value, separator) {
		slice = append(slice, strings.Trim(part, " \r\t\v"))
	}

	return slice
} // }}}

//获取指定字符连接的字符串并转换成[]int
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

//获取所有参数
func (this *BaseController) GetParams() map[string]string { // {{{
	if this.IR.Form == nil {
		return nil
	}

	params := map[string]string{}
	for k, v := range this.IR.Form {
		if len(v) > 0 {
			params[k] = strings.TrimSpace(v[0])
		}
	}

	return params
} // }}}

//获取数组类型参数
func (this *BaseController) GetArray(key string) []string { // {{{
	if this.IR.Form == nil {
		return nil
	}

	ret := []string{}
	retry := true
	for {
		if vs := this.IR.Form[key]; len(vs) > 0 {
			for _, v := range vs {
				ret = append(ret, strings.TrimSpace(v))
			}
			break
		}

		if !retry {
			break
		}

		if strings.HasSuffix(key, "[]") {
			key = key[:len(key)-2]
		} else {
			key = key + "[]"
		}

		retry = false
	}

	return ret
} // }}}

//获取MAP类型参数
func (this *BaseController) GetMap(key string) map[string]string { // {{{
	if this.IR.Form == nil {
		return nil
	}

	ret := map[string]string{}
	for k, v := range this.IR.Form {
		if strings.HasPrefix(k, key+"[") && k != key+"[]" && k[len(k)-1] == ']' && len(v) > 0 {
			idx := k[len(key)+1 : len(k)-1]
			ret[idx] = strings.TrimSpace(v[0])
		}
	}

	return ret
} // }}}

//获取Int类型参数
func (this *BaseController) GetInt(key string, defaultValues ...int) int { // {{{
	ret, err := strconv.Atoi(this.getFormValue(key, true))
	if err != nil {
		if len(defaultValues) > 0 {
			return defaultValues[0]
		}
	}

	return ret
} // }}}

//获取Int32类型参数
func (this *BaseController) GetInt32(key string, defaultValues ...int32) int32 { // {{{
	ret, err := strconv.Atoi(this.getFormValue(key, true))
	if err != nil {
		if len(defaultValues) > 0 {
			return defaultValues[0]
		}
	}

	return int32(ret)
} // }}}

//获取Int64类型参数
func (this *BaseController) GetInt64(key string, defaultValues ...int64) int64 { // {{{
	ret, err := strconv.ParseInt(this.getFormValue(key, true), 10, 64)
	if err != nil {
		if len(defaultValues) > 0 {
			return defaultValues[0]
		}
	}

	return ret
} // }}}

//获取bool类型参数
func (this *BaseController) GetBool(key string, defaultValues ...bool) bool { // {{{
	ret, err := strconv.ParseBool(this.getFormValue(key, true))
	if err != nil {
		if len(defaultValues) > 0 {
			return defaultValues[0]
		}
	}

	return ret
} // }}}

//获取json字符串并转换为MAP
func (this *BaseController) GetJsonMap(key string) x.MAP { // {{{
	ret := this.getFormValue(key, true)
	if ret != "" {
		json := x.JsonDecode(ret)
		if m, ok := json.(x.MAP); ok {
			return m
		}
	}
	return nil
} // }}}

//获取上传文件
func (this *BaseController) GetFile(key string) (multipart.File, *multipart.FileHeader, error) { // {{{
	return this.R.FormFile(key)
} // }}}

func (this *BaseController) GetIp() string { // {{{
	if HTTP_MODE == this.Mode {
		return x.GetIp(this.R)
	}

	/* run in norpc
	if RPC_MODE == this.Mode {
		pr, ok := peer.FromContext(this.Ctx)
		if !ok {
			return ""
		}

		if pr.Addr == net.Addr(nil) {
			return ""
		}

		addr := strings.Split(pr.Addr.String(), ":")
		return addr[0]
	}
	*/

	return ""
} // }}}

func (this *BaseController) GetRequestUri() string { // {{{
	if HTTP_MODE == this.Mode && nil != this.R {
		return fmt.Sprint(this.R.URL)
	}

	if RPC_MODE == this.Mode && nil != this.IR {
		return this.Uri
	}

	return ""
} // }}}

func (this *BaseController) GetUA() string { // {{{
	if HTTP_MODE == this.Mode && nil != this.R {
		return this.R.UserAgent()
	}

	return ""
} // }}}

//lifetime<0时删除cookie
//options: domain,secure,httponly,path
func (this *BaseController) SetCookie(key, val string, lifetime int, options ...interface{}) { // {{{
	x.SetCookie(this.RW, key, val, lifetime, options...)
} // }}}

func (this *BaseController) SetHeader(key, val string) { // {{{
	if HTTP_MODE == this.Mode {
		this.RW.Header().Set(key, val)
	} else if RPC_MODE == this.Mode {
		if this.rpcOutHeaders == nil {
			this.rpcOutHeaders = map[string]string{}
		}
		this.rpcOutHeaders[key] = val
	}
} // }}}

func (this *BaseController) SetHeaders(headers http.Header) { // {{{
	this_header := this.RW.Header()
	for k, v := range headers {
		this_header.Set(k, v[0])
	}
} // }}}

//接口正常输出json, 若要改变返回json格式，可在业务代码中重写此方法
func (this *BaseController) Render(data ...interface{}) { // {{{
	var retdata interface{}
	if len(data) > 0 {
		retdata = data[0]
	} else {
		retdata = make(map[string]interface{})
	}

	res := this.RenderResponser(x.ERR_SUC.GetCode(), x.ERR_SUC.GetMessage(), retdata)

	this.RenderJson(res)
} // }}}

//接口异常输出json，在HttpApiServer中回调, 若要改变返回json格式，可在业务代码中重写此方法
func (this *BaseController) RenderError(err interface{}) { // {{{
	errno, errmsg, retdata := this.GetErrorResponse(err)

	res := this.RenderResponser(errno, errmsg, retdata)

	if LOG_ERROR {
		x.Logger.Warn(this.GenLog(), res)
	}

	this.RenderJson(res)
} // }}}

//根据捕获的错误获取需要返回的错误码、错误信息及数据
func (this *BaseController) GetErrorResponse(err interface{}) (int, string, map[string]interface{}) { // {{{
	var errno int
	var errmsg string
	var isbizerr bool

	var retdata = make(map[string]interface{})

	switch errinfo := err.(type) {
	case string:
		errno = x.ERR_SYSTEM.GetCode()
		errmsg = errinfo
	case *x.Error:
		errno = errinfo.GetCode()
		errmsg = errinfo.GetMessage()
		isbizerr = true
	case *x.Errorf:
		lang := this.GetString("lang")
		errno = errinfo.GetCode()
		errmsg = errinfo.GetMessage(lang)
		retdata = errinfo.GetData()
		isbizerr = true
	case error:
		errno = x.ERR_SYSTEM.GetCode()
		errmsg = errinfo.Error()
	default:
		errno = x.ERR_SYSTEM.GetCode()
		errmsg = fmt.Sprint(errinfo)
	}

	if !isbizerr {
		debug_trace := debug.Stack()
		if LOG_ERROR {
			x.Logger.Error(this.GenLog(), errmsg, string(debug_trace))
		}

		fmt.Println(errmsg)
		os.Stderr.Write(debug_trace)

		if x.Conf.Get("env_mode") != "DEV" {
			errmsg = x.ERR_SYSTEM.GetMessage()
		}
	}

	if len(retdata) == 0 {
		retdata = map[string]interface{}{}
	}

	return errno, errmsg, retdata
} // }}}

//格式化输出
func (this *BaseController) RenderResponser(errno, errmsg, retdata interface{}) map[string]interface{} { // {{{
	return map[string]interface{}{
		"code":    errno,
		"msg":     errmsg,
		"time":    time.Now().Unix(),
		"consume": this.Cost(),
		"data":    retdata, //错误时，也可附带一些数据
	}
} // }}}

//输出JSON
func (this *BaseController) RenderJson(res interface{}) { // {{{
	if nil != this.RW {
		this.RW.Header().Set("Content-Type", "application/json;charset=UTF-8")
	}

	this.render(x.JsonEncodeBytes(res))
} // }}}

//输出文本
func (this *BaseController) RenderText(res interface{}) { // {{{
	var ret []byte

	switch v := res.(type) {
	case []byte:
		ret = v
	case string:
		ret = []byte(v)
	default:
		ret = []byte(fmt.Sprint(v))
	}

	this.render(ret)
} // }}}

//输出HTTP状态码
func (this *BaseController) RenderStatus(code int) { // {{{
	this.RW.WriteHeader(code)
} // }}}

//渲染html模板
func (this *BaseController) RenderHtml(files ...string) { // {{{
	file := ""
	if len(files) > 0 {
		file = files[0]
	}

	if "" == file {
		file = strings.ReplaceAll(this.Uri, "/", "_") + x.TemplateSuffix
	}

	err := this.Tpl.Render(this.RW, this.Uri, file)

	if nil != err {
		fmt.Println(err)
	}
} // }}}

//重定向URL
func (this *BaseController) Redirect(url string, codes ...int) { // {{{
	code := http.StatusFound //302
	if len(codes) > 0 {
		code = codes[0]
	}
	http.Redirect(this.RW, this.R, url, code)
} // }}}

func (this *BaseController) render(data []byte) { // {{{
	if LOG_ACCESS {
		x.Logger.Access(this.GenLog(), string(data))
	}

	if this.Mode == RPC_MODE {
		this.rpcContent = data
	} else if this.Mode == CLI_MODE {
		fmt.Printf("%s", data)
	} else {
		this.RW.Write(data)
	}
} // }}}

//获取日志内容
func (this *BaseController) GenLog() map[string]interface{} { // {{{
	ret := make(map[string]interface{})

	if HTTP_MODE == this.Mode && nil != this.R {
		//访问ip
		ret["ip"] = this.GetIp()
		//请求路径
		ret["uri"] = this.R.URL

		if this.R.Method == "POST" {
			ret["post"] = this.R.PostForm
		}

		ret["ua"] = this.R.UserAgent()
	}

	if RPC_MODE == this.Mode && nil != this.IR {
		delete(this.IR.Form, "secret")
		ret["ip"] = this.GetIp()
		ret["uri"] = this.Uri
		ret["post"] = this.IR.Form
	}

	for k, v := range this.logParams {
		ret[k] = v
	}

	if len(this.logOmitParams) > 0 && nil != ret["post"] {
		ret_post := ret["post"].(url.Values)
		if len(ret_post) > 0 {
			for _, v := range this.logOmitParams {
				if _, ok := ret_post[v]; ok {
					delete(ret_post, v)
				}
			}
		}
	}

	return ret
} // }}}

//在业务日志中添加自定义字段
func (this *BaseController) AddLog(k string, v interface{}) { // {{{
	if nil == this.logParams {
		this.logParams = map[string]interface{}{}
	}

	this.logParams[k] = v
} // }}}

//在业务日志中删除字段(比如密码等敏感字段)
func (this *BaseController) OmitLog(v string) { // {{{
	if nil == this.logOmitParams {
		this.logOmitParams = []string{}
	}

	this.logOmitParams = append(this.logOmitParams, v)
} // }}}

func (this *BaseController) Cost() int64 {
	return time.Now().Sub(this.startTime).Nanoseconds() / 1000 / 1000
}

func (this *BaseController) GetRpcContent() []byte { // {{{
	/* run in norpc
	if this.rpcOutHeaders != nil {
		header := metadata.New(this.rpcOutHeaders)
		grpc.SendHeader(this.Ctx, header)
	}
	*/

	return this.rpcContent
} // }}}

//便于在controller中调式
func (this *BaseController) Println(s ...interface{}) { // {{{
	fmt.Println(s...)
} // }}}
