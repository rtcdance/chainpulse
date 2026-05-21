function downloadFile(content: string, filename: string, mimeType: string): void {
  const blob = new Blob([content], { type: mimeType })
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
}

export function exportToCSV(rows: Record<string, unknown>[], columns: Array<{ key: string; label: string }>, filename: string): void {
  const header = columns.map((c) => `"${c.label}"`).join(',')
  const body = rows.map((row) =>
    columns.map((c) => {
      const value = String(row[c.key] ?? '')
      return `"${value.replace(/"/g, '""')}"`
    }).join(',')
  ).join('\n')
  downloadFile(`${header}\n${body}`, filename, 'text/csv;charset=utf-8')
}

export function exportToJSON(data: unknown, filename: string): void {
  downloadFile(JSON.stringify(data, null, 2), filename, 'application/json')
}