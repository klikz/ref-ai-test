import { useCallback, useRef, useState, type ReactNode } from "react"
import {
  Barcode,
  Camera,
  CheckCircle2,
  FlaskConical,
  ImageOff,
  Loader2,
  Search,
  XCircle,
} from "lucide-react"
import { Global_Data } from "@/config/config"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ShowErrorToast } from "@/components/showToast"
import { cn } from "@/lib/utils"

type SerialInfoCatalog = {
  model_id?: number
  artikul?: string
  modeli?: string
  model_nomi?: string
  import_code?: string
  seriya_raqami?: string
  acc_serial_rule?: string
  compressor_serial_rule?: string
  gs1_shablon?: string
}

type SerialInfoProduct = {
  product_id?: number
  acc_serial?: string
  status?: string
  line_id?: number
  line_name?: string
  registered_at?: string
  transferred_at?: string
  user_name?: string
  gs_code?: string
  gs_code_at?: string
  odoo_code?: string
  brend?: string
  gs1_ean13?: string
  modeli?: string
  model_nomi?: string
}

type SerialInfoCompressor = {
  params_id?: number
  serial_number?: string
  compressor_serial?: string
  acc_serial?: string
  door_serial?: string
  freeze_door_serial?: string
  ref_door_serial?: string
  gscode?: string
  model_id?: number
  modeli?: string
  created_at?: string
  user_name?: string
  matched_by?: string
}

type ScanPhoto = {
  id?: number
  file_path?: string
  captured_at?: string
  line_id?: number
}

type LabBxRow = {
  serial?: string
  compressor?: string
  bx_model?: string
  line_num?: string
  point_num?: string
  start_time?: string
  stop_time?: string
  test_time?: string
  bx_result?: string
}

type SerialInfoLab = {
  source?: string
  latest?: LabBxRow
  rows?: LabBxRow[]
}

type SerialInfoResponse = {
  serial: string
  in_products: boolean
  catalog?: SerialInfoCatalog
  product?: SerialInfoProduct
  compressor?: SerialInfoCompressor
  scan_photo?: ScanPhoto
  lab?: SerialInfoLab
}

const productStatusLabel: Record<string, string> = {
  active: "Balansda",
  transferred: "O'tkazilgan",
  written_off: "Hisobdan chiqarilgan",
}

const productStatusClass: Record<string, string> = {
  active: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
  transferred: "bg-sky-500/15 text-sky-700 dark:text-sky-300",
  written_off: "bg-rose-500/15 text-rose-700 dark:text-rose-300",
}

function photoURL(path: string) {
  return `${Global_Data.server_ip}${path}`
}

function labResultOk(value?: string) {
  return stringsTrim(value) === "01"
}

function stringsTrim(value?: string) {
  return (value ?? "").trim()
}

function InfoRow({
  label,
  value,
  mono,
}: {
  label: string
  value?: string | number | null
  mono?: boolean
}) {
  const text = value == null || value === "" ? "—" : String(value)
  return (
    <div className="grid gap-0.5 border-b border-border/40 py-1.5 last:border-0 sm:grid-cols-[minmax(0,130px)_1fr] sm:items-baseline sm:gap-2">
      <span className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </span>
      <span className={cn("text-sm leading-snug text-foreground break-all", mono && "font-mono")}>
        {text}
      </span>
    </div>
  )
}

function SectionCard({
  title,
  icon: Icon,
  children,
  className,
  badge,
}: {
  title: string
  icon: typeof Barcode
  children: ReactNode
  className?: string
  badge?: ReactNode
}) {
  return (
    <div
      className={cn(
        "flex h-full flex-col overflow-hidden rounded-2xl border border-border/60 bg-card shadow-sm",
        className,
      )}
    >
      <div className="flex shrink-0 items-center gap-2 border-b border-border/50 bg-muted/30 px-4 py-2">
        <div className="rounded-lg bg-primary/10 p-1.5 text-primary">
          <Icon className="size-4" />
        </div>
        <h3 className="min-w-0 flex-1 text-sm font-semibold text-foreground">{title}</h3>
        {badge}
      </div>
      <div className="flex-1 px-4 pb-2 pt-1">{children}</div>
    </div>
  )
}

export default function SerialInfoPage() {
  const inputRef = useRef<HTMLInputElement>(null)
  const [serial, setSerial] = useState("")
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<SerialInfoResponse | null>(null)
  const [searched, setSearched] = useState(false)

  const search = useCallback(async (value?: string) => {
    const q = (value ?? serial).trim()
    if (!q) {
      ShowErrorToast("Serial kiriting")
      inputRef.current?.focus()
      return
    }

    setLoading(true)
    setSearched(true)
    const result = await Backend_Request<SerialInfoResponse>({ serial: q }, "/api/serial/info")
    setLoading(false)

    if (result.result !== "ok") {
      setData(null)
      ShowErrorToast(result.error || "Ma'lumot topilmadi")
      return
    }

    setData(result.data ?? null)
    setSerial(q)
  }, [serial])

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    void search()
  }

  const status = data?.product?.status ?? ""
  const statusLabel = productStatusLabel[status] ?? status
  const labLatest = data?.lab?.latest
  const labRows = data?.lab?.rows ?? []
  const labOk = labResultOk(labLatest?.bx_result)

  return (
    <PageContainer
      title="Serial Info"
      description="Serial yoki kompressor nomer bo'yicha model, ishlab chiqarish, Photo va laboratory"
    >
      <Panel className="overflow-hidden border-primary/20 bg-gradient-to-br from-primary/5 via-card to-card p-0">
        <form
          onSubmit={onSubmit}
          className="flex flex-col gap-4 p-5 sm:flex-row sm:items-end"
        >
          <div className="flex-1 space-y-2">
            <label
              htmlFor="serial-search"
              className="flex items-center gap-2 text-sm font-medium text-foreground"
            >
              <Barcode className="size-4 text-primary" />
              Serial raqam
            </label>
            <Input
              id="serial-search"
              ref={inputRef}
              value={serial}
              onChange={(e) => setSerial(e.target.value)}
              placeholder="AC serial yoki kompressor nomer"
              className="h-12 rounded-xl border-border/70 bg-background/80 text-base font-mono shadow-inner"
              autoComplete="off"
              spellCheck={false}
            />
          </div>
          <Button
            type="submit"
            size="lg"
            className="h-12 rounded-xl px-6 shadow-md shadow-primary/20"
            disabled={loading}
          >
            {loading ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <Search className="size-4" />
            )}
            Qidirish
          </Button>
        </form>
      </Panel>

      {loading && (
        <div className="flex items-center justify-center gap-2 rounded-2xl border border-dashed border-border/70 bg-muted/20 py-16 text-muted-foreground">
          <Loader2 className="size-5 animate-spin" />
          Ma'lumot yuklanmoqda…
        </div>
      )}

      {!loading && searched && !data && (
        <div className="flex flex-col items-center justify-center gap-3 rounded-2xl border border-dashed border-rose-500/30 bg-rose-500/5 py-16 text-center">
          <XCircle className="size-10 text-rose-500/80" />
          <p className="text-sm text-muted-foreground">
            Ushbu serial bo'yicha ma'lumot topilmadi
          </p>
        </div>
      )}

      {!loading && data && (
        <div className="space-y-5">
          <div className="glass-card flex flex-wrap items-center gap-3 border-border/60 p-4">
            <div className="rounded-xl bg-primary/10 p-2 text-primary">
              <Barcode className="size-5" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-xs uppercase tracking-wide text-muted-foreground">Serial</p>
              <p className="truncate font-mono text-lg font-semibold text-foreground">
                {data.serial}
              </p>
            </div>
            {data.in_products ? (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/15 px-3 py-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
                <CheckCircle2 className="size-3.5" />
                Bazada mavjud
              </span>
            ) : data.compressor ? (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-sky-500/15 px-3 py-1 text-xs font-medium text-sky-800 dark:text-sky-200">
                Kompressor bog'langan
              </span>
            ) : (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-amber-500/15 px-3 py-1 text-xs font-medium text-amber-800 dark:text-amber-200">
                Faqat katalog / surat
              </span>
            )}
            {status && (
              <span
                className={cn(
                  "rounded-full px-3 py-1 text-xs font-medium",
                  productStatusClass[status] ?? "bg-muted text-muted-foreground",
                )}
              >
                {statusLabel}
              </span>
            )}
            {labLatest ? (
              <span
                className={cn(
                  "inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium",
                  labOk
                    ? "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300"
                    : "bg-rose-500/15 text-rose-700 dark:text-rose-300",
                )}
              >
                <FlaskConical className="size-3.5" />
                Lab {labOk ? "OK" : "muammo"}
              </span>
            ) : null}
          </div>

          <div className="grid items-stretch gap-4 xl:grid-cols-[minmax(280px,360px)_1fr]">
            <SectionCard title="Photo" icon={Camera} className="xl:row-span-2">
              {data.scan_photo?.file_path ? (
                <div className="space-y-2 py-1">
                  <div className="overflow-hidden rounded-xl border border-border/60 bg-black/5 shadow-inner">
                    <img
                      src={photoURL(data.scan_photo.file_path)}
                      alt={`Photo: ${data.serial}`}
                      className="aspect-[4/3] w-full object-contain"
                    />
                  </div>
                  <InfoRow label="Surat vaqti" value={data.scan_photo.captured_at} />
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center gap-2 py-12 text-center text-muted-foreground">
                  <ImageOff className="size-10 opacity-50" />
                  <p className="text-sm">Photo topilmadi</p>
                </div>
              )}
            </SectionCard>

            <div className="grid items-stretch gap-4 md:grid-cols-2">
              {data.catalog && (
                <SectionCard title="Model (katalog)" icon={Barcode}>
                  <InfoRow label="Model ID" value={data.catalog.model_id} />
                  <InfoRow label="Modeli" value={data.catalog.modeli} mono />
                  <InfoRow label="Model nomi" value={data.catalog.model_nomi} />
                  <InfoRow label="Artikul" value={data.catalog.artikul} mono />
                  <InfoRow label="Seriya raqami" value={data.catalog.seriya_raqami} mono />
                  <InfoRow label="Import kodi" value={data.catalog.import_code} mono />
                  <InfoRow label="GS1 shablon" value={data.catalog.gs1_shablon} mono />
                  <InfoRow label="Acc serial qoidasi" value={data.catalog.acc_serial_rule} mono />
                </SectionCard>
              )}

              {(data.product || data.compressor) && (
                <SectionCard title="Ishlab chiqarish" icon={CheckCircle2}>
                  {data.product && (
                    <>
                      <InfoRow label="Mahsulot ID" value={data.product.product_id} />
                      <InfoRow label="Liniya" value={data.product.line_name} />
                      <InfoRow label="Acc serial" value={data.product.acc_serial} mono />
                      <InfoRow label="Modeli" value={data.product.modeli} mono />
                      <InfoRow label="Model nomi" value={data.product.model_nomi} />
                      <InfoRow label="Brend" value={data.product.brend} />
                      <InfoRow label="Odoo kodi" value={data.product.odoo_code} mono />
                      <InfoRow label="GS1 EAN13" value={data.product.gs1_ean13} mono />
                      <InfoRow label="GS Code" value={data.product.gs_code} mono />
                      <InfoRow label="GS Code vaqti" value={data.product.gs_code_at} />
                      <InfoRow label="Ro'yxatdan o'tgan" value={data.product.registered_at} />
                      <InfoRow label="O'tkazilgan" value={data.product.transferred_at} />
                      <InfoRow label="Operator" value={data.product.user_name} />
                    </>
                  )}
                  <InfoRow
                    label="Kompressor nomer"
                    value={data.compressor?.compressor_serial}
                    mono
                  />
                  <InfoRow label="Door serial" value={data.compressor?.door_serial} mono />
                  <InfoRow label="Freeze door" value={data.compressor?.freeze_door_serial} mono />
                  <InfoRow label="Ref door" value={data.compressor?.ref_door_serial} mono />
                  {data.compressor?.serial_number &&
                    data.compressor.serial_number !== data.serial && (
                      <InfoRow
                        label="Bog'langan serial"
                        value={data.compressor.serial_number}
                        mono
                      />
                    )}
                </SectionCard>
              )}
            </div>
          </div>

          <SectionCard
            title="Laboratory"
            icon={FlaskConical}
            badge={
              labLatest ? (
                <span
                  className={cn(
                    "rounded-full px-2.5 py-0.5 text-[11px] font-semibold",
                    labOk
                      ? "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300"
                      : "bg-rose-500/15 text-rose-700 dark:text-rose-300",
                  )}
                >
                  {labOk ? "bx_result=01 OK" : `bx_result=${labLatest.bx_result || "—"}`}
                </span>
              ) : (
                <span className="rounded-full bg-muted px-2.5 py-0.5 text-[11px] font-medium text-muted-foreground">
                  Ma'lumot yo‘q
                </span>
              )
            }
          >
            {!labLatest ? (
              <div className="flex flex-col items-center justify-center gap-2 py-10 text-center text-muted-foreground">
                <FlaskConical className="size-10 opacity-40" />
                <p className="text-sm">Laboratoriya natijasi topilmadi</p>
              </div>
            ) : (
              <div className="space-y-4 py-2">
                <div
                  className={cn(
                    "grid gap-3 rounded-xl border p-4 sm:grid-cols-2 lg:grid-cols-4",
                    labOk
                      ? "border-emerald-500/30 bg-emerald-500/5"
                      : "border-rose-500/30 bg-rose-500/5",
                  )}
                >
                  <div>
                    <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                      Natija
                    </p>
                    <p className="font-mono text-xl font-bold">{labLatest.bx_result || "—"}</p>
                  </div>
                  <div>
                    <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                      Kompressor
                    </p>
                    <p className="break-all font-mono text-sm font-semibold">
                      {labLatest.compressor || "—"}
                    </p>
                  </div>
                  <div>
                    <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                      Bx model
                    </p>
                    <p className="break-all text-sm font-semibold">{labLatest.bx_model || "—"}</p>
                  </div>
                  <div>
                    <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                      Test / Stop
                    </p>
                    <p className="text-sm font-semibold">
                      {labLatest.test_time || "—"}
                      {labLatest.stop_time ? (
                        <span className="mt-0.5 block text-xs font-normal text-muted-foreground">
                          {labLatest.stop_time}
                        </span>
                      ) : null}
                    </p>
                  </div>
                  <div>
                    <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                      Line / Point
                    </p>
                    <p className="text-sm font-semibold">
                      {[labLatest.line_num, labLatest.point_num].filter(Boolean).join(" / ") || "—"}
                    </p>
                  </div>
                  <div>
                    <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                      Start
                    </p>
                    <p className="text-sm font-semibold">{labLatest.start_time || "—"}</p>
                  </div>
                  <div>
                    <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                      Stop
                    </p>
                    <p className="text-sm font-semibold">{labLatest.stop_time || "—"}</p>
                  </div>
                  <div>
                    <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                      Manba
                    </p>
                    <p className="text-sm font-semibold">
                      {data.lab?.source === "mssql" ? "VTM MSSQL" : "Postgres"}
                    </p>
                  </div>
                </div>

                {labRows.length > 1 ? (
                  <div className="overflow-hidden rounded-xl border border-border/60">
                    <div className="border-b bg-muted/30 px-3 py-2 text-xs font-medium text-muted-foreground">
                      Barcha testlar ({labRows.length})
                    </div>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Stop</TableHead>
                          <TableHead>BxModel</TableHead>
                          <TableHead>Compressor</TableHead>
                          <TableHead>Line</TableHead>
                          <TableHead>Point</TableHead>
                          <TableHead>Test</TableHead>
                          <TableHead>Result</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {[...labRows].reverse().map((row, index) => {
                          const ok = labResultOk(row.bx_result)
                          return (
                            <TableRow key={`${row.stop_time}-${index}`}>
                              <TableCell className="whitespace-nowrap font-mono text-xs">
                                {row.stop_time || "—"}
                              </TableCell>
                              <TableCell className="text-xs">{row.bx_model || "—"}</TableCell>
                              <TableCell className="font-mono text-xs">{row.compressor || "—"}</TableCell>
                              <TableCell className="text-xs">{row.line_num || "—"}</TableCell>
                              <TableCell className="text-xs">{row.point_num || "—"}</TableCell>
                              <TableCell className="text-xs">{row.test_time || "—"}</TableCell>
                              <TableCell>
                                <span
                                  className={cn(
                                    "rounded-full px-2 py-0.5 font-mono text-xs font-semibold",
                                    ok
                                      ? "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300"
                                      : "bg-rose-500/15 text-rose-700 dark:text-rose-300",
                                  )}
                                >
                                  {row.bx_result || "—"}
                                </span>
                              </TableCell>
                            </TableRow>
                          )
                        })}
                      </TableBody>
                    </Table>
                  </div>
                ) : null}
              </div>
            )}
          </SectionCard>
        </div>
      )}
    </PageContainer>
  )
}
