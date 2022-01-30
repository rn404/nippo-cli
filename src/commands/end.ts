import { LOG_DIR } from '../const.ts';
import { getLogFile, updateLogFile } from '../features/logFile.ts';
import { finishTaskItem } from '../features/log.ts';
import { generateFinishedTaskItem } from '../features/generate.ts'

export const endCommand = async (
  hash: string,
): Promise<void> => {
  const { fileName, body: log } = await getLogFile(LOG_DIR);
  const { log: newLog, finished } = finishTaskItem(log, hash);

  generateFinishedTaskItem(finished);

  await updateLogFile(
    LOG_DIR,
    fileName,
    newLog,
  );
};
