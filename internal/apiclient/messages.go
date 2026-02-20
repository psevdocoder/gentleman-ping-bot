package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/psevdocoder/gentleman-ping-bot/internal/sender"
)

func (c *Client) SendMessage(ctx context.Context, requestURL string, cookie string, headers map[string]string, messageBody *sender.Body) error {
	if requestURL == "" {
		return errors.New("requestURL is empty")
	}

	if cookie == "" {
		return errors.New("cookie is empty")
	}

	if headers == nil {
		return errors.New("headers is empty")
	}

	bodyBytes, err := json.Marshal(messageBody)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(bodyBytes))

	if err != nil {
		return err
	}

	request.Header.Set("Cookie", cookie)
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer closeAndDiscard(response)

	respBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	log.Println("Client.SendMessage response", response.StatusCode, string(respBytes))
	return nil
}
