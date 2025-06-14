package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/LubyRuffy/localdumper/httpdumper"
	"github.com/LubyRuffy/localdumper/llmparser"
	"github.com/google/gopacket"
)

type Notifier struct {
	llmRequests sync.Map
}

func (n *Notifier) OnRequest(req *httpdumper.Request) {
	// 目前请求大模型基本都是json
	// 同时url相对比较固定
	llmReq := llmparser.ParseRequest(req)
	if llmReq == nil {
		return
	}

	n.llmRequests.Store(req.ID, time.Now())

	fmt.Println(strings.Repeat(">", 58))
	fmt.Printf("New request: %s\n", req.URL.String())

	if llmReq.Model != "" {
		fmt.Printf("Model: %s\n", llmReq.Model)
		if llmReq.System != "" {
			fmt.Printf("System: %s\n", llmReq.System)
		}
		if llmReq.Prompt != "" {
			fmt.Printf("Prompt: %s\n", llmReq.Prompt)
		}
		if len(llmReq.Messages) > 0 {
			for _, msg := range llmReq.Messages {
				switch msg.Role {
				case "system":
					fmt.Printf("System: %s\n", msg.Content)
				case "user":
					fmt.Printf("User: %s\n", msg.Content)
				case "tool":
					fmt.Printf("Tool response: %s\n", msg.Content)
				case "assistant":
					fmt.Printf("Assistant: ")
					if msg.Content != "" {
						fmt.Printf("%s\n", msg.Content)
					}
					if msg.ToolCalls != nil {
						fmt.Printf("%s\n", msg.ToolCallsString())
					}
				default:
					fmt.Printf("Message: %s\n", msg.Content)
				}
			}
		}
		// if len(llmReq.Tools) > 0 {
		// 	for _, tool := range llmReq.Tools {
		// 		fmt.Printf("Tool: %s\n", tool.Function.Name)
		// 	}
		// }
	}
	fmt.Println(strings.Repeat(">", 58))

}

func (n *Notifier) OnResponse(resp *httpdumper.Response) {
	// 对应的请求是llm请求
	if _, ok := n.llmRequests.Load(resp.Request.ID); ok {
		fmt.Println(strings.Repeat("<", 58))
		fmt.Printf("New response: %s\n", resp.Request.URL)
		ct := resp.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "application/x-ndjson") {
			response := ""
			lines := strings.Split(string(resp.Body), "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}
				var llmResp llmparser.LLMResponse
				if err := json.Unmarshal([]byte(line), &llmResp); err != nil {
					continue
				}
				response += llmResp.String()
			}
			fmt.Printf("%s\n", response)
		} else if strings.HasPrefix(ct, "application/json") {
			var llmResp llmparser.LLMResponse
			if err := json.Unmarshal(resp.Body, &llmResp); err != nil {
				return
			}
			fmt.Printf("%s\n", llmResp.String())
		} else if strings.HasPrefix(ct, "text/event-stream") {
			response := ""
			lines := strings.Split(string(resp.Body), "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}
				if !strings.HasPrefix(line, "data:") {
					continue
				}
				line = strings.TrimPrefix(line, "data:")
				line = strings.TrimSpace(line)
				if line == "" || line == "[DONE]" {
					continue
				}

				var llmResp llmparser.LLMResponse
				if err := json.Unmarshal([]byte(line), &llmResp); err != nil {
					log.Printf("Failed to unmarshal response: %s\n", err)
					continue
				}
				response += llmResp.String()
			}
			fmt.Printf("%s\n", response)
		} else {
			fmt.Printf("unknown content type: %s\n", ct)
		}
		fmt.Println(strings.Repeat("<", 58))

		n.llmRequests.Delete(resp.Request.ID)
	}
}

func (n *Notifier) OnTcpSession(id string, net, transport gopacket.Flow) {
	fmt.Printf("New TCP session: %s\n", id)
}

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	hd := httpdumper.New(&httpdumper.Config{
		Device:    "lo0", // 本地抓包
		BPFFilter: "tcp and (port 11434 or port 1234)",
	}, &Notifier{})
	doneChan := make(chan struct{}, 1)
	go func() {
		defer close(doneChan)
		if err := hd.Start(context.Background()); err != nil {
			panic(err)
		}
	}()

	<-signalChan
	fmt.Println("\nReceived interrupt, shutting down...")
	hd.Stop()
	<-doneChan
}
