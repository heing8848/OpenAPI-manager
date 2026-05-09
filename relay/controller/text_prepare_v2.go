package controller

import (
	"bytes"
	"io"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

type PreparedTextRequestV2 struct {
	TargetURL       string
	Headers         map[string]string
	Body            []byte
	ActualModelName string
	IsStream        bool
}

func BuildPreparedTextRequestV2(c *gin.Context, relayPath string) (*PreparedTextRequestV2, *model.ErrorWithStatusCode) {
	parsedRelayURL, err := url.ParseRequestURI(relayPath)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "invalid_worker_relay_path", http.StatusBadRequest)
	}

	originalURL := *c.Request.URL
	originalRequestURI := c.Request.RequestURI
	c.Request.URL.Path = parsedRelayURL.Path
	c.Request.URL.RawPath = parsedRelayURL.RawPath
	c.Request.URL.RawQuery = parsedRelayURL.RawQuery
	c.Request.RequestURI = parsedRelayURL.RequestURI()
	defer func() {
		c.Request.URL = &originalURL
		c.Request.RequestURI = originalRequestURI
	}()

	metaInfo := meta.GetByContext(c)
	if !isWorkerSupportedTextRelayModeV2(metaInfo.Mode) {
		return nil, openai.ErrorWrapper(
			http.ErrNotSupported,
			"worker_prepare_mode_not_supported",
			http.StatusForbidden,
		)
	}

	textRequest, err := getAndValidateTextRequest(c, metaInfo.Mode)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "invalid_text_request", http.StatusBadRequest)
	}
	c.Set(ctxkey.RequestModel, textRequest.Model)
	metaInfo.IsStream = textRequest.Stream
	metaInfo.OriginModelName = textRequest.Model
	mappedModelName, mappingApplied := getMappedModelName(textRequest.Model, metaInfo.ModelMapping)
	textRequest.Model = openai.NormalizeProviderCompatibleModelNameV2(metaInfo.ChannelType, metaInfo.BaseURL, mappedModelName, mappingApplied)
	metaInfo.ActualModelName = textRequest.Model
	setSystemPrompt(c.Request.Context(), textRequest, metaInfo.ForcedSystemPrompt)

	requestAdaptor := relay.GetAdaptor(metaInfo.APIType)
	if requestAdaptor == nil {
		return nil, openai.ErrorWrapper(
			http.ErrNotSupported,
			"invalid_api_type",
			http.StatusBadRequest,
		)
	}
	requestAdaptor.Init(metaInfo)

	requestBody, err := getRequestBody(c, metaInfo, textRequest, requestAdaptor)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "convert_request_failed", http.StatusInternalServerError)
	}

	preparedBody, err := io.ReadAll(requestBody)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "read_prepared_request_failed", http.StatusInternalServerError)
	}

	targetURL, err := requestAdaptor.GetRequestURL(metaInfo)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "get_request_url_failed", http.StatusInternalServerError)
	}

	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(preparedBody))
	if err != nil {
		return nil, openai.ErrorWrapper(err, "new_request_failed", http.StatusInternalServerError)
	}
	if err = requestAdaptor.SetupRequestHeader(c, req, metaInfo); err != nil {
		return nil, openai.ErrorWrapper(err, "setup_request_header_failed", http.StatusInternalServerError)
	}

	headers := make(map[string]string, len(req.Header))
	for name, values := range req.Header {
		if len(values) == 0 {
			continue
		}
		headers[name] = values[0]
	}
	if headers["Content-Type"] == "" {
		headers["Content-Type"] = "application/json"
	}

	return &PreparedTextRequestV2{
		TargetURL:       targetURL,
		Headers:         headers,
		Body:            preparedBody,
		ActualModelName: metaInfo.ActualModelName,
		IsStream:        metaInfo.IsStream,
	}, nil
}

func isWorkerSupportedTextRelayModeV2(relayMode int) bool {
	switch relayMode {
	case relaymode.ChatCompletions, relaymode.Completions, relaymode.Embeddings, relaymode.Moderations:
		return true
	default:
		return false
	}
}
