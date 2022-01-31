import { LOG_DIR } from '../const.ts';
import { getLogFile } from '../features/logFile.ts';
import { listItems } from '../features/log.ts';
import { generateItemList, generateLogFileStat, generateHeader } from '../features/generate.ts';

export const listCommand = async (
  options: { all?: boolean; stat?: boolean },
): Promise<void> => {
  if (options.all !== true) {
    const { fileName, body: log } = await getLogFile(LOG_DIR);

    if (options.stat === true) {
      const { tasks, memos } = listItems(log);
      const unfinishedTaskCount = tasks.filter((item) => item.closed === false).length;

      generateHeader(`Today's log stats are...`);

      generateLogFileStat({
        fileName,
        isFreezed: log.freezed,
        items: {
          tasks,
          memos
        },
        unfinishedTaskCount
      })
      return;
    }

    generateHeader(`Today's logs are...`);
    generateItemList(listItems(log));
    return;
  }

  console.log('TBD');
};
