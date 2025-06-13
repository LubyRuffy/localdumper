package httpdumper

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// tcpState 两端共享的状态信息
type tcpState struct {
	mutex    sync.Mutex
	discard  atomic.Bool
	requests []*Request // 请求列表
	reqIndex int        // 当前读取的偏移
}

func (ts *tcpState) appendRequest(req *Request) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	ts.requests = append(ts.requests, req)
}

func (ts *tcpState) getLastRequest() *Request {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	// 收到了response但是还没有request
	if len(ts.requests) < ts.reqIndex+1 {
		return nil
	}

	req := ts.requests[ts.reqIndex]
	ts.reqIndex++
	return req
}

// httpStreamFactory 实现了 tcpassembly.StreamFactory 接口
type httpStreamFactory struct {
	m        sync.Map
	wg       sync.WaitGroup
	notifier Notifier
}

type RequestOrResponse int

const (
	RequestOrResponseWait RequestOrResponse = iota
	RequestOrResponseRequest
	RequestOrResponseResponse
	RequestOrResponseError
)

// httpStream 用于处理一个独立的 TCP 流，它现在只包含 ReaderStream
type httpStream struct {
	tcpreader.ReaderStream

	id                string             // net和transport的连接关系的友好显示
	net, transport    gopacket.Flow      // 网络层和传输层的流
	isFirstPkt        bool               // 是否收到了第一个有数据的包
	requestOrResponse RequestOrResponse  // 类型，区分是request还是response
	factory           *httpStreamFactory // 创建工厂
	state             *tcpState          // 两端共享的状态
	closeOnce         sync.Once          // 确保ReassemblyComplete只被调用一次
}

// ReassemblyComplete implements tcpassembly.Stream's ReassemblyComplete function.
// 使用 sync.Once 来确保底层的 ReaderStream.ReassemblyComplete 只被调用一次。
func (r *httpStream) ReassemblyComplete() {
	r.closeOnce.Do(func() {
		r.ReaderStream.ReassemblyComplete()
	})
}

// Reassembled implements tcpassembly.Stream's Reassembled function.
func (r *httpStream) Reassembled(reassembly []tcpassembly.Reassembly) {
	if r.state.discard.Load() {
		return
	}

	//for _, pkt := range reassembly {
	//	log.Printf("reassembled: %s:%s -> %s:%s, %d bytes, skip=%v, start=%v, end=%v\n",
	//		r.net.Src(), r.transport.Src(), r.net.Dst(), r.transport.Dst(), len(pkt.Bytes), pkt.Skip, pkt.Start, pkt.End)
	//}

	// 检查第一个包
	if r.isFirstPkt && len(reassembly) > 0 && len(reassembly[0].Bytes) > 0 {
		r.isFirstPkt = false
		firstData := reassembly[0].Bytes

		if bytes.HasPrefix(firstData, []byte("HTTP/")) {
			r.requestOrResponse = RequestOrResponseResponse
		} else if bytes.HasPrefix(firstData, []byte("GET ")) ||
			bytes.HasPrefix(firstData, []byte("HEAD ")) ||
			bytes.HasPrefix(firstData, []byte("POST ")) {
			r.requestOrResponse = RequestOrResponseRequest
		} else {
			r.requestOrResponse = RequestOrResponseError
			// 不再进行处理
			log.Println("not http protocol:", r.id)
			r.state.discard.Store(true)
			r.ReassemblyComplete()
			return
		}
	}
	r.ReaderStream.Reassembled(reassembly)
}

func (s *httpStream) readRequest(buf *bufio.Reader) error {
	req, err := http.ReadRequest(buf)
	if err != nil {
		return err
	}

	newReq := NewRequest(req.Clone(context.Background()), s.net, s.transport)
	s.state.appendRequest(newReq)

	body, _ := io.ReadAll(req.Body)
	req.Body.Close()

	newReq.SetBody(body)

	s.factory.notifier.OnRequest(newReq)
	return nil
}

func (s *httpStream) readResponse(buf *bufio.Reader) error {
	_, err := buf.Peek(1)
	if err != nil {
		return err
	}

	req := s.state.getLastRequest()
	var rawReq *http.Request
	if req != nil {
		rawReq = req.Request
		i := 0
		for !req.processedBody && i < 10 {
			time.Sleep(time.Millisecond * 100)
			i++
		}
	}

	resp, err := http.ReadResponse(buf, rawReq)
	if err != nil {
		return err
	}
	newResp := NewResponse(req, resp, s.net, s.transport)

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	newResp.SetBody(body)

	s.factory.notifier.OnResponse(newResp)
	return nil
}

func (s *httpStream) run() {
	defer func() {
		log.Println("out:", s.id)
		s.factory.wg.Done()
	}()

	buf := bufio.NewReader(s)
	_, err := buf.Peek(1) // 必须读一次，确保s.requestOrResponse被设置

	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return
		}
		log.Println("buf peek failed:", err)
		return
	}

_out:
	for {
		switch s.requestOrResponse {
		case RequestOrResponseRequest:
			err := s.readRequest(buf)
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					return
				}
				log.Println("request buf read failed:", err)
				break _out
			}
		case RequestOrResponseResponse:
			err := s.readResponse(buf)
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					return
				}
				log.Println("response buf read failed:", err)
				break _out
			}
		default:
			s.ReassemblyComplete()
		}
	}
}

// createConnectionKey 创建一个不区分流向的唯一连接标识
func createConnectionKey(netFlow, tcpFlow gopacket.Flow) string {
	srcIP, dstIP := netFlow.Endpoints()
	srcPort, dstPort := tcpFlow.Endpoints()
	if srcIP.String() > dstIP.String() || (srcIP.String() == dstIP.String() && srcPort.String() > dstPort.String()) {
		srcIP, dstIP = dstIP, srcIP
		srcPort, dstPort = dstPort, srcPort
	}
	return fmt.Sprintf("%s:%s-%s:%s", srcIP, srcPort, dstIP, dstPort)
}

func (f *httpStreamFactory) getHttpStream(net, transport gopacket.Flow) *httpStream {
	id := createConnectionKey(net, transport)

	// 设置共享状态
	state, loaded := f.m.LoadOrStore(net.String()+":"+transport.String(), &tcpState{})
	f.m.LoadOrStore(net.Reverse().String()+":"+transport.Reverse().String(), state)

	if !loaded {
		f.notifier.OnTcpSession(id, net, transport)
	}

	return &httpStream{
		id:           net.String() + ":" + transport.String(),
		ReaderStream: tcpreader.NewReaderStream(),
		factory:      f,
		net:          net,
		transport:    transport,
		state:        state.(*tcpState),
		isFirstPkt:   true,
	}
}

// New 方法在检测到新的 TCP 流时被调用
// 这是关键的修改点：我们在这里判断流的方向
func (f *httpStreamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	s := f.getHttpStream(net, transport)
	f.wg.Add(1)
	go s.run()

	return s
}
