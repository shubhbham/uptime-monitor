package utils

import "time"

func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return "< 1 minute"
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return string(rune(minutes)) + " minutes"
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return string(rune(hours)) + " hours"
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return string(rune(days)) + " days"
}

func TimePtr(t time.Time) *time.Time {
	return &t
}

func BoolPtr(b bool) *bool {
	return &b
}

func IntPtr(i int) *int {
	return &i
}

func StringPtr(s string) *string {
	return &s
}