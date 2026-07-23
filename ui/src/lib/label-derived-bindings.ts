import { formatLabelDate, isTodayBinding, TODAY_BINDING } from "@/lib/label-date-format"
import { generateSerial } from "@/lib/generate-serial"

export const GSCODE_DATA38_BINDING = "gscode.data38"
export const GSCODE_DATA38_LENGTH = 38
export const RADIATOR_SERIAL_BINDING = "radiator.serial"
export const KLAPAN_SERIAL_BINDING = "klapan.serial"
export const ESHIK_SERIAL_BINDING = "eshik.serial"

export function isRadiatorSerialBinding(binding?: string): boolean {
  return binding === RADIATOR_SERIAL_BINDING
}

export function isKlapanSerialBinding(binding?: string): boolean {
  return binding === KLAPAN_SERIAL_BINDING
}

export function isEshikSerialBinding(binding?: string): boolean {
  return binding === ESHIK_SERIAL_BINDING
}

export function isGS1Data38Binding(binding?: string): boolean {
  return binding === GSCODE_DATA38_BINDING
}

export function isDerivedBinding(binding?: string): boolean {
  return (
    isTodayBinding(binding) ||
    isGS1Data38Binding(binding) ||
    isRadiatorSerialBinding(binding) ||
    isKlapanSerialBinding(binding) ||
    isEshikSerialBinding(binding)
  )
}

export function truncateGS1DataPrefix(value: string, length = GSCODE_DATA38_LENGTH): string {
  return value.slice(0, length)
}

export function resolveGS1Data38FromData(data: Record<string, unknown>): string {
  const gscode = data.gscode
  if (gscode == null || typeof gscode !== "object") {
    return ""
  }
  const raw = (gscode as Record<string, unknown>).data
  if (raw == null) {
    return ""
  }
  return truncateGS1DataPrefix(String(raw))
}

export function resolveDerivedBinding(
  data: Record<string, unknown>,
  binding?: string,
  dateFormat?: string,
): string {
  if (!binding) {
    return ""
  }
  if (isTodayBinding(binding)) {
    return formatLabelDate(new Date(), dateFormat)
  }
  if (isGS1Data38Binding(binding)) {
    return resolveGS1Data38FromData(data)
  }
  if (isRadiatorSerialBinding(binding)) {
    const existing = typeof data.serial === "string" ? data.serial.trim() : ""
    if (existing) {
      return existing
    }
    const rawCounter = data.radiator_counter
    const counter =
      typeof rawCounter === "number" && Number.isFinite(rawCounter) ? Math.trunc(rawCounter) : 1
    const seriyaRaqami = typeof data.seriya_raqami === "string" ? data.seriya_raqami : ""
    return generateSerial(seriyaRaqami, counter)
  }
  if (isKlapanSerialBinding(binding)) {
    const existing = typeof data.serial === "string" ? data.serial.trim() : ""
    if (existing) {
      return existing
    }
    const rawCounter = data.klapan_counter
    const counter =
      typeof rawCounter === "number" && Number.isFinite(rawCounter) ? Math.trunc(rawCounter) : 1
    const seriyaRaqami = typeof data.seriya_raqami === "string" ? data.seriya_raqami : ""
    return generateSerial(seriyaRaqami, counter)
  }
  if (isEshikSerialBinding(binding)) {
    const existing = typeof data.serial === "string" ? data.serial.trim() : ""
    if (existing) {
      return existing
    }
    const rawCounter = data.eshik_counter
    const counter =
      typeof rawCounter === "number" && Number.isFinite(rawCounter) ? Math.trunc(rawCounter) : 1
    const seriyaRaqami = typeof data.seriya_raqami === "string" ? data.seriya_raqami : ""
    return generateSerial(seriyaRaqami, counter)
  }
  return ""
}

export { TODAY_BINDING }
