package monitor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

type CheckCursor struct {
	CheckedAt time.Time `json:"t"`
	ID        int64     `json:"i"`
}

// EncodeCursor creates a base64 encoded cursor string from a time and ID
func EncodeCursor(t time.Time, id int64) string {
	cursor := CheckCursor{
		CheckedAt: t,
		ID:        id,
	}
	
	data, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeCursor parses a base64 encoded cursor string
func DecodeCursor(s string) (time.Time, int64, error) {
	if s == "" {
		return time.Time{}, 0, nil
	}

	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor encoding")
	}

	var cursor CheckCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor format")
	}

	return cursor.CheckedAt, cursor.ID, nil
}
