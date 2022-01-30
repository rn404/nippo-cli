import { LOG_DIR } from '../const.ts';
import { getLogFile, updateLogFile } from '../features/logFile.ts';
import { addItem } from '../features/log.ts';

export const addCommand = async (
  options: { memo?: boolean },
  content: string,
): Promise<void> => {
  const { fileName, body: log } = await getLogFile(LOG_DIR);
  const newLog = addItem(log, content, options.memo !== true);

  await updateLogFile(
    LOG_DIR,
    fileName,
    newLog,
  );
};
