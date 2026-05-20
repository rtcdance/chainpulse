export function formatTimestamp(value: number | null): string {
  if (!value) {
    return '-'
  }

  const millis = value > 1_000_000_000_000 ? value : value * 1000
  const date = new Date(millis)
  if (Number.isNaN(date.getTime())) {
    return String(value)
  }
  return date.toLocaleString()
}