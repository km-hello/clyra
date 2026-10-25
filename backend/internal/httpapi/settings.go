package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"clyra/backend/internal/problem"
	"clyra/backend/internal/settings"
)

func (a *API) settingsResponse(s settings.Settings) map[string]any {
	return map[string]any{"default_reminder_days": s.DefaultReminderDays, "timezone": s.Timezone, "notification_time": s.NotificationTime, "telegram_chat_id": s.TelegramChatID, "theme": s.Theme, "bot_configured": strings.TrimSpace(a.config.TelegramBotToken) != "", "scheduler_enabled": s.SchedulerEnabled}
}
func (a *API) getSettings(w http.ResponseWriter, r *http.Request) {
	s, err := settings.Read(a.db)
	if err != nil {
		writeError(w, problem.New(500, "无法读取设置"))
		return
	}
	writeJSON(w, 200, map[string]any{"data": a.settingsResponse(s)})
}

func (a *API) patchSettings(w http.ResponseWriter, r *http.Request) {
	var input map[string]json.RawMessage
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	s, err := settings.Patch(a.db, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"data": a.settingsResponse(s)})
}
