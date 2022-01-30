import { createHash } from './../dependencies.ts';
import { DateFromISOString } from './Date.ts';

export class MemoItem {
  public readonly hash: string;
  public readonly createdAt: DateFromISOString;
  public updatedAt: DateFromISOString;

  constructor(
    public content: string,
    createdAt?: DateFromISOString,
    updateAt?: DateFromISOString,
    hash?: string,
  ) {
    const currentTimeStamp = new Date().toISOString();
    this.createdAt = createdAt ?? currentTimeStamp;
    this.updatedAt = updateAt ?? currentTimeStamp;
    this.hash = hash ?? createHash('md5')
      .update(this.createdAt.toString()).toString();
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
  const hash = createHash('md5').update(createdAt.toString()).toString();
  return new MemoItem(
    content,
    createdAt,
    updatedAt,
    hash,
  );
};
