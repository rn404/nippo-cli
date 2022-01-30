import { format } from '../dependencies.ts';
import { LOG_FILE_EXT } from '../const.ts';
import { DateString } from './Date.ts';

export class LogFileName {
  public name: DateString;
  public withExtension: `${DateString}.${typeof LOG_FILE_EXT}`;
  constructor(targetDay: Date) {
    const fileName = format(targetDay, 'yyyy-MM-dd', {}) as DateString;
    this.name = fileName;
    this.withExtension = `${fileName}.${LOG_FILE_EXT}`;
  }
}
