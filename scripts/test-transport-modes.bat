@echo off
echo Testing Dev MCP Server Transport Modes
echo =====================================

echo.
echo 1. Testing default mode (should force to SSE):
echo Command: dev-mcp.exe mcp
echo.

echo 2. Testing explicit SSE mode:
echo Command: dev-mcp.exe mcp --sse
echo.

echo 3. Testing HTTP mode (should use SSE):
echo Command: dev-mcp.exe mcp --http
echo.

echo 4. Testing stdio mode (should force to SSE):
echo Command: dev-mcp.exe mcp --stdio
echo.

echo 5. Testing transport parameter:
echo Command: dev-mcp.exe mcp --transport sse
echo.

echo Available transport modes:
echo - sse (Server-Sent Events) - DEFAULT/FORCED
echo - http (HTTP-based, uses SSE implementation)
echo - stdio (Forced to SSE mode)
echo.

echo To run in debug mode, add --debug or -d flag
echo Example: dev-mcp.exe mcp --sse --debug
echo.

echo Note: All modes are currently forced to use SSE transport as requested