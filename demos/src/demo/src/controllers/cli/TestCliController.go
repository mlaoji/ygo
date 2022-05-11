package cli

import (
	"github.com/mlaoji/ygo/controllers"
	"github.com/mlaoji/ygo/x"
	//"github.com/mlaoji/yqueue"
	//  "demo/src/workers"
)

func init() {
	//注册CLI方法 (Action 结尾)
	x.AddCli(&TestCliController{})
}

type TestCliController struct {
	controllers.BaseController
}

func (this *TestCliController) HelloAction() { // {{{

	msg := this.GetString("msg")

	x.Interceptor(len(msg) > 0, x.ERR_PARAMS, "msg")

	ret := map[string]interface{}{
		"msg": msg,
	}

	this.Render(ret)

} // }}}

func (this *TestCliController) QueueAction() { // {{{

	//queue := yqueue.NewYQueue()
	//queue.SetDebug(this.Debug)
	//queue.AddWorker("test", &workers.TestWorker{}, yqueue.WithWorkerOptionChildren(1))

	//queue.Run()
} // }}}
