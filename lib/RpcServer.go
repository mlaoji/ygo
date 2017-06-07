package lib

import (
	"fmt"
	pb "github.com/mlaoji/yclient/pb"
	"github.com/mlaoji/ygo/lib/endless"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"time"
	//"runtime"
	"runtime/debug"
	"strings"
)

func NewRpcServer(port int, timout int, pprof bool) *RpcServer {
	return &RpcServer{
		Port:    port,
		Timeout: timout,
		hanlder: &rpcHandler{routMap: make(map[string]map[string]reflect.Type), enablePprof: pprof},
	}
}

//http服务监听,路由
type RpcServer struct {
	Port    int
	Timeout int
	hanlder *rpcHandler
}

func (this *RpcServer) AddController(c interface{}) {
	this.hanlder.addController(c)
}

/*
func (this *RpcServer) Run() {
	//runtime.GOMAXPROCS(runtime.NumCPU())
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", this.Port))

	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()
	pb.RegisterYGOServiceServer(s, this.hanlder)
	fmt.Println("RpcServer Listen: ", this.Port)
	s.Serve(lis)
}
*/
//run with endless
func (this *RpcServer) Run() {
	//runtime.GOMAXPROCS(runtime.NumCPU())
	s := grpc.NewServer()
	pb.RegisterYGOServiceServer(s, this.hanlder)
	fmt.Println("RpcServer Listen: ", this.Port)

	endless.DefaultReadTimeOut = time.Duration(this.Timeout) * time.Millisecond
	endless.DefaultWriteTimeOut = time.Duration(this.Timeout) * time.Millisecond
	endless.HandleGRPC = s.Serve
	if 1 == Conf.GetInt("dev_mode") {
		endless.DevMode = true
	}

	monitor_port := Conf.Get("rpc_monitor_port")
	if "" != monitor_port {
		go this.monitorStatus(monitor_port)
	}

	log.Println(endless.ListenAndServe(fmt.Sprintf(":%d", this.Port), nil))
}

func (this *RpcServer) monitorStatus(port string) {
	http.HandleFunc("/status", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("ok\n"))
	})

	fmt.Println("lvs monitor Listen: ", port)
	http.ListenAndServe(":"+port, nil)
}

type rpcHandler struct {
	routMap     map[string]map[string]reflect.Type //key:controller: {key:method value:reflect.type}
	enablePprof bool
}

func (this *rpcHandler) Call(ctx context.Context, in *pb.Request) (*pb.Reply, error) {
	method := in.Method
	params := in.Params

	//fmt.Println(method)
	//fmt.Println(params)

	res := this.ServeTCP(method, params)

	return &pb.Reply{Response: res}, nil
}

func (this *rpcHandler) ServeTCP(requesturi string, params map[string]string) (ret string) {
	defer func() {
		if err := recover(); err != nil {
			var errmsg string
			switch errinfo := err.(type) {
			case *Error:
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

	var cname, mname string
	uri := strings.Trim(requesturi, " \r\t\v/")

	path := strings.Split(uri, "/")

	Interceptor(len(path) > 1, ERR_METHOD_INVALID, uri)

	cname = strings.Title(path[0])
	mname = strings.Title(path[1])

	//只能调用以Action结尾的方法
	mname = mname + METHOD_EXPORT_TAG
	canhandler := false
	var contollerType reflect.Type
	if cname != "" && mname != "" {
		if methodMap, ok := this.routMap[cname]; ok {
			if contollerType, ok = methodMap[mname]; ok {
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

	in = make([]reflect.Value, 2)
	in[0] = reflect.ValueOf(rpc_params)
	in[1] = reflect.ValueOf(uri)
	method = vc.MethodByName("PrepareRpc")
	method.Call(in)

	in = make([]reflect.Value, 0)
	method = vc.MethodByName(mname)
	method.Call(in)

	in = make([]reflect.Value, 0)
	method = vc.MethodByName("GetRpcContent")
	res := method.Call(in)
	ret = fmt.Sprintf("%s", res[0])

	return
}

func (this *rpcHandler) addController(c interface{}) {
	reflectVal := reflect.ValueOf(c)
	rt := reflectVal.Type()
	ct := reflect.Indirect(reflectVal).Type()
	firstParam := strings.TrimSuffix(ct.Name(), "Controller")
	if _, ok := this.routMap[firstParam]; ok {
		return
	} else {
		this.routMap[firstParam] = make(map[string]reflect.Type)
	}
	var mname string
	for i := 0; i < rt.NumMethod(); i++ {
		mname = rt.Method(i).Name
		if strings.HasSuffix(mname, METHOD_EXPORT_TAG) {
			this.routMap[firstParam][rt.Method(i).Name] = ct
		}
	}
}
