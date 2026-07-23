import type { LabelBinding } from "@/lib/label-bindings"
import { LABEL_BINDINGS } from "@/lib/label-bindings"

export function groupLabelBindings(bindings: LabelBinding[] = LABEL_BINDINGS): [string, LabelBinding[]][] {
  const map = new Map<string, LabelBinding[]>()
  for (const item of bindings) {
    const list = map.get(item.group) ?? []
    list.push(item)
    map.set(item.group, list)
  }
  return Array.from(map.entries())
}

/** Shtrix kod uchun backend maydonlar */
export const BARCODE_BINDINGS = LABEL_BINDINGS.filter(
  (b) => !b.key.startsWith("gscode.") && b.key !== "today",
)

/** DataMatrix uchun backend maydonlar */
export const DATAMATRIX_BINDINGS = LABEL_BINDINGS.filter((b) => b.key === "gscode.data")

/** Matn elementi uchun backend maydonlar */
export const TEXT_BINDINGS = LABEL_BINDINGS
