import {
  join,
  ensureDir,
  ensureFile,
  format,
  createHash
} from './dependencies.ts'

import { Log, LogFileInfo, LogItem } from './models/Log.ts';
import { createMemoItem, MemoItem } from './models/MemoItem.ts';
import { createTaskItem, TaskItem } from './models/TaskItem.ts';
import { LOG_FILE_EXT } from './const.ts'

export const getCurrentFile = async (logDir: string): Promise<LogFileInfo> => {
  const currentTime = new Date();
  const createdAt = currentTime.toISOString();
  const todayLogFileName = format(currentTime, 'yyyyMMdd', {}) +
    `.${LOG_FILE_EXT}`;
  const path = join(Deno.cwd(), logDir);
  const hash = createHash('md5').update(currentTime.toString()).toString()
  const freezed = false

  try {
    const stat = await Deno.lstat(join(path, todayLogFileName));
    if (stat.isFile === false) {
      throw new Error('Log is not file.');
    }
  } catch (error) {
    if (error instanceof Deno.errors.NotFound) {
      await create(path, todayLogFileName);
      return {
        path,
        fileName: todayLogFileName,
        createdAt,
        data: {
          hash, freezed, logs: []
        },
      };
    }
  }

  const resource: Log = JSON.parse(
    await Deno.readTextFile(join(path, todayLogFileName)),
  );

  return {
    path,
    fileName: todayLogFileName,
    createdAt,
    data: resource,
  };
};

export const create = async (path: string, fileName: string): Promise<void> => {
  await ensureDir(path);
  await ensureFile(join(path, fileName));

  const encoder = new TextEncoder();
  const data = encoder.encode(JSON.stringify({}));
  await Deno.writeFile(join(path, fileName), data);
};

export const update = async (
  content = {},
  path: string,
  fileName: string,
): Promise<void> => {
  const targetFile = join(path, fileName);
  const encoder = new TextEncoder();
  const data = encoder.encode(JSON.stringify(content));
  await Deno.writeFile(targetFile, data);
};

export const addItem = async (newItem: LogItem, oldData: Log): Promise<Log> => {
  const data = Object.assign({}, oldData);
  data.logs.push(newItem);
  return data;
};

export const parse = (log: Log): {
  tasks: Array<TaskItem>;
  memos: Array<MemoItem>;
} => {
  const tasks: Array<TaskItem> = [];
  const memos: Array<MemoItem> = [];

  log.logs.forEach((item) => {
    if (item.closed !== undefined) {
      tasks.push(createTaskItem(
        item.content,
        item.createdAt,
        item.updatedAt,
        item.closed === 'true',
      ));
    } else {
      memos.push(createMemoItem(
        item.content,
        item.createdAt,
        item.updatedAt,
      ));
    }
  });

  // TODO sort する

  return {
    tasks,
    memos,
  };
};
