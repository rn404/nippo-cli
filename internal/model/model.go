// Package model defines the log data structures persisted as JSON.
// The JSON layout must stay compatible with files written by the
// former Deno implementation (see testdata/legacy-samples/).
package model

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// DateLayout is the file-name date format (yyyy-MM-dd, local time).
const DateLayout = "2006-01-02"

// isoLayout mirrors JavaScript's Date.toISOString() (UTC, milliseconds).
const isoLayout = "2006-01-02T15:04:05.000Z"

// idBytes is the entropy of an item/file ID (hex-encoded to 8 chars).
const idBytes = 4

// Item is a single log entry. The presence of Closed distinguishes a
// task (non-nil) from a memo (nil).
type Item struct {
	Hash      string `json:"hash"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Closed    *bool  `json:"closed,omitempty"`
}

// NewTaskItem creates an open task with a fresh ID and timestamps.
func NewTaskItem(content string) Item {
	now := NowISO()
	closed := false

	return Item{
		Hash:      NewID(),
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
		Closed:    &closed,
	}
}

// NewMemoItem creates a memo with a fresh ID and timestamps.
func NewMemoItem(content string) Item {
	now := NowISO()

	return Item{
		Hash:      NewID(),
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsTask reports whether the item is a task (has a closed flag).
func (i Item) IsTask() bool {
	return i.Closed != nil
}

// IsClosed reports whether the item is a finished task.
func (i Item) IsClosed() bool {
	return i.Closed != nil && *i.Closed
}

// Log is the body of one daily log file.
type Log struct {
	Hash    string `json:"hash"`
	Freezed bool   `json:"freezed"`
	Items   []Item `json:"items"`
}

// NewLog creates an empty, unfrozen log body.
func NewLog() Log {
	return Log{
		Hash:    NewID(),
		Freezed: false,
		Items:   []Item{},
	}
}

// NewID returns a random 8-character hex ID used as item/file hash.
func NewID() string {
	buf := make([]byte, idBytes)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand never fails on supported platforms; fall back to time.
		return fmt.Sprintf("%08x", time.Now().UnixNano()&0xffffffff)
	}

	return hex.EncodeToString(buf)
}

// NowISO returns the current UTC time in JavaScript toISOString() format.
func NowISO() string {
	return time.Now().UTC().Format(isoLayout)
}

// ParseDate strictly parses a yyyy-MM-dd string.
func ParseDate(value string) (time.Time, error) {
	t, err := time.ParseInLocation(DateLayout, value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q: expected format yyyy-MM-dd", value)
	}

	return t, nil
}

// IsDateString reports whether value is a strict yyyy-MM-dd date.
func IsDateString(value string) bool {
	_, err := ParseDate(value)
	return err == nil
}

// Today returns the current local date as yyyy-MM-dd, matching the
// daily log file naming.
func Today() string {
	return time.Now().Format(DateLayout)
}
