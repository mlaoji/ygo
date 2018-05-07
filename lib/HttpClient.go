package lib

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"
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

func (this *HttpClient) Get(url string, headers ...http.Header) (*httpResponse, error) { // {{{
	return this.Request("GET", url, nil, headers...)
} //}}}

func (this *HttpClient) Post(url string, post_data io.Reader, headers ...http.Header) (*httpResponse, error) { // {{{
	return this.Request("POST", url, post_data, headers...)
} //}}}

func (this *HttpClient) Request(method, url string, post_data io.Reader, headers ...http.Header) (*httpResponse, error) { // {{{
	req, err := http.NewRequest(method, url, post_data)
	if err != nil {
		return nil, err
	}

	if nil != post_data && "post" == strings.ToLower(method) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if len(headers) > 0 {
		header := headers[0]
		for k, v := range header {
			req.Header.Set(k, v[0])
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
