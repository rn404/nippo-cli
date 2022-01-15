import {
  parse,
  format
} from './dependencies.ts'

import { addItem, getCurrentFile, parse as parseLog, update } from './log.ts';
import { MemoItem } from './models/MemoItem.ts';
import { LOG_DIR } from './const.ts'

const listCommand = async (): Promise<void> => {
  const { createdAt, data } = await getCurrentFile(LOG_DIR);
  const { tasks, memos } = parseLog(data);

  const isNoData = tasks.length === 0 && memos.length === 0;
  const hasTask = tasks.length !== 0;
  const hasMemo = memos.length !== 0;
  const spacer = ' ';

  if (isNoData) {
    prompt('There is no data...')
    Deno.exit(0);
  }

  const title = format(new Date(createdAt), 'yyyy/MM/dd cccc', {});
  console.log(title);

  // TODO format どうしよう?
  if (hasTask) {
    console.log('Task ->');
    tasks.forEach((task) => {
      console.log(
        [
          '-',
          task.closed ? '[x]' : '[ ]',
          `[${format(new Date(task.createdAt), 'HH:mm:ss', {})}]`,
          task.content,
        ].join(spacer),
      );
    });
  }

  if (hasMemo) {
    console.log('Memo ->');
    memos.forEach((memo) => {
      console.log(
        [
          '-',
          `[${format(new Date(memo.createdAt), 'HH:mm:ss', {})}]`,
          memo.content,
        ].join(spacer),
      );
    });
  }
  Deno.exit(0);
}

const addMemoCommand = async (content: string): Promise<void> => {
  const newItem: MemoItem = new MemoItem(content);
  const { path, fileName, data } = await getCurrentFile(LOG_DIR);
  const newLog = await addItem(newItem, data);
  await update(newLog, path, fileName);
}

const { _: args } = parse(Deno.args);
const [topCommands, subCommands] = args;

console.log('DEBUG: ', parse(Deno.args));
console.log('DEBUG: ', ...args);

// NOTE そのうちいらなくなるかも
if (topCommands !== 'todo') {
  Deno.exit(0);
}

if (subCommands === 'list') {
  listCommand()
}

if (subCommands !== undefined) {
  const [, content] = args;
  addMemoCommand(content.toString())
}
