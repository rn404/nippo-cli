import { createHash } from 'node:crypto';
import { DateFromISOString } from './Date.ts';

export class TaskItem {
  public readonly hash: string;
  public readonly createdAt: DateFromISOString;
  public updatedAt: DateFromISOString;
  public closed: boolean;

  constructor(
    public content: string,
    createdAt?: DateFromISOString,
    updatedAt?: DateFromISOString,
    closed?: boolean,
    hash?: string,
  ) {
    const currentTimeStamp = new Date().toISOString();
    this.createdAt = createdAt ?? currentTimeStamp;
    this.updatedAt = updatedAt ?? currentTimeStamp;
    this.hash = hash ?? createHash('md5')
      .update(this.createdAt.toString()).toString();
    this.closed = closed ?? false;
  }

  public updateContent(newVal: string) {
    if (this.closed === true) {
      return;
    }

    this.content = newVal;
    this.updatedAt = new Date().toISOString();
  }

  public close() {
    if (this.closed === true) {
      return;
    }

    this.closed = true;
    this.updatedAt = new Date().toISOString();
  }
}

export const createTaskItem = (
  content: string,
  createdAt: DateFromISOString,
  updatedAt: DateFromISOString,
  closed: boolean,
): TaskItem => {
  const hash = createHash('md5').update(createdAt.toString()).toString();
  return new TaskItem(
    content,
    createdAt,
    updatedAt,
    closed,
    hash,
  );
};
