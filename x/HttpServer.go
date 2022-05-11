package x

import (
	"embed"
	"fmt"
	"github.com/mlaoji/ygo/x/endless"
	"io/fs"
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

var (
	StaticPath = "static"
	StaticRoot = "../src/www"
)

var (
	defaultApis               = [][]interface{}{}
	defaultHttpMaxHeaderBytes = 0 //0时, 将使用默认配置DefaultMaxHeaderBytes(1M)
)

var (
	staticUseEmbed  bool
	embedStatic     embed.FS
	embedStaticPath string
)

//设置使用embed.FS
func StaticEmbed(filesys embed.FS, fs_path string) { // {{{
	staticUseEmbed = true
	embedStatic = filesys
	embedStaticPath = fs_path
} // }}}

//添加http 方法对应的controller实例, 支持分组; 默认url路径: controller/action, 分组时路径: group/controller/action
func AddApi(c interface{}, groups ...string) { // {{{
	group := ""
	if len(groups) > 0 {
		group = groups[0]
	}
	defaultApis = append(defaultApis, []interface{}{c, group})
} // }}}

//设置MaxHeaderBytes, 单位M
func SetHttpMaxHeaderBytes(m int) { // {{{
	if m > 0 {
		defaultHttpMaxHeaderBytes = m << 20
	}
} // }}}

func NewHttpServer(addr string, port, timeout int, enable_static bool, static_path, static_root string) *HttpServer { // {{{
	if "" == static_path {
		static_path = StaticPath
	}

	if "" == static_root {
		static_root = StaticRoot
	}

	if "" != static_root && '/' != static_root[0] && "" != AppRoot {
		static_root = AppRoot + "/" + static_root
	}

	server := &HttpServer{
		HttpAddr:       addr,
		HttpPort:       port,
		Timeout:        timeout,
		maxHeaderBytes: defaultHttpMaxHeaderBytes,
		handler: &httpHandler{
			routMap:      make(map[string]map[string]reflect.Type),
			enableStatic: enable_static,
			staticPath:   "/" + strings.Trim(static_path, "/"),
			staticRoot:   static_root,
		},
	}

	for _, v := range defaultApis {
		server.AddController(v[0], AsString(v[1]))
	}

	return server
} // }}}

type HttpServer struct {
	HttpAddr       string
	HttpPort       int
	Timeout        int
	maxHeaderBytes int
	handler        *httpHandler
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
//使用endless, 支持graceful reload
func (this *HttpServer) Run() {
	if len(this.handler.routMap) == 0 {
		return
	}

	//runtime.GOMAXPROCS(runtime.NumCPU())
	addr := fmt.Sprintf("%s:%d", this.HttpAddr, this.HttpPort)

	log.Println("HttpServer Listen", addr)

	rtimeout := time.Duration(this.Timeout) * time.Millisecond
	wtimeout := time.Duration(this.Timeout) * time.Millisecond
	mhbytes := this.maxHeaderBytes

	log.Println(endless.ListenAndServe(addr, this.handler, rtimeout, wtimeout, mhbytes))
}

//controller中以此后缀结尾的方法会参与路由
const ACTION_SUFFIX = "Action"

//默认controller/action
const DEFAULT_CONTROLLER = "Index"
const DEFAULT_ACTION = "Index"

type httpHandler struct {
	routMap      map[string]map[string]reflect.Type //key:controller: {key:method value:reflect.type}
	enableStatic bool                               //是否解析静态资源
	staticPath   string                             //静态资源访问路径前缀
	staticRoot   string                             //静态资源文件根目录
}

func (this *httpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) { // {{{
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
	if Conf.GetBool("pprof_enable") && strings.HasPrefix(r.URL.Path, "/debug/pprof") { //如果开启了pprof, 相关请求走DefaultServeMux
		this.monitorPprof(rw, r)
		return
	} else if this.enableStatic && strings.HasPrefix(r.URL.Path, this.staticPath) { //如果开启了静态资源服务, 相关请求走fileServrer
		this.serveFile(rw, r)
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
} // }}}

//静态资源服务
func (this *httpHandler) serveFile(rw http.ResponseWriter, r *http.Request) { // {{{
	var filesys http.FileSystem
	if staticUseEmbed {
		if embedStaticPath != "" {
			embedStaticPath = strings.Trim(embedStaticPath, "/")
			embedSub, err := fs.Sub(embedStatic, embedStaticPath)
			if err != nil {
				panic(err)
			}
			filesys = http.FS(embedSub)
		} else {
			filesys = http.FS(embedStatic)
		}
	} else {
		filesys = http.Dir(this.staticRoot)
	}
	http.StripPrefix(this.staticPath, http.FileServer(filesys)).ServeHTTP(rw, r)
} // }}}

//pprof监控
func (this *httpHandler) monitorPprof(rw http.ResponseWriter, r *http.Request) {
	http.DefaultServeMux.ServeHTTP(rw, r)
}

//用于lvs监控
func (this *httpHandler) monitorStatus(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte("ok\n"))
}

func (this *httpHandler) addController(c interface{}, group ...string) { // {{{
	reflectVal := reflect.ValueOf(c)
	rt := reflectVal.Type()
	ct := reflect.Indirect(reflectVal).Type()
	controller_name := strings.TrimSuffix(ct.Name(), "Controller")
	if len(group) > 0 && group[0] != "" {
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
} // }}}
