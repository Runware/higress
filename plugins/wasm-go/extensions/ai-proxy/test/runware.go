package test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基本 Runware 配置
var basicRunwareConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "runware",
			"apiTokens": []string{"rw-test123456789"},
			"modelMapping": map[string]string{
				"*": "qwen3.5-9b",
			},
		},
	})
	return data
}()

// 测试配置：Runware 多模型配置
var runwareMultiModelConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "runware",
			"apiTokens": []string{"rw-multi-model"},
			"modelMapping": map[string]string{
				"gpt-4":         "qwen3.5-9b",
				"gpt-3.5-turbo": "qwen3.5-4b",
				"*":             "qwen3.5-4b",
			},
		},
	})
	return data
}()

// 测试配置：无效 Runware 配置（缺少 apiToken）
var invalidRunwareConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":         "runware",
			"apiTokens":    []string{},
			"modelMapping": map[string]string{},
		},
	})
	return data
}()

// 测试配置：完整 Runware 配置
var completeRunwareConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "runware",
			"apiTokens": []string{"rw-complete-test"},
			"modelMapping": map[string]string{
				"gpt-4":         "qwen3.5-9b",
				"gpt-3.5-turbo": "qwen3.5-4b",
				"*":             "qwen3.5-4b",
			},
		},
	})
	return data
}()

// RunRunwareParseConfigTests 测试 Runware 配置解析
func RunRunwareParseConfigTests(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试基本 Runware 配置解析
		t.Run("basic runware config", func(t *testing.T) {
			host, status := test.NewTestHost(basicRunwareConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试 Runware 多模型配置解析
		t.Run("runware multi model config", func(t *testing.T) {
			host, status := test.NewTestHost(runwareMultiModelConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试无效 Runware 配置（缺少 apiToken）
		t.Run("invalid runware config - missing apiToken", func(t *testing.T) {
			host, status := test.NewTestHost(invalidRunwareConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试完整 Runware 配置解析
		t.Run("runware complete config", func(t *testing.T) {
			host, status := test.NewTestHost(completeRunwareConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

// RunRunwareOnHttpRequestHeadersTests 测试 Runware 请求头处理
func RunRunwareOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Runware 聊天完成请求头处理
		t.Run("runware chat completion request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicRunwareConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 应该返回 HeaderStopIteration，因为需要处理请求体
			require.Equal(t, types.HeaderStopIteration, action)

			// 验证请求头是否被正确处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证 Host 是否被改为 Runware 域名
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost, "Host header should exist")
			require.Equal(t, "api.runware.ai", hostValue, "Host should be changed to Runware domain")

			// 验证 Authorization 是否被设置
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth, "Authorization header should exist")
			require.Contains(t, authValue, "Bearer rw-test123456789", "Authorization should contain Runware API token with Bearer prefix")

			// 验证 Path 保持 OpenAI 兼容格式
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			require.Equal(t, "/v1/chat/completions", pathValue, "Path should remain OpenAI compatible")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			hasRunwareLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "runware") || strings.Contains(log, "ai-proxy") {
					hasRunwareLogs = true
					break
				}
			}
			require.True(t, hasRunwareLogs, "Should have Runware or ai-proxy processing logs")
		})

		// 测试 Runware 模型列表请求头处理
		t.Run("runware models request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicRunwareConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/models"},
				{":method", "GET"},
			})

			// TODO: Due to the limitations of the test framework, we just treat it as a request with body here.
			//require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.HeaderStopIteration, action)

			// 验证请求头处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证 Host 转换
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost)
			require.Equal(t, "api.runware.ai", hostValue)

			// 验证 Path 保持 OpenAI 兼容格式
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath)
			require.Equal(t, "/v1/models", pathValue, "Path should remain OpenAI compatible for models")

			// 验证 Authorization 设置
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth, "Authorization header should exist for models")
			require.Contains(t, authValue, "Bearer rw-test123456789", "Authorization should contain Runware API token")
		})
	})
}

// RunRunwareOnHttpRequestBodyTests 测试 Runware 请求体处理
func RunRunwareOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Runware 聊天完成请求体处理
		t.Run("runware chat completion request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicRunwareConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 测试请求体
			requestBody := `{
				"model": "gpt-3.5-turbo",
				"messages": [
					{"role": "user", "content": "Hello, world!"}
				],
				"stream": false
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体被正确处理
			actualRequestBody := host.GetRequestBody()
			require.NotNil(t, actualRequestBody)

			// 验证模型映射
			require.Contains(t, string(actualRequestBody), "qwen3.5-9b",
				"Model should be mapped to Runware model")
			require.Contains(t, string(actualRequestBody), "Hello, world!",
				"Request content should be preserved")
		})

		// 测试 Runware 流式聊天完成请求体处理
		t.Run("runware streaming chat completion request body", func(t *testing.T) {
			host, status := test.NewTestHost(runwareMultiModelConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 测试流式请求体
			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "Write a poem about AI"}
				],
				"stream": true
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体被正确处理
			actualRequestBody := host.GetRequestBody()
			require.NotNil(t, actualRequestBody)

			// 验证模型映射（gpt-4 应该映射到 qwen3.5-9b 模型）
			require.Contains(t, string(actualRequestBody), "qwen3.5-9b",
				"GPT-4 should be mapped to Runware qwen3.5-9b model")
			require.Contains(t, string(actualRequestBody), "Write a poem about AI",
				"Request content should be preserved")
			require.Contains(t, string(actualRequestBody), `"stream": true`,
				"Stream flag should be preserved")
		})
	})
}
