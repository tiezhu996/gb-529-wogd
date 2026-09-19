export const number = new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 2 })
export const compactNumber = new Intl.NumberFormat('zh-CN', { notation: 'compact', maximumFractionDigits: 2 })

export function kg(value: number): string {
  return number.format(value) + ' kg'
}

export function cubicMeters(value: number): string {
  return number.format(value) + ' m³'
}

export function dateTime(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false
  }).format(new Date(value))
}

export function localInputDate(value: Date): string {
  const offset = value.getTimezoneOffset() * 60_000
  return new Date(value.getTime() - offset).toISOString().slice(0, 16)
}
