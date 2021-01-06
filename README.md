# YGO
### version 0.1.1

golang微服务框架

支持http 、 rpc 和 cli 模式

## Installation

    go get -u golang.org/x/net (https://github.com/golang/net)
    go get -u golang.org/x/text (https://github.com/golang/text)
    go get -u google.golang.org/grpc (https://github.com/grpc/grpc-go/)
    go get -u google.golang.org/genproto/googleapis/rpc/status  (https://github.com/google/go-genproto/blob/master/googleapis/rpc/status)
    go get -u github.com/go-sql-driver/mysql
    go get -u github.com/golang/protobuf
    go get -u github.com/mediocregopher/radix.v2
    go get -u github.com/mlaoji/go-cache
    
    go get -u github.com/mlaoji/ygo
    go get -u github.com/mlaoji/yclient
    go get -u github.com/mlaoji/yqueue

## Quick Start

1. 把github.com/mlaoji/ygo/demo目录拷贝到你的GOPATH/src路径下,作为一个演示项目
2. 进入demo目录，执行 ./run, 启动一个http服务，浏览器访问：http://127.0.0.1:6002/testHttp/hello
3. 再执行 ./run rpc, 启动一个RPC服务，可以到 tool目录下运行testRpc.go 测试一下rpc效果
4. run 命令实际执行了 (go run main.go -f xxxx.conf -m rpc)，-f 后面是配置文件 -m 后面是服务模式(http、rpc、 cli)
5. 运行make脚本, 编译并输出执行文件到./bin 目录, 文件名称默认为:"app.bin", 也可指定名称, 如  make test, 则文件名为 test.app.bin 
6. 运行控制台脚本 ./serverctl 启动、停止、重启、热重启, ./serverctl start|stop|restart|reload

详细文档梳理中……
