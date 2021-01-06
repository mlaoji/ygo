package lib

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func NewHttpClient(timeouts ...int) *HttpClient {
	conn_timeout := 3
	read_timeout := 5
	if len(timeouts) > 0 {
		conn_timeout = timeouts[0]
		read_timeout = conn_timeout

		if len(timeouts) > 1 {
			read_timeout = timeouts[1]
		}
	}

	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				conn, err := net.DialTimeout(netw, addr, time.Second*time.Duration(conn_timeout))
				if err != nil {
					return nil, err
				}
				conn.SetDeadline(time.Now().Add(time.Second * time.Duration(read_timeout)))
				return conn, nil
			},
			ResponseHeaderTimeout: time.Second * time.Duration(read_timeout),
			DisableKeepAlives:     true,
		},
	}

	return &HttpClient{client}
}

type HttpClient struct {
	client *http.Client
}

type httpResponse struct {
	response string
	code     int
	header   http.Header
}

func (r *httpResponse) GetResponse() string {
	return r.response
}

func (r *httpResponse) GetCode() int {
	return r.code
}

func (r *httpResponse) GetHeader() http.Header {
	return r.header
}

func (this *HttpClient) Get(requrl string, headers ...http.Header) (*httpResponse, error) { // {{{
	return this.Request("GET", requrl, nil, headers...)
} //}}}

func (this *HttpClient) Post(requrl string, post_data interface{}, headers ...http.Header) (*httpResponse, error) { // {{{
	return this.Request("POST", requrl, post_data, headers...)
} //}}}

//post_data 支持map[string]interface{} 和 io.Reader 两种参数类型
func (this *HttpClient) Request(method, requrl string, post_data interface{}, headers ...http.Header) (*httpResponse, error) { // {{{
	var data io.Reader
	if params, ok := post_data.(map[string]interface{}); ok {
		urlparams := url.Values{}
		for k, v := range params {
			urlparams.Add(k, AsString(v))
		}

		query := urlparams.Encode()
		data = strings.NewReader(query)
	} else if reader, ok := post_data.(io.Reader); ok {
		data = reader
	}

	req, err := http.NewRequest(method, requrl, data)
	if err != nil {
		return nil, err
	}

	if nil != data && "post" == strings.ToLower(method) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if len(headers) > 0 {
		header := headers[0]
		for k, v := range header {
			req.Header.Set(k, v[0])
			//不支持header中设置host(go 当前版本1.14.2) ???
			if strings.ToLower(k) == "host" {
				req.Host = v[0]
			}
		}
	}

	resp, err := this.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &httpResponse{
		response: string(body),
		code:     resp.StatusCode,
		header:   resp.Header,
	}, nil
} // }}}
