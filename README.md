# YGO

这是你需要的golang微服务框架，绝对不是简单的URL路由

## Installation

    go get golang.org/x/net
    go get golang.org/x/text
    go get google.golang.org/grpc
    go get github.com/go-sql-driver/mysql
    go get github.com/golang/protobuf
    go get github.com/mediocregopher/radix.v2
    go get github.com/patrickmn/go-cache
    go get github.com/mlaoji/ygo
    go get github.com/mlaoji/yclient
    go get github.com/mlaoji/yqueue

## Quick Start

1. 把github.com/mlaoji/ygo/demo目录拷贝到你的GOPATH/src路径下,作为一个演示项目
2. 进入demo目录，执行 ./run, 看看效果(实际执行的是go run main.go -f xxxx.conf -m http), 这样已经启动了一个http服务，浏览器访问：http://127.0.0.1:6002/testHttp/hello
3. 再执行 ./run rpc, 看看效果(实际执行的是go run main.go -f xxxx.conf -m rpc), 这样就启动了一个RPC服务，可以到 tool目录下运行testRpc.go 测试一下rpc效果
4. 应该已经知道，-f 后面跟的就是配置文件 -m 后面跟的就是服务模式(http、rpc、 cli)
5. 运行make脚本, 编译并输出执行文件到./bin 目录, 文件名称默认为:"app.bin", 也可指定名称, 如  make passport, 则文件名为 passport.add.bin 
6. 运行控制台脚本 ./serverctl 启动、停止、重启, ./serverctl start|stop|restart|reload

文档梳理中……
