export const requiredDateFormatHash = (hash: string): boolean => {
  if (/(\d{4})-(\d{1,2})-(\d{1,2})/.test(hash) === false) return true;

  return Number.isNaN(new Date(hash).getDate())
}
