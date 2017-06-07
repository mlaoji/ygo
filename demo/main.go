package main

import (
	"demo/src/controllers"
	"demo/src/controllers/cli"
	"demo/src/controllers/service"
	"github.com/mlaoji/ygo"
)

func main() {
	new_ygo := ygo.NewYgo()

	//注册http接口
	new_ygo.AddApi(&controllers.TestHttpController{})

	//注册rpc接口
	new_ygo.AddService(&service.TestRpcController{})

	//注册cli接口
	new_ygo.AddCli(&cli.TestCliController{})

	new_ygo.Run()
}
