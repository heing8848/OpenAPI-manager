package controller

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/songquanpeng/one-api/relay/model"
)

var cloudflareChallengeZonePatternV2 = regexp.MustCompile(`cZone:\s*['"]([^'"]+)['"]`)

func buildUpstreamChallengeErrorV2(resp *http.Response, bodyText string, statusCode int) (model.Error, bool) {
	if !isCloudflareChallengeBodyV2(bodyText) {
		return model.Error{}, false
	}

	host := extractChallengeHostV2(resp, bodyText)
	message := "upstream returned a Cloudflare challenge page"
	if host != "" {
		message = fmt.Sprintf("%s from %s", message, host)
	}
	message += " (possible egress IP, region, or anti-bot block)"

	return model.Error{
		Message: message,
		Type:    "upstream_error",
		Code:    "upstream_cloudflare_challenge",
		Param:   fmt.Sprintf("%d", statusCode),
	}, true
}

func isCloudflareChallengeBodyV2(bodyText string) bool {
	lower := strings.ToLower(bodyText)
	return strings.Contains(lower, "enable javascript and cookies to continue") ||
		strings.Contains(lower, "window._cf_chl_opt") ||
		strings.Contains(lower, "__cf_chl_tk") ||
		strings.Contains(lower, "just a moment...")
}

func extractChallengeHostV2(resp *http.Response, bodyText string) string {
	if matches := cloudflareChallengeZonePatternV2.FindStringSubmatch(bodyText); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
		return strings.TrimSpace(resp.Request.URL.Host)
	}
	return ""
}
