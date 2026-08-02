package data

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const dateTimeLayout = "2006-01-02 15:04:05"

var dateTimeInputLayouts = []string{
	time.RFC3339,
	dateTimeLayout,
	"2006-01-02",
}

// DateTime accepts common API datetime strings such as "2006-01-02 15:04:05".
type DateTime time.Time

func (dt *DateTime) UnmarshalJSON(b []byte) error {
	raw := strings.TrimSpace(string(b))
	if raw == "" || raw == "null" {
		return nil
	}
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	parsed, err := parseDateTime(raw)
	if err != nil {
		return err
	}
	*dt = DateTime(parsed)
	return nil
}

func (dt DateTime) MarshalJSON() ([]byte, error) {
	t := time.Time(dt)
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Format(dateTimeLayout))
}

func (dt *DateTime) TimePtr() *time.Time {
	if dt == nil {
		return nil
	}
	t := time.Time(*dt)
	if t.IsZero() {
		return nil
	}
	return &t
}

func DateTimeFromPtr(t *time.Time) *DateTime {
	if t == nil || t.IsZero() {
		return nil
	}
	dt := DateTime(*t)
	return &dt
}

func parseDateTime(value string) (time.Time, error) {
	for _, layout := range dateTimeInputLayouts {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid datetime %q, expected formats like %q or RFC3339", value, dateTimeLayout)
}
