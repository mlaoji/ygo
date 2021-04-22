package ygo

import (
	"flag"
	"fmt"
	"github.com/mlaoji/ygo/controllers"
	"github.com/mlaoji/ygo/lib"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	SYSTEM_VERSION = "v0.1.1"
	SERVER_HTTP    = "http"
	SERVER_RPC     = "rpc"
	SERVER_CLI     = "cli"
)

type Ygo struct {
	Mode       string
	HttpServer *lib.HttpServer
	RpcServer  *lib.RpcServer
	CliServer  *lib.CliServer
}

func NewYgo() *Ygo {
	ygo := &Ygo{}
	ygo.Init()

	return ygo
}

func (this *Ygo) Init() {
	logo := `

    _/      _/    _/_/_/    _/_/
     _/  _/    _/        _/    _/
      _/      _/  _/_/  _/    _/
     _/      _/    _/  _/    _/
    _/        _/_/_/    _/_/

` +
		"Author: mlaoji\n" +
		"Version: " + SYSTEM_VERSION + "\n" +
		"Home: https://github.com/mlaoji/ygo\n"

	fmt.Printf("\033[1;31;33m%s\033[0m\n", logo)

	this.envInit()
	this.genPidFile()
}

//初始化
func (this *Ygo) envInit() {
	os.Chdir(path.Dir(os.Args[0]))
	confiPath := flag.String("f", "../conf/app.conf", "config file")
	logPath := flag.String("o", "", "log path")
	mode := flag.String("m", "", "http or rpc or cli ?")
	debug := flag.Bool("d", false, "use debug mode")
	flag.Parse()

	controllers.DEBUG = *debug

	err := lib.Conf.Init(*confiPath)
	if nil != err {
		fmt.Println("Error: ", err)
		os.Exit(0)
	}

	if *mode != "" {
		this.Mode = *mode
	}

	if *logPath == "" {
		*logPath = lib.Conf.Get("log_root")

		if "" != this.Mode && "http" != this.Mode {
			*logPath = strings.TrimRight(*logPath, "/") + "_" + this.Mode
		}
	}

	log_level := lib.Conf.GetInt("log_level")
	if *debug {
		log_level = 0
	}

	lib.Logger.Init(*logPath, lib.Conf.Get("log_name"), log_level)

	lib.LocalCache.Init()

	fmt.Println("run cmd: ", os.Args)
	fmt.Println("time: ", time.Now().Format("2006-01-02 15:04:05"))
}

//添加http 方法对应的controller, 支持分组; 默认url路径: controller/action, 分组时路径: group/controller/action
func (this *Ygo) AddApi(c interface{}, group ...string) {
	if this.HttpServer == nil {
		this.initHttpServer()
	}

	this.HttpServer.AddController(c, group...)
}

//添加rpc 方法对应的controller
func (this *Ygo) AddService(c interface{}) {
	if this.RpcServer == nil {
		this.initRpcServer()
	}

	this.RpcServer.AddController(c)
}

//添加cli 方法对应的controller
func (this *Ygo) AddCli(c interface{}) {
	if this.CliServer == nil {
		this.initCliServer()
	}

	this.CliServer.AddController(c)
}

func (this *Ygo) initHttpServer() {
	this.HttpServer = lib.NewHttpServer("", lib.Conf.GetInt("http_port"), lib.Conf.GetInt("http_timeout"), lib.Conf.GetBool("pprof_enable"))
}

func (this *Ygo) initRpcServer() {
	this.RpcServer = lib.NewRpcServer(lib.Conf.GetInt("rpc_port"), lib.Conf.GetInt("rpc_timeout"), lib.Conf.GetBool("rpc_pprof_enable"))
}

func (this *Ygo) initCliServer() {
	this.CliServer = lib.NewCliServer()
}

func (this *Ygo) RunRpc() {
	this.run(SERVER_RPC)
}

func (this *Ygo) RunHttp() {
	this.run(SERVER_HTTP)
}

func (this *Ygo) RunCli() {
	this.run(SERVER_CLI)
}

//支持命令行参数 -m 指定运行模式
func (this *Ygo) Run() { // {{{
	this.run(this.Mode)
} // }}}

func (this *Ygo) run(mode string) { // {{{
	defer func() {
		this.removePidFile()
		println("======= Server Exit ======")
	}()

	println("======= " + mode + " Server Start ======")

	if mode == SERVER_RPC {
		if this.RpcServer != nil {
			this.RpcServer.Run()
		}
	} else if mode == SERVER_HTTP {
		if this.HttpServer != nil {
			this.HttpServer.Run()
		}
	} else if mode == SERVER_CLI {
		if this.CliServer != nil {
			this.CliServer.Run()
		}
	} else {
		if this.HttpServer != nil && this.RpcServer != nil {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				this.RpcServer.Run()
			}()

			this.HttpServer.Run()

			wg.Wait()
		} else if this.HttpServer != nil {
			this.HttpServer.Run()
		} else if this.RpcServer != nil {
			this.RpcServer.Run()
		}
	}
} // }}}

//生成pid文件
func (this *Ygo) genPidFile() {
	pid := os.Getpid()
	pidString := strconv.Itoa(pid)
	ioutil.WriteFile(lib.Conf.Get("app_pid_file"), []byte(pidString), 0777)

	fmt.Println("pid: ", pidString)
}

//删除pid文件
func (this *Ygo) removePidFile() {
	os.Remove(lib.Conf.Get("app_pid_file"))
}
