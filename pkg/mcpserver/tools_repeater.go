package mcpserver

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/yhy0/ChYing/pkg/Jie/pkg/protocols/httpx"
)

// --- send_request ---

func sendRequestTool() mcp.Tool {
	return mcp.NewTool("send_request",
		mcp.WithDescription(`Send a raw HTTP request (Repeater). Useful for testing and verifying vulnerabilities.

Example raw_request:
GET /api/users HTTP/1.1
Host: example.com
Cookie: session=abc123

Example with POST body:
POST /api/login HTTP/1.1
Host: example.com
Content-Type: application/json

{"username":"admin","password":"test"}`),
		mcp.WithString("target",
			mcp.Required(),
			mcp.Description("Target URL including scheme (e.g., 'https://example.com')"),
		),
		mcp.WithString("raw_request",
			mcp.Required(),
			mcp.Description("Raw HTTP request text (headers and optional body)"),
		),
	)
}

func handleSendRequest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	target, err := req.RequireString("target")
	if err != nil {
		return errorResult("target is required"), nil
	}

	rawRequest, err := req.RequireString("raw_request")
	if err != nil {
		return errorResult("raw_request is required"), nil
	}

	resp, reqErr := httpx.Raw(rawRequest, target)
	if reqErr != nil {
		return errorResult("request failed: %v", reqErr), nil
	}

	result := fmt.Sprintf("=== SENT REQUEST ===\n%s\n\n=== RESPONSE ===\nStatus: %s (Code: %d)\nContent-Length: %d\nTime: %.2fms\n\n%s",
		resp.RequestDump,
		resp.Status,
		resp.StatusCode,
		resp.ContentLength,
		resp.ServerDurationMs,
		resp.ResponseDump,
	)

	return textResult(result), nil
}
