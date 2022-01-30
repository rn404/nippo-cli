import { LOG_DIR } from '../const.ts';
import { getLogFile } from '../features/logFile.ts';
import { listItems } from '../features/log.ts';
import { generateItemList } from '../features/generate.ts';

export const listCommand = async (
  options: { all?: boolean; stat?: boolean },
): Promise<void> => {
  if (options.all !== true && options.stat !== true) {
    const { body: log } = await getLogFile(LOG_DIR);
    generateItemList(listItems(log));
    return;
  }
  console.log('TBD');
};
