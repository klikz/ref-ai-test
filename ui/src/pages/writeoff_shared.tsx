import { Global_Data } from "@/config/config"
import { randomId } from "@/lib/random-id"
import { Backend_Request_Blob } from "@/services/backend"

export type WriteoffDocument = {
  id: number
  status: string
  created_by: number
  created_by_name: string
  submitted_at: string
  approved_by: number
  approved_by_name: string
  approved_at: string
  rejected_by: number
  rejected_by_name: string
  rejected_at: string
  reject_comment: string
  c_time: string
  item_count: number
  approval_required?: number
  approval_done?: number
}

export type WriteoffDocumentApproval = {
  id: number
  document_id: number
  user_id: number
  user_name: string
  user_login: string
  approved_at: string
  is_approved: boolean
}

export type WriteoffImportError = {
  row: number
  column: string
  message: string
}

export type WriteoffImportResult = {
  imported_rows?: number
  errors?: WriteoffImportError[]
}

export type WriteoffApproveResult = {
  all_approved: boolean
  approval_done: number
  approval_required: number
}

export type WriteoffDocumentItem = {
  id: number
  document_id: number
  line_id: number
  line_name: string
  item_type: string
  model_id: number
  component_id: number
  product_id: number
  serial: string
  item_label: string
  quantity: number
  comment: string
  sort_order: number
}

export type WriteoffCatalogItem = {
  item_id: number
  item_type: string
  label: string
  line_id: number
  line_name: string
}

export type WriteoffRecord = {
  id: number
  document_id: number
  line_id: number
  line_name: string
  item_type: string
  model_id: number
  component_id: number
  product_id: number
  serial: string
  item_label: string
  quantity: number
  comment: string
  balance_before: number
  balance_after: number
  written_off_by: number
  written_off_by_name: string
  written_off_at: string
}

export type WriteoffRowDraft = {
  key: string
  line_id: number
  line_name?: string
  item_type: string
  model_id: number
  component_id: number
  item_label?: string
  serial: string
  quantity: number
  comment: string
}

export const WRITEOFF_PRODUCT_LINE_IDS = new Set([4, 5, 6, 7])
export const WRITEOFF_AUX_LINE_IDS = new Set([8, 9])

export function isWriteoffProductLine(lineId: number) {
  return WRITEOFF_PRODUCT_LINE_IDS.has(lineId)
}

export function isWriteoffAuxLine(lineId: number) {
  return WRITEOFF_AUX_LINE_IDS.has(lineId)
}

export function writeoffStatusLabel(status: string) {
  switch (status) {
    case "draft":
      return "Qoralama"
    case "pending":
      return "Tasdiqlash kutilmoqda"
    case "approved":
      return "Tasdiqlangan"
    case "rejected":
      return "Rad etilgan"
    default:
      return status
  }
}

export function emptyWriteoffRow(): WriteoffRowDraft {
  return {
    key: randomId(),
    line_id: 0,
    item_type: "",
    model_id: 0,
    component_id: 0,
    serial: "",
    quantity: 1,
    comment: "",
  }
}

export function documentItemToRow(item: WriteoffDocumentItem): WriteoffRowDraft {
  return {
    key: String(item.id || randomId()),
    line_id: item.line_id,
    line_name: item.line_name,
    item_type: item.item_type,
    model_id: item.model_id,
    component_id: item.component_id,
    item_label: item.item_label,
    serial: item.serial,
    quantity: item.quantity,
    comment: item.comment,
  }
}

export function rowToSavePayload(row: WriteoffRowDraft) {
  const productLine = isWriteoffProductLine(row.line_id)
  return {
    line_id: row.line_id,
    item_type: productLine ? "product" : "component",
    model_id: productLine ? 0 : row.model_id,
    component_id: productLine ? 0 : row.component_id,
    serial: row.serial,
    quantity: productLine ? 1 : row.quantity,
    comment: row.comment,
  }
}

export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement("a")
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}

export async function downloadWriteoffTemplate() {
  const result = await Backend_Request_Blob({}, "/api/writeoff/documents/template")
  if (result.result !== "ok" || !result.data) {
    throw new Error(result.error || "Shablon yuklanmadi")
  }
  downloadBlob(result.data, "writeoff_template.xlsx")
}

export async function downloadWriteoffExport(documentId: number) {
  const result = await Backend_Request_Blob({ document_id: documentId }, "/api/writeoff/documents/export")
  if (result.result !== "ok" || !result.data) {
    throw new Error(result.error || "Export xatolik")
  }
  downloadBlob(result.data, `writeoff_${documentId}.xlsx`)
}

export async function uploadWriteoffImport(documentId: number, file: File) {
  const token = Global_Data.getAccessToken()
  const form = new FormData()
  form.append("document_id", String(documentId))
  form.append("file", file)
  const response = await fetch(`${Global_Data.server_ip}/api/writeoff/documents/import`, {
    method: "POST",
    headers: token ? { Authorization: token } : {},
    body: form,
  })
  const payload = await response.json().catch(() => null)
  if (!response.ok || payload?.result === "error") {
    return {
      result: "error" as const,
      error: String(payload?.error ?? "Import xatolik"),
      data: payload?.data as WriteoffImportResult | undefined,
    }
  }
  return { result: "ok" as const, data: payload?.data as WriteoffImportResult | undefined }
}

export function writeoffImportColumnLabel(column: string) {
  switch (column) {
    case "line_name":
      return "Liniya"
    case "serial":
      return "Serial"
    case "item_label":
      return "Komponent"
    case "quantity":
      return "Miqdor"
    case "item_type":
      return "Tur"
    default:
      return column || "Umumiy"
  }
}
