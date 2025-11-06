@echo off
echo Testing MCP initialization...

echo {"method":"initialize","params":{"protocolVersion":"1.0.0","capabilities":[],"clientInfo":{"name":"test-client","version":"1.0.0"}}} | dev-mcp.exe mcp

echo.
echo Testing tools list...

echo {"method":"tools/list","id":1} | dev-mcp.exe mcp

echo.
echo Testing database query tool...

echo {"method":"tools/call","params":{"name":"database_query","arguments":{"query":"SELECT * FROM users LIMIT 1"}},"id":2} | dev-mcp.exe mcp

echo.
echo Testing Loki query tool...

echo {"method":"tools/call","params":{"name":"loki_query","arguments":{"query":"{job=\"example\"}"}},"id":3} | dev-mcp.exe mcp

echo.
echo Testing S3 query tool...

echo {"method":"tools/call","params":{"name":"s3_query","arguments":{"url":"s3://example-bucket/example-key.json"}},"id":4} | dev-mcp.exe mcp

echo.
echo Testing Sentry query tool...

echo {"method":"tools/call","params":{"name":"sentry_query","arguments":{}},"id":5} | dev-mcp.exe mcp

echo.
echo Testing Swagger query tool...

echo {"method":"tools/call","params":{"name":"swagger_query","arguments":{}},"id":6} | dev-mcp.exe mcp

echo.
echo Testing LLM tool...

echo {"method":"tools/call","params":{"name":"llm_chat","arguments":{"prompt":"Hello, world!"}},"id":7} | dev-mcp.exe mcp

echo.
echo Testing HTTP request tool...

echo {"method":"tools/call","params":{"name":"http_request","arguments":{"url":"https://httpbin.org/get"}},"id":8} | dev-mcp.exe mcp