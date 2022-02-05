import { LOG_DIR } from '../const.ts';
import { getLogFile, listLogFile } from '../features/logFile.ts';
import { listItems } from '../features/log.ts';
import {
  generateHeader,
  generateItemList,
  generateListItem,
  generateLogFileStat,
} from '../features/generate.ts';
import { requiredDateFormatHash } from '../features/hash.ts';
import { DateString } from '../models/Date.ts';
import { LogFile, Log } from '../models/LogFile.ts'

const viewLogFileStat = (fileName: DateString, log: Log): void => {
  const { tasks, memos } = listItems(log);
  const unfinishedTaskCount = tasks.filter((item) =>
    item.closed === false
  ).length;

  generateLogFileStat({
    fileName,
    isFreezed: log.freezed,
    items: {
      tasks,
      memos,
    },
    unfinishedTaskCount,
  });
}

export const listCommand = async (
  options: { all?: boolean; stat?: boolean },
  hash?: string,
): Promise<void> => {
  if (options.all !== true) {
    if (hash !== undefined && requiredDateFormatHash(hash) === true) {
      throw new Error('Invalid hash string.');
    }

    const { fileName, body: log } = await getLogFile(
      LOG_DIR,
      hash as DateString,
    );

    if (options.stat === true) {
      if (hash === undefined || hash === '') {
        generateHeader(`Today's log stats are...`);
      } else {
        generateHeader(`Log stats for ${hash} are...`);
      }

      viewLogFileStat(fileName, log);
      return;
    }

    if (hash === undefined || hash === '') {
      generateHeader(`Today's logs are...`);
    } else {
      generateHeader(`Log for ${hash} are...`);
    }
    generateItemList(listItems(log));
    return;
  }

  const listLogFileNames = await listLogFile(LOG_DIR);

  if (options.stat === true) {
    const FILE_STATS_LIMIT_DAYS = 10

    generateHeader(`View all log statistics. There are ${listLogFileNames.length} total.`);

    if (listLogFileNames.length > FILE_STATS_LIMIT_DAYS) {
      const acceptedAllReading = confirm(`
        The log was found to be more than ${FILE_STATS_LIMIT_DAYS} days old.
        It may take some time to display all of them.
        Are you sure you want to view them?
      `);

      if (acceptedAllReading === false) return;
    }

    const logFiles: Array<LogFile> = await Promise.all(
      listLogFileNames.map(async(item) => {
        return getLogFile(LOG_DIR, item.fileName)
      })
    )
    logFiles.forEach((logFile) => viewLogFileStat(logFile.fileName, logFile.body))
    return;
  }

  generateHeader(`The logs here are...`);

  listLogFileNames.forEach((logFile) => {
    generateListItem(logFile.fileName);
  });
};
