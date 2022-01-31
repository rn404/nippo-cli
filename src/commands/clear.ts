import { LOG_DIR } from '../const.ts';
import { listLogFile } from '../features/logFile.ts'

const STORAGE_PERIOD_DAY = -30

const clearAllFiles = async (logDir: string): Promise<void> => {
  const isDeleteAllLogFiles = confirm('Do you want to delete all the files?');

  if (isDeleteAllLogFiles === false) return;

  const logFiles = await listLogFile(logDir)
  if (logFiles.length === 0) {
    console.log('There is no log files.')
    return;
  }

  logFiles.forEach(async (log) => {
    console.log(`Deleted... ${log.fileName} logs.`);
    await Deno.remove(log.path);
  });

  console.log('Deleted all files.');
}

const clearOldFiles = async (logDir: string): Promise<void> => {
  console.log(`
    Delete logs that are past their storage period. ( Storage period: 30 days )
  `);

  const periodDate = new Date(new Date().setDate(STORAGE_PERIOD_DAY))
  const logFiles = await listLogFile(logDir)
  logFiles.forEach(async (log) => {
    if (
      new Date(log.fileName).valueOf() < periodDate.valueOf()
    ) {
      console.log(`Deleted... ${log.fileName} logs.`);
      await Deno.remove(log.path);
    }
  });
}

export const clearCommand = async (
  options: { all?: boolean }
): Promise<void> => {
  if (options.all === true) {
    await clearAllFiles(LOG_DIR);
    return;
  }
  await clearOldFiles(LOG_DIR);
};
