package httpdumper

import (
	"github.com/google/gopacket"
	"github.com/google/uuid"
	"net/http"
)

// Config http dumper的配置
type Config struct {
	Device          string `json:"device"`          // 设备接口，比如lo0
	PcapFile        string `json:"pcapFile"`        // pcap本地文件，跟Device冲突，必须二选一
	BPFFilter       string `json:"bpfFilter"`       // 抓包语法过滤器
	PromiscuousMode bool   `json:"promiscuousMode"` // 混杂模式，默认本地抓包就不需要

	snapLen int // 最多获取多长的数据包，这里必须是0，所有包都获取，不然http解析就被截断了。不能直接设置，仅用于调试
}

// Notifier 通知器
type Notifier interface {
	OnTcpSession(id string, net, transport gopacket.Flow) // 新的TCP会话，只通知一次
	OnRequest(req *Request)                               // http请求
	OnResponse(resp *Response)                            // http响应
}

// Request http请求
type Request struct {
	*http.Request

	ID             string
	Net, Transport gopacket.Flow
	Body           []byte
	processedBody  bool
}

// SetBody 设置请求体，只能设置一次
func (r *Request) SetBody(body []byte) {
	if r.processedBody {
		return
	}
	r.processedBody = true
	r.Body = body
}

// NewRequest 创建一个请求
func NewRequest(req *http.Request, net, transport gopacket.Flow) *Request {
	return &Request{
		ID:        uuid.New().String(),
		Request:   req,
		Net:       net,
		Transport: transport,
	}
}

// Response http响应
type Response struct {
	*http.Response

	Request        *Request
	Net, Transport gopacket.Flow
	Body           []byte
	processedBody  bool
}

// SetBody 设置响应体，只能设置一次
func (r *Response) SetBody(body []byte) {
	if r.processedBody {
		return
	}
	r.processedBody = true
	r.Body = body
}

// NewResponse 创建一个响应
func NewResponse(request *Request, resp *http.Response, net, transport gopacket.Flow) *Response {
	return &Response{
		Request:   request,
		Response:  resp,
		Net:       net,
		Transport: transport,
	}
}
