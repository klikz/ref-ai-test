import { useCallback, useEffect, useMemo, useState } from "react"
import { Eye, Loader2, Printer } from "lucide-react"
import { Link } from "react-router-dom"

import { Label } from "@/components/ui/label"
import { Backend_Request } from "@/services/backend"
import type { LabelDefinition, LabelPrintRotationDeg } from "@/lib/label-types"

type PrinterV2Item = {
  id: number
  line_id: number
  printer_name: string
  label_template_id: number
  label_template_name: string
}

type PreviewResponse = {
  preview_png: string
  width_px: number
  height_px: number
  printer_name: string
  dpi: number
  width_mm: number
  height_mm: number
}

type LabelPrinterPreviewProps = {
  templateId: number
  widthMm: number
  heightMm: number
  dpi: number
  printRotationDeg?: LabelPrintRotationDeg
  definition: LabelDefinition
  previewSerial?: string
  previewAccSerial?: string
}

export function LabelPrinterPreview({
  templateId,
  widthMm,
  heightMm,
  dpi,
  printRotationDeg = 0,
  definition,
  previewSerial = "",
  previewAccSerial = "",
}: LabelPrinterPreviewProps) {
  const [printers, setPrinters] = useState<PrinterV2Item[]>([])
  const [selectedPrinterId, setSelectedPrinterId] = useState("")
  const [preview, setPreview] = useState<PreviewResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const linkedPrinters = useMemo(
    () => printers.filter((p) => p.label_template_id === templateId),
    [printers, templateId],
  )

  const loadPrinters = useCallback(async () => {
    const result = await Backend_Request<PrinterV2Item[]>({}, "/api/tech/printers-v2/all")
    if (result.result === "ok" && result.data) {
      setPrinters(result.data)
    }
  }, [])

  const fetchPreview = useCallback(async () => {
    const printerId = Number(selectedPrinterId)
    if (!printerId) {
      setPreview(null)
      setError(null)
      return
    }

    setLoading(true)
    setError(null)
    const result = await Backend_Request<PreviewResponse>(
      {
        id: templateId,
        printer_v2_id: printerId,
        width_mm: widthMm,
        height_mm: heightMm,
        dpi,
        print_rotation_deg: printRotationDeg,
        definition,
        serial: previewSerial,
        acc_serial: previewAccSerial,
      },
      "/api/tech/label-templates/preview",
    )
    setLoading(false)

    if (result.result === "ok" && result.data) {
      setPreview(result.data)
    } else {
      setPreview(null)
      setError(result.error || "Ko'rish yuklanmadi")
    }
  }, [templateId, selectedPrinterId, widthMm, heightMm, dpi, printRotationDeg, definition, previewSerial, previewAccSerial])

  useEffect(() => {
    void loadPrinters()
  }, [loadPrinters])

  useEffect(() => {
    if (linkedPrinters.length === 0) {
      setSelectedPrinterId("")
      return
    }
    if (!linkedPrinters.some((p) => String(p.id) === selectedPrinterId)) {
      setSelectedPrinterId(String(linkedPrinters[0].id))
    }
  }, [linkedPrinters, selectedPrinterId])

  useEffect(() => {
    if (!selectedPrinterId) {
      return
    }
    const timer = window.setTimeout(() => {
      void fetchPreview()
    }, 400)
    return () => window.clearTimeout(timer)
  }, [fetchPreview, selectedPrinterId])

  const selectedPrinter = linkedPrinters.find((p) => String(p.id) === selectedPrinterId)

  return (
    <div className="space-y-3">
      {linkedPrinters.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          Ushbu shablon uchun V2 printer yo&apos;q.{" "}
          <Link to="/printers-v2" className="text-primary underline">
            Printerlar V2
          </Link>{" "}
          sahifasida shablonni bog&apos;lang.
        </p>
      ) : (
        <>
          <div className="space-y-1">
            <Label className="text-xs">Printer</Label>
            <select
              className="flex h-9 w-full rounded-md border border-input bg-background px-2 text-sm"
              value={selectedPrinterId}
              onChange={(e) => setSelectedPrinterId(e.target.value)}
            >
              {linkedPrinters.map((printer) => (
                <option key={printer.id} value={printer.id}>
                  {printer.printer_name}
                </option>
              ))}
            </select>
          </div>

          {selectedPrinter ? (
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Printer className="size-3.5 shrink-0" />
              <span>
                {selectedPrinter.printer_name} · {dpi} DPI · {widthMm}×{heightMm} mm
              </span>
            </div>
          ) : null}

          <div className="relative min-h-[120px] rounded-xl border border-dashed border-border bg-muted/30 p-3">
            {loading ? (
              <div className="flex min-h-[100px] items-center justify-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="size-4 animate-spin" />
                Ko&apos;rish tayyorlanmoqda...
              </div>
            ) : error ? (
              <p className="text-sm text-destructive">{error}</p>
            ) : preview?.preview_png ? (
              <div className="flex flex-col items-center gap-2">
                <img
                  src={preview.preview_png}
                  alt="Etiketka ko'rinishi"
                  className="max-h-[280px] max-w-full rounded-md border bg-white shadow-sm"
                  style={{
                    width: Math.min(preview.width_px, 360),
                    height: "auto",
                  }}
                />
                <p className="flex items-center gap-1 text-[10px] text-muted-foreground">
                  <Eye className="size-3" />
                  Chop etishdagi ko&apos;rinish ({preview.width_px}×{preview.height_px} px)
                </p>
              </div>
            ) : (
              <p className="text-center text-sm text-muted-foreground">Printer tanlang</p>
            )}
          </div>
        </>
      )}
    </div>
  )
}
