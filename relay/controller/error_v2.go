package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/model"
)

func RelayErrorHandlerV2(resp *http.Response) *model.ErrorWithStatusCode {
	if resp == nil {
		return &model.ErrorWithStatusCode{
			StatusCode: http.StatusInternalServerError,
			Error: model.Error{
				Message: "resp is nil",
				Type:    "upstream_error",
				Code:    "bad_response",
			},
		}
	}

	errWithStatus := &model.ErrorWithStatusCode{
		StatusCode: resp.StatusCode,
		Error: model.Error{
			Message: "",
			Type:    "upstream_error",
			Code:    "bad_response_status_code",
			Param:   strconv.Itoa(resp.StatusCode),
		},
	}

	responseBody, bodyText, err := readRelayErrorBodyV2(resp)
	if err != nil {
		return errWithStatus
	}

	if config.DebugEnabled {
		logger.SysLog(fmt.Sprintf("error happened, status code: %d, response: \n%s", resp.StatusCode, bodyText))
	}

	if challengeError, ok := buildUpstreamChallengeErrorV2(resp, bodyText, errWithStatus.StatusCode); ok {
		errWithStatus.Error = challengeError
		return errWithStatus
	}

	var errResponse GeneralErrorResponse
	if json.Unmarshal(responseBody, &errResponse) == nil {
		if errResponse.Error.Message != "" {
			errWithStatus.Error = errResponse.Error
		} else {
			errWithStatus.Error.Message = errResponse.ToMessage()
		}
	}

	if strings.TrimSpace(errWithStatus.Error.Message) == "" {
		errWithStatus.Error.Message = normalizeRelayErrorBodyTextV2(bodyText, resp.StatusCode)
	}
	return errWithStatus
}

func readRelayErrorBodyV2(resp *http.Response) ([]byte, string, error) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if err = resp.Body.Close(); err != nil {
		return nil, "", err
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))
	return responseBody, string(responseBody), nil
}

func normalizeRelayErrorBodyTextV2(bodyText string, statusCode int) string {
	bodyText = strings.TrimSpace(bodyText)
	if bodyText == "" {
		return fmt.Sprintf("bad response status code %d", statusCode)
	}

	bodyText = stripSimpleHTMLTagsV2(bodyText)
	bodyText = strings.TrimSpace(bodyText)
	if bodyText == "" {
		return fmt.Sprintf("bad response status code %d", statusCode)
	}
	if len(bodyText) > 1024 {
		bodyText = bodyText[:1024]
	}
	return bodyText
}

func stripSimpleHTMLTagsV2(raw string) string {
	if !strings.Contains(raw, "<") || !strings.Contains(raw, ">") {
		return raw
	}

	var builder strings.Builder
	builder.Grow(len(raw))
	inTag := false
	inIgnoredTag := false
	ignoredTagName := ""
	tagBuffer := make([]rune, 0, 16)
	lastWasSpace := false
	for _, r := range raw {
		switch r {
		case '<':
			inTag = true
			tagBuffer = tagBuffer[:0]
		case '>':
			if inTag {
				tagContent := strings.TrimSpace(string(tagBuffer))
				tagName := strings.ToLower(strings.TrimPrefix(tagContent, "/"))
				if fields := strings.Fields(tagName); len(fields) > 0 {
					tagName = fields[0]
				} else {
					tagName = ""
				}
				switch strings.ToLower(tagName) {
				case "script", "style":
					if strings.HasPrefix(tagContent, "/") {
						inIgnoredTag = false
						ignoredTagName = ""
					} else {
						inIgnoredTag = true
						ignoredTagName = tagName
					}
				default:
					if strings.HasPrefix(tagContent, "/") && ignoredTagName == tagName {
						inIgnoredTag = false
						ignoredTagName = ""
					}
				}
				inTag = false
				if !lastWasSpace {
					builder.WriteByte(' ')
					lastWasSpace = true
				}
			}
		default:
			if inTag {
				tagBuffer = append(tagBuffer, r)
				continue
			}
			if inIgnoredTag {
				continue
			}
			if r == '\n' || r == '\r' || r == '\t' {
				r = ' '
			}
			if r == ' ' {
				if lastWasSpace {
					continue
				}
				lastWasSpace = true
			} else {
				lastWasSpace = false
			}
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
