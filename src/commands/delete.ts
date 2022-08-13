import { getLogFile, updateLogFile } from '../features/logFile.ts';
import { deleteItem } from '../features/log.ts';
import { logDir } from '../features/path.ts';

export const deleteCommand = async (
  hash: string,
): Promise<void> => {
  const dir = logDir()
  const { fileName, body: log } = await getLogFile(dir);
  const newLog = deleteItem(log, hash);

  await updateLogFile(
    dir,
    fileName,
    newLog,
  );
};
