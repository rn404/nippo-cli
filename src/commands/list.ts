import { FILE_STATS_LIMIT_DAYS } from '../const.ts';
import { getLogFile, listLogFile, statLogFile } from '../features/logFile.ts';
import { listItems } from '../features/log.ts';
import {
  generateHeader,
  generateItemList,
  generateListItem,
  generateLogFileStat,
} from '../features/generate.ts';
import { requiredDateFormatHash } from '../features/hash.ts';
import { logDir } from '../features/path.ts';
import { DateString } from '../models/Date.ts';
import { Log, LogFile } from '../models/LogFile.ts';

const viewLogFileStat = (fileName: DateString, log: Log): void => {
  const { tasks, memos } = listItems(log);
  const unfinishedTaskCount =
    tasks.filter((item) => item.closed === false).length;

  generateLogFileStat({
    fileName,
    isFreezed: log.freezed,
    items: {
      tasks,
      memos,
    },
    unfinishedTaskCount,
  });
};

export const listCommand = async (
  options: { all?: boolean; stat?: boolean },
  hash?: string,
): Promise<void> => {
  const dir = logDir();

  if (options.all !== true) {
    if (hash !== undefined && requiredDateFormatHash(hash) === true) {
      throw new Error('Invalid hash string.');
    }

    const logFile = await statLogFile(
      dir,
      hash as DateString,
    );

    if (logFile === undefined) {
      // TODO(@rn404) handle empty log case
      generateHeader(`Today's logs are...`);
      console.log('There is no body...');
      return;
    }
    const { fileName, body: log } = logFile;

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

  const listLogFileNames = await listLogFile(dir);

  if (options.stat === true) {
    generateHeader(
      `View all log statistics. There are ${listLogFileNames.length} total.`,
    );

    if (listLogFileNames.length > FILE_STATS_LIMIT_DAYS) {
      const acceptedAllReading = confirm(`
        The log was found to be more than ${FILE_STATS_LIMIT_DAYS} days old.
        It may take some time to display all of them.
        Are you sure you want to view them?
      `);

      if (acceptedAllReading === false) return;
    }

    const logFiles: Array<LogFile> = await Promise.all(
      listLogFileNames.map(async (item) => {
        return await getLogFile(dir, item.fileName);
      }),
    );
    logFiles.forEach((logFile) =>
      viewLogFileStat(logFile.fileName, logFile.body)
    );
    return;
  }

  generateHeader(`The logs here are...`);

  listLogFileNames.forEach((logFile) => {
    generateListItem(logFile.fileName);
  });
};
