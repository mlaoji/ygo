package x

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"strings"
)

func RunMonitor(port string) { // {{{
	fmt.Println("monitor Listen: ", port)
	http.ListenAndServe(":"+port, &monitorHandler{})
} // }}}

type monitorHandler struct {
}

func (m *monitorHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) { // {{{
	if strings.HasPrefix(r.URL.Path, "/debug/pprof") {
		if Conf.GetBool("pprof_enable") { //如果开启了pprof, 相关请求走DefaultServeMux
			http.DefaultServeMux.ServeHTTP(rw, r)
		} else {
			rw.Write([]byte("unavailable\n"))
		}
	}

	if strings.HasPrefix(r.URL.Path, "/status") { //用于lvs监控
		rw.Write([]byte("ok\n"))
	}
} // }}}
