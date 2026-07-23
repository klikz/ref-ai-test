export const TODAY_BINDING = "today"

export const LABEL_DATE_FORMATS = [
  { id: "DD.MM.YYYY", label: "16.06.2026 (DD.MM.YYYY)" },
  { id: "DD.MM.YY", label: "16.06.26 (DD.MM.YY)" },
  { id: "DD/MM/YYYY", label: "16/06/2026 (DD/MM/YYYY)" },
  { id: "YYYY-MM-DD", label: "2026-06-16 (YYYY-MM-DD)" },
] as const

export type LabelDateFormat = (typeof LABEL_DATE_FORMATS)[number]["id"]

export const DEFAULT_LABEL_DATE_FORMAT: LabelDateFormat = "DD.MM.YYYY"

export function formatLabelDate(date: Date, format: string = DEFAULT_LABEL_DATE_FORMAT): string {
  const dd = String(date.getDate()).padStart(2, "0")
  const mm = String(date.getMonth() + 1).padStart(2, "0")
  const yyyy = String(date.getFullYear())
  const yy = yyyy.slice(-2)

  return format.replaceAll("YYYY", yyyy).replaceAll("YY", yy).replaceAll("DD", dd).replaceAll("MM", mm)
}

export function isTodayBinding(binding?: string): boolean {
  return binding === TODAY_BINDING
}
