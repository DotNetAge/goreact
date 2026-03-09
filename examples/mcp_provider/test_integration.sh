#!/bin/bash

# 测试 Engine 集成 MCP Provider 的简化示例

echo "=== Engine with MCP Provider Integration Test ==="
echo ""

# 1. 编译并启动 Mock Server
echo "Starting Mock MCP Server..."
cd server
go build -o /tmp/mock_mcp_server main.go
/tmp/mock_mcp_server > /tmp/mcp_server.log 2>&1 &
SERVER_PID=$!
echo "Server PID: $SERVER_PID"
sleep 2

# 2. 检查服务器
HEALTH=$(curl -s http://localhost:55987/health 2>/dev/null)
if [ $? -ne 0 ]; then
    echo "❌ Server failed to start"
    cat /tmp/mcp_server.log
    exit 1
fi
echo "✓ Server is running"
echo ""

# 3. 运行简化的集成示例
echo "Running simple integration example..."
cd ..
go run simple_integration.go
EXIT_CODE=$?

# 4. 清理
echo ""
echo "Cleaning up..."
kill $SERVER_PID 2>/dev/null
echo "✓ Server stopped"

if [ $EXIT_CODE -eq 0 ]; then
    echo ""
    echo "✅ Integration test passed!"
else
    echo ""
    echo "❌ Integration test failed"
fi

exit $EXIT_CODE
