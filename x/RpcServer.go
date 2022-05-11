//go:build !norpc
// +build !norpc

package x

import (
	"fmt"
	"github.com/mlaoji/ygo/x/endless"
	"github.com/mlaoji/ygo/x/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"net/url"
	"reflect"
	//"runtime"
	"runtime/debug"
	"strings"
)

var (
	defaultServices = []interface{}{}
)

//添加rpc 方法对应的controller实例
func AddService(c interface{}) { // {{{
	defaultServices = append(defaultServices, c)
} // }}}

func NewRpcServer(addr string, port int) *RpcServer { // {{{
	server := &RpcServer{
		Addr:    addr,
		Port:    port,
		hanlder: &rpcHandler{routMap: make(map[string]map[string]reflect.Type)},
	}

	for _, v := range defaultServices {
		server.AddController(v)
	}

	return server
} // }}}

type RpcServer struct {
	Addr    string
	Port    int
	hanlder *rpcHandler
}

func (this *RpcServer) AddController(c interface{}) {
	this.hanlder.addController(c)
}

/*
func (this *RpcServer) Run() {// {{{
	//runtime.GOMAXPROCS(runtime.NumCPU())
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", this.Port))

	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()
	pb.RegisterYGOServiceServer(s, this.hanlder)
	fmt.Println("RpcServer Listen: ", this.Port)
	s.Serve(lis)
}// }}}
*/
//run with endless
func (this *RpcServer) Run() { // {{{
	if len(this.hanlder.routMap) == 0 {
		return
	}

	//runtime.GOMAXPROCS(runtime.NumCPU())
	rpcServer := grpc.NewServer()
	pb.RegisterYGOServiceServer(rpcServer, this.hanlder)

	addr := fmt.Sprintf("%s:%d", this.Addr, this.Port)
	log.Println("RpcServer Listen:", addr)

	log.Println(endless.ListenAndServeOther(addr, "", rpcServer))
} // }}}

type rpcHandler struct { // {{{
	routMap map[string]map[string]reflect.Type //key:controller: {key:method value:reflect.type}
} // }}}

func (this *rpcHandler) Call(ctx context.Context, in *pb.Request) (*pb.Reply, error) { // {{{
	method := in.Method
	params := in.Params

	res := this.ServeTCP(method, params, ctx)

	return &pb.Reply{Response: res}, nil
} // }}}

func (this *rpcHandler) ServeTCP(requesturi string, params map[string]string, ctx context.Context) (ret string) { // {{{
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

			fmt.Println("ServeTCP: ", errmsg)
		}
	}()

	uri := strings.Trim(requesturi, " \r\t\v/")
	path := strings.Split(uri, "/")

	Interceptor(len(path) > 1, ERR_METHOD_INVALID, uri)

	controller_name := strings.Title(path[0])
	action_name := strings.Title(path[1])

	canhandler := false
	var contollerType reflect.Type
	if controller_name != "" && action_name != "" {
		if methodMap, ok := this.routMap[controller_name]; ok {
			if contollerType, ok = methodMap[action_name]; ok {
				canhandler = true
			}
		}
	}

	if !canhandler {
		fmt.Println("method not exits ")
		return ""
	}

	vc := reflect.New(contollerType)
	var in []reflect.Value
	var method reflect.Value

	defer func() {
		if err := recover(); err != nil {
			in = []reflect.Value{reflect.ValueOf(err)}
			method := vc.MethodByName("RenderError")
			method.Call(in)

			in = make([]reflect.Value, 0)
			method = vc.MethodByName("GetRpcContent")
			res := method.Call(in)
			ret = fmt.Sprintf("%s", res[0])
		}
	}()

	rpc_params := url.Values{}
	for k, v := range params {
		rpc_params.Set(k, v)
	}

	in = make([]reflect.Value, 4)
	in[0] = reflect.ValueOf(rpc_params)
	in[1] = reflect.ValueOf(ctx)
	in[2] = reflect.ValueOf(controller_name)
	in[3] = reflect.ValueOf(action_name)
	method = vc.MethodByName("PrepareRpc")
	method.Call(in)

	//call Init method if exists
	in = make([]reflect.Value, 0)
	method = vc.MethodByName("Init")
	method.Call(in)

	in = make([]reflect.Value, 0)
	method = vc.MethodByName(action_name + ACTION_SUFFIX)
	method.Call(in)

	in = make([]reflect.Value, 0)
	method = vc.MethodByName("GetRpcContent")
	res := method.Call(in)
	ret = fmt.Sprintf("%s", res[0])

	return
} // }}}

func (this *rpcHandler) addController(c interface{}) { // {{{
	reflectVal := reflect.ValueOf(c)
	rt := reflectVal.Type()
	ct := reflect.Indirect(reflectVal).Type()
	controller_name := strings.TrimSuffix(ct.Name(), "Controller")
	if _, ok := this.routMap[controller_name]; ok {
		return
	} else {
		this.routMap[controller_name] = make(map[string]reflect.Type)
	}
	var action_fullname string
	for i := 0; i < rt.NumMethod(); i++ {
		action_fullname = rt.Method(i).Name
		if strings.HasSuffix(action_fullname, ACTION_SUFFIX) {
			this.routMap[controller_name][strings.TrimSuffix(action_fullname, ACTION_SUFFIX)] = ct
		}
	}
} // }}}
