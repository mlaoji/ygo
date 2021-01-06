package lib

import (
	"fmt"
	"github.com/mlaoji/ygo/lib/endless"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"reflect"
	//"runtime"
	"runtime/debug"
	"strings"
	"time"
)

func NewHttpServer(addr string, port int, timout int, pprof bool) *HttpServer {
	return &HttpServer{
		HttpAddr: addr,
		HttpPort: port,
		Timeout:  timout,
		handler:  &httpHandler{routMap: make(map[string]map[string]reflect.Type), enablePprof: pprof},
	}
}

//http服务监听,路由
type HttpServer struct {
	HttpAddr string
	HttpPort int
	Timeout  int
	handler  *httpHandler
}

func (this *HttpServer) AddController(c interface{}, group ...string) {
	this.handler.addController(c, group...)
}

/*
func (this *HttpServer) Run() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	addr := fmt.Sprintf("%s:%d", this.HttpAddr, this.HttpPort)
	s := &http.Server{
		Addr:         addr,
		Handler:      this.handler,
		ReadTimeout:  time.Duration(this.Timeout) * time.Millisecond,
		WriteTimeout: time.Duration(this.Timeout) * time.Millisecond,
	}
	log.Println("HttpServer Listen: ", addr)
	log.Println(s.ListenAndServe())
}
*/
//run with endless
func (this *HttpServer) Run() {
	//runtime.GOMAXPROCS(runtime.NumCPU())
	addr := fmt.Sprintf("%s:%d", this.HttpAddr, this.HttpPort)

	log.Println("HttpServer Listen", addr)
	endless.DefaultReadTimeOut = time.Duration(this.Timeout) * time.Millisecond
	endless.DefaultWriteTimeOut = time.Duration(this.Timeout) * time.Millisecond
	if 1 == Conf.GetInt("dev_mode") {
		endless.DevMode = true
	}
	log.Println(endless.ListenAndServe(addr, this.handler))
}

//controller中以此后缀结尾的方法会参与路由
const ACTION_SUFFIX = "Action"

//默认controller/action
const DEFAULT_CONTROLLER = "Index"
const DEFAULT_ACTION = "Index"

type httpHandler struct {
	routMap     map[string]map[string]reflect.Type //key:controller: {key:method value:reflect.type}
	enablePprof bool
}

func (this *httpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
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
				log.Println(errmsg)
				debug.PrintStack()
			default:
				errmsg = fmt.Sprint(errinfo)
			}

			log.Println("ServeHTTP: ", errmsg)
			http.Error(rw, errmsg, http.StatusInternalServerError)
		}
	}()

	rw.Header().Set("Server", "YGOServer")
	//跨域设置
	ref := r.Referer()
	if "" == ref {
		ref = r.Header.Get("Origin")
	}
	if ref != "" {
		if u, err := url.Parse(ref); nil == err {
			cors_domain := Conf.Get("cors_domain")
			if len(cors_domain) > 0 {
				allowed := false
				if "*" == cors_domain || strings.Contains(","+cors_domain+",", ","+u.Host+",") {
					allowed = true
				} else if strings.Contains(","+cors_domain, ",*.") {
					domains := strings.Split(cors_domain, ",")
					for _, v := range domains {
						if v[0] == '*' && strings.Contains(u.Host+",", string(v[1:])+",") {
							allowed = true
							break
						}
					}
				}

				if allowed {
					rw.Header().Set("Access-Control-Allow-Origin", u.Scheme+"://"+u.Host)
					rw.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}
		}
	}

	var uri, controller_name, action_name string
	if this.enablePprof && strings.HasPrefix(r.URL.Path, "/debug/pprof") { //如果开启了pprof, 相关请求走DefaultServeMux
		this.monitorPprof(rw, r)
		return
	} else if strings.HasPrefix(r.URL.Path, "/status") { //用于lvs监控
		this.monitorStatus(rw, r)
		return
	} else { //根据路径路由: User/GetUserInfo
		uri = strings.Trim(r.URL.Path, " \r\t\v/")

		if "" != uri {
			path := strings.Split(uri, "/")
			l := len(path)
			if l > 0 {
				controller_name = strings.Title(path[0])

				if l > 1 {
					action_name = strings.Title(path[1])
				}

				if l > 2 {
					if _, ok := this.routMap[path[0]+"/"+action_name]; ok {
						controller_name = path[0] + "/" + action_name
						action_name = strings.Title(path[2])
					}
				}
			}
		}

		if "" == controller_name {
			controller_name = Conf.Get("default_controller")
			if "" == controller_name {
				controller_name = DEFAULT_CONTROLLER
			}
		}

		if "" == action_name {
			action_name = Conf.Get("default_action")
			if "" == action_name {
				action_name = DEFAULT_ACTION
			}
		}
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
		http.NotFound(rw, r)
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

	in = make([]reflect.Value, 4)
	in[0] = reflect.ValueOf(rw)
	in[1] = reflect.ValueOf(r)
	in[2] = reflect.ValueOf(controller_name)
	in[3] = reflect.ValueOf(action_name)
	method = vc.MethodByName("Prepare")
	method.Call(in)

	//call Init method if exists
	in = make([]reflect.Value, 0)
	method = vc.MethodByName("Init")
	method.Call(in)

	in = make([]reflect.Value, 0)
	method = vc.MethodByName(action_name + ACTION_SUFFIX)
	method.Call(in)
}

//pprof监控
func (this *httpHandler) monitorPprof(rw http.ResponseWriter, r *http.Request) {
	http.DefaultServeMux.ServeHTTP(rw, r)
}

//用于lvs监控
func (this *httpHandler) monitorStatus(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte("ok\n"))
}

func (this *httpHandler) addController(c interface{}, group ...string) {
	reflectVal := reflect.ValueOf(c)
	rt := reflectVal.Type()
	ct := reflect.Indirect(reflectVal).Type()
	controller_name := strings.TrimSuffix(ct.Name(), "Controller")
	if len(group) > 0 {
		controller_name = group[0] + "/" + controller_name
	}

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
