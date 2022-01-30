import type { DateFromISOString, DateString } from './Date.ts';
import { MemoItem } from './MemoItem.ts';
import { TaskItem } from './TaskItem.ts';

export interface LogFile {
  path: string; // File full path includes file name
  fileName: DateString; // File name exclude extension
  body: Log;
}

export interface Item {
  hash: string;
  createdAt: DateFromISOString;
  updatedAt: DateFromISOString;
  content: string;
  closed?: boolean;
}

export interface Log {
  hash: string;
  freezed: boolean;
  items: Array<Item>;
}

export const convertToLogItem = (item: MemoItem | TaskItem): Item => {
  if (item instanceof MemoItem) {
    return {
      hash: item.hash,
      content: item.content,
      createdAt: item.createdAt,
      updatedAt: item.updatedAt,
    };
  } else if (item instanceof TaskItem) {
    return {
      hash: item.hash,
      content: item.content,
      createdAt: item.createdAt,
      updatedAt: item.updatedAt,
      closed: item.closed,
    };
  }

  throw new Error('item is unspecified.');
};
