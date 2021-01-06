package lib

import (
	"fmt"
)

var (
	//多语言时指定默认语言
	DEFAULT_LANG = "CN"
	//成功
	ERR_SUC = &Error{0, "OK"}

	//系统错误码
	ERR_SYSTEM         = &Error{10, "系统错误"}
	ERR_METHOD_INVALID = &Error{11, "请求不合法: %+v"}
	ERR_FREQ           = &Error{12, "接口访问过于频繁: %+v"}
	ERR_RPCAUTH        = &Error{13, "rpc认证失败:%+v"}
	ERR_OTHER          = &Error{14, "%+v"}

	//业务级别错误码，需要定义到业务代码中
	//ERR_USER_NOT_EXIST = &Error{1101, "用户不存在: %s"}
)

type Error struct {
	Code int
	Msg  interface{}
}

func (this *Error) GetCode() int {
	return this.Code
}

func (this *Error) GetMessage() string {
	return fmt.Sprint(this.Msg)
}

func (this *Error) Error() string {
	return fmt.Sprint(this.Msg)
}

//格式化输出错误信息
//用于 Interceptor 拦截抛错
//国际化产品,多语言时，Msg 可以设置为map[string]string ,如:{"CN":"系统错误", "EN":"system error"}
type Errorf struct {
	Code int
	Msg  interface{}
	fmts []interface{}
	data map[string]interface{}
}

func (this *Errorf) GetCode() int {
	return this.Code
}

func (this *Errorf) GetMessage(langs ...string) string { // {{{
	if len(this.fmts) > 0 {
		//fmts的可用值为string, 若fmts最后一个值为map, 则认为它是异常时返回的data
		if data, ok := this.fmts[len(this.fmts)-1].(map[string]interface{}); ok {
			this.fmts = this.fmts[0 : len(this.fmts)-1]

			this.data = data
		}
	}

	if msg, ok := this.Msg.(string); ok {
		return fmt.Sprintf(msg, this.fmts...)
	} else if global_msg, ok := this.Msg.(map[string]string); ok {
		lang := ""
		if len(langs) > 0 {
			lang = langs[0]
		}

		if lang != "" {
			if msg, ok := global_msg[lang]; ok {
				return fmt.Sprintf(msg, this.fmts...)
			}
		}

		if msg, ok := global_msg[DEFAULT_LANG]; ok {
			return fmt.Sprintf(msg, this.fmts...)
		}

		for _, v := range global_msg {
			return fmt.Sprintf(v, this.fmts...)
		}
	}

	return fmt.Sprint(this.Msg)
} // }}}

func (this *Errorf) GetData() map[string]interface{} {
	if this.data == nil && len(this.fmts) > 0 {
		if data, ok := this.fmts[len(this.fmts)-1].(map[string]interface{}); ok {
			return data
		}
	}

	return this.data
}
func (this *Errorf) Error() string {
	return this.GetMessage()
}

//捕获异常时，可同时返回data(通过fmts参数最后一个类型为map的值)
func Interceptor(guard bool, errmsg *Error, fmts ...interface{}) {
	if !guard {
		panic(&Errorf{errmsg.Code, errmsg.Msg, fmts, nil})
	}
}
