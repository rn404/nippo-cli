import { DateFromISOString } from '../models/Date.ts';

export const requiredDateFormatHash = (hash: string): boolean => {
  if (/(\d{4})-(\d{1,2})-(\d{1,2})/.test(hash) === false) return true;

  return Number.isNaN(new Date(hash).getDate());
};

// Sort by date in descending order
export const compareDatesInDescent = (
  first: DateFromISOString,
  second: DateFromISOString,
) => {
  return new Date(first).valueOf() - new Date(second).valueOf();
};
