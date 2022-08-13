import { getLogFile, updateLogFile } from '../features/logFile.ts';
import { finishTaskItem } from '../features/log.ts';
import { generateFinishedTaskItem } from '../features/generate.ts';
import { logDir } from '../features/path.ts';

export const endCommand = async (
  hash: string,
): Promise<void> => {
  const dir = logDir();
  const { fileName, body: log } = await getLogFile(dir);
  const { log: newLog, finished } = finishTaskItem(log, hash);

  generateFinishedTaskItem(finished);

  await updateLogFile(
    dir,
    fileName,
    newLog,
  );
};
