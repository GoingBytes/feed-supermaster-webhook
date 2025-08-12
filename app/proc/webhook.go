package proc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/go-pkgz/lgr"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"

	"github.com/umputun/feed-master/app/feed"
)

type WebhookClient struct {
	URL     string
	Retries int
}

type WebhookPayload struct {
	ItemTitle string `json:"item_title"`
	ItemURL   string `json:"item_url"`
}

func NewWebhookClient(url string, retries int) *WebhookClient {
	return &WebhookClient{
		URL:     url,
		Retries: retries,
	}
}

func (client WebhookClient) sendHook(rssFeed feed.Rss2, item feed.Item) (message feed.Item, err error) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = client.Retries

	standardClient := retryClient.StandardClient()

	data := WebhookPayload{
		ItemTitle: item.Title,
		ItemURL:   item.Link,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return item, errors.Wrap(err, "can't marshal json")
	}

	req, err := http.NewRequest("POST", client.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return item, errors.Wrap(err, "can't create request")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := standardClient.Do(req)
	if err != nil {
		return item, errors.Wrap(err, "can't send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return item, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return item, nil
}

// Send message
func (client WebhookClient) Send(rssFeed feed.Rss2, item feed.Item) (err error) {
	_, err = client.sendHook(rssFeed, item)
	if err != nil {
		return errors.Wrapf(err, "can't send to webhook for %+v", item.Enclosure)
	}

	log.Printf("[DEBUG] webhook message sent")

	return nil
}
