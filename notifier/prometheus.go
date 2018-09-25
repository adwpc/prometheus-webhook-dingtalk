package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/adwpc/prometheus-webhook-dingtalk/models"
	"github.com/adwpc/prometheus-webhook-dingtalk/template"
	"github.com/pkg/errors"
)

func BuildDingTalkNotification(promMessage *models.WebhookMessage) (*models.DingTalkNotification, error) {
	title, err := template.ExecuteTextString(`{{ template "ding.link.title" . }}`, promMessage)
	if err != nil {
		return nil, err
	}
	content, err := template.ExecuteTextString(`{{ template "ding.link.content" . }}`, promMessage)
	if err != nil {
		return nil, err
	}
	var buttons []models.DingTalkNotificationButton
	for i, alert := range promMessage.Alerts.Firing() {
		buttons = append(buttons, models.DingTalkNotificationButton{
			Title:     fmt.Sprintf("Graph for alert #%d", i+1),
			ActionURL: alert.GeneratorURL,
		})
	}

	notification := &models.DingTalkNotification{
		MessageType: "markdown",
		Markdown: &models.DingTalkNotificationMarkdown{
			Title: title,
			Text:  content,
		},
	}
	return notification, nil
}

func SendDingTalkNotification(httpClient *http.Client, webhookURL string, at string, notification *models.DingTalkNotification) (*models.DingTalkNotificationResponse, error) {
	if at != "" {
		//186xxxxxxx,186xxxxxx...
		phones := strings.Split(at, ",")
		if notification != nil {
			if notification.At == nil {
				notification.At = &models.DingTalkNotificationAt{}
			}
			for i := 0; i < len(phones); i++ {
				notification.At.AtMobiles = append(notification.At.AtMobiles, phones[i])
				if notification.Markdown != nil {
					notification.Markdown.Text = notification.Markdown.Text + "@" + phones[i]
				}
			}
			notification.At.IsAtAll = false
		}
	}
	body, err := json.Marshal(&notification)

	if err != nil {
		return nil, errors.Wrap(err, "error encoding DingTalk request")
	}

	httpReq, err := http.NewRequest("POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "error building DingTalk request")
	}
	httpReq.Header.Set("Content-Type", "application/json")

	req, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "error sending notification to DingTalk")
	}
	defer req.Body.Close()

	if req.StatusCode != 200 {
		return nil, errors.Errorf("unacceptable response code %d", req.StatusCode)
	}

	var robotResp models.DingTalkNotificationResponse
	enc := json.NewDecoder(req.Body)
	if err := enc.Decode(&robotResp); err != nil {
		return nil, errors.Wrap(err, "error decoding response from DingTalk")
	}

	return &robotResp, nil
}
