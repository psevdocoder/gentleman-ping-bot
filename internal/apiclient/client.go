package apiclient

import (
	"io"
	"net/http"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	client httpClient
}

func NewClient() *Client {
	return &Client{
		client: &http.Client{},
	}
}

func closeAndDiscard(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}
