import { DateFromISOString } from './Date.ts';

export class MemoItem {
  public readonly createdAt: DateFromISOString;
  public updatedAt: DateFromISOString;

  constructor(
    public content: string,
    createdAt?: DateFromISOString,
    updateAt?: DateFromISOString,
  ) {
    const currentTimeStamp = new Date().toISOString();
    this.createdAt = createdAt ?? currentTimeStamp;
    this.updatedAt = updateAt ?? currentTimeStamp;
  }

  public updateContent(newVal: string) {
    this.content = newVal;
    this.updatedAt = new Date().toISOString();
  }
}

export const createMemoItem = (
  content: string,
  createdAt: DateFromISOString,
  updatedAt: DateFromISOString,
): MemoItem => {
  return new MemoItem(
    content,
    createdAt,
    updatedAt,
  );
};
