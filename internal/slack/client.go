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
	Send(ctx context.Context, webhookURL string, token string, channel string, titleTmpl string, messageTmpl string, color string, data any) error
}

type slackClient struct {
	httpClient *http.Client
}

func NewClient() Client {
	return &slackClient{
		httpClient: &http.Client{},
	}
}

func (c *slackClient) Send(ctx context.Context, webhookURL string, token string, channel string, titleTmpl string, messageTmpl string, color string, data any) error {
	// Render message
	tmplMsg, err := template.New("msg").Parse(messageTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse message template: %w", err)
	}
	var msgBuf bytes.Buffer
	if err := tmplMsg.Execute(&msgBuf, data); err != nil {
		return fmt.Errorf("failed to execute message template: %w", err)
	}
	message := msgBuf.String()

	// Render title
	title := ""
	if titleTmpl != "" {
		tmplTitle, err := template.New("title").Parse(titleTmpl)
		if err != nil {
			return fmt.Errorf("failed to parse title template: %w", err)
		}
		var titleBuf bytes.Buffer
		if err := tmplTitle.Execute(&titleBuf, data); err != nil {
			return fmt.Errorf("failed to execute title template: %w", err)
		}
		title = titleBuf.String()
	}

	attachment := slack.Attachment{
		Title: title,
		Color: color, // Valid values: "good", "warning", "danger", or hex
		Fields: []slack.AttachmentField{
			{
				Value: message,
				Short: false,
			},
		},
	}

	// Send via Token (API)
	if token != "" {
		api := slack.New(token)
		// If channel is not provided, we must fail or rely on default
		if channel == "" {
			return fmt.Errorf("channel is required when using token authentication")
		}

		options := []slack.MsgOption{
			slack.MsgOptionAttachments(attachment),
		}
		// Fallback text for notifications
		options = append(options, slack.MsgOptionText(message, false))

		_, _, err := api.PostMessageContext(ctx, channel, options...)
		if err != nil {
			return fmt.Errorf("failed to post message to slack via API: %w", err)
		}
		return nil
	}

	// Send via Webhook
	if webhookURL != "" {
		msg := &slack.WebhookMessage{
			Attachments: []slack.Attachment{attachment},
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
