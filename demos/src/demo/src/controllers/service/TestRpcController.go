package service

import (
	"demo/src/models"
	"github.com/mlaoji/ygo/controllers"
	"github.com/mlaoji/ygo/x"
	"time"
)

func init() {
	//注册RPC方法 (Action 结尾)
	x.AddService(&TestRpcController{})
}

type TestRpcController struct {
	controllers.BaseController
}

func (this *TestRpcController) HelloAction() { // {{{

	msg := this.GetString("msg")

	x.Interceptor(len(msg) > 0, x.ERR_PARAMS, "msg")

	ret := map[string]interface{}{
		"msg":         msg,
		"header-guid": this.GetHeader("guid"),
	}

	this.SetHeader("test-header", "1234")

	this.Render(ret)

} // }}}

func (this *TestRpcController) GetUserInfoAction() { // {{{

	uid := this.GetInt("uid")

	x.Interceptor(uid > 0, x.ERR_PARAMS, "uid")

	user_info := models.User().GetUserInfo(uid)

	ret := x.MAP{
		"user_info": user_info,
	}

	time.Sleep(9e9)

	this.Render(ret)

} // }}}
