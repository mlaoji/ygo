package main

import (
	_ "demo/src/controllers"
	_ "demo/src/controllers/cli"
	_ "demo/src/controllers/service"
	"github.com/mlaoji/ygo"
	//"github.com/mlaoji/ygo/x"
)

func main() {
	//设置TCP服务的回调
	//x.SetTcpHandler(h)

	//设置Websocket服务的回调
	//x.SetWebsocketHandler("/", wh)

	//按模式启动, 同时启用多个逗号分隔: ./run http,tcp,ws,rpc,cli
	//也可以访问独立方法启动,如: RunRpc()
	ygo.NewYgo().Run()
}
