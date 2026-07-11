// Package log provides operations on the items of a daily log.
package log

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/rn404/nippo-cli/internal/model"
)

var (
	// ErrFreezed is returned when modifying items of a frozen log.
	ErrFreezed = errors.New("this log file is freezed, no updates")
	// ErrNotTask is returned when finishing an item that is a memo.
	ErrNotTask = errors.New("target item is not a task")
	// ErrAlreadyFinished is returned when finishing a closed task.
	ErrAlreadyFinished = errors.New("target item is already finished")
	// ErrAlreadyStarted is returned when starting a started task.
	ErrAlreadyStarted = errors.New("target item is already started")
	// ErrEmptyTag is returned when a tag is empty after trimming.
	ErrEmptyTag = errors.New("tag must not be empty")
)

// Add appends a new task or memo to the log and returns the created item.
func Add(l *model.Log, content string, isTask bool) (model.Item, error) {
	if l.Freezed {
		return model.Item{}, ErrFreezed
	}

	var item model.Item
	if isTask {
		item = model.NewTaskItem(content)
	} else {
		item = model.NewMemoItem(content)
	}
	l.Items = append(l.Items, item)
	return item, nil
}

// Delete removes all items matching hash from the log.
func Delete(l *model.Log, hash string) {
	items := l.Items[:0]
	for _, item := range l.Items {
		if item.Hash != hash {
			items = append(items, item)
		}
	}
	l.Items = items
}

// Finish closes the task matching hash and returns the updated item.
func Finish(l *model.Log, hash string) (model.Item, error) {
	for i, item := range l.Items {
		if item.Hash != hash {
			continue
		}
		if !item.IsTask() {
			return model.Item{}, ErrNotTask
		}
		if item.IsClosed() {
			return model.Item{}, ErrAlreadyFinished
		}

		closed := true
		item.Closed = &closed
		item.UpdatedAt = model.NowISO()
		l.Items[i] = item
		return item, nil
	}

	return model.Item{}, fmt.Errorf("target item %q is not found", hash)
}

// Start marks the task matching hash as started and returns the
// updated item.
func Start(l *model.Log, hash string) (model.Item, error) {
	for i, item := range l.Items {
		if item.Hash != hash {
			continue
		}
		if !item.IsTask() {
			return model.Item{}, ErrNotTask
		}
		if item.IsClosed() {
			return model.Item{}, ErrAlreadyFinished
		}
		if item.IsStarted() {
			return model.Item{}, ErrAlreadyStarted
		}

		now := model.NowISO()
		item.StartedAt = &now
		item.UpdatedAt = now
		l.Items[i] = item
		return item, nil
	}

	return model.Item{}, fmt.Errorf("target item %q is not found", hash)
}

// normalizeTags trims whitespace and deduplicates tags while keeping
// their order. Empty tags and tags containing whitespace are rejected.
func normalizeTags(tags []string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return nil, ErrEmptyTag
		}
		if strings.ContainsAny(tag, " \t") {
			return nil, fmt.Errorf("tag %q must not contain whitespace", tag)
		}
		if !seen[tag] {
			seen[tag] = true
			out = append(out, tag)
		}
	}
	return out, nil
}

// AddTags adds tags to the item matching hash (tasks and memos alike)
// and returns the updated item. Already present tags are skipped.
func AddTags(l *model.Log, hash string, tags []string) (model.Item, error) {
	tags, err := normalizeTags(tags)
	if err != nil {
		return model.Item{}, err
	}

	for i, item := range l.Items {
		if item.Hash != hash {
			continue
		}

		changed := false
		for _, tag := range tags {
			if !item.HasTag(tag) {
				item.Tags = append(item.Tags, tag)
				changed = true
			}
		}
		if changed {
			item.UpdatedAt = model.NowISO()
		}
		l.Items[i] = item
		return item, nil
	}

	return model.Item{}, fmt.Errorf("target item %q is not found", hash)
}

// RemoveTags removes tags from the item matching hash and returns the
// updated item. Tags the item does not carry are ignored.
func RemoveTags(l *model.Log, hash string, tags []string) (model.Item, error) {
	tags, err := normalizeTags(tags)
	if err != nil {
		return model.Item{}, err
	}

	drop := map[string]bool{}
	for _, tag := range tags {
		drop[tag] = true
	}

	for i, item := range l.Items {
		if item.Hash != hash {
			continue
		}

		kept := item.Tags[:0]
		for _, tag := range item.Tags {
			if !drop[tag] {
				kept = append(kept, tag)
			}
		}
		if len(kept) != len(item.Tags) {
			item.UpdatedAt = model.NowISO()
		}
		if len(kept) == 0 {
			kept = nil
		}
		item.Tags = kept
		l.Items[i] = item
		return item, nil
	}

	return model.Item{}, fmt.Errorf("target item %q is not found", hash)
}

// FilterByTags returns the items matching the tags: all of them by
// default, or at least one when anyMatch is true.
func FilterByTags(items []model.Item, tags []string, anyMatch bool) []model.Item {
	var out []model.Item
	for _, item := range items {
		matched := 0
		for _, tag := range tags {
			if item.HasTag(tag) {
				matched++
			}
		}
		if (anyMatch && matched > 0) || (!anyMatch && matched == len(tags)) {
			out = append(out, item)
		}
	}
	return out
}

// Split separates the log items into tasks and memos, each sorted by
// creation time in ascending order.
func Split(l model.Log) (tasks, memos []model.Item) {
	for _, item := range l.Items {
		if item.IsTask() {
			tasks = append(tasks, item)
		} else {
			memos = append(memos, item)
		}
	}

	// createdAt is a fixed-width UTC ISO string, so lexicographic
	// order equals chronological order.
	byCreatedAt := func(items []model.Item) func(i, j int) bool {
		return func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt }
	}
	sort.Slice(tasks, byCreatedAt(tasks))
	sort.Slice(memos, byCreatedAt(memos))

	return tasks, memos
}

// CountUnfinished returns the number of open tasks.
func CountUnfinished(tasks []model.Item) int {
	count := 0
	for _, task := range tasks {
		if !task.IsClosed() {
			count++
		}
	}
	return count
}
