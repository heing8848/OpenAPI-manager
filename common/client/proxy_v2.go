package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common/config"
)

var relayHTTPClientCacheV2 sync.Map

func GetRelayHTTPClientByProxyV2(proxyValue string) (*http.Client, error) {
	normalizedProxy := strings.TrimSpace(proxyValue)
	if normalizedProxy == "" || normalizedProxy == strings.TrimSpace(config.RelayProxy) {
		return HTTPClient, nil
	}

	if cachedClient, ok := relayHTTPClientCacheV2.Load(normalizedProxy); ok {
		return cachedClient.(*http.Client), nil
	}

	parsedProxy, err := url.Parse(normalizedProxy)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream proxy: %w", err)
	}

	client := &http.Client{
		Transport: newHTTPTransport(parsedProxy),
	}
	if config.RelayTimeout != 0 {
		client.Timeout = time.Duration(config.RelayTimeout) * time.Second
	}

	actualClient, _ := relayHTTPClientCacheV2.LoadOrStore(normalizedProxy, client)
	return actualClient.(*http.Client), nil
}
