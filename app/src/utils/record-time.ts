export function toBrowserDateInputValue(date: Date) {
  const year = date.getFullYear();
  const month = `${date.getMonth() + 1}`.padStart(2, "0");
  const day = `${date.getDate()}`.padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function formatBrowserRecordTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value || "—";
  return date.toLocaleString();
}

function formatBrowserOffset(date: Date) {
  const offset = -date.getTimezoneOffset();
  const sign = offset >= 0 ? "+" : "-";
  const abs = Math.abs(offset);
  const hours = `${Math.floor(abs / 60)}`.padStart(2, "0");
  const minutes = `${abs % 60}`.padStart(2, "0");
  return `${sign}${hours}:${minutes}`;
}

function formatBrowserDateTimeWithOffset(date: Date) {
  const year = date.getFullYear();
  const month = `${date.getMonth() + 1}`.padStart(2, "0");
  const day = `${date.getDate()}`.padStart(2, "0");
  const hour = `${date.getHours()}`.padStart(2, "0");
  const minute = `${date.getMinutes()}`.padStart(2, "0");
  const second = `${date.getSeconds()}`.padStart(2, "0");
  return `${year}-${month}-${day}T${hour}:${minute}:${second}${formatBrowserOffset(date)}`;
}

export function toBrowserRecordBoundary(
  value: string,
  boundary: "start" | "end",
) {
  if (!value) return undefined;

  const [year, month, day] = value.split("-").map(Number);
  if (!year || !month || !day) return undefined;

  const date =
    boundary === "start"
      ? new Date(year, month - 1, day, 0, 0, 0)
      : new Date(year, month - 1, day, 23, 59, 59);

  return formatBrowserDateTimeWithOffset(date);
}
