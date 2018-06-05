package lib

import (
	"fmt"
)

var (
	//多语言时指定默认语言
	DEFAULT_LANG = "CN"
	//成功
	ERR_SUC = &Error{0, "OK"}

	//系统级别，10X
	ERR_SYSTEM         = &Error{100, "系统错误"}
	ERR_METHOD_INVALID = &Error{101, "请求不合法: %+v"}
	ERR_PARAMS         = &Error{102, "参数格式错误: %+v"}
	ERR_GUID           = &Error{103, "签名不合法: %+v"}
	ERR_FLOOD          = &Error{104, "不能重复请求: %+v"}
	ERR_FREQ           = &Error{105, "接口访问过于频繁: %+v"}
	ERR_TOKEN          = &Error{106, "您的登录已过期或在其他设备登录,请重新登录"}
	ERR_SIGN           = &Error{107, "令牌[sign]不正确:%+v"}
	ERR_RPCAUTH        = &Error{108, "rpc认证失败:%+v"}
	ERR_OTHER          = &Error{109, "%+v"}
	ERR_BAN            = &Error{110, "帐号被封禁"}

	//业务级别，需要定义到业务代码中, 使用4位数字, 不同业务使用不同号段
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
	return fmt.Sprint(this.Msg)
}
