package x

import (
	"bufio"
	"fmt"
	"github.com/mlaoji/ygo/x/endless"
	"log"
	"net"
	"runtime/debug"
)

var (
	defaultTcpHandler = func(conn net.Conn) { // {{{

		defer conn.Close()

		for {
			reader := bufio.NewReader(conn)
			var buf [128]byte
			n, err := reader.Read(buf[:])
			if err != nil {
				fmt.Println("read from client failed, err: ", err)
				break
			}
			recvStr := string(buf[:n])
			fmt.Println("Received msg from Client：", conn.RemoteAddr(), recvStr)
			conn.Write([]byte(recvStr))
		}
	} // }}}
)

//设置回调处理
func SetTcpHandler(h func(net.Conn)) { // {{{
	defaultTcpHandler = h
} // }}}

func NewTcpServer(addr string, port int) *TcpServer {
	server := &TcpServer{
		Addr:    addr,
		Port:    port,
		handler: defaultTcpHandler,
	}

	return server
}

type TcpServer struct {
	Addr    string
	Port    int
	handler func(net.Conn)
}

func (this *TcpServer) SetTcpHandler(h func(net.Conn)) {
	this.handler = h
}

/*
func (this *TcpServer) Run() { // {{{
	if this.handler == nil {
		return
	}

	addr := fmt.Sprintf("%s:%d", this.Addr, this.Port)

	log.Println("TcpServer Listen", addr)

	listener, err := this.listenTCP(addr)
	if err != nil {
		log.Println("TcpServer Listen error:", err)
	}

	this.serve(listener)
} // }}}
*/

//使用endless, 支持graceful reload
func (this *TcpServer) Run() { // {{{
	if this.handler == nil {
		return
	}

	addr := fmt.Sprintf("%s:%d", this.Addr, this.Port)

	log.Println("TcpServer Listen", addr)

	log.Println(endless.ListenAndServeOther(addr, "tcp4", this))
} // }}}

func (this *TcpServer) listenTCP(addrStr string) (net.Listener, error) { // {{{
	addr, err := net.ResolveTCPAddr("tcp4", addrStr)
	if err != nil {
		return nil, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	return listener, err
} // }}}

func (this *TcpServer) Serve(listener net.Listener) error { // {{{
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("listener Accept error: %v", err)
			return err
		}

		go this.process(conn)
	}

	return nil
} // }}}

func (this *TcpServer) process(conn net.Conn) { // {{{
	defer func() {
		if err := recover(); err != nil {
			var errmsg string
			switch errinfo := err.(type) {
			case *Error:
				errmsg = errinfo.GetMessage()
			case *Errorf:
				errmsg = errinfo.GetMessage()
			case error:
				errmsg = errinfo.Error()
				fmt.Println(errmsg)
				debug.PrintStack()
			default:
				errmsg = fmt.Sprint(errinfo)
			}

			fmt.Println("ServeTcp: ", errmsg)
		}
	}()

	defer conn.Close() // 关闭连接

	this.handler(conn)
} // }}}

func (this *TcpServer) GracefulStop() {
}
