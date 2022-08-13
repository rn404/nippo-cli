import { getLogFile, updateLogFile } from '../features/logFile.ts';
import { addItem } from '../features/log.ts';
import { logDir } from '../features/path.ts';

export const addCommand = async (
  options: { memo?: boolean },
  content: string,
): Promise<void> => {
  const dir = logDir();
  const { fileName, body: log } = await getLogFile(dir);
  const newLog = addItem(log, content, options.memo !== true);

  await updateLogFile(
    dir,
    fileName,
    newLog,
  );
};
