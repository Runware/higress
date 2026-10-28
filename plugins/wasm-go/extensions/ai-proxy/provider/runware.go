package provider

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

// runwareProvider is the provider for Runware service.

const (
	runwareDomain = "api.runware.ai"
)

type runwareProviderInitializer struct{}

func (r *runwareProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (r *runwareProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion): PathOpenAIChatCompletions,
		string(ApiNameModels):         PathOpenAIModels,
		string(ApiNameResponses):      PathOpenAIResponses,
	}
}

func (r *runwareProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(r.DefaultCapabilities())
	return &runwareProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type runwareProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (r *runwareProvider) GetProviderType() string {
	return providerTypeRunware
}

func (r *runwareProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	r.config.handleRequestHeaders(r, ctx, apiName)
	return nil
}

func (r *runwareProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !r.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	return r.config.handleRequestBody(r, r.contextCache, ctx, apiName, body)
}

func (r *runwareProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), r.config.capabilities)
	util.OverwriteRequestHostHeader(headers, runwareDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+r.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

func (r *runwareProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, PathOpenAIChatCompletions) {
		return ApiNameChatCompletion
	}
	if strings.Contains(path, PathOpenAIModels) {
		return ApiNameModels
	}
	if strings.Contains(path, PathOpenAIResponses) {
		return ApiNameResponses
	}
	return ""
}
