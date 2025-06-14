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
	"github.com/google/gopacket"
)

type Notifier struct {
	llmRequests sync.Map
}

func isLLMRequest(req *httpdumper.Request) bool {
	if !strings.Contains(req.Header.Get("Content-Type"), "application/json") {
		return false
	}

	urls := []string{
		"/api/chat",                // ollama 对话
		"/api/generate",            // ollama 生成
		"/v1/chat/completions",     // openai 兼容的api，ollama/lmstudio
		"/v1/completions",          // openai 兼容的api，ollama/lmstudio
		"/api/v0/chat/completions", // lmstudio 对话
		"/api/v0/completions",      // lmstudio 生成
	}
	url := req.URL.String()
	for _, u := range urls {
		if strings.Contains(url, u) {
			return true
		}
	}
	return false
}

type llmMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	ToolCalls []llmTool `json:"tool_calls"`
}

type llmTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"function"`
}

type llmRequest struct {
	Model string `json:"model"`

	// generate
	System string `json:"system"`
	Prompt string `json:"prompt"`

	// chat
	Messages []llmMessage `json:"messages"`
	Tools    []llmTool    `json:"tools"`
}

type llmResponse struct {
	// /v1/chat/completions
	ID     string `json:"id"`
	Object string `json:"object"` // chat.completion.chunk

	// /api/chat
	// /api/generate
	// /v1/completions
	Model     string     `json:"model"`
	CreatedAt any        `json:"created_at"`
	Response  string     `json:"response"`
	Done      bool       `json:"done"`
	Message   llmMessage `json:"message"`
	Choices   []struct {
		Index        int        `json:"index"`
		FinishReason string     `json:"finish_reason"`
		Text         string     `json:"text"`
		Message      llmMessage `json:"message"`
		Delta        llmMessage `json:"delta"`
	} `json:"choices"`
}

func (r *llmResponse) String() string {
	response := ""
	if r.Message.Content != "" {
		response += r.Message.Content
	}
	if len(r.Message.ToolCalls) > 0 {
		for _, tool := range r.Message.ToolCalls {
			response += fmt.Sprintf("Tool call: %s\n", tool.Function.Name)
		}
	}
	if r.Response != "" {
		response += r.Response
	}
	if len(r.Choices) > 0 {
		for _, choice := range r.Choices {
			if choice.Delta.Content != "" {
				response += choice.Delta.Content
			}
			if choice.Text != "" {
				response += choice.Text
			}
			if choice.Message.Content != "" {
				response += choice.Message.Content
			}
			if len(choice.Message.ToolCalls) > 0 {
				for _, tool := range choice.Message.ToolCalls {
					response += fmt.Sprintf("Tool call: %s\n", tool.Function.Name)
				}
			}
		}
	}

	return response
}

func (n *Notifier) OnRequest(req *httpdumper.Request) {
	// 目前请求大模型基本都是json
	// 同时url相对比较固定
	if isLLMRequest(req) {
		n.llmRequests.Store(req.ID, time.Now())

		var llmReq llmRequest
		if err := json.Unmarshal(req.Body, &llmReq); err != nil {
			fmt.Printf("Failed to unmarshal request: %s\n", err)
			return
		}

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
						fmt.Printf("Tool: %s\n", msg.Content)
					case "assistant":
						fmt.Printf("Assistant: %s\n", msg.Content)
					default:
						fmt.Printf("Message: %s\n", msg.Content)
					}
				}
			}
			if len(llmReq.Tools) > 0 {
				for _, tool := range llmReq.Tools {
					fmt.Printf("Tool: %s\n", tool.Function.Name)
				}
			}
		}
		fmt.Println(strings.Repeat(">", 58))
	}
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
				var llmResp llmResponse
				if err := json.Unmarshal([]byte(line), &llmResp); err != nil {
					continue
				}
				response += llmResp.String()
			}
			fmt.Printf("%s\n", response)
		} else if strings.HasPrefix(ct, "application/json") {
			var llmResp llmResponse
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

				var llmResp llmResponse
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
