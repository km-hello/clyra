package settings

import (
	"database/sql"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"clyra/backend/internal/problem"
)

type Settings struct {
	SchedulerEnabled        bool    `json:"scheduler_enabled"`
	DefaultReminderDays     int64   `json:"default_reminder_days"`
	Timezone                string  `json:"timezone"`
	NotificationTime        string  `json:"notification_time"`
	TelegramChatID          *string `json:"telegram_chat_id"`
	TelegramMessageThreadID *int64  `json:"telegram_message_thread_id"`
	Theme                   string  `json:"theme"`
}

func Read(q interface{ QueryRow(string, ...any) *sql.Row }) (Settings, error) {
	var s Settings
	err := q.QueryRow(`SELECT scheduler_enabled,default_reminder_days,timezone,notification_time,telegram_chat_id,telegram_message_thread_id,theme FROM settings WHERE id=1`).Scan(&s.SchedulerEnabled, &s.DefaultReminderDays, &s.Timezone, &s.NotificationTime, &s.TelegramChatID, &s.TelegramMessageThreadID, &s.Theme)
	return s, err
}

var notificationTimePattern = regexp.MustCompile(`^(?:[01]\d|2[0-3]):[0-5]\d$`)

func Patch(db *sql.DB, input map[string]json.RawMessage) (Settings, error) {
	if len(input) == 0 {
		return Settings{}, problem.Validation("", "请至少修改一个设置字段")
	}
	tx, err := db.Begin()
	if err != nil {
		return Settings{}, problem.New(500, "设置保存失败")
	}
	defer tx.Rollback()
	s, err := Read(tx)
	if err != nil {
		return Settings{}, problem.New(500, "设置保存失败")
	}
	for k, v := range input {
		invalid := func(message string) error { return problem.Validation(k, message) }
		if string(v) == "null" && k != "telegram_chat_id" && k != "telegram_message_thread_id" {
			return Settings{}, invalid("字段不能为空")
		}
		switch k {
		case "scheduler_enabled":
			var enabled *bool
			if json.Unmarshal(v, &enabled) != nil || enabled == nil {
				return Settings{}, invalid("请选择是否开启自动提醒")
			}
			s.SchedulerEnabled = *enabled
		case "default_reminder_days":
			var days float64
			if json.Unmarshal(v, &days) != nil || days < 0 || days > 365 || days != float64(int64(days)) {
				return Settings{}, invalid("提醒天数必须为 0–365 的整数")
			}
			s.DefaultReminderDays = int64(days)
		case "timezone":
			if json.Unmarshal(v, &s.Timezone) != nil {
				return Settings{}, invalid("输入有效时区，如 Asia/Singapore")
			}
			if _, e := time.LoadLocation(s.Timezone); e != nil || s.Timezone == "" || s.Timezone == "Local" || strings.HasPrefix(s.Timezone, "+") || strings.HasPrefix(s.Timezone, "-") {
				return Settings{}, invalid("输入有效时区，如 Asia/Singapore")
			}
		case "notification_time":
			if json.Unmarshal(v, &s.NotificationTime) != nil || !notificationTimePattern.MatchString(s.NotificationTime) {
				return Settings{}, invalid("时间格式应为 HH:mm")
			}
		case "telegram_chat_id":
			if json.Unmarshal(v, &s.TelegramChatID) != nil {
				return Settings{}, invalid("请输入有效的 Chat ID")
			}
			if s.TelegramChatID != nil {
				value := strings.TrimSpace(*s.TelegramChatID)
				if strings.ContainsRune(value, 0) {
					return Settings{}, invalid("Chat ID 不能包含空字符")
				}
				s.TelegramChatID = &value
				if value == "" {
					s.TelegramChatID = nil
				}
			}
		case "telegram_message_thread_id":
			if json.Unmarshal(v, &s.TelegramMessageThreadID) != nil || (s.TelegramMessageThreadID != nil && (*s.TelegramMessageThreadID < 1 || *s.TelegramMessageThreadID > 2147483647)) {
				return Settings{}, invalid("Topic ID 必须为 1–2147483647 的整数")
			}
		case "theme":
			if json.Unmarshal(v, &s.Theme) != nil || (s.Theme != "system" && s.Theme != "light" && s.Theme != "dark") {
				return Settings{}, invalid("请选择有效的主题")
			}
		default:
			return Settings{}, invalid("不支持的设置字段")
		}
	}
	_, err = tx.Exec(`UPDATE settings SET scheduler_enabled=?,default_reminder_days=?,timezone=?,notification_time=?,telegram_chat_id=?,telegram_message_thread_id=?,theme=?,updated_at=? WHERE id=1`, s.SchedulerEnabled, s.DefaultReminderDays, s.Timezone, s.NotificationTime, s.TelegramChatID, s.TelegramMessageThreadID, s.Theme, time.Now().UnixMilli())
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		return Settings{}, problem.New(500, "设置保存失败")
	}
	return s, nil
}
