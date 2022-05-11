package main

import (
	"flag"
	"fmt"
	"github.com/mlaoji/yclient"
	"net/url"
)

func main() {
	host := flag.String("h", "127.0.0.1:9002", "host")
	app := flag.String("a", "test", "app")
	secret := flag.String("s", "test", "secret")
	method := flag.String("m", "testRpc/Hello", "method")
	params := flag.String("p", "", "params")

	flag.Parse()

	fmt.Printf("host: %#v\n", *host)

	c, err := yclient.NewYClient(*host, *app, *secret, 3, 1)
	if err != nil {
		fmt.Printf("%#v\n", err)
		return
	}

	m := map[string][]string{}
	p := map[string]interface{}{}

	if len(*params) > 1 {
		m, err = url.ParseQuery(*params)
		if nil != err {
			fmt.Println("params parse error")
			return
		}

		for k, v := range m {
			p[k] = v[0]
		}
	}

	data, err := c.Request(*method, p)

	if err != nil {
		fmt.Printf("%#v\n", err)

		//获取错误码
		errno := c.Errno(err)
		fmt.Printf("%#v\n", errno)

	} else {
		fmt.Println("ok")
		fmt.Printf("%#v\n", data)
	}
}
