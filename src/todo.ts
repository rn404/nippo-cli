import { parse } from 'https://deno.land/std@0.100.0/flags/mod.ts';
import format from 'https://deno.land/x/date_fns@v2.22.1/format/index.js';
import { addItem, getCurrentFile, parse as parseLog, update } from './log.ts';
import { MemoItem } from './models/MemoItem.ts';
import { LOG_DIR } from './const.ts'

const { _: args } = parse(Deno.args);
const [topCommands, subCommands] = args;

console.log('DEBUG: ', parse(Deno.args));
console.log('DEBUG: ', ...args);

// NOTE そのうちいらなくなるかも
if (topCommands !== 'todo') {
  Deno.exit(0);
}

if (subCommands === 'list') {
  const { createdAt, data } = await getCurrentFile(LOG_DIR);
  const { tasks, memos } = parseLog(data);

  const isNoData = tasks.length === 0 && memos.length === 0;
  const hasTask = tasks.length !== 0;
  const hasMemo = memos.length !== 0;
  const spacer = ' ';

  if (isNoData) {
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

if (subCommands !== undefined) {
  const [, content] = args;
  const newItem: MemoItem = new MemoItem(content.toString());

  const { path, fileName, data } = await getCurrentFile(LOG_DIR);
  const newLog = await addItem(newItem, data);
  await update(newLog, path, fileName);
}
