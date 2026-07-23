import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import { ArrowLeft, Download, Plus, Save, Send, Trash2, Upload } from "lucide-react"
import { useDropzone } from "react-dropzone"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import {
  type WriteoffCatalogItem,
  type WriteoffDocument,
  type WriteoffDocumentItem,
  type WriteoffRowDraft,
  documentItemToRow,
  downloadWriteoffExport,
  downloadWriteoffTemplate,
  emptyWriteoffRow,
  isWriteoffAuxLine,
  isWriteoffProductLine,
  rowToSavePayload,
  type WriteoffImportError,
  uploadWriteoffImport,
  writeoffImportColumnLabel,
  writeoffStatusLabel,
} from "@/pages/writeoff_shared"
import { formatQty } from "@/lib/ware-quantity"
import { cn } from "@/lib/utils"

const selectClass = cn(
  "h-9 min-w-[140px] rounded-lg border border-input bg-background px-2 text-sm shadow-sm",
  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
)

export default function WriteoffIdPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const documentId = Number(id)

  const [document, setDocument] = useState<WriteoffDocument | null>(null)
  const [rows, setRows] = useState<WriteoffRowDraft[]>([emptyWriteoffRow()])
  const [lines, setLines] = useState<WriteoffCatalogItem[]>([])
  const [catalogByLine, setCatalogByLine] = useState<Record<number, WriteoffCatalogItem[]>>({})
  const [saving, setSaving] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [importing, setImporting] = useState(false)
  const [importErrorsOpen, setImportErrorsOpen] = useState(false)
  const [importErrors, setImportErrors] = useState<WriteoffImportError[]>([])
  const [importErrorMessage, setImportErrorMessage] = useState("")
  const loadedCatalogLinesRef = useRef(new Set<number>())

  const editable = document?.status === "draft" || document?.status === "rejected"

  const loadCatalog = useCallback(async (lineId: number) => {
    if (!lineId || loadedCatalogLinesRef.current.has(lineId)) {
      return
    }
    loadedCatalogLinesRef.current.add(lineId)
    const result = await Backend_Request<WriteoffCatalogItem[]>({ line_id: lineId }, "/api/writeoff/catalog/items")
    if (result.result === "ok") {
      setCatalogByLine((current) => ({ ...current, [lineId]: result.data ?? [] }))
    } else {
      loadedCatalogLinesRef.current.delete(lineId)
    }
  }, [])

  const loadDocument = useCallback(async () => {
    if (!documentId) {
      return
    }
    const result = await Backend_Request<{ document: WriteoffDocument; items: WriteoffDocumentItem[] }>(
      { document_id: documentId },
      "/api/writeoff/documents/get",
    )
    if (result.result !== "ok" || !result.data) {
      ShowErrorToast(result.error || "Hujjat topilmadi")
      navigate("/writeoff")
      return
    }
    setDocument(result.data.document)
    const nextRows = result.data.items.length
      ? result.data.items.map(documentItemToRow)
      : [emptyWriteoffRow()]
    setRows(nextRows)
    for (const row of nextRows) {
      if (isWriteoffAuxLine(row.line_id)) {
        void loadCatalog(row.line_id)
      }
    }
  }, [documentId, loadCatalog, navigate])

  useEffect(() => {
    void Backend_Request<WriteoffCatalogItem[]>({}, "/api/writeoff/catalog/lines").then((result) => {
      if (result.result === "ok") {
        setLines(result.data ?? [])
      }
    })
  }, [])

  useEffect(() => {
    loadedCatalogLinesRef.current.clear()
    setCatalogByLine({})
    void loadDocument()
  }, [documentId, loadDocument])

  const title = useMemo(() => (document ? `Hisobdan chiqarish #${document.id}` : "Hisobdan chiqarish"), [document])

  function updateRow(key: string, patch: Partial<WriteoffRowDraft>) {
    setRows((current) =>
      current.map((row) => {
        if (row.key !== key) {
          return row
        }
        const next = { ...row, ...patch }
        if (patch.line_id !== undefined && patch.line_id !== row.line_id) {
          next.model_id = 0
          next.component_id = 0
          next.serial = ""
          next.item_type = isWriteoffProductLine(patch.line_id) ? "product" : "component"
          next.quantity = isWriteoffProductLine(patch.line_id) ? 1 : next.quantity
          if (isWriteoffAuxLine(patch.line_id)) {
            void loadCatalog(patch.line_id)
          }
        }
        return next
      }),
    )
  }

  function addRow() {
    setRows((current) => [...current, emptyWriteoffRow()])
  }

  function removeRow(key: string) {
    setRows((current) => (current.length <= 1 ? [emptyWriteoffRow()] : current.filter((row) => row.key !== key)))
  }

  function buildSavePayload() {
    return rows.filter((row) => row.line_id > 0).map(rowToSavePayload)
  }

  async function saveRows() {
    const payload = buildSavePayload()
    if (payload.length === 0) {
      ShowErrorToast("Kamida bitta to'ldirilgan qator kerak")
      return false
    }
    setSaving(true)
    const result = await Backend_Request(
      { document_id: documentId, items: payload },
      "/api/writeoff/documents/items/save",
    )
    setSaving(false)
    if (result.result === "ok") {
      ShowOKToast("Saqlangan")
      await loadDocument()
      return true
    }
    ShowErrorToast(result.error || "Saqlashda xatolik")
    return false
  }

  async function submitDocument() {
    setSubmitting(true)
    const payload = buildSavePayload()
    if (payload.length === 0) {
      ShowErrorToast("Kamida bitta to'ldirilgan qator kerak")
      setSubmitting(false)
      return
    }
    const saveResult = await Backend_Request(
      { document_id: documentId, items: payload },
      "/api/writeoff/documents/items/save",
    )
    if (saveResult.result !== "ok") {
      setSubmitting(false)
      ShowErrorToast(saveResult.error || "Saqlashda xatolik")
      return
    }
    const result = await Backend_Request({ document_id: documentId }, "/api/writeoff/documents/submit")
    setSubmitting(false)
    if (result.result === "ok") {
      ShowOKToast("Tasdiqlashga yuborildi")
      await loadDocument()
    } else {
      ShowErrorToast(result.error || "Yuborishda xatolik")
    }
  }

  async function deleteDocument() {
    const result = await Backend_Request({ document_id: documentId }, "/api/writeoff/documents/delete")
    if (result.result === "ok") {
      ShowOKToast("Hujjat o'chirildi")
      navigate("/writeoff")
    } else {
      ShowErrorToast(result.error || "O'chirishda xatolik")
    }
  }

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    disabled: !editable || importing,
    accept: {
      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": [".xlsx"],
    },
    maxFiles: 1,
    onDrop: (files) => {
      const file = files[0]
      if (!file) {
        return
      }
      void (async () => {
        setImporting(true)
        const result = await uploadWriteoffImport(documentId, file)
        setImporting(false)
        if (result.result !== "ok") {
          const errors = result.data?.errors ?? []
          if (errors.length > 0) {
            setImportErrors(errors)
            setImportErrorMessage(result.error || "Import xatoliklari")
            setImportErrorsOpen(true)
          } else {
            ShowErrorToast(result.error || "Import xatolik")
          }
          return
        }
        ShowOKToast(`Import qilindi: ${result.data?.imported_rows ?? 0} qator`)
        await loadDocument()
      })()
    },
  })

  return (
    <PageContainer
      title={title}
      description={document ? writeoffStatusLabel(document.status) : ""}
      actions={
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" className="gap-2" asChild>
            <Link to="/writeoff">
              <ArrowLeft className="size-4" />
              Orqaga
            </Link>
          </Button>
          {editable ? (
            <>
              <Button variant="outline" className="gap-2" onClick={() => void downloadWriteoffTemplate()}>
                <Download className="size-4" />
                Shablon
              </Button>
              <Button variant="outline" className="gap-2" onClick={() => void downloadWriteoffExport(documentId)}>
                <Download className="size-4" />
                Export
              </Button>
              <Button variant="outline" className="gap-2 text-destructive" onClick={() => void deleteDocument()}>
                <Trash2 className="size-4" />
                O'chirish
              </Button>
              <Button className="gap-2" disabled={saving} onClick={() => void saveRows()}>
                <Save className="size-4" />
                {saving ? "Saqlanmoqda..." : "Saqlash"}
              </Button>
              <Button className="gap-2" disabled={submitting} onClick={() => void submitDocument()}>
                <Send className="size-4" />
                {submitting ? "Yuborilmoqda..." : "Tasdiqlashga yuborish"}
              </Button>
            </>
          ) : null}
        </div>
      }
    >
      {editable ? (
        <Panel title="Excel import">
          <div
            {...getRootProps()}
            className={cn(
              "flex cursor-pointer flex-col items-center justify-center rounded-xl border border-dashed p-8 text-center transition",
              isDragActive ? "border-primary bg-primary/5" : "border-border hover:bg-muted/40",
            )}
          >
            <input {...getInputProps()} />
            <Upload className="mb-2 size-8 text-muted-foreground" />
            <p className="font-medium">{importing ? "Import qilinmoqda..." : "Excel faylni bu yerga tashlang"}</p>
            <p className="text-sm text-muted-foreground">
              T1/T2/T3/I1 uchun faqat serial; Fin Press/Radiator uchun komponent tanlanadi
            </p>
          </div>
        </Panel>
      ) : null}

      {document?.status === "rejected" && document.reject_comment ? (
        <Panel title="Rad etish sababi">
          <p className="text-sm text-rose-700">{document.reject_comment}</p>
        </Panel>
      ) : null}

      <Panel title="Qatorlar">
        <div className="mb-3 flex justify-end">
          {editable ? (
            <Button variant="outline" size="sm" className="gap-2" onClick={addRow}>
              <Plus className="size-4" />
              Qator
            </Button>
          ) : null}
        </div>

        <div className="overflow-auto rounded-xl border border-border/60">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Liniya</TableHead>
                <TableHead>Komponent</TableHead>
                <TableHead>Serial</TableHead>
                <TableHead>Miqdor</TableHead>
                <TableHead>Izoh</TableHead>
                {editable ? <TableHead /> : null}
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => {
                const catalog = catalogByLine[row.line_id] ?? []
                const productLine = isWriteoffProductLine(row.line_id)
                return (
                  <TableRow key={row.key}>
                    <TableCell>
                      {editable ? (
                        <select
                          className={selectClass}
                          value={row.line_id || ""}
                          onChange={(e) => updateRow(row.key, { line_id: Number(e.target.value) })}
                        >
                          <option value="">Tanlang</option>
                          {lines.map((line) => (
                            <option key={line.line_id} value={line.line_id}>
                              {line.line_name}
                            </option>
                          ))}
                        </select>
                      ) : (
                        row.line_name ?? lines.find((l) => l.line_id === row.line_id)?.line_name ?? row.line_id
                      )}
                    </TableCell>
                    <TableCell>
                      {editable ? (
                        productLine ? (
                          <span className="text-sm text-muted-foreground">Serial orqali aniqlanadi</span>
                        ) : (
                          <select
                            className={selectClass}
                            value={row.component_id || ""}
                            disabled={!row.line_id}
                            onChange={(e) => updateRow(row.key, { component_id: Number(e.target.value) })}
                          >
                            <option value="">Tanlang</option>
                            {catalog.map((item) => (
                              <option key={item.item_id} value={item.item_id}>
                                {item.label}
                              </option>
                            ))}
                          </select>
                        )
                      ) : productLine ? (
                        row.item_label || "—"
                      ) : (
                        row.item_label ??
                        catalog.find((c) => c.item_id === row.component_id)?.label ??
                        "—"
                      )}
                    </TableCell>
                    <TableCell>
                      {editable && productLine ? (
                        <Input
                          value={row.serial}
                          onChange={(e) => updateRow(row.key, { serial: e.target.value })}
                          placeholder="Serial"
                          className="h-9 min-w-[120px]"
                        />
                      ) : productLine ? (
                        row.serial || "—"
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell>
                      {editable && !productLine ? (
                        <Input
                          type="number"
                          min={0}
                          step="any"
                          value={row.quantity}
                          onChange={(e) => updateRow(row.key, { quantity: Number(e.target.value) })}
                          className="h-9 w-24"
                        />
                      ) : (
                        formatQty(row.quantity)
                      )}
                    </TableCell>
                    <TableCell>
                      {editable ? (
                        <Input
                          value={row.comment}
                          onChange={(e) => updateRow(row.key, { comment: e.target.value })}
                          className="h-9 min-w-[140px]"
                        />
                      ) : (
                        row.comment || "—"
                      )}
                    </TableCell>
                    {editable ? (
                      <TableCell>
                        <Button variant="ghost" size="icon" onClick={() => removeRow(row.key)}>
                          <Trash2 className="size-4 text-destructive" />
                        </Button>
                      </TableCell>
                    ) : null}
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      </Panel>

      <Dialog open={importErrorsOpen} onOpenChange={setImportErrorsOpen}>
        <DialogContent className="max-h-[92svh] overflow-y-auto sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>Import xatoliklari</DialogTitle>
            <DialogDescription>
              {importErrorMessage || "Quyidagi qatorlarda xatolik topildi. Excel faylni tuzatib qayta yuklang."}
            </DialogDescription>
          </DialogHeader>
          <div className="overflow-auto rounded-xl border border-border/60">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Qator</TableHead>
                  <TableHead>Maydon</TableHead>
                  <TableHead>Xabar</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {importErrors.map((item, index) => (
                  <TableRow key={`${item.row}-${item.column}-${index}`}>
                    <TableCell className="font-medium">{item.row}</TableCell>
                    <TableCell>{writeoffImportColumnLabel(item.column)}</TableCell>
                    <TableCell>{item.message}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setImportErrorsOpen(false)}>
              Yopish
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PageContainer>
  )
}
