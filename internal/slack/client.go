package slack

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"text/template"

	"github.com/slack-go/slack"
)

type Client interface {
	Send(ctx context.Context, webhookURL string, token string, channel string, messageTmpl string, data any) error
}

type slackClient struct {
	httpClient *http.Client
}

func NewClient() Client {
	return &slackClient{
		httpClient: &http.Client{},
	}
}

func (c *slackClient) Send(ctx context.Context, webhookURL string, token string, channel string, messageTmpl string, data any) error {
	// Render message
	tmpl, err := template.New("msg").Parse(messageTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse message template: %w", err)
	}
	var msgBuf bytes.Buffer
	if err := tmpl.Execute(&msgBuf, data); err != nil {
		return fmt.Errorf("failed to execute message template: %w", err)
	}
	message := msgBuf.String()

	// Send via Token (API)
	if token != "" {
		api := slack.New(token)
		// If channel is not provided, we cannot send via API easily unless we know the default,
		// but usually explicit channel is good.
		if channel == "" {
			return fmt.Errorf("channel is required when using token authentication")
		}
		_, _, err := api.PostMessageContext(ctx, channel, slack.MsgOptionText(message, false))
		if err != nil {
			return fmt.Errorf("failed to post message to slack via API: %w", err)
		}
		return nil
	}

	// Send via Webhook
	if webhookURL != "" {
		msg := &slack.WebhookMessage{
			Text: message,
		}
		if channel != "" {
			msg.Channel = channel
		}
		err := slack.PostWebhookContext(ctx, webhookURL, msg)
		if err != nil {
			return fmt.Errorf("failed to post webhook: %w", err)
		}
		return nil
	}

	return fmt.Errorf("neither token nor webhookURL provided")
}
