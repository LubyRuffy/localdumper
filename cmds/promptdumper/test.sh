# chat/generate  stream/non-stream tool/no-tool system/no-system think/no-think single/multi
# 至少32个组合测试

# ollama chat stream no-tool system think single
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "system", "content": "You are a helpful assistant."}, {"role": "user", "content": "Hello, how are you?"}]}'

# ollama chat stream no-tool system think multi
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "system", "content": "You are a helpful assistant."}, {"role": "user", "content": "Hello, how are you?"}, {"role": "assistant", "content": "I am good, thank you!"}, {"role": "user", "content": "What is the capital of France?"}], "think": true}'

# ollama chat stream tool system think multi
curl http://localhost:11434/api/chat \
-X POST \
-H "Content-Type: application/json" \
-d @- <<'EOF'
{
  "model": "qwen3:0.6b",
  "stream": true,
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_current_weather",
        "description": "获取指定城市的当前天气信息",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {
              "type": "string",
              "description": "城市名称, 例如: 北京"
            }
          },
          "required": ["location"]
        }
      }
    }
  ],
  "messages": [
    {
      "role": "user",
      "content": "你好"
    },
    {
      "role": "assistant",
      "content": "你好！有什么可以帮助你的吗？"
    },
    {
      "role": "user",
      "content": "北京今天天气怎么样？"
    },
    {
      "role": "assistant",
      "content": null,
      "tool_calls": [
        {
          "function": {
            "name": "get_current_weather",
            "arguments": { "location": "Beijing" }
          }
        }
      ]
    },
    {
      "role": "tool",
      "content": "{\"temperature\": \"25\", \"unit\": \"celsius\", \"description\": \"晴朗\"}"
    },
    {
      "role": "assistant",
      "content": "北京今天气温25摄氏度，天气晴朗。需要查询其他城市的天气吗？"
    },
    {
      "role": "user",
      "content": "hi"
    }
  ]
}
EOF

# ollama content-type 不是json，但是有model字段，工具动态返回
curl http://localhost:11434/api/chat -d '{
  "model": "qwen3",
  "messages": [
    {
      "role": "user",
      "content": "What is the weather today in Toronto? /no_think"
    }
  ],
  "stream": true,
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_current_weather",
        "description": "Get the current weather for a location",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {
              "type": "string",
              "description": "The location to get the weather for, e.g. San Francisco, CA"
            },
            "format": {
              "type": "string",
              "description": "The format to return the weather in, e.g. celsius or fahrenheit",
              "enum": ["celsius", "fahrenheit"]
            }
          },
          "required": ["location", "format"]
        }
      }
    }
  ]
}'

# ollama chat stream no-tool system no-think single
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "system", "content": "You are a helpful assistant."}, {"role": "user", "content": "Hello, how are you?"}], "think": false}'

# ollama chat stream no-tool no-system think single
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "user", "content": "Hello, how are you?"}]}'

# ollama chat stream no-tool no-system no-think single
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "user", "content": "Hello, how are you?"}], "think": false}'

# ollama chat non-stream no-tool no-system no-think single
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "user", "content": "Hello, how are you?"}], "think": false, "stream": false}'

# ollama chat non-stream tool no-system think single
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:4b", "messages": [{"role": "user", "content": "what is the current time?"}], "think": true, "tools": [{"type": "function", "function": {"name": "get_current_time", "description": "Get the current time", "parameters": {"type": "object", "properties": {"timezone": {"type": "string", "description": "The timezone to get the time in"}}}}}]}'

# ollama chat non-stream tool no-system no-think single
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "user", "content": "what is the current time?"}], "think": false, "tools": [{"type": "function", "function": {"name": "get_current_time", "description": "Get the current time", "parameters": {"type": "object", "properties": {"timezone": {"type": "string", "description": "The timezone to get the time in"}}}}}], "think": false}'

# ollama chat non-stream tool system think single


# ollama generate stream no-tool system think single
curl -X POST http://localhost:11434/api/generate -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "prompt": "Hello, how are you?"}'

# openai chat stream no-tool system think single
curl -X POST http://localhost:11434/v1/chat/completions -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "system", "content": "You are a helpful assistant."}, {"role": "user", "content": "Hello, how are you?"}], "stream": true}'

# openai chat stream no-tool system no-think single
curl -X POST http://localhost:11434/v1/chat/completions -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "system", "content": "You are a helpful assistant."}, {"role": "user", "content": "Hello, how are you?"}], "stream": true, "think": false}'

# openai chat stream tool system no-think multi
curl http://localhost:11434/v1/chat/completions \
-X POST \
-H "Content-Type: application/json" \
-d @- <<'EOF'
{
  "model": "qwen3:0.6b",
  "stream": true,
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_current_weather",
        "description": "获取指定城市的当前天气信息",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {
              "type": "string",
              "description": "城市名称, 例如: 北京"
            }
          },
          "required": ["location"]
        }
      }
    }
  ],
  "messages": [
    {
      "role": "system",
      "content": "You are a helpful assistant. /no_think"
    },
    {
      "role": "user",
      "content": "你好"
    },
    {
      "role": "assistant",
      "content": "你好！有什么可以帮助你的吗？"
    },
    {
      "role": "user",
      "content": "北京今天天气怎么样？"
    }
  ]
}
EOF

# openai chat stream no-tool no-system think single
curl -X POST http://localhost:11434/v1/chat/completions -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "user", "content": "Hello, how are you?"}], "stream": true, "think": false}'


# openai generate single
curl -X POST http://localhost:11434/v1/completions -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "prompt": "Hello, how are you?"}'

# lmstudio chat