package webrouter

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/adwpc/prometheus-webhook-dingtalk/models"
	"github.com/adwpc/prometheus-webhook-dingtalk/notifier"
	"github.com/go-chi/chi"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type DingTalkResource struct {
	Logger     log.Logger
	Profiles   map[string]string
	ProfilesAt map[string]string
	HttpClient *http.Client
}

func (rs *DingTalkResource) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/{profile}/send", rs.SendNotification)
	return r
}

func (rs *DingTalkResource) SendNotification(w http.ResponseWriter, r *http.Request) {
	logger := rs.Logger
	profile := chi.URLParam(r, "profile")

	webhookURL, ok := rs.Profiles[profile]
	if !ok || webhookURL == "" {
		http.NotFound(w, r)
		return
	}

	at, ok := rs.ProfilesAt[profile]
	if !ok || at == "" {
		http.NotFound(w, r)
		return
	}

	var promMessage models.WebhookMessage
	if err := json.NewDecoder(r.Body).Decode(&promMessage); err != nil {
		level.Error(logger).Log("msg", "Cannot decode prometheus webhook JSON request", "err", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	notification, err := notifier.BuildDingTalkNotification(&promMessage)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to build notification", "err", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	robotResp, err := notifier.SendDingTalkNotification(rs.HttpClient, webhookURL, at, notification)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to send notification", "err", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if robotResp.ErrorCode != 0 {
		level.Error(logger).Log("msg", "Failed to send notification to DingTalk", "respCode", robotResp.ErrorCode, "respMsg", robotResp.ErrorMessage)
		http.Error(w, "Unable to talk to DingTalk", http.StatusUnprocessableEntity)
		return
	}

	io.WriteString(w, "OK")
}
