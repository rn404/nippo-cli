import { LOG_DIR } from '../const.ts';
import { getLogFile, updateLogFile } from '../features/logFile.ts';
import { deleteItem } from '../features/log.ts';

export const deleteCommand = async (
  hash: string,
): Promise<void> => {
  const { fileName, body: log } = await getLogFile(LOG_DIR);
  const newLog = deleteItem(log, hash);

  await updateLogFile(
    LOG_DIR,
    fileName,
    newLog,
  );
};
