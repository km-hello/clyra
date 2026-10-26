package notification

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"

	"clyra/backend/internal/calendar"
	"clyra/backend/internal/config"
	"clyra/backend/internal/database"
	"clyra/backend/internal/integrations"
	"clyra/backend/internal/problem"
	"clyra/backend/internal/settings"
)

type Service struct {
	DB         *sql.DB
	Config     config.Config
	HTTPClient *http.Client
	Context    context.Context
}

type NotificationRun struct {
	ID            string  `json:"id"`
	RunDate       string  `json:"run_date"`
	ScheduledTime *string `json:"scheduled_time"`
	RunType       string  `json:"run_type"`
	Channel       string  `json:"channel"`
	Status        string  `json:"status"`
	MatchedCount  int64   `json:"matched_count"`
	MessageCount  int64   `json:"message_count"`
	StartedAt     int64   `json:"started_at"`
	FinishedAt    *int64  `json:"finished_at"`
	ErrorMessage  *string `json:"error_message"`
}

const notificationColumns = `id,run_date,scheduled_time,run_type,channel,status,matched_count,message_count,started_at,finished_at,error_message`
const missingConfiguration = "请先配置 Telegram Bot Token 和 Chat ID"
const uncertainFailure = "通知发送结果未确认"

func scanNotification(s interface{ Scan(...any) error }) (NotificationRun, error) {
	var n NotificationRun
	err := s.Scan(&n.ID, &n.RunDate, &n.ScheduledTime, &n.RunType, &n.Channel, &n.Status, &n.MatchedCount, &n.MessageCount, &n.StartedAt, &n.FinishedAt, &n.ErrorMessage)
	return n, err
}
func (a *Service) List() (map[string]any, error) {
	tx, err := a.DB.Begin()
	if err != nil {
		return nil, problem.New(500, "无法处理通知记录")
	}
	defer tx.Rollback()
	s, err := settings.Read(tx)
	if err != nil {
		return nil, problem.New(500, "无法处理通知记录")
	}
	rows, err := tx.Query(`SELECT ` + notificationColumns + ` FROM notification_runs ORDER BY started_at DESC,id ASC LIMIT 10`)
	if err != nil {
		return nil, problem.New(500, "无法处理通知记录")
	}
	runs := []NotificationRun{}
	for rows.Next() {
		n, e := scanNotification(rows)
		if e != nil {
			err = e
			break
		}
		runs = append(runs, n)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return nil, problem.New(500, "无法处理通知记录")
	}
	if err = tx.Commit(); err != nil {
		return nil, problem.New(500, "无法处理通知记录")
	}
	return map[string]any{"data": runs, "meta": map[string]any{"timezone": s.Timezone}}, nil
}
func (a *Service) finishNotification(id string, started int64, status string, matched, messageCount int, errorMessage *string) error {
	finished := time.Now().UnixMilli()
	if finished < started {
		finished = started
	}
	if errorMessage != nil {
		runes := []rune(*errorMessage)
		if len(runes) > 500 {
			v := string(runes[:500])
			errorMessage = &v
		}
	}
	result, err := a.DB.Exec(`UPDATE notification_runs SET status=?,matched_count=?,message_count=?,error_message=?,finished_at=? WHERE id=?`, status, matched, messageCount, errorMessage, finished, id)
	if err == nil {
		var n int64
		n, err = result.RowsAffected()
		if err == nil && n != 1 {
			err = sql.ErrNoRows
		}
	}
	if err != nil {
		return problem.New(500, "无法处理通知记录")
	}
	return nil
}
func (a *Service) Test() (map[string]any, error) {
	started := time.Now().UnixMilli()
	tx, err := a.DB.Begin()
	if err != nil {
		return nil, problem.New(500, "无法处理通知记录")
	}
	defer tx.Rollback()
	s, err := settings.Read(tx)
	if err != nil {
		return nil, problem.New(500, "无法处理通知记录")
	}
	today, err := calendar.Today(time.UnixMilli(started), s.Timezone)
	if err != nil {
		return nil, problem.New(500, "无法处理通知记录")
	}
	id := database.NewID()
	_, err = tx.Exec(`INSERT INTO notification_runs(id,run_date,run_type,status,started_at) VALUES(?,?,'manual_test','running',?)`, id, today, started)
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		return nil, problem.New(500, "无法处理通知记录")
	}
	if strings.TrimSpace(a.Config.TelegramBotToken) == "" || s.TelegramChatID == nil {
		message := missingConfiguration
		if err = a.finishNotification(id, started, "failed", 0, 0, &message); err != nil {
			return nil, err
		}
		return nil, &problem.Error{Status: 400, Message: message, Data: map[string]any{"code": "NOTIFICATION_CONFIGURATION_ERROR", "run_id": id}}
	}
	// Browser disconnects do not cancel delivery, but application shutdown does.
	result := integrations.SendTelegram(a.Context, a.HTTPClient, a.Config.TelegramBotToken, *s.TelegramChatID, []string{"<b>Clyra 测试消息</b>\n\nTelegram 通知连接测试。"}, false)
	status := "succeeded"
	if result.ErrorMessage != nil {
		status = "failed"
	}
	if err = a.finishNotification(id, started, status, 0, result.MessageCount, result.ErrorMessage); err != nil {
		return nil, err
	}
	if result.ErrorMessage != nil {
		return nil, &problem.Error{Status: 502, Message: *result.ErrorMessage, Data: map[string]any{"code": "NOTIFICATION_SEND_FAILED", "run_id": id}}
	}
	return map[string]any{"data": map[string]any{"run_id": id, "message_count": result.MessageCount, "status": "succeeded"}}, nil
}
func (a *Service) scheduledTick(ctx context.Context, now time.Time) error {
	started := now.UnixMilli()
	tx, err := a.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var userID string
	if err = tx.QueryRow(`SELECT id FROM users LIMIT 1`).Scan(&userID); err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return err
	}
	s, err := settings.Read(tx)
	if err != nil {
		return err
	}
	if !s.SchedulerEnabled {
		return nil
	}
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		return err
	}
	local := now.In(loc)
	today := local.Format("2006-01-02")
	if local.Format("15:04") < s.NotificationTime {
		return nil
	}
	if strings.TrimSpace(a.Config.TelegramBotToken) == "" || s.TelegramChatID == nil {
		return nil
	}
	rows, err := tx.Query(`SELECT name,next_payment,amount_minor,currency FROM subscriptions WHERE status='active' AND notify_enabled=1 AND julianday(next_payment)-julianday(?) BETWEEN 0 AND COALESCE(reminder_days,?) ORDER BY next_payment ASC,id ASC`, today, s.DefaultReminderDays)
	if err != nil {
		return err
	}
	items := []integrations.ReminderItem{}
	for rows.Next() {
		var item integrations.ReminderItem
		if err = rows.Scan(&item.Name, &item.NextPayment, &item.AmountMinor, &item.Currency); err != nil {
			break
		}
		items = append(items, item)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	id := database.NewID()
	claim, err := tx.Exec(`INSERT INTO notification_runs(id,run_date,scheduled_time,run_type,status,matched_count,started_at) VALUES(?,?,?,'scheduled','running',?,?) ON CONFLICT DO NOTHING`, id, today, s.NotificationTime, len(items), started)
	if err != nil {
		return err
	}
	n, err := claim.RowsAffected()
	if err != nil || n != 1 {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	messages, err := integrations.FormatReminders(items, today)
	if err != nil {
		message := uncertainFailure
		return a.finishNotification(id, started, "failed", len(items), 0, &message)
	}
	result := integrations.SendTelegram(ctx, a.HTTPClient, a.Config.TelegramBotToken, *s.TelegramChatID, messages, true)
	status := "succeeded"
	if result.ErrorMessage != nil {
		status = "failed"
	}
	return a.finishNotification(id, started, status, len(items), result.MessageCount, result.ErrorMessage)
}
func (a *Service) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		if err := a.scheduledTick(ctx, time.Now()); err != nil {
			log.Print("通知定时检查失败")
		}
		delay := time.Until(time.Now().Truncate(time.Minute).Add(time.Minute))
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
