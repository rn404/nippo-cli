// date of iso format
export type DateFromISOString = string;

type ItemHash = string;
type ItemIndex = number;
type DateYear = string; /* yyyy */
type DateMonth = string; /* MM, zero pad */
type DateDay = string; /* dd, zero pad */

export type HashDateString =
  | `${ItemHash}{${ItemIndex}}`
  | `${DateYear}-${DateMonth}-${DateDay}{${ItemIndex}}`;

/**
 * Format is `yyyy-MM-dd`.
 * This is because it is easy to convert when using new Date().
 */
export type DateString = `${DateYear}-${DateMonth}-${DateDay}`;
