package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// RunRunwareSmokeTests exercises the runware provider end to end: the plugin
// (loaded into a test host) transforms an OpenAI-style chat completion request,
// and the transformed request is then sent to the real Runware endpoint
// (RUNWARE_BASE_URL, default https://api.runware.ai) using RUNWARE_API_KEY.
// The test is skipped when RUNWARE_API_KEY is not set.
func RunRunwareSmokeTests(t *testing.T) {
	apiKey := os.Getenv("RUNWARE_API_KEY")
	if apiKey == "" {
		t.Skip("RUNWARE_API_KEY is not set; skipping Runware smoke test")
	}
	baseURL := os.Getenv("RUNWARE_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.runware.ai/v1"
	}
	model := os.Getenv("RUNWARE_SMOKE_MODEL")
	if model == "" {
		// LLM id taken from the Runware /v1/models catalog; override with RUNWARE_SMOKE_MODEL.
		model = "qwen3.5-9b"
	}

	cfg, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "runware",
			"apiTokens": []string{apiKey},
			"modelMapping": map[string]string{
				"gpt-4": model,
				"*":     model,
			},
		},
	})

	host, status := test.NewTestHost(cfg)
	defer host.Reset()
	require.Equal(t, types.OnPluginStartStatusOK, status)

	action := host.CallOnHttpRequestHeaders([][2]string{
		{":authority", "example.com"},
		{":path", "/v1/chat/completions"},
		{":method", "POST"},
		{"Content-Type", "application/json"},
	})
	require.Equal(t, types.HeaderStopIteration, action)

	origBody := `{"model":"gpt-4","messages":[{"role":"user","content":"Reply with the single word: pong"}],"stream":false,"max_tokens":32}`
	action = host.CallOnHttpRequestBody([]byte(origBody))
	require.Equal(t, types.ActionContinue, action)

	// Collect the request transformed by the plugin.
	transformedPath := ""
	hdr := http.Header{}
	for _, kv := range host.GetRequestHeaders() {
		if strings.HasPrefix(kv[0], ":") {
			if kv[0] == ":path" {
				transformedPath = kv[1]
			}
			continue
		}
		hdr.Set(kv[0], kv[1])
	}
	require.Equal(t, "/v1/chat/completions", transformedPath, "path should stay OpenAI compatible")
	require.Contains(t, hdr.Get("Authorization"), "Bearer ", "Authorization should carry the Runware key")
	transformedBody := host.GetRequestBody()
	require.NotNil(t, transformedBody)
	require.Contains(t, string(transformedBody), model, "model should be mapped to the Runware model id")

	// Send the transformed request to the real endpoint.
	u, err := url.Parse(baseURL)
	require.NoError(t, err)
	basePath := strings.TrimSuffix(u.Path, "/")
	reqPath := transformedPath
	if strings.HasSuffix(basePath, "/v1") && strings.HasPrefix(reqPath, "/v1") {
		reqPath = strings.TrimPrefix(reqPath, "/v1")
	}
	u.Path = basePath + reqPath

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(transformedBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", hdr.Get("Authorization"))

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	t.Logf("POST %s -> %d", u.String(), resp.StatusCode)
	require.Equal(t, http.StatusOK, resp.StatusCode, "upstream returned non-200: %s", string(body))

	var chatResp struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	require.NoError(t, json.Unmarshal(body, &chatResp), "response body: %s", string(body))
	require.NotEmpty(t, chatResp.Choices, "response should contain choices: %s", string(body))
	require.NotEmpty(t, chatResp.Choices[0].Message.Content, "choice content should not be empty")
	t.Logf("smoke ok: model=%s content=%q", chatResp.Model, chatResp.Choices[0].Message.Content)
}
