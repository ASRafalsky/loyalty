package client

import (
	"time"

	"github.com/gojek/heimdall/v7/httpclient"
)

func NewClient() *httpclient.Client {
	// Create a new HTTP client with a default timeout
	timeout := 10 * time.Second
	return httpclient.NewClient(httpclient.WithHTTPTimeout(timeout))
}
