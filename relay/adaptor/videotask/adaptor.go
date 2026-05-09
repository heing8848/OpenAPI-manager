package videotask

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

type Adaptor struct{}

func (a *Adaptor) Init(_ *meta.Meta) {}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	switch meta.Mode {
	case relaymode.VideosGenerationsV1:
		return BuildCreateTaskURLV1(meta.BaseURL), nil
	case relaymode.VideoGenerationsTasksV1:
		return BuildTaskQueryURLV1(meta.BaseURL, ""), nil
	default:
		return "", fmt.Errorf("unsupported video relay mode %d", meta.Mode)
	}
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error {
	adaptor.SetupCommonRequestHeader(c, req, meta)
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	if meta.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+meta.APIKey)
	}
	return nil
}

func (a *Adaptor) ConvertRequest(_ *gin.Context, _ int, request *relaymodel.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}

func (a *Adaptor) ConvertImageRequest(request *relaymodel.ImageRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	return adaptor.DoRequestHelper(a, c, meta, requestBody)
}

func (a *Adaptor) DoResponse(_ *gin.Context, _ *http.Response, _ *meta.Meta) (*relaymodel.Usage, *relaymodel.ErrorWithStatusCode) {
	return nil, openai.ErrorWrapper(errors.New("video task adaptor requires dedicated response handling"), "video_task_generic_response_unsupported_v1", http.StatusInternalServerError)
}

func (a *Adaptor) GetModelList() []string {
	return []string{}
}

func (a *Adaptor) GetChannelName() string {
	return "video-task-v1"
}

func BuildCreateTaskURLV1(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/videos/generations"
}

func BuildTaskQueryURLV1(baseURL string, upstreamTaskID string) string {
	base := strings.TrimRight(baseURL, "/") + "/videos/generations/tasks"
	if upstreamTaskID == "" {
		return base
	}
	return base + "?id=" + upstreamTaskID
}
