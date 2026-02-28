package mcpserver

import (
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
	"github.com/yhy0/ChYing/conf"
	"github.com/yhy0/logging"
)

// NewChYingMCPServer 创建并注册所有 MCP tools
func NewChYingMCPServer() *server.MCPServer {
	s := server.NewMCPServer(
		"ChYing Security Platform",
		conf.Version,
		server.WithToolCapabilities(true),
	)

	// 查询类工具
	s.AddTool(getHttpHistoryTool(), handleGetHttpHistory)
	s.AddTool(getTrafficDetailTool(), handleGetTrafficDetail)
	s.AddTool(queryByDSLTool(), handleQueryByDSL)
	s.AddTool(getHostsTool(), handleGetHosts)
	s.AddTool(getTrafficByHostTool(), handleGetTrafficByHost)
	s.AddTool(getVulnerabilitiesTool(), handleGetVulnerabilities)
	s.AddTool(getStatisticsTool(), handleGetStatistics)

	// 主动测试类工具
	s.AddTool(sendRequestTool(), handleSendRequest)
	s.AddTool(runIntruderTool(), handleRunIntruder)

	// 工具类
	s.AddTool(getCurrentProjectTool(), handleGetCurrentProject)

	return s
}

// StartHTTPServer 启动 MCP HTTP SSE Server，仅绑定到 localhost
func StartHTTPServer(port int) {
	s := NewChYingMCPServer()
	httpServer := server.NewStreamableHTTPServer(s)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	logging.Logger.Infof("MCP server listening on %s", addr)

	mux := http.NewServeMux()
	mux.Handle("/mcp", httpServer)

	if err := http.ListenAndServe(addr, mux); err != nil {
		logging.Logger.Errorf("MCP server error: %v", err)
	}
}
