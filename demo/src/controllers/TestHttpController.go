package controllers

import (
	"github.com/mlaoji/ygo/controllers"
	"github.com/mlaoji/ygo/lib"
)

type TestHttpController struct {
	controllers.BaseController
}

func (this *TestHttpController) HelloAction() { // {{{
	msg := this.GetString("msg")

	lib.Interceptor(len(msg) > 0, lib.ERR_PARAMS, "msg")

	ret := map[string]interface{}{
		"msg": msg,
	}

	this.Render(ret)

}
