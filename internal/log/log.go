// Package log provides operations on the items of a daily log.
package log

import (
	"errors"
	"fmt"
	"sort"

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
