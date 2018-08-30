# YGO

这是你需要的golang微服务框架

可支持同一代码在 http 和 rpc 服务模式的切换

## Installation

    go get -u golang.org/x/net (https://github.com/golang/net)
    go get -u golang.org/x/text (https://github.com/golang/text)
    go get -u google.golang.org/grpc (https://github.com/grpc/grpc-go/)
    go get -u google.golang.org/genproto/googleapis/rpc/status  (https://github.com/google/go-genproto/blob/master/googleapis/rpc/status)
    go get -u github.com/go-sql-driver/mysql
    go get -u github.com/golang/protobuf
    go get -u github.com/mediocregopher/radix.v2
    go get -u github.com/patrickmn/go-cache
    
    go get -u github.com/mlaoji/ygo
    go get -u github.com/mlaoji/yclient
    go get -u github.com/mlaoji/yqueue

## Quick Start

1. 把github.com/mlaoji/ygo/demo目录拷贝到你的GOPATH/src路径下,作为一个演示项目
2. 进入demo目录，执行 ./run, 看看效果(实际执行的是go run main.go -f xxxx.conf -m http), 这样已经启动了一个http服务，浏览器访问：http://127.0.0.1:6002/testHttp/hello
3. 再执行 ./run rpc, 看看效果(实际执行的是go run main.go -f xxxx.conf -m rpc), 这样就启动了一个RPC服务，可以到 tool目录下运行testRpc.go 测试一下rpc效果
4. 应该已经知道，-f 后面跟的就是配置文件 -m 后面跟的就是服务模式(http、rpc、 cli)
5. 运行make脚本, 编译并输出执行文件到./bin 目录, 文件名称默认为:"app.bin", 也可指定名称, 如  make passport, 则文件名为 passport.app.bin 
6. 运行控制台脚本 ./serverctl 启动、停止、重启、热重启, ./serverctl start|stop|restart|reload

详细文档梳理中……
