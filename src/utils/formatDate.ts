const formatDate = (date: Date): string => {
  return new Intl.DateTimeFormat('en-CA').format(date);
};

export { formatDate };
