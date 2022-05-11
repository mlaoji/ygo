//go:build norpc
// +build norpc

package x

func AddService(c interface{}) {
}

func NewRpcServer(addr string, port int) *RpcServer {
	return &RpcServer{}
}

type RpcServer struct {
}

func (this *RpcServer) AddController(c interface{}) {
}

func (this *RpcServer) Run() {
}
