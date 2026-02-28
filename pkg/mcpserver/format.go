package mcpserver

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// toJSON 将任意对象序列化为 JSON 字符串
func toJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":"json marshal failed: %s"}`, err.Error())
	}
	return string(data)
}

// textResult 创建文本类型的 MCP 工具结果
func textResult(text string) *mcp.CallToolResult {
	return mcp.NewToolResultText(text)
}

// jsonResult 创建 JSON 类型的 MCP 工具结果
func jsonResult(v interface{}) *mcp.CallToolResult {
	return mcp.NewToolResultText(toJSON(v))
}

// errorResult 创建错误类型的 MCP 工具结果
func errorResult(format string, args ...interface{}) *mcp.CallToolResult {
	return mcp.NewToolResultError(fmt.Sprintf(format, args...))
}
