//go:build nows
// +build nows

package x

func SetWebsocketMaxHeaderBytes(m int) {
}

func SetWebsocketHandler(p string, h interface{}) {
}

func NewWebsocketServer(addr string, port, timeout int) *WebsocketServer {
	return &WebsocketServer{}
}

type WebsocketServer struct {
}

func (this *WebsocketServer) Run() {
}
