import { LOG_DIR } from '../const.ts';
import { getLogFile } from '../features/logFile.ts';
import { listItems } from '../features/log.ts';
import { generateItemList, generateLogFileStat, generateHeader } from '../features/generate.ts';
import { requiredDateFormatHash } from '../features/hash.ts'
import { DateString } from '../models/Date.ts'

export const listCommand = async (
  options: { all?: boolean; stat?: boolean },
  hash?: string
): Promise<void> => {
  if (options.all !== true) {
    if (hash !== undefined && requiredDateFormatHash(hash) === true) {
      throw new Error('Invalid hash string.')
    }

    const { fileName, body: log } = await getLogFile(LOG_DIR, hash as DateString);

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
