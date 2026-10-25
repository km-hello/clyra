package subscription

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"clyra/backend/internal/calendar"
	"clyra/backend/internal/database"
	"clyra/backend/internal/problem"
	"clyra/backend/internal/settings"
	"clyra/backend/internal/textutil"
)

type Subscription struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	AmountMinor      int64   `json:"amount_minor"`
	Currency         string  `json:"currency"`
	Icon             *string `json:"icon"`
	WebsiteURL       *string `json:"website_url"`
	SubscriptionType string  `json:"subscription_type"`
	StartDate        *string `json:"start_date"`
	NextPayment      string  `json:"next_payment"`
	IntervalCount    *int64  `json:"interval_count"`
	IntervalUnit     *string `json:"interval_unit"`
	AnchorMonth      *int64  `json:"anchor_month"`
	AnchorDay        *int64  `json:"anchor_day"`
	NotifyEnabled    bool    `json:"notify_enabled"`
	ReminderDays     *int64  `json:"reminder_days"`
	Status           string  `json:"status"`
	Notes            *string `json:"notes"`
	Version          int64   `json:"version"`
	CreatedAt        int64   `json:"created_at"`
	UpdatedAt        int64   `json:"updated_at"`
}

const subscriptionColumns = "id,name,amount_minor,currency,icon,website_url,subscription_type,start_date,next_payment,interval_count,interval_unit,anchor_month,anchor_day,notify_enabled,reminder_days,status,notes,version,created_at,updated_at"

type subscriptionScanner interface{ Scan(...any) error }

func scanSubscription(row subscriptionScanner) (Subscription, error) {
	var s Subscription
	err := row.Scan(&s.ID, &s.Name, &s.AmountMinor, &s.Currency, &s.Icon, &s.WebsiteURL, &s.SubscriptionType, &s.StartDate, &s.NextPayment, &s.IntervalCount, &s.IntervalUnit, &s.AnchorMonth, &s.AnchorDay, &s.NotifyEnabled, &s.ReminderDays, &s.Status, &s.Notes, &s.Version, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}
func subscriptionFailure(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return problem.New(404, "订阅不存在")
	}
	return problem.New(500, "无法处理订阅数据")
}
func subInteger(raw json.RawMessage, min, max int64) (int64, error) {
	var value float64
	if json.Unmarshal(raw, &value) != nil || string(raw) == "null" || value < float64(min) || value > float64(max) || value != float64(int64(value)) {
		return 0, errors.New("invalid integer")
	}
	return int64(value), nil
}
func subString(raw json.RawMessage) (string, bool) {
	var s string
	ok := string(raw) != "null" && json.Unmarshal(raw, &s) == nil
	return s, ok
}
func subOptionalString(raw json.RawMessage, max int) (*string, error) {
	if string(raw) == "null" {
		return nil, nil
	}
	s, ok := subString(raw)
	if !ok {
		return nil, errors.New("invalid text")
	}
	s = strings.TrimSpace(s)
	if textutil.Length(s) > max {
		return nil, errors.New("text too long")
	}
	if s == "" {
		return nil, nil
	}
	return &s, nil
}

var subScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
var subWebsiteScheme = regexp.MustCompile(`(?i)^https?://`)

func validSubscriptionIcon(s string) bool {
	u, err := url.Parse(s)
	if err == nil && u.IsAbs() {
		return strings.EqualFold(u.Scheme, "https") && u.Host != "" && u.User == nil
	}
	return !subScheme.MatchString(s) && !strings.HasPrefix(s, "//") && utf8.RuneCountInString(s) <= 16
}
func validSubscriptionWebsite(s string) bool {
	if !subWebsiteScheme.MatchString(s) || strings.ContainsRune(s, '\\') || strings.ContainsFunc(s, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return false
	}
	u, err := url.Parse(s)
	if err != nil || u.Hostname() == "" || u.User != nil {
		return false
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n > 65535 {
			return false
		}
	}
	return true
}

func normalizeSubscription(raw map[string]json.RawMessage, current *Subscription) (Subscription, int64, error) {
	return parseSubscriptionInput(raw, current, false)
}
func parseSubscriptionInput(raw map[string]json.RawMessage, current *Subscription, validateOnly bool) (Subscription, int64, error) {
	s := Subscription{NotifyEnabled: true, Status: "active"}
	one := int64(1)
	month := "month"
	s.IntervalCount = &one
	s.IntervalUnit = &month
	if current != nil {
		s = *current
	}
	allowed := map[string]bool{"name": true, "amount_minor": true, "currency": true, "icon": true, "website_url": true, "subscription_type": true, "start_date": true, "next_payment": true, "interval_count": true, "interval_unit": true, "anchor_day": true, "anchor_month": true, "notify_enabled": true, "reminder_days": true, "notes": true}
	if current != nil {
		allowed["status"] = true
		allowed["expected_version"] = true
	}
	for key := range raw {
		if !allowed[key] {
			return s, 0, problem.Validation(key, "未知字段")
		}
	}
	var version int64
	if current != nil {
		var err error
		version, err = subInteger(raw["expected_version"], 1, maxInteger)
		if err != nil {
			return s, 0, problem.Validation("expected_version", "请输入有效版本")
		}
		if len(raw) < 2 {
			return s, 0, problem.Validation("", "请至少修改一个字段")
		}
	}
	if current == nil {
		for _, k := range []string{"name", "amount_minor", "currency", "next_payment", "subscription_type"} {
			if _, ok := raw[k]; !ok {
				return s, 0, problem.Validation(k, "必填项")
			}
		}
	}
	for k, v := range raw {
		bad := func() (Subscription, int64, error) { return s, 0, problem.Validation(k, "请输入有效值") }
		switch k {
		case "amount_minor":
			if strings.TrimSpace(string(v)) == "null" || json.Unmarshal(v, &s.AmountMinor) != nil || s.AmountMinor < 0 || s.AmountMinor > maxInteger {
				return bad()
			}
		case "currency":
			x, ok := subString(v)
			if !ok {
				return bad()
			}
			switch x {
			case "CNY", "USD", "EUR", "GBP", "JPY", "HKD", "TWD", "SGD", "AUD", "CAD":
				s.Currency = x
			default:
				return bad()
			}
		case "name":
			x, ok := subString(v)
			x = strings.TrimSpace(x)
			if !ok || x == "" || textutil.Length(x) > 100 || strings.ContainsRune(x, 0) {
				return bad()
			}
			s.Name = x
		case "next_payment":
			x, ok := subString(v)
			if _, err := calendar.Parse(x); !ok || err != nil {
				return bad()
			}
			s.NextPayment = x
		case "subscription_type":
			x, ok := subString(v)
			if !ok || (x != "one_time" && x != "recurring") {
				return bad()
			}
			s.SubscriptionType = x
		case "status":
			x, ok := subString(v)
			if !ok || (x != "active" && x != "inactive") {
				return bad()
			}
			s.Status = x
		case "website_url":
			x, err := subOptionalString(v, 2048)
			if err != nil || x != nil && !validSubscriptionWebsite(*x) {
				return s, 0, problem.Validation(k, "链接须为 HTTP/HTTPS，不含账号密码，最多 2048 个字符")
			}
			s.WebsiteURL = x
		case "icon", "notes":
			max := 2000
			if k == "icon" {
				max = 2048
			}
			x, err := subOptionalString(v, max)
			if err != nil || (k == "icon" && x != nil && !validSubscriptionIcon(*x)) {
				return bad()
			}
			if k == "icon" {
				s.Icon = x
			} else {
				s.Notes = x
			}
		case "start_date":
			if string(v) == "null" {
				s.StartDate = nil
			} else {
				x, ok := subString(v)
				if _, err := calendar.Parse(x); !ok || err != nil {
					return bad()
				}
				s.StartDate = &x
			}
		case "interval_count", "reminder_days", "anchor_day", "anchor_month":
			var x *int64
			if string(v) != "null" {
				min, max := int64(1), maxInteger
				if k == "reminder_days" {
					min, max = 0, 365
				} else if k == "anchor_day" {
					max = 31
				} else if k == "anchor_month" {
					max = 12
				}
				n, err := subInteger(v, min, max)
				if err != nil {
					return bad()
				}
				x = &n
			}
			switch k {
			case "interval_count":
				s.IntervalCount = x
			case "reminder_days":
				s.ReminderDays = x
			case "anchor_day":
				s.AnchorDay = x
			case "anchor_month":
				s.AnchorMonth = x
			}
		case "interval_unit":
			if string(v) == "null" {
				s.IntervalUnit = nil
			} else {
				x, ok := subString(v)
				if !ok || (x != "day" && x != "week" && x != "month" && x != "year") {
					return bad()
				}
				s.IntervalUnit = &x
			}
		case "notify_enabled":
			if string(v) == "null" || json.Unmarshal(v, &s.NotifyEnabled) != nil {
				return bad()
			}
		}
	}
	if validateOnly {
		return s, version, nil
	}
	if current == nil && s.SubscriptionType == "recurring" {
		for _, k := range []string{"interval_count", "interval_unit"} {
			if len(raw[k]) == 0 || string(raw[k]) == "null" {
				return s, 0, problem.Validation(k, "请输入周期配置")
			}
		}
	}
	if s.SubscriptionType == "one_time" {
		s.IntervalCount = nil
		s.IntervalUnit = nil
		s.StartDate = nil
		s.AnchorDay = nil
		s.AnchorMonth = nil
	} else {
		if s.IntervalCount == nil {
			return s, 0, problem.Validation("interval_count", "请输入正整数周期倍数")
		}
		if s.IntervalUnit == nil {
			return s, 0, problem.Validation("interval_unit", "请选择周期单位")
		}
		changed := current == nil || s.NextPayment != current.NextPayment || s.SubscriptionType != current.SubscriptionType || !subTextEqual(s.IntervalUnit, current.IntervalUnit)
		if changed {
			date, _ := calendar.Parse(s.NextPayment)
			day, month := int64(date.Day()), int64(date.Month())
			if _, supplied := raw["anchor_day"]; !supplied {
				s.AnchorDay = nil
				if *s.IntervalUnit == "month" || *s.IntervalUnit == "year" {
					s.AnchorDay = &day
				}
			}
			if _, supplied := raw["anchor_month"]; !supplied {
				s.AnchorMonth = nil
				if *s.IntervalUnit == "year" {
					s.AnchorMonth = &month
				}
			}
		}
		needsDay := *s.IntervalUnit == "month" || *s.IntervalUnit == "year"
		if needsDay != (s.AnchorDay != nil) {
			return s, 0, problem.Validation("anchor_day", "续订日期配置无效")
		}
		if (*s.IntervalUnit == "year") != (s.AnchorMonth != nil) {
			return s, 0, problem.Validation("anchor_month", "续订日期配置无效")
		}
		anchored, err := calendar.Advance(s.NextPayment, 0, *s.IntervalUnit, s.AnchorDay, s.AnchorMonth)
		if err != nil || anchored != s.NextPayment {
			return s, 0, problem.Validation("anchor_day", "续订日期配置与到期日不一致")
		}
		if s.StartDate == nil {
			return s, 0, problem.Validation("start_date", "请输入开始日期")
		}
		if *s.StartDate >= s.NextPayment {
			return s, 0, problem.Validation("start_date", "开始日期必须早于到期日期")
		}
	}
	return s, version, nil
}
func subTextEqual(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func subValues(s Subscription) []any {
	return []any{s.Name, s.AmountMinor, s.Currency, s.Icon, s.WebsiteURL, s.SubscriptionType, s.StartDate, s.NextPayment, s.IntervalCount, s.IntervalUnit, s.AnchorMonth, s.AnchorDay, s.NotifyEnabled, s.ReminderDays, s.Status, s.Notes}
}

type Service struct{ DB *sql.DB }

const maxInteger int64 = 9007199254740991

func (a Service) Create(raw map[string]json.RawMessage) (Subscription, error) {
	s, _, err := normalizeSubscription(raw, nil)
	if err != nil {
		return Subscription{}, err
	}
	s.ID = database.NewID()
	s.Version = 1
	s.CreatedAt = time.Now().UnixMilli()
	s.UpdatedAt = s.CreatedAt
	values := append([]any{s.ID}, subValues(s)...)
	values = append(values, s.Version, s.CreatedAt, s.UpdatedAt)
	_, err = a.DB.Exec("INSERT INTO subscriptions ("+subscriptionColumns+") VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", values...)
	if err != nil {
		return Subscription{}, subscriptionFailure(err)
	}
	return s, nil
}
func (a Service) Mutate(id string, raw map[string]json.RawMessage, renew bool) (Subscription, error) {
	var err error
	// Validate the request before inspecting records, matching the public API contract.
	var expected int64
	if renew {
		expected, err = subInteger(raw["expected_version"], 1, maxInteger)
		if len(raw) != 1 || err != nil {
			return Subscription{}, problem.Validation("expected_version", "请输入有效版本")
		}
	} else {
		_, expected, err = parseSubscriptionInput(raw, &Subscription{SubscriptionType: "one_time"}, true)
		if err != nil {
			return Subscription{}, err
		}
	}
	tx, err := a.DB.Begin()
	if err != nil {
		return Subscription{}, subscriptionFailure(err)
	}
	defer tx.Rollback()
	current, err := scanSubscription(tx.QueryRow("SELECT "+subscriptionColumns+" FROM subscriptions WHERE id=?", id))
	if err != nil {
		return Subscription{}, subscriptionFailure(err)
	}
	if current.Version != expected {
		return Subscription{}, problem.New(409, "订阅已变化，请刷新后重试")
	}
	next := current
	if renew {
		if current.Status != "active" || current.SubscriptionType != "recurring" {
			return Subscription{}, problem.Validation("", "仅启用的周期性订阅可以续订")
		}
		date, err := calendar.Advance(current.NextPayment, *current.IntervalCount, *current.IntervalUnit, current.AnchorDay, current.AnchorMonth)
		if err != nil {
			return Subscription{}, problem.Validation("next_payment", "续订日期超出有效范围")
		}
		next.NextPayment = date
	} else {
		next, _, err = normalizeSubscription(raw, &current)
		if err != nil {
			return Subscription{}, err
		}
	}
	next.Version = current.Version + 1
	next.UpdatedAt = time.Now().UnixMilli()
	args := append(subValues(next), next.Version, next.UpdatedAt, id, expected)
	result, err := tx.Exec("UPDATE subscriptions SET name=?,amount_minor=?,currency=?,icon=?,website_url=?,subscription_type=?,start_date=?,next_payment=?,interval_count=?,interval_unit=?,anchor_month=?,anchor_day=?,notify_enabled=?,reminder_days=?,status=?,notes=?,version=?,updated_at=? WHERE id=? AND version=?", args...)
	if err != nil {
		return Subscription{}, subscriptionFailure(err)
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return Subscription{}, problem.New(409, "订阅已变化，请刷新后重试")
	}
	if err = tx.Commit(); err != nil {
		return Subscription{}, subscriptionFailure(err)
	}
	return next, nil
}

var subQueryInteger = regexp.MustCompile(`^[1-9][0-9]*$`)

func subQueryInt(value string, max int64) (int64, error) {
	if !subQueryInteger.MatchString(value) {
		return 0, errors.New("invalid integer")
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n > max {
		return 0, errors.New("invalid integer")
	}
	return n, nil
}
func (a Service) Delete(id string, q url.Values) error {
	expected, err := subQueryInt(q.Get("expected_version"), maxInteger)
	if err != nil || len(q) != 1 || len(q["expected_version"]) != 1 {
		return problem.Validation("expected_version", "请输入有效版本")
	}
	tx, err := a.DB.Begin()
	if err != nil {
		return subscriptionFailure(err)
	}
	defer tx.Rollback()
	var version int64
	if err = tx.QueryRow("SELECT version FROM subscriptions WHERE id=?", id).Scan(&version); err != nil {
		return subscriptionFailure(err)
	}
	result, err := tx.Exec("DELETE FROM subscriptions WHERE id=? AND version=?", id, expected)
	if err != nil {
		return subscriptionFailure(err)
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return problem.New(409, "订阅已变化，请刷新后重试")
	}
	if err = tx.Commit(); err != nil {
		return subscriptionFailure(err)
	}
	return nil
}
func (a Service) List(q url.Values) (map[string]any, error) {
	allowed := map[string]bool{"search": true, "status": true, "due": true, "sort": true, "page": true, "page_size": true}
	for k, v := range q {
		if !allowed[k] || len(v) != 1 {
			return nil, problem.Validation(k, "请输入有效查询条件")
		}
	}
	defaults := func(k, fallback string) string {
		if _, ok := q[k]; !ok {
			return fallback
		}
		return q.Get(k)
	}
	search := strings.TrimSpace(q.Get("search"))
	status := defaults("status", "active")
	due := defaults("due", "all")
	sort := defaults("sort", "next_payment_asc")
	page, err := subQueryInt(defaults("page", "1"), maxInteger)
	size, sizeErr := subQueryInt(defaults("page_size", "50"), 100)
	if textutil.Length(search) > 100 || strings.ContainsRune(search, 0) {
		return nil, problem.Validation("search", "请输入有效搜索条件")
	}
	if status != "active" && status != "inactive" && status != "all" {
		return nil, problem.Validation("status", "请输入有效状态")
	}
	if due != "all" && due != "overdue" && due != "today" && due != "window" && due != "next_30_days" {
		return nil, problem.Validation("due", "请输入有效到期范围")
	}
	if sort != "next_payment_asc" && sort != "next_payment_desc" && sort != "name_asc" {
		return nil, problem.Validation("sort", "请输入有效排序")
	}
	if err != nil || sizeErr != nil {
		return nil, problem.Validation("page", "请输入有效分页参数")
	}
	tx, err := a.DB.Begin()
	if err != nil {
		return nil, subscriptionFailure(err)
	}
	defer tx.Rollback()
	preferences, err := settings.Read(tx)
	if err != nil {
		return nil, subscriptionFailure(err)
	}
	today, err := calendar.Today(time.Now(), preferences.Timezone)
	if err != nil {
		return nil, subscriptionFailure(err)
	}
	filters := []string{"1=1"}
	args := []any{}
	if status != "all" {
		filters = append(filters, "status=?")
		args = append(args, status)
	}
	if search != "" {
		escaped := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(search)
		filters = append(filters, "name LIKE ? ESCAPE '\\'")
		args = append(args, "%"+escaped+"%")
	}
	switch due {
	case "overdue":
		filters = append(filters, "next_payment<?")
		args = append(args, today)
	case "today":
		filters = append(filters, "next_payment=?")
		args = append(args, today)
	case "window":
		filters = append(filters, "julianday(next_payment)-julianday(?) BETWEEN 0 AND COALESCE(reminder_days,?)")
		args = append(args, today, preferences.DefaultReminderDays)
	case "next_30_days":
		filters = append(filters, "julianday(next_payment)-julianday(?) BETWEEN 0 AND 30")
		args = append(args, today)
	}
	where := strings.Join(filters, " AND ")
	var total int64
	if err = tx.QueryRow("SELECT count(*) FROM subscriptions WHERE "+where, args...).Scan(&total); err != nil {
		return nil, subscriptionFailure(err)
	}
	pages := (total + size - 1) / size
	if pages < 1 {
		pages = 1
	}
	data := []Subscription{}
	if page <= pages {
		order := "next_payment ASC"
		if sort == "next_payment_desc" {
			order = "next_payment DESC"
		}
		if sort == "name_asc" {
			order = "name ASC"
		}
		queryArgs := append(append([]any{}, args...), size, (page-1)*size)
		rows, err := tx.Query("SELECT "+subscriptionColumns+" FROM subscriptions WHERE "+where+" ORDER BY (status='inactive') ASC,"+order+",id ASC LIMIT ? OFFSET ?", queryArgs...)
		if err != nil {
			return nil, subscriptionFailure(err)
		}
		for rows.Next() {
			s, scanErr := scanSubscription(rows)
			if scanErr != nil {
				rows.Close()
				return nil, subscriptionFailure(scanErr)
			}
			data = append(data, s)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, subscriptionFailure(err)
		}
	}
	var hasAny bool
	if err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM subscriptions)").Scan(&hasAny); err != nil {
		return nil, subscriptionFailure(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, subscriptionFailure(err)
	}
	return map[string]any{"data": data, "meta": map[string]any{"page": page, "page_size": size, "total": total, "total_pages": pages, "has_any_subscriptions": hasAny, "today": today, "timezone": preferences.Timezone, "default_reminder_days": preferences.DefaultReminderDays, "theme": preferences.Theme}}, nil
}
