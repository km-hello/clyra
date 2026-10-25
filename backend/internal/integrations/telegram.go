package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"clyra/backend/internal/calendar"
	"clyra/backend/internal/textutil"
)

type ReminderItem struct {
	Name        string
	NextPayment string
}

var nameLineBreaks = regexp.MustCompile(`[\r\n\x{0085}\x{2028}\x{2029}]+`)

func FormatReminders(items []ReminderItem, today string) ([]string, error) {
	if len(items) == 0 {
		return []string{}, nil
	}
	base, err := calendar.Parse(today)
	if err != nil {
		return nil, err
	}
	heading := "🔔 <b>订阅到期提醒</b>"
	summary := fmt.Sprintf("%s · 共 %d 项", today, len(items))
	reserved := fmt.Sprintf("%s (%d/%d)\n\n%s\n\n", heading, len(items), len(items), summary)
	groups := []string{}
	body := ""
	for _, item := range items {
		date, e := calendar.Parse(item.NextPayment)
		if e != nil {
			return nil, e
		}
		days := int(date.Unix()/86400 - base.Unix()/86400)
		due := fmt.Sprintf("%d 天后到期", days)
		if days == 0 {
			due = "今天到期"
		} else if days == 1 {
			due = "明天到期"
		} else if days < 0 {
			due = fmt.Sprintf("已过期 %d 天", -days)
		}
		name := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;").Replace(nameLineBreaks.ReplaceAllString(item.Name, " "))
		entry := fmt.Sprintf("<b>%s</b>\n%s · %s", name, due, item.NextPayment)
		if textutil.Length(reserved+entry) > 4000 {
			return nil, fmt.Errorf("单个提醒条目过长")
		}
		if body != "" && textutil.Length(reserved+body+"\n\n"+entry) > 4000 {
			groups = append(groups, body)
			body = ""
		}
		if body != "" {
			body += "\n\n"
		}
		body += entry
	}
	groups = append(groups, body)
	for i, group := range groups {
		prefix := heading
		if len(groups) > 1 {
			prefix += fmt.Sprintf(" (%d/%d)", i+1, len(groups))
		}
		groups[i] = prefix + "\n\n" + summary + "\n\n" + group
	}
	return groups, nil
}

type TelegramResult struct {
	MessageCount int
	ErrorMessage *string
}

func telegramFailure(count int, message string) TelegramResult {
	return TelegramResult{count, &message}
}
func retrySeconds(payload map[string]json.RawMessage, header string, fallback int) int64 {
	var parameters map[string]json.RawMessage
	if json.Unmarshal(payload["parameters"], &parameters) == nil {
		var value float64
		if raw := parameters["retry_after"]; len(raw) > 0 && string(raw) != "null" && json.Unmarshal(raw, &value) == nil && value >= 0 && value <= float64(9007199254740991) && value == float64(int64(value)) {
			return int64(value)
		}
	}
	if header != "" {
		if n, e := strconv.ParseInt(strings.TrimSpace(header), 10, 64); e == nil && regexp.MustCompile(`^[0-9]+$`).MatchString(strings.TrimSpace(header)) && n >= 0 && n <= 9007199254740991 {
			return n
		}
		if d, e := http.ParseTime(header); e == nil && d.Format(http.TimeFormat) == header {
			seconds := int64(time.Until(d).Seconds())
			if d.After(time.Now()) {
				seconds++
			}
			if seconds < 0 {
				return 0
			}
			return seconds
		}
	}
	return int64(fallback)
}
func SendTelegram(ctx context.Context, base *http.Client, token string, chatID string, messages []string, automatic bool) TelegramResult {
	count := 0
	if strings.TrimSpace(token) == "" || strings.TrimSpace(chatID) == "" {
		return telegramFailure(count, "Telegram 通知配置不完整")
	}
	client := externalClient(base)
	for _, text := range messages {
		for attempt := 1; attempt <= 3; attempt++ {
			data, _ := json.Marshal(map[string]any{"chat_id": chatID, "text": text, "parse_mode": "HTML"})
			requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, "https://api.telegram.org/bot"+token+"/sendMessage", bytes.NewReader(data))
			if err != nil {
				cancel()
				return telegramFailure(count, "Telegram 发送结果未确认，未自动重试")
			}
			req.Header.Set("Content-Type", "application/json")
			response, err := client.Do(req)
			if err != nil {
				cancel()
				return telegramFailure(count, "Telegram 发送结果未确认，未自动重试")
			}
			var payload map[string]json.RawMessage
			raw, readErr := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
			err = json.Unmarshal(raw, &payload)
			response.Body.Close()
			cancel()
			if readErr != nil {
				err = readErr
			}
			if err != nil || payload == nil {
				return telegramFailure(count, "Telegram 响应无效，发送结果未确认，未自动重试")
			}
			var ok bool
			json.Unmarshal(payload["ok"], &ok)
			successHTTP := response.StatusCode >= 200 && response.StatusCode < 300
			if successHTTP && ok {
				count++
				break
			}
			if response.StatusCode >= 500 {
				return telegramFailure(count, "Telegram 服务异常，发送结果未确认，未自动重试")
			}
			var code float64
			json.Unmarshal(payload["error_code"], &code)
			if response.StatusCode != 429 && !(successHTTP && strings.TrimSpace(string(payload["ok"])) == "false" && code == 429) {
				return telegramFailure(count, "Telegram 发送结果未确认，未自动重试")
			}
			if !automatic {
				return telegramFailure(count, "Telegram 请求过于频繁，请稍后再试")
			}
			if attempt == 3 {
				return telegramFailure(count, "Telegram 限流重试次数已达上限")
			}
			wait := retrySeconds(payload, response.Header.Get("Retry-After"), attempt)
			if wait > 60 {
				return telegramFailure(count, "Telegram 限流等待超过 60 秒，已停止发送")
			}
			timer := time.NewTimer(time.Duration(wait) * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return telegramFailure(count, "Telegram 发送结果未确认，未自动重试")
			case <-timer.C:
			}
		}
	}
	return TelegramResult{MessageCount: count}
}
