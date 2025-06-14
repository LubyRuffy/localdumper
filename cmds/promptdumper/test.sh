# chat/generate  stream/non-stream tool/no-tool system/no-system think/no-think
# 至少16个组合测试

# ollama chat stream no-tool system think
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "system", "content": "You are a helpful assistant."}, {"role": "user", "content": "Hello, how are you?"}]}'

# ollama chat stream no-tool system no-think
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "system", "content": "You are a helpful assistant."}, {"role": "user", "content": "Hello, how are you?"}], "think": false}'

# ollama chat stream no-tool no-system think
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "user", "content": "Hello, how are you?"}]}'

# ollama chat stream no-tool no-system no-think
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "user", "content": "Hello, how are you?"}], "think": false}'

# ollama chat non-stream no-tool no-system no-think
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "user", "content": "Hello, how are you?"}], "think": false, "stream": false}'

# ollama chat non-stream tool no-system think
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:4b", "messages": [{"role": "user", "content": "what is the current time?"}], "think": true, "tools": [{"type": "function", "function": {"name": "get_current_time", "description": "Get the current time", "parameters": {"type": "object", "properties": {"timezone": {"type": "string", "description": "The timezone to get the time in"}}}}}]}'

# ollama chat non-stream tool no-system no-think
curl -X POST http://localhost:11434/api/chat -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "user", "content": "what is the current time?"}], "think": false, "tools": [{"type": "function", "function": {"name": "get_current_time", "description": "Get the current time", "parameters": {"type": "object", "properties": {"timezone": {"type": "string", "description": "The timezone to get the time in"}}}}}], "think": false}'

# ollama chat non-stream tool system think


# ollama generate stream no-tool system think
curl -X POST http://localhost:11434/api/generate -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "prompt": "Hello, how are you?"}'

# openai chat stream no-tool system think
curl -X POST http://localhost:11434/v1/chat/completions -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "system", "content": "You are a helpful assistant."}, {"role": "user", "content": "Hello, how are you?"}], "stream": true}'

# openai chat stream no-tool system no-think
curl -X POST http://localhost:11434/v1/chat/completions -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "messages": [{"role": "system", "content": "You are a helpful assistant."}, {"role": "user", "content": "Hello, how are you?"}], "stream": true, "think": false}'

# openai chat stream no-tool no-system think


# openai generate
curl -X POST http://localhost:11434/v1/completions -H "Content-Type: application/json" -d '{"model": "qwen3:0.6b", "prompt": "Hello, how are you?"}'

# lmstudio chat