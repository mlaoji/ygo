package cli

import (
	"github.com/mlaoji/ygo/controllers"
	"github.com/mlaoji/ygo/lib"
)

type TestCliController struct {
	controllers.BaseController
}

func (this *TestCliController) HelloAction() { // {{{
	msg := this.GetString("msg")

	lib.Interceptor(len(msg) > 0, lib.ERR_PARAMS, "msg")

	ret := map[string]interface{}{
		"msg": msg,
	}

	this.Render(ret)

}
