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

	var controller_name, action_name string
	uri := strings.Trim(params[0], "/")

	//根据路径路由: User.GetUserInfo
	path := strings.Split(uri, "/")
	controller_name = strings.Title(path[0])
	if len(path) > 1 {
		action_name = strings.Title(path[1])
	}

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

	in = make([]reflect.Value, 3)
	in[0] = reflect.ValueOf(m)
	in[1] = reflect.ValueOf(controller_name)
	in[2] = reflect.ValueOf(action_name)
	method = vc.MethodByName("PrepareCli")
	method.Call(in)

	//call Init method if exists
	in = make([]reflect.Value, 0)
	method = vc.MethodByName("Init")
	method.Call(in)

	in = make([]reflect.Value, 0)
	method = vc.MethodByName(action_name + ACTION_SUFFIX)
	method.Call(in)

	return
}

func (this *CliServer) addController(c interface{}) {
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
}
