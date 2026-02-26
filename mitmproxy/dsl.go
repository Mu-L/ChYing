package mitmproxy

import (
	"bufio"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/projectdiscovery/dsl"
	"github.com/yhy0/ChYing/pkg/db"
)

/**
   @author yhy
   @since 2025/6/4
   @desc DSL查询功能实现
   该文件实现了基于DSL(Domain Specific Language)的HTTP历史记录查询
   利用proxify的DSL引擎匹配HTTP请求/响应，为前端提供强大的查询能力
**/

// 注意：DSL查询数据源为SQLite数据库中的HTTPHistory表和Traffic表
// 不再依赖内存中的HTTPBodyMap

// QueryHistoryByDSL 根据提供的DSL查询字符串过滤HTTP历史记录
// dslQuery: DSL查询表达式
// 返回匹配的HTTP历史记录摘要列表
func QueryHistoryByDSL(dslQuery string) ([]HTTPHistory, error) {
	if dslQuery == "" {
		// 如果DSL查询为空，返回空列表
		// 这样前端可以区分"清除过滤器"和"执行查询"的情况
		return []HTTPHistory{}, nil
	}

	// 验证DSL表达式
	if err := validateDSL(dslQuery); err != nil {
		return nil, fmt.Errorf("DSL语法错误: %w", err)
	}

	// 从数据库获取所有HTTPHistory记录
	histories, err := db.GetAllHistory("", "", 0, 0)
	if err != nil {
		return nil, fmt.Errorf("查询数据库失败: %w", err)
	}

	var matchedHistory []HTTPHistory

	// 遍历所有历史记录并应用DSL匹配
	for _, history := range histories {
		// 跳过 Hid 为 0 的记录（数据异常，无法关联请求/响应）
		if history.Hid == 0 {
			continue
		}

		// 通过 Hid 获取对应的请求和响应原始报文
		req, res := db.GetTraffic(int(history.Hid))

		var requestRaw, responseRaw string
		if req != nil {
			requestRaw = req.RequestRaw
		}
		if res != nil {
			responseRaw = res.ResponseRaw
		}

		// 解析请求和响应以构建DSL上下文
		requestData := parseRawHTTP(requestRaw, true)
		responseData := parseRawHTTP(responseRaw, false)

		// 合并请求和响应数据，构建完整的DSL评估上下文
		dslContext := mergeDSLContext(history, requestData, responseData)

		// 评估DSL表达式
		matched, err := evaluateDSL(dslQuery, dslContext)
		if err != nil {
			log.Printf("评估ID为%d的请求时DSL错误: %v", history.ID, err)
			continue
		}

		// 如果结果为true，则此历史记录匹配查询条件
		if matched {
			matchedHistory = append(matchedHistory, dbHistoryToMitmHistory(history))
		}
	}

	// 按照ID排序结果
	sort.Slice(matchedHistory, func(i, j int) bool {
		return matchedHistory[i].Id < matchedHistory[j].Id
	})

	return matchedHistory, nil
}

// 辅助函数，用于验证DSL表达式
func validateDSL(dslQuery string) error {
	// 创建一个包含所有可能字段的模拟上下文，用于验证DSL表达式
	mockContext := map[string]interface{}{
		// 摘要字段
		"id":           int64(1),
		"url":          "https://example.com/api/test",
		"path":         "/api/test",
		"method":       "GET",
		"host":         "example.com",
		"status":       "200",
		"length":       "1024",
		"content_type": "application/json",
		"timestamp":    "2023-01-01T12:00:00Z",

		// 请求相关字段
		"request":         "GET /api/test HTTP/1.1\r\nHost: example.com\r\n\r\n",
		"request_body":    "test body",
		"request_headers": map[string]string{"host": "example.com"},

		// 响应相关字段
		"response":         "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"status\":\"ok\"}",
		"response_body":    "{\"status\":\"ok\"}",
		"response_headers": map[string]string{"content-type": "application/json"},
		"status_reason":    "OK",
	}

	// 使用完整的模拟上下文测试DSL表达式
	_, err := dsl.EvalExpr(dslQuery, mockContext)
	return err
}

// 辅助函数，用于评估DSL表达式
func evaluateDSL(dslQuery string, context map[string]interface{}) (bool, error) {
	// 使用 projectdiscovery/dsl 包评估表达式
	result, err := dsl.EvalExpr(dslQuery, context)
	if err != nil {
		return false, err
	}

	if resultBool, ok := result.(bool); ok {
		return resultBool, nil
	}
	return false, fmt.Errorf("DSL结果不是布尔值: %v", result)
}

// parseRawHTTP 解析原始HTTP请求/响应字符串
// 返回可用于DSL匹配的键值对
func parseRawHTTP(rawHTTP string, isRequest bool) map[string]interface{} {
	if rawHTTP == "" {
		return make(map[string]interface{})
	}

	result := make(map[string]interface{})

	// 添加原始内容以支持直接对整个内容进行匹配
	if isRequest {
		result["request"] = rawHTTP
	} else {
		result["response"] = rawHTTP
	}

	// 分离头部和主体
	parts := strings.SplitN(rawHTTP, "\r\n\r\n", 2)
	headers := parts[0]
	var body string
	if len(parts) > 1 {
		body = parts[1]
	}

	// 添加主体内容
	if isRequest {
		result["request_body"] = body
	} else {
		result["response_body"] = body
	}

	// 解析请求/响应行和头部
	scanner := bufio.NewScanner(strings.NewReader(headers))
	lineNum := 0
	headerMap := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()
		if lineNum == 0 {
			// 处理请求行或状态行
			if isRequest {
				parts := strings.SplitN(line, " ", 3)
				if len(parts) >= 2 {
					result["method"] = parts[0]
					result["path"] = parts[1]
					if len(parts) > 2 {
						result["http_version"] = parts[2]
					}
				}
			} else {
				parts := strings.SplitN(line, " ", 3)
				if len(parts) >= 3 {
					result["status"] = parts[1]
					result["status_reason"] = parts[2]
				}
			}
		} else {
			// 处理头部
			colonIdx := strings.IndexByte(line, ':')
			if colonIdx > 0 {
				key := strings.TrimSpace(line[:colonIdx])
				value := strings.TrimSpace(line[colonIdx+1:])
				headerMap[strings.ToLower(key)] = value

				// 提取一些重要头部作为独立字段
				lowerKey := strings.ToLower(key)
				if lowerKey == "host" {
					result["host"] = value
				} else if lowerKey == "content-type" {
					result["content_type"] = value
				} else if lowerKey == "content-length" {
					result["content_length"] = value
				}
			}
		}
		lineNum++
	}

	// 添加所有头部
	if isRequest {
		result["request_headers"] = headerMap
	} else {
		result["response_headers"] = headerMap
	}

	return result
}

// mergeDSLContext 合并请求摘要、请求详情和响应详情，构建完整的DSL上下文
func mergeDSLContext(summary *db.HTTPHistory, requestData, responseData map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	// 添加从HTTP摘要中提取的字段
	merged["id"] = summary.ID
	merged["url"] = summary.FullUrl
	merged["path"] = summary.Path
	merged["method"] = summary.Method
	merged["host"] = summary.Host
	merged["status"] = summary.Status
	merged["length"] = summary.Length
	merged["content_type"] = summary.ContentType
	merged["timestamp"] = summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00")

	// 合并请求和响应数据
	for k, v := range requestData {
		merged[k] = v
	}
	for k, v := range responseData {
		merged[k] = v
	}

	return merged
}

// dbHistoryToMitmHistory 将数据库 HTTPHistory 转换为 mitmproxy HTTPHistory
func dbHistoryToMitmHistory(h *db.HTTPHistory) HTTPHistory {
	return HTTPHistory{
		Id:          h.ID,
		Host:        h.Host,
		Method:      h.Method,
		FullUrl:     h.FullUrl,
		Path:        h.Path,
		Status:      h.Status,
		Length:      h.Length,
		ContentType: h.ContentType,
		MIMEType:    h.MIMEType,
		Extension:   h.Extension,
		Title:       h.Title,
		IP:          h.IP,
		Note:        h.Note,
		Color:       h.Color,
		Timestamp:   h.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// 以下是用于测试的便捷函数，可以在开发环境中使用，生产环境可以移除

// ParseHTTPRequestForDSL 解析HTTP请求供DSL使用
func ParseHTTPRequestForDSL(reqRaw string) (map[string]interface{}, error) {
	parsedMap := parseRawHTTP(reqRaw, true)
	return parsedMap, nil
}

// ParseHTTPResponseForDSL 解析HTTP响应供DSL使用
func ParseHTTPResponseForDSL(respRaw string) (map[string]interface{}, error) {
	parsedMap := parseRawHTTP(respRaw, false)
	return parsedMap, nil
}

// TestDSL 测试DSL表达式是否有效
func TestDSL(dslQuery string, context map[string]interface{}) (bool, error) {
	return evaluateDSL(dslQuery, context)
}

