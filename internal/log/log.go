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
	// ErrNotTask is returned when finishing an item that is a memo.
	ErrNotTask = errors.New("target item is not a task")
	// ErrAlreadyFinished is returned when finishing a closed task.
	ErrAlreadyFinished = errors.New("target item is already finished")
	// ErrAlreadyStarted is returned when starting a started task.
	ErrAlreadyStarted = errors.New("target item is already started")
	// ErrEmptyTag is returned when a tag is empty after trimming.
	ErrEmptyTag = errors.New("tag must not be empty")
	// ErrEmptyContent is returned when content is empty after trimming.
	ErrEmptyContent = errors.New("content must not be empty")
)

// Add appends a new task or memo to the log and returns the created item.
// The item is given a fresh hash even if it happens to collide with an
// existing item's, so hashes stay unique within the log. content is
// rejected with ErrEmptyContent when empty after trimming whitespace;
// otherwise it is stored exactly as given, without trimming.
func Add(l *model.Log, content string, isTask bool) (model.Item, error) {
	if strings.TrimSpace(content) == "" {
		return model.Item{}, ErrEmptyContent
	}

	var item model.Item
	if isTask {
		item = model.NewTaskItem(content)
	} else {
		item = model.NewMemoItem(content)
	}
	item.Hash = uniqueID(l.Items, model.NewID)
	l.Items = append(l.Items, item)
	return item, nil
}

// uniqueID generates an ID that none of items already has, drawing
// candidates from next (model.NewID in production; tests can pass a
// stub to force a collision deterministically without any package-
// level mutable state).
func uniqueID(items []model.Item, next func() string) string {
	id := next()
	for HashExists(items, id) {
		id = next()
	}
	return id
}

// HashExists reports whether any item already has hash. Exported so
// callers outside this package (e.g. a multi-day search for a hash)
// can reuse the same check instead of re-scanning by hand.
func HashExists(items []model.Item, hash string) bool {
	return indexOf(items, hash) != -1
}

// CarryForward returns fresh copies of every unfinished task in items,
// ready to seed a new day's log: each copy gets its own fresh hash
// (unique among the copies), fresh CreatedAt/UpdatedAt, and content and
// tags carried over unchanged. StartedAt and Closed are not carried
// over, so a copy starts today as untouched, open work. CarriedFrom on
// each copy points back to "sourceDate:<original hash>". Memos and
// already-closed tasks are left where they are, not copied.
func CarryForward(items []model.Item, sourceDate string) []model.Item {
	var out []model.Item
	for _, item := range items {
		if !item.IsTask() || item.IsClosed() {
			continue
		}

		clone := model.NewTaskItem(item.Content)
		clone.Hash = uniqueID(out, model.NewID)
		if len(item.Tags) > 0 {
			clone.Tags = append([]string(nil), item.Tags...)
		}
		from := sourceDate + ":" + item.Hash
		clone.CarriedFrom = &from

		out = append(out, clone)
	}
	return out
}

// Delete removes the item matching hash from the log and returns it.
// Only the first match is removed, so behavior stays well-defined even
// if two items were ever created with colliding hashes.
func Delete(l *model.Log, hash string) (model.Item, error) {
	i := indexOf(l.Items, hash)
	if i == -1 {
		return model.Item{}, fmt.Errorf("target item %q is not found", hash)
	}
	item := l.Items[i]
	l.Items = append(l.Items[:i], l.Items[i+1:]...)
	return item, nil
}

// indexOf returns the index of the first item with hash, or -1.
func indexOf(items []model.Item, hash string) int {
	for i, item := range items {
		if item.Hash == hash {
			return i
		}
	}
	return -1
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

// Find returns the item matching hash without modifying the log.
func Find(l *model.Log, hash string) (model.Item, error) {
	i := indexOf(l.Items, hash)
	if i == -1 {
		return model.Item{}, fmt.Errorf("target item %q is not found", hash)
	}
	return l.Items[i], nil
}

// Edit replaces the content of the item matching hash and updates
// UpdatedAt. Unlike Start, Edit does not validate the item's
// lifecycle state or kind — a memo or a task in any state can be
// edited. newContent is not validated either (an empty string is
// accepted as-is), consistent with Add's lack of content validation.
func Edit(l *model.Log, hash, newContent string) (model.Item, error) {
	for i, item := range l.Items {
		if item.Hash != hash {
			continue
		}

		item.Content = newContent
		item.UpdatedAt = model.NowISO()
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

	sort.SliceStable(tasks, byCreatedAt(tasks))
	sort.SliceStable(memos, byCreatedAt(memos))

	return tasks, memos
}

// SplitByStatus separates the log items into three status groups —
// closed tasks, non-closed tasks (open or started), and memos — each
// sorted by creation time in ascending order.
func SplitByStatus(l model.Log) (closedTasks, openTasks, memos []model.Item) {
	for _, item := range l.Items {
		switch {
		case item.IsClosed():
			closedTasks = append(closedTasks, item)
		case item.IsTask():
			openTasks = append(openTasks, item)
		default:
			memos = append(memos, item)
		}
	}

	sort.SliceStable(closedTasks, byCreatedAt(closedTasks))
	sort.SliceStable(openTasks, byCreatedAt(openTasks))
	sort.SliceStable(memos, byCreatedAt(memos))

	return closedTasks, openTasks, memos
}

// Timeline returns every item (tasks and memos together) sorted by
// creation time, for a single chronological list.
func Timeline(l model.Log) []model.Item {
	items := make([]model.Item, len(l.Items))
	copy(items, l.Items)
	sort.SliceStable(items, byCreatedAt(items))
	return items
}

// byCreatedAt orders items by creation time, oldest first. createdAt
// is a fixed-width UTC ISO string, so lexicographic order equals
// chronological order. Ties (items created within the same
// millisecond) are broken by insertion order via a stable sort, so
// output order is deterministic even when timestamps collide.
func byCreatedAt(items []model.Item) func(i, j int) bool {
	return func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt }
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
