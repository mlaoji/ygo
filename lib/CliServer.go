package lib

import (
	"flag"
	"fmt"
	"net/url"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
)

func NewCliServer() *CliServer {
	return &CliServer{
		routMap: make(map[string]map[string]reflect.Type),
	}
}

type CliServer struct {
	routMap map[string]map[string]reflect.Type //key:controller: {key:method value:reflect.type}
}

func (this *CliServer) AddController(c interface{}) {
	this.addController(c)
}

//run with endless
func (this *CliServer) Run() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	this.serveCli()
}

func (this *CliServer) serveCli() {
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

			fmt.Println("ServeCli: ", errmsg)
		}
	}()

	params := flag.Args()
	Interceptor(nil != params && len(params) > 0, ERR_METHOD_INVALID, params)

	var cname, mname string
	uri := strings.Trim(params[0], "/")

	//根据路径路由: User.GetUserInfo
	path := strings.Split(uri, "/")
	cname = strings.Title(path[0])
	if len(path) > 1 {
		mname = strings.Title(path[1])
	}

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
		return
	}

	vc := reflect.New(contollerType)
	var in []reflect.Value
	var method reflect.Value

	defer func() {
		if err := recover(); err != nil {
			in = []reflect.Value{reflect.ValueOf(err)}
			method := vc.MethodByName("RenderError")
			method.Call(in)
		}
	}()

	m := map[string][]string{}
	var err error
	if len(params) > 1 {
		m, err = url.ParseQuery(params[1])
		if nil != err {
			fmt.Println("params parse error")
			return
		}
	}

	in = make([]reflect.Value, 2)
	in[0] = reflect.ValueOf(m)
	in[1] = reflect.ValueOf(uri)
	method = vc.MethodByName("PrepareCli")
	method.Call(in)

	in = make([]reflect.Value, 0)
	method = vc.MethodByName(mname)
	method.Call(in)

	return
}

func (this *CliServer) addController(c interface{}) {
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
