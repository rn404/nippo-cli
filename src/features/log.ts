import { MemoItem } from '../models/MemoItem.ts';
import { TaskItem } from '../models/TaskItem.ts';
import { convertToLogItem, Item, Log } from '../models/LogFile.ts';

export const addItem = (
  log: Log,
  content: Item['content'],
  isTask: boolean,
): Log => {
  if (log.freezed === true) {
    throw new Error('This log file is freezed. No updates.');
  }

  const newItem = isTask === true
    ? new TaskItem(content)
    : new MemoItem(content);

  log.items.push(convertToLogItem(newItem));

  return log;
};

export const deleteItem = (
  log: Log,
  hash: Item['hash'],
): Log => {
  const newItems = log.items.filter((item) => item.hash !== hash);
  log.items = newItems;

  return log;
};

export const finishTaskItem = (
  log: Log,
  hash: Item['hash'],
): {
  log: Log;
  finished: TaskItem;
} => {
  const index = log.items.findIndex((item) => item.hash === hash);
  const targetItem = log.items[index];

  if (targetItem === undefined) {
    throw new Error('Target item is not founded.');
  }

  if (targetItem.closed === undefined) {
    throw new Error('Target item is not task.');
  }

  if (targetItem.closed === true) {
    throw new Error('Target item is already finished.');
  }

  const taskItem = new TaskItem(
    targetItem.content,
    targetItem.createdAt,
    targetItem.updatedAt,
    targetItem.closed,
    targetItem.hash,
  );
  taskItem.close();

  log.items.splice(index, 1, convertToLogItem(taskItem));

  return {
    log,
    finished: taskItem,
  };
};

export const listItems = (
  log: Log,
): { tasks: Array<TaskItem>; memos: Array<MemoItem> } => {
  const tasks: Array<TaskItem> = [];
  const memos: Array<MemoItem> = [];

  log.items.forEach((item) => {
    if (item.closed !== undefined) {
      tasks.push(
        new TaskItem(
          item.content,
          item.createdAt,
          item.updatedAt,
          item.closed,
          item.hash,
        ),
      );
    } else {
      memos.push(
        new MemoItem(
          item.content,
          item.createdAt,
          item.updatedAt,
          item.hash,
        ),
      );
    }
  });

  tasks.sort((a, b) => {
    return new Date(a.createdAt).valueOf() - new Date(b.createdAt).valueOf();
  });
  memos.sort((a, b) => {
    return new Date(a.createdAt).valueOf() - new Date(b.createdAt).valueOf();
  });

  return {
    tasks,
    memos,
  };
};
