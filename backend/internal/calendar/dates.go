package calendar

import (
	"fmt"
	"time"
)

func Parse(value string) (time.Time, error) {
	if len(value) != 10 {
		return time.Time{}, fmt.Errorf("invalid date")
	}
	d, err := time.Parse("2006-01-02", value)
	if err != nil || d.Year() < 1 || d.Year() > 9999 {
		return time.Time{}, fmt.Errorf("invalid date")
	}
	return d, nil
}
func Today(now time.Time, timezone string) (string, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return "", err
	}
	d := now.In(loc)
	if d.Year() < 1 || d.Year() > 9999 {
		return "", fmt.Errorf("date out of range")
	}
	return d.Format("2006-01-02"), nil
}
func Advance(value string, count int64, unit string, anchorDay, anchorMonth *int64) (string, error) {
	d, err := Parse(value)
	if err != nil {
		return "", err
	}
	invalid := fmt.Errorf("date out of range")
	if count > 9007199254740991 || count < -9007199254740991 {
		return "", invalid
	}
	year, month, day := int64(d.Year()), int64(d.Month()), int64(d.Day())
	switch unit {
	case "day", "week":
		if unit == "week" {
			if count > 4000000 || count < -4000000 {
				return "", invalid
			}
			count *= 7
		}
		if count > 4000000 || count < -4000000 {
			return "", invalid
		}
		d = d.AddDate(0, 0, int(count))
	case "month", "year":
		if anchorDay != nil {
			day = *anchorDay
		}
		if day < 1 || day > 31 {
			return "", invalid
		}
		if unit == "month" {
			target := year*12 + month - 1 + count
			if target < 12 || target > 9999*12+11 {
				return "", invalid
			}
			year = target / 12
			month = target%12 + 1
		} else {
			year += count
			if anchorMonth != nil {
				month = *anchorMonth
			}
			if year < 1 || year > 9999 || month < 1 || month > 12 {
				return "", invalid
			}
		}
		last := int64(time.Date(int(year), time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day())
		if day > last {
			day = last
		}
		d = time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC)
	default:
		return "", invalid
	}
	if d.Year() < 1 || d.Year() > 9999 {
		return "", invalid
	}
	return d.Format("2006-01-02"), nil
}
