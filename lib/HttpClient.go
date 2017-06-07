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
	timeout := 3
	if len(timeouts) > 0 {
		timeout = timeouts[0]
	}
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				conn, err := net.DialTimeout(netw, addr, time.Second*time.Duration(timeout))
				if err != nil {
					return nil, err
				}
				conn.SetDeadline(time.Now().Add(time.Second * time.Duration(timeout)))
				return conn, nil
			},
			ResponseHeaderTimeout: time.Second * time.Duration(timeout),
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
