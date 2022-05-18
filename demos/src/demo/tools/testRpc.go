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

	fmt.Println("host: ", *host)

	c, err := yclient.NewYClient(*host, *app, *secret)
	if err != nil {
		fmt.Println("init err: ", err)
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

	res, err := c.Request(*method, p)

	if err != nil {
		fmt.Println("res err: ", err)
		return
	}

	fmt.Println("code: ", res.GetCode())
	fmt.Println("msg: ", res.GetMsg())
	fmt.Println("data: ", res.GetData())
	fmt.Println("header: ", res.GetHeaders())
	fmt.Println("trailer: ", res.GetTrailers())
}
