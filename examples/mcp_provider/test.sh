#!/bin/bash

# 测试 MCP Provider 的脚本

echo "=== MCP Provider Test Script ==="
echo ""

# 1. 编译 Mock Server
echo "Building Mock MCP Server..."
cd server
go build -o /tmp/mock_mcp_server main.go
if [ $? -ne 0 ]; then
    echo "❌ Failed to build server"
    exit 1
fi
echo "✓ Server built successfully"
echo ""

# 2. 启动 Mock Server（后台）
echo "Starting Mock MCP Server on port 55987..."
/tmp/mock_mcp_server > /tmp/mcp_server.log 2>&1 &
SERVER_PID=$!
echo "Server PID: $SERVER_PID"
sleep 2

# 3. 检查服务器是否启动
echo "Checking server health..."
HEALTH=$(curl -s http://localhost:55987/health 2>/dev/null)
if [ $? -eq 0 ]; then
    echo "✓ Server is healthy"
    echo "Response: $HEALTH"
else
    echo "❌ Server health check failed"
    cat /tmp/mcp_server.log
    kill $SERVER_PID 2>/dev/null
    exit 1
fi
echo ""

# 4. 测试工具列表
echo "Testing /tools endpoint..."
TOOLS=$(curl -s http://localhost:55987/tools 2>/dev/null)
if [ $? -eq 0 ]; then
    echo "✓ Tools endpoint working"
    echo "$TOOLS" | python3 -m json.tool 2>/dev/null || echo "$TOOLS"
else
    echo "❌ Tools endpoint failed"
fi
echo ""

# 5. 运行客户端
echo "Running MCP Provider client..."
cd ..
go run main.go
CLIENT_EXIT=$?
echo ""

# 6. 清理
echo "Cleaning up..."
kill $SERVER_PID 2>/dev/null
echo "✓ Server stopped"

if [ $CLIENT_EXIT -eq 0 ]; then
    echo ""
    echo "✅ Test completed successfully!"
else
    echo ""
    echo "❌ Test failed with exit code $CLIENT_EXIT"
    echo "Server logs:"
    cat /tmp/mcp_server.log
fi

exit $CLIENT_EXIT
