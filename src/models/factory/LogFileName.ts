import { formatDate } from '../../dependencies.ts';
import { DateString } from '../Date.ts';
import { LogFileName } from '../LogFileName.ts';

export class LogFileNameFactory {
  #date: DateConstructor;

  constructor(dateConstructor: DateConstructor) {
    this.#date = dateConstructor;
  }

  private formatToDateString(date: Date): DateString {
    return formatDate(date) as DateString;
  }

  public create(date?: Date): LogFileName;
  public create(dateString?: DateString): LogFileName;
  public create(target?: DateString | Date): LogFileName {
    if (target === undefined) {
      const date = new this.#date();
      return new LogFileName(this.formatToDateString(date));
    }

    if (target instanceof Date) {
      return new LogFileName(this.formatToDateString(target));
    }

    const date = new this.#date(target);
    return new LogFileName(this.formatToDateString(date));
  }
}
