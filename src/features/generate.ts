import { format } from '../dependencies.ts';
import { DateFromISOString } from '../models/Date.ts';
import { MemoItem } from '../models/MemoItem.ts';
import { TaskItem } from '../models/TaskItem.ts';

const WORD_SPACER = ' ';
const LIST_BULLET = '-';

const generateBreakLine = (): void => {
  console.log(); // really?
};

const formatDateString = (item: DateFromISOString): string => {
  // return `(${format(new Date(item), 'HH:mm:ss', {})})`
  return `(${format(new Date(item), 'HH:mm', {})})`;
};

const generateTaskListHeader = (): void => {
  console.log('Task ->');
};

const generateTaskItem = (item: TaskItem): void => {
  console.log(
    [
      LIST_BULLET,
      item.closed ? '[x]' : '[ ]',
      item.content,
      formatDateString(item.createdAt),
      item.hash,
    ].join(WORD_SPACER),
  );
};

const generateMemoListHeader = (): void => {
  console.log('Memo ->');
};

const generateMemoItem = (item: MemoItem): void => {
  console.log(
    [
      LIST_BULLET,
      item.content,
      formatDateString(item.createdAt),
      item.hash,
    ].join(WORD_SPACER),
  );
};

export const generateItemList = (contents: {
  tasks: Array<TaskItem>;
  memos: Array<MemoItem>;
}) => {
  if (contents.tasks.length === 0 && contents.memos.length === 0) {
    prompt('There is no body...');
    return;
  }

  if (contents.tasks.length > 0) {
    generateTaskListHeader();
    contents.tasks.forEach(generateTaskItem);
  }

  if (contents.tasks.length > 0 && contents.memos.length > 0) {
    generateBreakLine();
  }

  if (contents.memos.length > 0) {
    generateMemoListHeader();
    contents.memos.forEach(generateMemoItem);
  }
};

export const generateFinishedTaskItem = (item: TaskItem): void => {
  console.log('Finished!!');
  console.log(
    [
      '>',
      item.content,
      formatDateString(item.createdAt),
    ].join(WORD_SPACER),
  );
}