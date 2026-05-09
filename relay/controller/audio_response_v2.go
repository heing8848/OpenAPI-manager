package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func RelayAudioSpeechResponseHandlerV2(c *gin.Context, resp *http.Response, metaInfo *meta.Meta) *relaymodel.ErrorWithStatusCode {
	if resp == nil {
		return buildBadAudioPayloadErrorV2("provider returned an empty audio response")
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return buildBadAudioPayloadErrorV2(fmt.Sprintf("failed to read audio response body: %s", err.Error()))
	}
	if err = resp.Body.Close(); err != nil {
		return buildBadAudioPayloadErrorV2(fmt.Sprintf("failed to close audio response body: %s", err.Error()))
	}

	modelName := getStandaloneOutputModelNameV2(metaInfo)
	channelName := ""
	channelID := 0
	if metaInfo != nil {
		channelName = getStandaloneOutputChannelNameV2(metaInfo.ChannelType)
		channelID = metaInfo.ChannelId
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if resp.StatusCode == http.StatusOK && resp.ContentLength > 0 && shouldStreamAudioDirectlyV2(contentType) {
		defer func() {
			_ = resp.Body.Close()
		}()
		for k, v := range resp.Header {
			if len(v) == 0 {
				continue
			}
			c.Writer.Header().Set(k, v[0])
		}
		c.Status(resp.StatusCode)
		c.Writer.WriteHeaderNow()
		if _, err = io.Copy(c.Writer, resp.Body); err != nil {
			return buildBadAudioPayloadErrorV2(fmt.Sprintf("failed to stream audio response body: %s", err.Error()))
		}
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		logger.Errorf(
			c.Request.Context(),
			"audio speech upstream returned non-success status=%d channel_type=%s channel_id=%d model=%s body=%s",
			resp.StatusCode,
			channelName,
			channelID,
			modelName,
			string(responseBody),
		)
		resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))
		return RelayErrorHandlerV2(resp)
	}

	trimmedBody := strings.TrimSpace(string(responseBody))
	if trimmedBody == "" {
		logger.Errorf(
			c.Request.Context(),
			"audio speech upstream returned empty payload channel_type=%s channel_id=%d model=%s",
			channelName,
			channelID,
			modelName,
		)
		return buildBadAudioPayloadErrorV2("provider returned an empty audio response payload")
	}

	if strings.Contains(contentType, "application/json") || strings.HasPrefix(trimmedBody, "{") {
		var errPayload GeneralErrorResponse
		if err = json.Unmarshal(responseBody, &errPayload); err == nil {
			if message := errPayload.ToMessage(); message != "" {
				logger.Errorf(
					c.Request.Context(),
					"audio speech upstream returned embedded error payload channel_type=%s channel_id=%d model=%s body=%s",
					channelName,
					channelID,
					modelName,
					string(responseBody),
				)
				return &relaymodel.ErrorWithStatusCode{
					StatusCode: http.StatusBadGateway,
					Error: relaymodel.Error{
						Message: message,
						Type:    "upstream_error",
						Code:    "bad_audio_response",
					},
				}
			}
		}
		logger.Errorf(
			c.Request.Context(),
			"audio speech upstream returned unexpected json payload channel_type=%s channel_id=%d model=%s body=%s",
			channelName,
			channelID,
			modelName,
			string(responseBody),
		)
		return buildBadAudioPayloadErrorV2("provider returned a JSON payload instead of audio data")
	}

	resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))
	for k, v := range resp.Header {
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.WriteHeader(resp.StatusCode)
	if _, err = io.Copy(c.Writer, resp.Body); err != nil {
		return buildBadAudioPayloadErrorV2(fmt.Sprintf("failed to copy audio response body: %s", err.Error()))
	}
	if err = resp.Body.Close(); err != nil {
		return buildBadAudioPayloadErrorV2(fmt.Sprintf("failed to close audio response body: %s", err.Error()))
	}
	return nil
}

func shouldStreamAudioDirectlyV2(contentType string) bool {
	return strings.HasPrefix(contentType, "audio/") || strings.Contains(contentType, "application/octet-stream")
}

func buildBadAudioPayloadErrorV2(message string) *relaymodel.ErrorWithStatusCode {
	return &relaymodel.ErrorWithStatusCode{
		StatusCode: http.StatusBadGateway,
		Error: relaymodel.Error{
			Message: message,
			Type:    "upstream_error",
			Code:    "bad_audio_response",
		},
	}
}
