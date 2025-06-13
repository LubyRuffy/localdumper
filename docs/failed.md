# 异常记录

## 协议支持
### HEAD结果
HEAD的结果虽然有Content-Length，但是实际上不预期返回数据。
```shell
curl -I http://127.0.0.1:1234
HTTP/1.1 200 OK
X-Powered-By: Express
Content-Type: application/json; charset=utf-8
Content-Length: 51
ETag: W/"33-+zPRS4E3EuAAgNdSqtzYcWgWx/w"
Date: Tue, 10 Jun 2025 11:13:52 GMT
Connection: keep-alive
Keep-Alive: timeout=5
```
做个HEAD判断就好，但是由于双向组包，导致response不知道request是什么消息，导致失败。
所以这里有两个要求：
- response绑定request
- 处理request.method==head的response

### 没有抓到完整的包
注意设置snaplen为-1，可以用一个超过100k的来进行测试。

### keep-alive的两个HEAD测试
```shell
echo 'HEAD /v1 HTTP/1.1\r\nUser-Agent: curl/8.12.1\r\nHost: 127.0.0.1:1234\r\n\r\nHEAD /v1 HTTP/1.1\r\nUser-Agent: curl/8.12.1\r\nHost: 127.0.0.1:1234\r\n\r\n' | nc 127.0.0.1 1234
HTTP/1.1 200 OK
X-Powered-By: Express
Content-Type: application/json; charset=utf-8
Content-Length: 53
ETag: W/"35-ombJbBVWYI1pD1D0EeEu8p2TMmw"
Date: Tue, 10 Jun 2025 14:36:00 GMT
Connection: keep-alive
Keep-Alive: timeout=5

HTTP/1.1 200 OK
X-Powered-By: Express
Content-Type: application/json; charset=utf-8
Content-Length: 53
ETag: W/"35-ombJbBVWYI1pD1D0EeEu8p2TMmw"
Date: Tue, 10 Jun 2025 14:36:00 GMT
Connection: keep-alive
Keep-Alive: timeout=5
```
正常是2个包，但是抓包提示`malformed HTTP request`

```text
2025/06/10 23:01:59 Starting capture on interface: lo0
2025/06/10 23:01:59 Using BPF filter: tcp and port 1234
2025/06/10 23:01:59 Waiting for packets...
2025/06/10 23:02:01 new connection: 127.0.0.1:1234-127.0.0.1:55800
2025/06/10 23:02:01 new connection: 127.0.0.1:1234-127.0.0.1:55800
2025/06/10 23:02:01 first chunk: HEAD 
2025/06/10 23:02:01 isRequest: true
2025/06/10 23:02:01 request buf read failed: malformed HTTP request ""
2025/06/10 23:02:01 first chunk: HTTP/
2025/06/10 23:02:01 isRequest: false
----------------------------------------------------------
>>> HTTP Request: 127.0.0.1:55800 -> 127.0.0.1:1234
HEAD /v1 HTTP/1.1
Host: 127.0.0.1:1234
----------------------------------------------------------
----------------------------------------------------------
>>> HTTP Request: 127.0.0.1:55800 -> 127.0.0.1:1234
HEAD /v1 HTTP/1.1
Host: 127.0.0.1:1234
----------------------------------------------------------
----------------------------------------------------------
<<< HTTP Response: 127.0.0.1:55800 <- 127.0.0.1:1234
    HTTP/1.1 200 OK
    Keep-Alive: timeout=5
    X-Powered-By: Express
    Content-Type: application/json; charset=utf-8
    Content-Length: 53
    Etag: W/"35-ombJbBVWYI1pD1D0EeEu8p2TMmw"
    Date: Tue, 10 Jun 2025 15:02:01 GMT
    Connection: keep-alive

HTTP/1.1 200 OK
X-Powered-By: Express
Content-Type:
----------------------------------------------------------
2025/06/10 23:02:30 response buf read failed: malformed HTTP status code "application/json;"
```
可以看到正是因为不知道是HEAD包，导致解析失败。