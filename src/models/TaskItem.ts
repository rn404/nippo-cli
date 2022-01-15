import { DateFromISOString } from './Date.ts';

export class TaskItem {
  public readonly createdAt: DateFromISOString;
  public updatedAt: DateFromISOString;
  public closed: boolean;

  constructor(
    public content: string,
    createdAt?: DateFromISOString,
    updatedAt?: DateFromISOString,
    closed?: boolean,
  ) {
    const currentTimeStamp = new Date().toISOString();
    this.createdAt = createdAt ?? currentTimeStamp;
    this.updatedAt = updatedAt ?? currentTimeStamp;
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
  return new TaskItem(
    content,
    createdAt,
    updatedAt,
    closed,
  );
};
