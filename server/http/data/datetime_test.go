package data

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDateTimeUnmarshalJSON(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "space separated", input: `"2026-07-02 00:00:00"`, want: "2026-07-02 00:00:00"},
		{name: "rfc3339", input: `"2026-07-02T00:00:00+08:00"`, want: "2026-07-02 00:00:00"},
		{name: "date only", input: `"2026-07-02"`, want: "2026-07-02 00:00:00"},
		{name: "invalid", input: `"02/07/2026"`, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var dt DateTime
			err := json.Unmarshal([]byte(tc.input), &dt)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := time.Time(dt).Format(dateTimeLayout)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDateTimeMarshalJSON(t *testing.T) {
	parsed, err := time.ParseInLocation(dateTimeLayout, "2026-07-02 15:04:05", time.Local)
	if err != nil {
		t.Fatal(err)
	}
	dt := DateTime(parsed)
	b, err := json.Marshal(dt)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"2026-07-02 15:04:05"` {
		t.Fatalf("unexpected marshal result: %s", b)
	}
}
