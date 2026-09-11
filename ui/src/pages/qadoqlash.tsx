import { useCallback, useEffect, useRef, useState } from "react"
import { CameraOff, Loader2, PackageCheck, Printer } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { Global_Data } from "@/config/config"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ShowErrorToast, ShowOKToast, ShowWarningToast } from "@/components/showToast"
import { cn } from "@/lib/utils"
import {
  LAST_RECORDS_PANEL_CLASS,
  LAST_RECORDS_VISIBLE_ROWS,
} from "@/lib/last-records"

const QADOQLASH_LINE_ID = 12

type PrinterV2 = {
  id: number
  line_id: number
  printer_name: string
  address?: string
  label_template_id?: number
  label_template_name?: string
}

type ScanPhoto = {
  serial?: string
  file_path?: string
  captured_at?: string
}

type CompleteResponse = {
  serial?: string
  acc_serial?: string
  door_serial?: string
  freeze_door_serial?: string
  ref_door_serial?: string
  bx_result?: string
  compressor?: string
  bx_model?: string
  modeli?: string
  model_id?: number
  scan_photo?: ScanPhoto
  scan_photo_error?: string
}

type LastScan = {
  serial: string
  acc_serial: string
  freeze_door_serial: string
  ref_door_serial: string
  modeli: string
  compressor: string
  bx_model: string
  bx_result: string
  time: string
  file_path?: string
  photo_error?: string
  photo_key: number
}

type SessionRow = {
  id: number
  serial: string
  acc_serial: string
  freeze_door_serial: string
  ref_door_serial: string
  modeli: string
  time: string
}

type SessionApiRow = SessionRow & {
  door_serial?: string
}

type SessionsLastResponse = {
  sessions?: SessionApiRow[]
  last_scan?: {
    serial?: string
    acc_serial?: string
    door_serial?: string
    freeze_door_serial?: string
    ref_door_serial?: string
    modeli?: string
    compressor?: string
    bx_model?: string
    bx_result?: string
    time?: string
    file_path?: string
  }
}

function InfoRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  if (!value) return null
  return (
    <div className="min-w-0">
      <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
      <p className={cn("truncate text-base font-semibold", mono && "font-mono")}>{value}</p>
    </div>
  )
}

export default function QadoqlashPage() {
  const accRef = useRef<HTMLInputElement>(null)
  const freezeDoorRef = useRef<HTMLInputElement>(null)
  const refDoorRef = useRef<HTMLInputElement>(null)
  const serialRef = useRef<HTMLInputElement>(null)

  const [accSerial, setAccSerial] = useState("")
  const [freezeDoorSerial, setFreezeDoorSerial] = useState("")
  const [refDoorSerial, setRefDoorSerial] = useState("")
  const [serial, setSerial] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [reprintOnce, setReprintOnce] = useState(false)
  const [lastScan, setLastScan] = useState<LastScan | null>(null)
  const [sessions, setSessions] = useState<SessionRow[]>([])
  const [printers, setPrinters] = useState<PrinterV2[]>([])
  const [printerId, setPrinterId] = useState(0)
  const [reprintingSerial, setReprintingSerial] = useState("")

  const focusAcc = useCallback(() => {
    requestAnimationFrame(() => accRef.current?.focus())
  }, [])

  const focusSerial = useCallback(() => {
    requestAnimationFrame(() => serialRef.current?.focus())
  }, [])

  const clearAll = useCallback(() => {
    setAccSerial("")
    setFreezeDoorSerial("")
    setRefDoorSerial("")
    setSerial("")
  }, [])

  const loadPrinters = useCallback(async () => {
    const result = await Backend_Request<PrinterV2[]>(
      { line_id: QADOQLASH_LINE_ID },
      "/api/tech/printers-v2/by-line",
    )
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Printerlar yuklanmadi")
      return
    }
    const list = result.data || []
    setPrinters(list)
    setPrinterId((current) => {
      if (current && list.some((p) => p.id === current)) {
        return current
      }
      return list[0]?.id ?? 0
    })
  }, [])

  const loadSessions = useCallback(async () => {
    const result = await Backend_Request<SessionsLastResponse>(
      { limit: LAST_RECORDS_VISIBLE_ROWS },
      "/api/lines/qadoqlash/sessions/last",
    )
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Oxirgi sessiyalar yuklanmadi")
      return
    }
    const rows = (result.data?.sessions ?? []).map((row) => ({
      id: Number(row.id) || 0,
      serial: String(row.serial ?? ""),
      acc_serial: String(row.acc_serial ?? ""),
      freeze_door_serial: String(row.freeze_door_serial ?? row.door_serial ?? ""),
      ref_door_serial: String(row.ref_door_serial ?? ""),
      modeli: String(row.modeli ?? ""),
      time: String(row.time ?? ""),
    }))
    setSessions(rows)

    const scan = result.data?.last_scan
    if (scan?.serial) {
      setLastScan({
        serial: scan.serial,
        acc_serial: scan.acc_serial || "",
        freeze_door_serial: scan.freeze_door_serial || scan.door_serial || "",
        ref_door_serial: scan.ref_door_serial || "",
        modeli: scan.modeli || "",
        compressor: scan.compressor || "",
        bx_model: scan.bx_model || "",
        bx_result: scan.bx_result || "",
        time: scan.time || "",
        file_path: scan.file_path || undefined,
        photo_key: Date.now(),
      })
    }
  }, [])

  useEffect(() => {
    focusAcc()
    void Promise.all([loadPrinters(), loadSessions()])
  }, [focusAcc, loadPrinters, loadSessions])

  useEffect(() => {
    if (reprintOnce) {
      setAccSerial("")
      setFreezeDoorSerial("")
      setRefDoorSerial("")
      focusSerial()
    } else {
      focusAcc()
    }
  }, [reprintOnce, focusAcc, focusSerial])

  const clearErrorOnType = useCallback(() => {
    if (error) setError("")
  }, [error])

  const reprintSession = useCallback(
    async (rowSerial: string) => {
      if (!printerId) {
        ShowErrorToast("Printer tanlang")
        return
      }
      setReprintingSerial(rowSerial)
      const result = await Backend_Request(
        { serial: rowSerial, printer_v2_id: printerId },
        "/api/lines/qadoqlash/reprint",
      )
      setReprintingSerial("")
      if (result.result !== "ok") {
        ShowErrorToast(result.error || "Qayta chop etilmadi")
        return
      }
      ShowOKToast(`Qayta chop: ${rowSerial}`)
    },
    [printerId],
  )

  const complete = useCallback(
    async (acc: string, freezeDoor: string, refDoor: string, product: string, reprint: boolean) => {
      const p = product.trim()
      if (!p || loading) return
      if (!reprint && (!acc.trim() || !freezeDoor.trim() || !refDoor.trim())) return

      setLoading(true)
      const payload = reprint
        ? { serial: p, reprint: true }
        : {
            acc_serial: acc.trim(),
            freeze_door_serial: freezeDoor.trim(),
            ref_door_serial: refDoor.trim(),
            serial: p,
            reprint: false,
          }
      const result = await Backend_Request<CompleteResponse>(
        payload,
        "/api/lines/qadoqlash/complete",
      )
      setLoading(false)
      setReprintOnce(false)

      if (result.result !== "ok") {
        setError(result.error || "Xatolik yuz berdi")
        clearAll()
        focusAcc()
        return
      }

      const now = new Date()
      const time = now.toLocaleTimeString("uz-UZ", {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      })
      const data = result.data
      const outSerial = data?.serial || p
      const outAcc = data?.acc_serial || acc.trim()
      const outFreeze = data?.freeze_door_serial || data?.door_serial || freezeDoor.trim()
      const outRef = data?.ref_door_serial || refDoor.trim()
      const outModel = data?.modeli || ""

      setLastScan({
        serial: outSerial,
        acc_serial: outAcc,
        freeze_door_serial: outFreeze,
        ref_door_serial: outRef,
        modeli: outModel,
        compressor: data?.compressor || "",
        bx_model: data?.bx_model || "",
        bx_result: data?.bx_result || "",
        time,
        file_path: data?.scan_photo?.file_path,
        photo_error: data?.scan_photo_error,
        photo_key: Date.now(),
      })

      setSessions((prev) =>
        [
          {
            id: Date.now(),
            serial: outSerial,
            acc_serial: outAcc,
            freeze_door_serial: outFreeze,
            ref_door_serial: outRef,
            modeli: outModel,
            time,
          },
          ...prev,
        ].slice(0, LAST_RECORDS_VISIBLE_ROWS),
      )
      ShowOKToast(reprint ? "Qayta chop OK" : "OK")
      if (data?.scan_photo_error) {
        ShowWarningToast(data.scan_photo_error)
      }
      clearAll()
      focusAcc()
    },
    [clearAll, focusAcc, loading],
  )

  function onAccKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key !== "Enter" || reprintOnce) return
    event.preventDefault()
    if (!accSerial.trim()) return
    freezeDoorRef.current?.focus()
  }

  function onFreezeDoorKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key !== "Enter" || reprintOnce) return
    event.preventDefault()
    if (!freezeDoorSerial.trim()) return
    refDoorRef.current?.focus()
  }

  function onRefDoorKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key !== "Enter" || reprintOnce) return
    event.preventDefault()
    if (!refDoorSerial.trim()) return
    serialRef.current?.focus()
  }

  function onSerialKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key !== "Enter") return
    event.preventDefault()
    if (!serial.trim()) return
    void complete(accSerial, freezeDoorSerial, refDoorSerial, serial, reprintOnce)
  }

  const photoSrc =
    lastScan?.file_path
      ? `${Global_Data.server_ip}${lastScan.file_path}?t=${lastScan.photo_key}`
      : ""

  return (
    <PageContainer
      title="Yakuniy yig'uv uchastkasi"
      description="Acc → Eshik → Product serial; lab OK bo‘lsa chop etiladi"
    >
      {error ? (
        <div
          className="sticky top-0 z-20 mb-4 rounded-xl border border-destructive/40 bg-destructive px-4 py-3 text-sm font-medium text-destructive-foreground shadow-md"
          role="alert"
        >
          {error}
        </div>
      ) : null}

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(480px,640px)]">
        <Panel className="space-y-4 p-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              <PackageCheck className="size-4 text-primary" />
              {reprintOnce ? "Qayta chop — faqat product serial" : "Ketma-ket skan"}
              {loading ? <Loader2 className="size-4 animate-spin text-primary" /> : null}
            </div>
            <label
              htmlFor="qadoq-reprint"
              className={cn(
                "inline-flex cursor-pointer select-none items-center gap-2 rounded-xl border px-3 py-2 text-sm font-medium transition",
                reprintOnce
                  ? "border-amber-500/50 bg-amber-500/15 text-amber-800 dark:text-amber-200"
                  : "border-border bg-muted/30 text-muted-foreground hover:bg-muted/50",
                loading && "pointer-events-none opacity-60",
              )}
            >
              <input
                id="qadoq-reprint"
                type="checkbox"
                className="size-4 rounded border-border"
                checked={reprintOnce}
                disabled={loading}
                onChange={(e) => setReprintOnce(e.target.checked)}
              />
              Qayta chop (bir marta)
            </label>
          </div>

          <div className={cn("grid gap-4", reprintOnce ? "md:grid-cols-1" : "md:grid-cols-2 xl:grid-cols-4")}>
            {!reprintOnce ? (
              <>
                <div className="space-y-2">
                  <label htmlFor="qadoq-acc" className="text-sm font-medium">
                    Acc serial
                  </label>
                  <Input
                    id="qadoq-acc"
                    ref={accRef}
                    value={accSerial}
                    disabled={loading}
                    autoComplete="off"
                    spellCheck={false}
                    className="h-12 font-mono text-base"
                    placeholder="Acc skanerlang"
                    onChange={(e) => {
                      clearErrorOnType()
                      setAccSerial(e.target.value)
                    }}
                    onKeyDown={onAccKeyDown}
                  />
                </div>

                <div className="space-y-2">
                  <label htmlFor="qadoq-freeze" className="text-sm font-medium">
                    Freeze door
                  </label>
                  <Input
                    id="qadoq-freeze"
                    ref={freezeDoorRef}
                    value={freezeDoorSerial}
                    disabled={loading}
                    autoComplete="off"
                    spellCheck={false}
                    className="h-12 font-mono text-base"
                    placeholder="Freeze eshik"
                    onChange={(e) => {
                      clearErrorOnType()
                      setFreezeDoorSerial(e.target.value)
                    }}
                    onKeyDown={onFreezeDoorKeyDown}
                  />
                </div>

                <div className="space-y-2">
                  <label htmlFor="qadoq-ref" className="text-sm font-medium">
                    Ref door
                  </label>
                  <Input
                    id="qadoq-ref"
                    ref={refDoorRef}
                    value={refDoorSerial}
                    disabled={loading}
                    autoComplete="off"
                    spellCheck={false}
                    className="h-12 font-mono text-base"
                    placeholder="Ref eshik"
                    onChange={(e) => {
                      clearErrorOnType()
                      setRefDoorSerial(e.target.value)
                    }}
                    onKeyDown={onRefDoorKeyDown}
                  />
                </div>
              </>
            ) : null}

            <div className="space-y-2">
              <label htmlFor="qadoq-serial" className="text-sm font-medium">
                Product serial
              </label>
              <Input
                id="qadoq-serial"
                ref={serialRef}
                value={serial}
                disabled={loading}
                autoComplete="off"
                spellCheck={false}
                className="h-12 font-mono text-base"
                placeholder={reprintOnce ? "Product serial skanerlang — DB dan chop" : "Mahsulot serial"}
                onChange={(e) => {
                  clearErrorOnType()
                  setSerial(e.target.value)
                }}
                onKeyDown={onSerialKeyDown}
              />
            </div>
          </div>
        </Panel>

        <Panel className="overflow-hidden border-primary/20 bg-gradient-to-br from-primary/5 via-card to-card p-0">
          {!lastScan ? (
            <div className="flex h-full min-h-56 flex-col items-center justify-center gap-2 px-6 py-10 text-center text-muted-foreground">
              <CameraOff className="size-10 opacity-50" strokeWidth={1.5} />
              <p className="text-sm font-medium">Oxirgi skan hali yo‘q</p>
              <p className="text-xs">Muvaffaqiyatli skandan keyin rasm va ma’lumot shu yerda chiqadi</p>
            </div>
          ) : (
            <div className="flex h-full flex-col">
              <div className="flex items-center justify-between border-b border-border/70 px-4 py-2.5">
                <p className="text-sm font-semibold">Oxirgi skan</p>
                <p className="text-xs text-muted-foreground">{lastScan.time}</p>
              </div>

              <div className="flex flex-1 flex-col gap-4 p-4 sm:flex-row">
                {photoSrc ? (
                  <img
                    src={photoSrc}
                    alt={lastScan.serial}
                    className="h-52 w-full shrink-0 rounded-xl border border-border bg-black/5 object-contain sm:h-64 sm:w-56"
                  />
                ) : (
                  <div
                    className={cn(
                      "flex h-52 w-full shrink-0 flex-col items-center justify-center gap-2 rounded-xl border border-dashed px-3 text-center sm:h-64 sm:w-56",
                      lastScan.photo_error
                        ? "border-destructive/40 bg-destructive/5 text-destructive"
                        : "border-border bg-muted/40 text-muted-foreground",
                    )}
                  >
                    <CameraOff className="size-10 opacity-70" strokeWidth={1.5} />
                    <p className="text-sm font-medium">Surat yo‘q</p>
                    {lastScan.photo_error ? (
                      <p className="text-xs leading-snug opacity-90">{lastScan.photo_error}</p>
                    ) : null}
                  </div>
                )}

                <div className="flex min-w-0 flex-1 flex-col justify-center gap-3">
                  <div>
                    <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                      Product serial
                    </p>
                    <p className="break-all font-mono text-2xl font-bold leading-tight tracking-tight sm:text-3xl">
                      {lastScan.serial}
                    </p>
                  </div>

                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                    <InfoRow label="Acc" value={lastScan.acc_serial} mono />
                    <InfoRow label="Freeze" value={lastScan.freeze_door_serial} mono />
                    <InfoRow label="Ref" value={lastScan.ref_door_serial} mono />
                    <InfoRow label="Model" value={lastScan.modeli} />
                    <InfoRow label="Kompressor" value={lastScan.compressor} mono />
                    <InfoRow label="Lab model" value={lastScan.bx_model} />
                    <InfoRow
                      label="Lab natija"
                      value={lastScan.bx_result ? `bx_result=${lastScan.bx_result}` : ""}
                      mono
                    />
                  </div>
                </div>
              </div>
            </div>
          )}
        </Panel>
      </div>

      <Panel className={cn("mt-4 overflow-hidden p-0", LAST_RECORDS_PANEL_CLASS)}>
        <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
          <div>
            <p className="text-sm font-medium">Oxirgi sessiyalar</p>
            <p className="text-xs text-muted-foreground">Tanlangan printerga qayta chop</p>
          </div>
          <select
            value={printerId || ""}
            onChange={(event) => setPrinterId(Number(event.target.value))}
            className="h-9 min-w-48 max-w-xs rounded-lg border border-input bg-background px-2 text-sm"
            title={printers.find((p) => p.id === printerId)?.address || undefined}
          >
            <option value="">Printer tanlang</option>
            {printers.map((printer) => (
              <option key={printer.id} value={printer.id}>
                {printer.printer_name}
                {printer.label_template_name ? ` · ${printer.label_template_name}` : ""}
              </option>
            ))}
          </select>
        </div>
        {sessions.length === 0 ? (
          <div className="px-4 py-8 text-center text-sm text-muted-foreground">Hali sessiyalar yo‘q</div>
        ) : (
          <div className="overflow-auto">
            <table className="w-full text-sm">
              <thead className="bg-muted/40 text-left text-muted-foreground">
                <tr>
                  <th className="px-4 py-2 font-medium">Vaqt</th>
                  <th className="px-4 py-2 font-medium">Serial</th>
                  <th className="px-4 py-2 font-medium">Acc</th>
                  <th className="px-4 py-2 font-medium">Freeze</th>
                  <th className="px-4 py-2 font-medium">Ref</th>
                  <th className="px-4 py-2 font-medium">Model</th>
                  <th className="w-14 px-2 py-2" />
                </tr>
              </thead>
              <tbody>
                {sessions.map((row) => {
                  const busy = reprintingSerial === row.serial
                  return (
                    <tr key={row.id} className="border-t">
                      <td className="px-4 py-2 whitespace-nowrap">{row.time}</td>
                      <td className="px-4 py-2 font-mono">{row.serial}</td>
                      <td className="px-4 py-2 font-mono">{row.acc_serial}</td>
                      <td className="px-4 py-2 font-mono">{row.freeze_door_serial}</td>
                      <td className="px-4 py-2 font-mono">{row.ref_door_serial}</td>
                      <td className="px-4 py-2">{row.modeli}</td>
                      <td className="px-2 py-1.5 text-right">
                        <Button
                          type="button"
                          size="icon"
                          variant="outline"
                          className="size-8"
                          disabled={!printerId || busy || loading}
                          title="Tanlangan printerga chop"
                          onClick={() => void reprintSession(row.serial)}
                        >
                          {busy ? (
                            <Loader2 className="size-4 animate-spin" />
                          ) : (
                            <Printer className="size-4" />
                          )}
                        </Button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </Panel>
    </PageContainer>
  )
}
