export const defaultRecordTimeZone = "Asia/Shanghai";

export function normalizeRecordTimeZone(timeZone?: string) {
  const value = timeZone?.trim() || defaultRecordTimeZone;
  try {
    new Intl.DateTimeFormat(undefined, { timeZone: value }).format(new Date());
    return value;
  } catch {
    return defaultRecordTimeZone;
  }
}

export function toTimeZoneDateInputValue(date: Date, timeZone?: string) {
  const formatter = new Intl.DateTimeFormat("en-CA", {
    timeZone: normalizeRecordTimeZone(timeZone),
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  });

  const parts = formatter.formatToParts(date);
  const get = (type: string) =>
    parts.find((part) => part.type === type)?.value || "";

  return `${get("year")}-${get("month")}-${get("day")}`;
}

export function formatRecordTime(value: string, timeZone?: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value || "—";
  return date.toLocaleString(undefined, {
    timeZone: normalizeRecordTimeZone(timeZone),
  });
}
