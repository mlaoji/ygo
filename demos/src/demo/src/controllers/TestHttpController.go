package controllers

import (
	"demo/src/models"
	"github.com/mlaoji/ygo/controllers"
	"github.com/mlaoji/ygo/x"
	//"time"
)

type TestHttpController struct {
	controllers.BaseController
}

func init() {
	//注册http api 方法 (Action 结尾)
	x.AddApi(&TestHttpController{})
}

func (this *TestHttpController) HelloAction() { // {{{

	msg := this.GetString("msg")

	x.Interceptor(len(msg) > 0, x.ERR_PARAMS, "msg")

	ret := x.MAP{
		"msg": msg,
	}

	this.Render(ret)

} // }}}

func (this *TestHttpController) GetUserInfoAction() { // {{{

	uid := this.GetInt("uid")

	x.Interceptor(uid > 0, x.ERR_PARAMS, "uid")

	user_info := models.User().GetUserInfo(uid)

	ret := x.MAP{
		"user_info": user_info,
	}
	//time.Sleep(10e9)

	this.Render(ret)

} // }}}
