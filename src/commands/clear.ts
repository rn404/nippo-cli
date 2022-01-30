import { LOG_DIR } from '../const.ts';
import { listLogFile } from '../features/logFile.ts'

export const clearCommand = async (
  options: { all?: boolean }
): Promise<void> => {
  const isDeleteAllLogFiles = confirm('Do you want to delete all the files?');

  if (isDeleteAllLogFiles === false) return;

  const logFiles = await listLogFile(LOG_DIR)
  if (logFiles.length === 0) {
    console.log('There is no log files.')
    return;
  }

  logFiles.forEach(async (file) => {
    console.log(`Deleted... ${file.fileName} logs.`);
    await Deno.remove(file.path);
  });

  console.log('Deleted all files.');
};
