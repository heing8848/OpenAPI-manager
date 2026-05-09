package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	dbmodel "github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/adaptor/videotask"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func RelayVideoHelperV1(c *gin.Context, relayMode int) *relaymodel.ErrorWithStatusCode {
	switch relayMode {
	case relaymode.VideosGenerationsV1:
		return relayVideoCreateTaskV1(c)
	case relaymode.VideoGenerationsTasksV1:
		return relayVideoQueryTasksV1(c)
	default:
		return openai.ErrorWrapper(fmt.Errorf("unsupported video relay mode %d", relayMode), "video_relay_mode_unsupported_v1", http.StatusBadRequest)
	}
}

func relayVideoCreateTaskV1(c *gin.Context) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()
	request, err := parseVideoGenerationRequestV1(c)
	if err != nil {
		return openai.ErrorWrapper(err, "invalid_video_request_v1", http.StatusBadRequest)
	}

	userID := c.GetInt(ctxkey.Id)
	group, channel, selectedKey, selectionErr := selectVideoChannelForCreateV1(c, request.Model, userID)
	if selectionErr != nil {
		return selectionErr
	}
	c.Set(ctxkey.Group, group)
	setupVideoChannelContextV1(c, channel, selectedKey, request.Model)

	metaInfo := meta.GetByContext(c)
	if capabilityErr := ValidateStandaloneOutputCapabilityV3(metaInfo, relaymode.VideosGenerationsV1); capabilityErr != nil {
		return capabilityErr
	}

	metaInfo.OriginModelName = request.Model
	mappedModelName, _ := getMappedModelName(request.Model, metaInfo.ModelMapping)
	metaInfo.ActualModelName = mappedModelName

	upstreamRequest := relaymodel.VideoGenerationRequestV1{
		Model:   mappedModelName,
		Content: request.Content,
	}
	requestBody, err := json.Marshal(upstreamRequest)
	if err != nil {
		return openai.ErrorWrapper(err, "marshal_video_request_failed_v1", http.StatusInternalServerError)
	}

	upstreamURL := videotask.BuildCreateTaskURLV1(metaInfo.BaseURL)
	req, err := http.NewRequest(http.MethodPost, upstreamURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return openai.ErrorWrapper(err, "new_video_request_failed_v1", http.StatusInternalServerError)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if metaInfo.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+metaInfo.APIKey)
	}

	logger.Infof(
		ctx,
		"creating standalone video task via channel_type=%s channel_id=%d model=%s",
		getStandaloneOutputChannelNameV3(metaInfo.ChannelType),
		metaInfo.ChannelId,
		metaInfo.ActualModelName,
	)

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return openai.ErrorWrapper(err, "video_create_upstream_request_failed_v1", http.StatusBadGateway)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusCreated &&
		resp.StatusCode != http.StatusAccepted {
		logger.Warnf(ctx, "video task create failed upstream: channel_id=%d model=%s status=%d", metaInfo.ChannelId, metaInfo.ActualModelName, resp.StatusCode)
		return RelayErrorHandlerV2(resp)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "read_video_create_response_failed_v1", http.StatusInternalServerError)
	}

	normalizedTask, err := videotask.ParseCreateTaskResponseV1(responseBody)
	if err != nil {
		logger.Warnf(ctx, "video task create returned malformed payload: channel_id=%d model=%s err=%s", metaInfo.ChannelId, metaInfo.ActualModelName, err.Error())
		return openai.ErrorWrapper(err, "bad_video_create_response_v1", http.StatusBadGateway)
	}

	upstreamTaskID := normalizedTask.ID
	alfredTaskID, err := relaymodel.EncodeVideoTaskHandleV1(relaymodel.VideoTaskHandleV1{
		UserID:         userID,
		ChannelID:      channel.Id,
		ChannelType:    channel.Type,
		RequestedModel: request.Model,
		UpstreamTaskID: upstreamTaskID,
	})
	if err != nil {
		return openai.ErrorWrapper(err, "encode_video_task_id_failed_v1", http.StatusInternalServerError)
	}

	normalizedTask.ID = alfredTaskID
	normalizedTask.Model = request.Model
	normalizedTask.PollingURL = buildVideoPollingURLV1(c, []string{alfredTaskID})
	if normalizedTask.Metadata == nil {
		normalizedTask.Metadata = map[string]any{}
	}
	normalizedTask.Metadata["upstream_task_id"] = upstreamTaskID

	c.JSON(http.StatusOK, normalizedTask)
	return nil
}

func relayVideoQueryTasksV1(c *gin.Context) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()
	userID := c.GetInt(ctxkey.Id)
	taskIDs, err := parseVideoTaskQueryIDsV1(c)
	if err != nil {
		return openai.ErrorWrapper(err, "invalid_video_task_query_v1", http.StatusBadRequest)
	}

	results := make([]relaymodel.VideoTaskResponseV1, 0, len(taskIDs))
	for _, taskID := range taskIDs {
		handle, decodeErr := relaymodel.DecodeVideoTaskHandleV1(taskID)
		if decodeErr != nil {
			return openai.ErrorWrapper(decodeErr, "invalid_video_task_id_v1", http.StatusBadRequest)
		}
		if handle.UserID != 0 && handle.UserID != userID {
			return openai.ErrorWrapper(errors.New("video task does not belong to current user"), "video_task_forbidden_v1", http.StatusForbidden)
		}

		channel, selectedKey, loadErr := loadVideoChannelForTaskQueryV1(handle.ChannelID)
		if loadErr != nil {
			return loadErr
		}
		baseURL := strings.TrimSpace(channel.GetBaseURL())
		if baseURL == "" {
			return openai.ErrorWrapper(errors.New("video task channel base url is empty"), "video_channel_base_url_missing_v1", http.StatusBadGateway)
		}

		queryURL := videotask.BuildTaskQueryURLV1(baseURL, url.QueryEscape(handle.UpstreamTaskID))
		req, reqErr := http.NewRequest(http.MethodGet, queryURL, nil)
		if reqErr != nil {
			return openai.ErrorWrapper(reqErr, "new_video_task_query_failed_v1", http.StatusInternalServerError)
		}
		req.Header.Set("Accept", "application/json")
		if selectedKey != nil && strings.TrimSpace(selectedKey.KeyValue) != "" {
			req.Header.Set("Authorization", "Bearer "+selectedKey.KeyValue)
		}

		logger.Infof(ctx, "querying standalone video task via channel_id=%d task_id=%s", channel.Id, taskID)

		resp, queryErr := client.HTTPClient.Do(req)
		if queryErr != nil {
			return openai.ErrorWrapper(queryErr, "video_task_query_upstream_failed_v1", http.StatusBadGateway)
		}
		if resp.StatusCode != http.StatusOK {
			return RelayErrorHandlerV2(resp)
		}

		responseBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return openai.ErrorWrapper(readErr, "read_video_task_query_response_failed_v1", http.StatusInternalServerError)
		}

		normalizedTasks, normalizeErr := videotask.ParseTaskQueryResponseV1(responseBody)
		if normalizeErr != nil {
			return openai.ErrorWrapper(normalizeErr, "bad_video_task_query_response_v1", http.StatusBadGateway)
		}

		selectedTask, matchErr := matchVideoTaskQueryResultV1(normalizedTasks, handle.UpstreamTaskID)
		if matchErr != nil {
			return openai.ErrorWrapper(matchErr, "video_task_not_found_v1", http.StatusBadGateway)
		}
		selectedTask.ID = taskID
		if selectedTask.Model == "" {
			selectedTask.Model = handle.RequestedModel
		}
		if selectedTask.Metadata == nil {
			selectedTask.Metadata = map[string]any{}
		}
		selectedTask.Metadata["upstream_task_id"] = handle.UpstreamTaskID
		results = append(results, selectedTask)
	}

	if len(results) == 1 {
		c.JSON(http.StatusOK, results[0])
		return nil
	}

	c.JSON(http.StatusOK, relaymodel.VideoTaskListResponseV1{
		Object: "list",
		Data:   results,
	})
	return nil
}

func parseVideoGenerationRequestV1(c *gin.Context) (*relaymodel.VideoGenerationRequestV1, error) {
	request := &relaymodel.VideoGenerationRequestV1{}
	if err := c.ShouldBindJSON(request); err != nil {
		return nil, err
	}
	request.Model = strings.TrimSpace(request.Model)
	if request.Model == "" {
		return nil, errors.New("model is required")
	}
	if len(request.Content) == 0 {
		return nil, errors.New("content is required")
	}

	textCount := 0
	imageCount := 0
	for index := range request.Content {
		request.Content[index].Type = strings.TrimSpace(request.Content[index].Type)
		switch request.Content[index].Type {
		case relaymodel.VideoContentTypeTextV1:
			request.Content[index].Text = strings.TrimSpace(request.Content[index].Text)
			if request.Content[index].Text == "" {
				return nil, errors.New("text content is required")
			}
			textCount++
		case relaymodel.VideoContentTypeImageURLV1:
			if request.Content[index].ImageURL == nil || strings.TrimSpace(request.Content[index].ImageURL.Url) == "" {
				return nil, errors.New("image_url.url is required")
			}
			request.Content[index].ImageURL.Url = strings.TrimSpace(request.Content[index].ImageURL.Url)
			imageCount++
		default:
			return nil, fmt.Errorf("unsupported content type: %s", request.Content[index].Type)
		}
	}
	if textCount == 0 {
		return nil, errors.New("at least one text content item is required")
	}
	if imageCount > 1 {
		return nil, errors.New("only one image_url content item is supported in video v1")
	}
	return request, nil
}

func parseVideoTaskQueryIDsV1(c *gin.Context) ([]string, error) {
	rawIDs := c.QueryArray("id")
	if rawCSV := strings.TrimSpace(c.Query("ids")); rawCSV != "" {
		rawIDs = append(rawIDs, strings.Split(rawCSV, ",")...)
	}
	seen := make(map[string]struct{})
	result := make([]string, 0, len(rawIDs))
	for _, rawID := range rawIDs {
		taskID := strings.TrimSpace(rawID)
		if taskID == "" {
			continue
		}
		if _, ok := seen[taskID]; ok {
			continue
		}
		seen[taskID] = struct{}{}
		result = append(result, taskID)
	}
	if len(result) == 0 {
		return nil, errors.New("at least one task id is required")
	}
	return result, nil
}

func selectVideoChannelForCreateV1(c *gin.Context, modelName string, userID int) (string, *dbmodel.Channel, *dbmodel.ChannelKey, *relaymodel.ErrorWithStatusCode) {
	if specificChannelID, ok := c.Get(ctxkey.SpecificChannelId); ok {
		channelID, err := strconv.Atoi(specificChannelID.(string))
		if err != nil {
			return "", nil, nil, openai.ErrorWrapper(errors.New("invalid channel id"), "invalid_channel_id_v1", http.StatusBadRequest)
		}
		channel, err := dbmodel.GetChannelById(channelID, true)
		if err != nil {
			return "", nil, nil, openai.ErrorWrapper(errors.New("invalid channel id"), "invalid_channel_id_v1", http.StatusBadRequest)
		}
		if channel.Status != dbmodel.ChannelStatusEnabled {
			return "", nil, nil, openai.ErrorWrapper(errors.New("channel is disabled"), "channel_disabled_v1", http.StatusForbidden)
		}
		key, keyErr := prepareVideoChannelKeyV1(channel)
		if keyErr != nil {
			return "", nil, nil, keyErr
		}
		group, _ := dbmodel.CacheGetUserGroup(userID)
		return group, channel, key, nil
	}

	group, err := dbmodel.CacheGetUserGroup(userID)
	if err != nil {
		return "", nil, nil, openai.ErrorWrapper(err, "get_user_group_failed_v1", http.StatusInternalServerError)
	}

	maxAttempts := config.RetryTimes + 3
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	attempted := make(map[int]bool)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		channel, channelErr := dbmodel.CacheGetRandomSatisfiedChannel(group, modelName, attempt > 0)
		if channelErr != nil {
			return group, nil, nil, openai.ErrorWrapper(channelErr, "video_channel_selection_failed_v1", http.StatusServiceUnavailable)
		}
		if attempted[channel.Id] {
			continue
		}
		attempted[channel.Id] = true
		key, keyErr := prepareVideoChannelKeyV1(channel)
		if keyErr != nil {
			logger.Warnf(c.Request.Context(), "skip video channel #%d during create selection: %s", channel.Id, keyErr.Message)
			continue
		}
		return group, channel, key, nil
	}

	return group, nil, nil, openai.ErrorWrapper(
		fmt.Errorf("no available channel for group %s and model %s", group, modelName),
		"video_channel_unavailable_v1",
		http.StatusServiceUnavailable,
	)
}

func prepareVideoChannelKeyV1(channel *dbmodel.Channel) (*dbmodel.ChannelKey, *relaymodel.ErrorWithStatusCode) {
	keys, err := dbmodel.PrepareChannelKeyCandidatesV2(channel)
	if err == nil && len(keys) > 0 {
		key := keys[0]
		return &key, nil
	}
	if channel != nil && channel.Type == channeltype.VideoTaskV1 && strings.TrimSpace(channel.Key) == "" {
		return &dbmodel.ChannelKey{
			ChannelId: channel.Id,
			KeyValue:  "",
			Position:  0,
			Status:    dbmodel.ChannelKeyStatusEnabled,
		}, nil
	}
	if err == nil {
		err = errors.New("channel has no available keys")
	}
	return nil, openai.ErrorWrapper(err, "video_channel_key_unavailable_v1", http.StatusServiceUnavailable)
}

func setupVideoChannelContextV1(c *gin.Context, channel *dbmodel.Channel, channelKey *dbmodel.ChannelKey, modelName string) {
	c.Set(ctxkey.ChannelObject, channel)
	c.Set(ctxkey.Channel, channel.Type)
	c.Set(ctxkey.ChannelId, channel.Id)
	c.Set(ctxkey.ChannelName, channel.Name)
	if channelKey != nil {
		c.Set(ctxkey.ChannelKeyId, channelKey.Id)
		c.Set(ctxkey.ChannelKeyIndex, channelKey.Position+1)
		c.Set(ctxkey.ChannelKeyValue, channelKey.KeyValue)
	} else {
		c.Set(ctxkey.ChannelKeyId, 0)
		c.Set(ctxkey.ChannelKeyIndex, 0)
		c.Set(ctxkey.ChannelKeyValue, "")
	}
	if channel.SystemPrompt != nil && *channel.SystemPrompt != "" {
		c.Set(ctxkey.SystemPrompt, *channel.SystemPrompt)
	}
	c.Set(ctxkey.ModelMapping, channel.GetModelMappingV2())
	c.Set(ctxkey.OriginalModel, modelName)
	c.Set(ctxkey.BaseURL, channel.GetBaseURL())
	if channelKey != nil && strings.TrimSpace(channelKey.KeyValue) != "" {
		c.Request.Header.Set("Authorization", "Bearer "+channelKey.KeyValue)
	} else {
		c.Request.Header.Del("Authorization")
	}
	cfg, _ := channel.LoadConfig()
	c.Set(ctxkey.Config, cfg)
}

func loadVideoChannelForTaskQueryV1(channelID int) (*dbmodel.Channel, *dbmodel.ChannelKey, *relaymodel.ErrorWithStatusCode) {
	channel, err := dbmodel.GetChannelById(channelID, true)
	if err != nil {
		return nil, nil, openai.ErrorWrapper(errors.New("video task channel not found"), "video_channel_not_found_v1", http.StatusNotFound)
	}
	key, keyErr := prepareVideoChannelKeyV1(channel)
	if keyErr != nil {
		return nil, nil, keyErr
	}
	return channel, key, nil
}

func matchVideoTaskQueryResultV1(tasks []relaymodel.VideoTaskResponseV1, upstreamTaskID string) (relaymodel.VideoTaskResponseV1, error) {
	for _, task := range tasks {
		if task.ID == upstreamTaskID {
			return task, nil
		}
	}
	if len(tasks) == 1 {
		return tasks[0], nil
	}
	return relaymodel.VideoTaskResponseV1{}, errors.New("video task not found in upstream response")
}

func buildVideoPollingURLV1(c *gin.Context, taskIDs []string) string {
	if len(taskIDs) == 0 {
		return ""
	}
	scheme := "http"
	if proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); proto != "" {
		scheme = strings.Split(proto, ",")[0]
	} else if c.Request.TLS != nil {
		scheme = "https"
	}
	host := strings.TrimSpace(c.Request.Host)
	if host == "" {
		return "/v1/videos/generations/tasks?id=" + url.QueryEscape(taskIDs[0])
	}
	query := make([]string, 0, len(taskIDs))
	for _, taskID := range taskIDs {
		query = append(query, "id="+url.QueryEscape(taskID))
	}
	return fmt.Sprintf("%s://%s/v1/videos/generations/tasks?%s", scheme, host, strings.Join(query, "&"))
}
