package llmparser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LubyRuffy/localdumper/httpdumper"
	"github.com/tidwall/gjson"
)

// IsLLMRequest 判断是否是llm请求
// 1. 请求头Content-Type不是application/json
// 2. 请求体中没有model字段
// 3. 请求url包含/api/chat、/api/generate、/v1/chat/completions、/v1/completions、/api/v0/chat/completions、/api/v0/completions
func IsLLMRequest(req *httpdumper.Request) bool {
	if !strings.Contains(req.Header.Get("Content-Type"), "application/json") &&
		!gjson.GetBytes(req.Body, "model").Exists() {
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

type LLMMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	ToolCalls []LLMTool `json:"tool_calls"`
}

// ToolCallsString 将tool_calls转换为字符串用于打印
func (m *LLMMessage) ToolCallsString() string {
	toolCalls := ""
	if m.ToolCalls != nil {
		var toolCallInfo []string
		for _, toolCall := range m.ToolCalls {
			args := ""
			if toolCall.Function.Arguments != nil {
				json, _ := json.Marshal(toolCall.Function.Arguments)
				args = string(json)
			}
			if toolCall.Function.Parameters != nil {
				json, _ := json.Marshal(toolCall.Function.Parameters)
				args = string(json)
			}

			toolCallInfo = append(toolCallInfo, fmt.Sprintf("Tool call: %s(%s)", toolCall.Function.Name, args))
		}
		toolCalls = strings.Join(toolCallInfo, "\n")
	}
	return toolCalls
}

// LLMTool 工具
type LLMTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
		Arguments   map[string]any `json:"arguments"`
	} `json:"function"`
}

// LLMRequest 请求
type LLMRequest struct {
	Model string `json:"model"`

	// generate
	System string `json:"system"`
	Prompt string `json:"prompt"`

	// chat
	Messages []LLMMessage `json:"messages"`
	Tools    []LLMTool    `json:"tools"`
}

// ParseRequest 解析请求
func ParseRequest(req *httpdumper.Request) *LLMRequest {
	if !IsLLMRequest(req) {
		return nil
	}

	var llmReq LLMRequest
	if err := json.Unmarshal(req.Body, &llmReq); err != nil {
		return nil
	}
	return &llmReq
}

// ParseResponse 解析响应
func ParseResponse(resp *httpdumper.Response) *LLMResponse {
	var llmResp LLMResponse
	if err := json.Unmarshal(resp.Body, &llmResp); err != nil {
		return nil
	}
	return &llmResp
}

// LLMResponse 响应
type LLMResponse struct {
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
	Message   LLMMessage `json:"message"`
	Choices   []struct {
		Index        int        `json:"index"`
		FinishReason string     `json:"finish_reason"`
		Text         string     `json:"text"`
		Message      LLMMessage `json:"message"`
		Delta        LLMMessage `json:"delta"`
	} `json:"choices"`
}

// String 将响应转换为字符串用于打印
func (r *LLMResponse) String() string {
	response := ""
	if r.Message.Content != "" {
		response += r.Message.Content
	}
	if len(r.Message.ToolCalls) > 0 {
		response += r.Message.ToolCallsString()
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
				response += choice.Message.ToolCallsString()
			}
		}
	}

	return response
}
