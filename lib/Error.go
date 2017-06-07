package lib

var (
	//成功
	ERR_SUC = &Error{0, "OK"}

	//系统级别，10X
	ERR_SYSTEM         = &Error{100, "系统错误"}
	ERR_METHOD_INVALID = &Error{101, "请求不合法: %+v"}
	ERR_PARAMS         = &Error{102, "参数格式错误: %+v"}
	ERR_GUID           = &Error{103, "签名不合法: %+v"}
	ERR_FLOOD          = &Error{104, "不能重复请求: %+v"}
	ERR_FREQ           = &Error{105, "接口访问过于频繁: %+v"}
	ERR_TOKEN          = &Error{106, "令牌[token]不正确:%+v"}
	ERR_SIGN           = &Error{107, "令牌[sign]不正确:%+v"}
	ERR_RPCAUTH        = &Error{108, "rpc认证失败:%+v"}
	ERR_OTHER          = &Error{109, "%+v"}
	ERR_BAN            = &Error{110, "帐号被封禁"}

	//业务级别，需要定义到业务代码中, 使用4位数字, 不同业务使用不同号段
	//ERR_TopicNotExist = &Error{1101, "用户不存在: %s"}
)

type Error struct {
	Code int
	Msg  string
}

func (this *Error) GetCode() int {
	return this.Code
}

func (this *Error) GetMessage() string {
	return this.Msg
}

func (this *Error) Error() string {
	return this.Msg
}
