export const WARE_QTY_DECIMALS = 8

export function normalizedQty(value: number) {
  if (!Number.isFinite(value)) {
    return 0
  }
  const scale = 10 ** WARE_QTY_DECIMALS
  return Math.round(value * scale) / scale
}

export function formatQty(value: number) {
  if (!Number.isFinite(value)) {
    return ""
  }
  const normalized = normalizedQty(value)
  if (normalized === 0) {
    return "0"
  }
  return parseFloat(normalized.toFixed(WARE_QTY_DECIMALS)).toString()
}

export function parseQty(value: string) {
  const parsed = Number(value.replace(",", "."))
  return Number.isFinite(parsed) ? normalizedQty(parsed) : 0
}
