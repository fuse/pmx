package cli

import (
	"testing"
	"time"
)

func TestFormatRemainingMinutes(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0 min"},
		{30 * time.Second, "1 min"},
		{90 * time.Second, "2 min"},
		{59 * time.Minute, "59 min"},
		{time.Hour + 30*time.Minute, "90 min"},
	}
	for _, tc := range tests {
		if got := formatRemainingMinutes(tc.d); got != tc.want {
			t.Fatalf("formatRemainingMinutes(%s) = %q, want %q", tc.d, got, tc.want)
		}
	}
}
