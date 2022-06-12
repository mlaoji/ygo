package ygo

import (
	"flag"
	"fmt"
	"github.com/mlaoji/ygo/controllers"
	"github.com/mlaoji/ygo/x"
	"github.com/mlaoji/ygo/x/endless"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	SYSTEM_VERSION = "v1.0"
	SERVER_HTTP    = "http"
	SERVER_RPC     = "rpc"
	SERVER_TCP     = "tcp"
	SERVER_WS      = "ws"
	SERVER_CLI     = "cli"
)

type Ygo struct {
	Mode []string
}

func NewYgo() *Ygo {
	ygo := &Ygo{}
	ygo.Init()

	return ygo
}

func (this *Ygo) Init() { // {{{
	logo := `

    _/      _/    _/_/_/    _/_/
     _/  _/    _/        _/    _/
      _/      _/  _/_/  _/    _/
     _/      _/    _/  _/    _/
    _/        _/_/_/    _/_/

` +
		"Author: YangJiguang\n" +
		"Version: " + SYSTEM_VERSION + "\n" +
		"Home: https://github.com/mlaoji/ygo\n"

	fmt.Printf("\033[1;31;33m%s\033[0m\n", logo)

	this.envInit()
	this.genPidFile()
} // }}}

//初始化
func (this *Ygo) envInit() { // {{{

	os.Chdir(path.Dir(os.Args[0]))
	configFile := flag.String("f", "../conf/app.conf", "config file")
	logPath := flag.String("o", "", "log path")
	appRoot := flag.String("t", "", "app root path")
	mode := flag.String("m", "", "run mode, http|rpc|tcp|ws|cli") // 支持同时运行多个逗号分隔
	debug := flag.Bool("d", false, "use debug mode")

	flag.Parse()

	controllers.DEBUG = *debug

	config, conf_path, err := x.NewConfig(*configFile)
	if nil != err {
		fmt.Println("Error: ", err)
		os.Exit(0)
	}

	fmt.Println("Config Init: ", conf_path)

	x.Conf = config

	if *mode != "" {
		this.Mode = strings.Split(*mode, ",")
	}

	if *appRoot != "" {
		x.AppRoot = *appRoot
	}

	log_root := x.Conf.Get("log_root")
	if *logPath != "" {
		log_root = *logPath
	}

	log_level := x.Conf.GetInt("log_level")
	if *debug {
		log_level = 0
	}

	x.Logger.Init(log_root, x.Conf.Get("log_name"), log_level)

	x.LocalCache = x.NewLocalCache()
	fmt.Println("LocalCache init")

	fmt.Println("run cmd: ", os.Args)
	fmt.Println("time: ", time.Now().Format("2006-01-02 15:04:05"))
} // }}}

//添加http 方法对应的controller实例, 支持分组; 默认url路径: controller/action, 分组时路径: group/controller/action
func (this *Ygo) AddApi(c interface{}, group ...string) {
	x.AddApi(c, group...)
}

//添加rpc 方法对应的controller实例
func (this *Ygo) AddService(c interface{}) {
	x.AddService(c)
}

//添加cli 方法对应的controller实例
func (this *Ygo) AddCli(c interface{}) {
	x.AddCli(c)
}

func (this *Ygo) RunTcp() {
	this.run(SERVER_TCP)
}

func (this *Ygo) RunWebsocket() {
	this.run(SERVER_WS)
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
func (this *Ygo) Run() {
	this.run(this.Mode...)
}

func (this *Ygo) run(modes ...string) { // {{{
	defer func() {
		this.removePidFile()
		fmt.Println("======= Server Exit ======")
	}()

	if len(modes) == 0 {
		fmt.Println("Error: 未指定运行模式")
		os.Exit(0)
	}

	if x.Conf.Get("env_mode") == "DEV" {
		endless.DevMode = true
	}

	//是否监听status
	run_moniter := true

	var wg sync.WaitGroup
	for _, mode := range modes { // {{{

		switch mode {
		case "http":
			run_moniter = false

			wg.Add(1)
			go func() {
				defer wg.Done()
				x.NewHttpServer(
					x.Conf.Get("http_addr"),
					x.Conf.GetInt("http_port"),
					x.Conf.GetInt("http_timeout"),
					x.Conf.GetBool("static_enable"),
					x.Conf.Get("static_path"),
					x.Conf.Get("static_root"),
				).Run()
			}()

		case "rpc":
			wg.Add(1)
			go func() {
				defer wg.Done()
				x.NewRpcServer(x.Conf.Get("rpc_addr"), x.Conf.GetInt("rpc_port")).Run()
			}()

		case "tcp":
			wg.Add(1)
			go func() {
				defer wg.Done()
				x.NewTcpServer(x.Conf.Get("tcp_addr"), x.Conf.GetInt("tcp_port")).Run()
			}()

		case "ws":
			wg.Add(1)
			go func() {
				defer wg.Done()
				x.NewWebsocketServer(x.Conf.Get("ws_addr"), x.Conf.GetInt("ws_port"), x.Conf.GetInt("ws_timeout")).Run()
			}()

		case "cli":
			wg.Add(1)
			go func() {
				defer wg.Done()
				x.NewCliServer().Run()
			}()

		default:
			fmt.Println("Error: 未指定正确的运行模式")
			os.Exit(0)
		}

		fmt.Println("======= " + mode + " Server Start ======")

	} // }}}

	monitor_port := x.Conf.Get("monitor_port")
	if "" != monitor_port && run_moniter {
		go x.RunMonitor(monitor_port)
	}

	wg.Wait()

} // }}}

//生成pid文件
func (this *Ygo) genPidFile() { // {{{
	pid := os.Getpid()
	pidString := strconv.Itoa(pid)
	ioutil.WriteFile(x.Conf.Get("app_pid_file"), []byte(pidString), 0777)

	fmt.Println("pid: ", pidString)
} // }}}

//删除pid文件
func (this *Ygo) removePidFile() {
	os.Remove(x.Conf.Get("app_pid_file"))
}
