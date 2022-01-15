import type { DateFromISOString } from './Date.ts';
import { MemoItem } from './MemoItem.ts';
import { TaskItem } from './TaskItem.ts';

export interface LogFileInfo {
  path: string;
  fileName: string;
  createdAt: DateFromISOString;
  data: Log;
}

export interface LogItem {
  createdAt: DateFromISOString;
  updatedAt: DateFromISOString;
  content: string;
  closed?: string;
}

export interface Log {
  hash: string;
  freezed: boolean;
  logs: Array<LogItem>;
}

export const convertToLogItem = (item: MemoItem | TaskItem): LogItem => {
  if (item instanceof MemoItem) {
    return {
      content: item.content,
      createdAt: item.createdAt,
      updatedAt: item.updatedAt,
    };
  } else if (item instanceof TaskItem) {
    return {
      content: item.content,
      createdAt: item.createdAt,
      updatedAt: item.updatedAt,
      closed: `${item.closed}`,
    };
  }

  throw new Error('item is unspecified.');
};
