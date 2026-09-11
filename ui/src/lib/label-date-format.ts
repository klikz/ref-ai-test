export const TODAY_BINDING = "today"
export const NOW_BINDING = "now"
export const NOW_DATE_FORMAT = "YYYY-MM-DD HH24:MI"

export const LABEL_DATE_FORMATS = [
  { id: "DD.MM.YYYY", label: "16.06.2026 (DD.MM.YYYY)" },
  { id: "DD.MM.YY", label: "16.06.26 (DD.MM.YY)" },
  { id: "DD/MM/YYYY", label: "16/06/2026 (DD/MM/YYYY)" },
  { id: "YYYY-MM-DD", label: "2026-06-16 (YYYY-MM-DD)" },
  { id: "YYYY-MM-DD HH24:MI", label: "2026-06-16 14:30 (YYYY-MM-DD HH24:MI)" },
] as const

export type LabelDateFormat = (typeof LABEL_DATE_FORMATS)[number]["id"]

export const DEFAULT_LABEL_DATE_FORMAT: LabelDateFormat = "DD.MM.YYYY"

export function formatLabelDate(date: Date, format: string = DEFAULT_LABEL_DATE_FORMAT): string {
  const dd = String(date.getDate()).padStart(2, "0")
  const mm = String(date.getMonth() + 1).padStart(2, "0")
  const yyyy = String(date.getFullYear())
  const yy = yyyy.slice(-2)
  const hh24 = String(date.getHours()).padStart(2, "0")
  const mi = String(date.getMinutes()).padStart(2, "0")

  return format
    .replaceAll("YYYY", yyyy)
    .replaceAll("YY", yy)
    .replaceAll("DD", dd)
    .replaceAll("MM", mm)
    .replaceAll("HH24", hh24)
    .replaceAll("MI", mi)
}

export function isTodayBinding(binding?: string): boolean {
  return binding === TODAY_BINDING
}

export function isNowBinding(binding?: string): boolean {
  return binding === NOW_BINDING
}

export function isDateTimeBinding(binding?: string): boolean {
  return isTodayBinding(binding) || isNowBinding(binding)
}
