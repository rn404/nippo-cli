import { LOG_FILE_EXT } from '../const.ts';
import { DateString } from './Date.ts';

export class LogFileName {
  public name: DateString;
  public withExtension: `${DateString}.${typeof LOG_FILE_EXT}`;

  constructor(date: DateString) {
    this.name = date;
    this.withExtension = `${date}.${LOG_FILE_EXT}`;
  }
}
