package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/model"
)

type imageGeneralErrorResponseV2 struct {
	Error    model.Error `json:"error"`
	Message  string      `json:"message"`
	Msg      string      `json:"msg"`
	Err      string      `json:"err"`
	ErrorMsg string      `json:"error_msg"`
	Header   struct {
		Message string `json:"message"`
	} `json:"header"`
	Response struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"response"`
}

func (e imageGeneralErrorResponseV2) ToMessageV2() string {
	if e.Error.Message != "" {
		return e.Error.Message
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != "" {
		return e.Err
	}
	if e.ErrorMsg != "" {
		return e.ErrorMsg
	}
	if e.Header.Message != "" {
		return e.Header.Message
	}
	if e.Response.Error.Message != "" {
		return e.Response.Error.Message
	}
	return ""
}

func ImageHandlerV2(c *gin.Context, resp *http.Response) (*model.ErrorWithStatusCode, *model.Usage) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorWrapper(err, "read_response_body_failed_v2", http.StatusInternalServerError), nil
	}
	err = resp.Body.Close()
	if err != nil {
		return ErrorWrapper(err, "close_response_body_failed_v2", http.StatusInternalServerError), nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		logger.Errorf(c.Request.Context(), "image upstream returned non-success status=%d body=%s", resp.StatusCode, string(responseBody))
		return buildImageUpstreamErrorV2(responseBody, resp.StatusCode), nil
	}

	var errPayload imageGeneralErrorResponseV2
	_ = json.Unmarshal(responseBody, &errPayload)

	var imageResponse ImageResponse
	if err = json.Unmarshal(responseBody, &imageResponse); err != nil {
		if errPayload.ToMessageV2() != "" {
			logger.Errorf(c.Request.Context(), "image upstream returned embedded error payload on success status body=%s", string(responseBody))
			return buildImageSuccessStatusErrorV2(errPayload), nil
		}
		logger.Errorf(c.Request.Context(), "image upstream returned malformed payload: %s", err.Error())
		return buildBadImagePayloadErrorV2("provider returned a malformed image response payload"), nil
	}
	if !hasUsableImageDataV2(imageResponse) {
		if errPayload.ToMessageV2() != "" {
			logger.Errorf(c.Request.Context(), "image upstream returned embedded error payload with success status body=%s", string(responseBody))
			return buildImageSuccessStatusErrorV2(errPayload), nil
		}
		logger.Errorf(c.Request.Context(), "image upstream returned empty image payload body=%s", string(responseBody))
		return buildBadImagePayloadErrorV2("provider returned an empty image response payload"), nil
	}

	resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))
	for k, v := range resp.Header {
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.WriteHeader(resp.StatusCode)

	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		return ErrorWrapper(err, "copy_response_body_failed_v2", http.StatusInternalServerError), nil
	}
	err = resp.Body.Close()
	if err != nil {
		return ErrorWrapper(err, "close_response_body_failed_v2", http.StatusInternalServerError), nil
	}
	return nil, nil
}

func hasUsableImageDataV2(response ImageResponse) bool {
	if len(response.Data) == 0 {
		return false
	}
	for _, item := range response.Data {
		if item.Url != "" || item.B64Json != "" {
			return true
		}
	}
	return false
}

func buildImageUpstreamErrorV2(responseBody []byte, statusCode int) *model.ErrorWithStatusCode {
	errWithStatus := &model.ErrorWithStatusCode{
		StatusCode: statusCode,
		Error: model.Error{
			Message: fmt.Sprintf("bad image response status code %d", statusCode),
			Type:    "upstream_error",
			Code:    "bad_response_status_code",
			Param:   strconv.Itoa(statusCode),
		},
	}

	var errPayload imageGeneralErrorResponseV2
	if err := json.Unmarshal(responseBody, &errPayload); err == nil {
		if errPayload.Error.Message != "" {
			errWithStatus.Error = errPayload.Error
			if errWithStatus.Error.Type == "" {
				errWithStatus.Error.Type = "upstream_error"
			}
			if errWithStatus.Error.Code == nil || errWithStatus.Error.Code == "" {
				errWithStatus.Error.Code = "bad_response_status_code"
			}
			if errWithStatus.Error.Param == "" {
				errWithStatus.Error.Param = strconv.Itoa(statusCode)
			}
			return errWithStatus
		}
		if message := errPayload.ToMessageV2(); message != "" {
			errWithStatus.Error.Message = message
		}
	}

	return errWithStatus
}

func buildImageSuccessStatusErrorV2(errPayload imageGeneralErrorResponseV2) *model.ErrorWithStatusCode {
	errMessage := errPayload.ToMessageV2()
	if errMessage == "" {
		errMessage = "provider returned an unexpected image error payload"
	}
	errType := errPayload.Error.Type
	if errType == "" {
		errType = "upstream_error"
	}
	errCode := errPayload.Error.Code
	if errCode == nil || errCode == "" {
		errCode = "bad_image_response"
	}
	return &model.ErrorWithStatusCode{
		StatusCode: http.StatusBadGateway,
		Error: model.Error{
			Message: errMessage,
			Type:    errType,
			Param:   errPayload.Error.Param,
			Code:    errCode,
		},
	}
}

func buildBadImagePayloadErrorV2(message string) *model.ErrorWithStatusCode {
	return &model.ErrorWithStatusCode{
		StatusCode: http.StatusBadGateway,
		Error: model.Error{
			Message: message,
			Type:    "upstream_error",
			Code:    "bad_image_response",
		},
	}
}
