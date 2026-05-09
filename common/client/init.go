package client

import (
	"fmt"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"net"
	"net/http"
	"net/url"
	"time"
)

var HTTPClient *http.Client
var ImpatientHTTPClient *http.Client
var UserContentRequestHTTPClient *http.Client

func newHTTPTransport(proxyURL *url.URL) *http.Transport {
	maxIdleConns := 32
	maxIdleConnsPerHost := 8
	maxConnsPerHost := 16
	if config.LowMemoryMode {
		maxIdleConns = 16
		maxIdleConnsPerHost = 4
		maxConnsPerHost = 8
	}
	return &http.Transport{
		Proxy:                 http.ProxyURL(proxyURL),
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:          maxIdleConns,
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
		MaxConnsPerHost:       maxConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

func Init() {
	var userContentProxyURL *url.URL
	if config.UserContentRequestProxy != "" {
		logger.SysLog(fmt.Sprintf("using %s as proxy to fetch user content", config.UserContentRequestProxy))
		proxyURL, err := url.Parse(config.UserContentRequestProxy)
		if err != nil {
			logger.FatalLog(fmt.Sprintf("USER_CONTENT_REQUEST_PROXY set but invalid: %s", config.UserContentRequestProxy))
		}
		userContentProxyURL = proxyURL
	}
	UserContentRequestHTTPClient = &http.Client{
		Transport: newHTTPTransport(userContentProxyURL),
		Timeout:   time.Second * time.Duration(config.UserContentRequestTimeout),
	}

	var relayProxyURL *url.URL
	var transport http.RoundTripper
	if config.RelayProxy != "" {
		logger.SysLog(fmt.Sprintf("using %s as api relay proxy", config.RelayProxy))
		proxyURL, err := url.Parse(config.RelayProxy)
		if err != nil {
			logger.FatalLog(fmt.Sprintf("RELAY_PROXY set but invalid: %s", config.RelayProxy))
		}
		relayProxyURL = proxyURL
	}
	transport = newHTTPTransport(relayProxyURL)

	if config.RelayTimeout == 0 {
		HTTPClient = &http.Client{
			Transport: transport,
		}
	} else {
		HTTPClient = &http.Client{
			Timeout:   time.Duration(config.RelayTimeout) * time.Second,
			Transport: transport,
		}
	}

	ImpatientHTTPClient = &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}
}
