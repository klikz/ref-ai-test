import { useCallback, useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { saveAs } from "file-saver"
import { CalendarDays, ChevronDown, FileDown, Search } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import { cn } from "@/lib/utils"
import { currentYearMonth, downloadPlanExport, todayDateInput, type PlanItemRow } from "./production_plan_shared"

type FilterOption = {
  value: string
  label: string
}

function normalizeSearchValue(value: unknown) {
  return String(value ?? "").trim().toLocaleLowerCase()
}

function uniqueOptionsFromRows(
  rows: PlanItemRow[],
  getValue: (row: PlanItemRow) => string,
): FilterOption[] {
  const values = new Set<string>()
  for (const row of rows) {
    const value = String(getValue(row)).trim()
    if (value) {
      values.add(value)
    }
  }
  return Array.from(values)
    .sort((a, b) => a.localeCompare(b))
    .map((value) => ({ value, label: value }))
}

function uniqueLineOptionsFromRows(rows: PlanItemRow[]): FilterOption[] {
  const byId = new Map<number, string>()
  for (const row of rows) {
    if (row.line_id > 0) {
      byId.set(row.line_id, String(row.line_name || row.line_id).trim())
    }
  }
  return Array.from(byId.entries())
    .sort((a, b) => a[1].localeCompare(b[1]))
    .map(([id, label]) => ({ value: String(id), label }))
}

function applyDatePreset(type: "month" | "year") {
  const now = new Date()
  if (type === "year") {
    const start = new Date(now.getFullYear(), 0, 1)
    const month = String(start.getMonth() + 1).padStart(2, "0")
    const day = String(start.getDate()).padStart(2, "0")
    const date1 = `${start.getFullYear()}-${month}-${day}`
    return { date1, date2: todayDateInput() }
  }
  const start = new Date(now.getFullYear(), now.getMonth(), 1)
  const month = String(start.getMonth() + 1).padStart(2, "0")
  const day = String(start.getDate()).padStart(2, "0")
  const date1 = `${start.getFullYear()}-${month}-${day}`
  return { date1, date2: todayDateInput() }
}

function toggleValue(list: string[], value: string): string[] {
  return list.includes(value) ? list.filter((item) => item !== value) : [...list, value]
}

function selectionLabel(selectedValues: string[], options: FilterOption[], emptyLabel: string) {
  if (selectedValues.length === 0) {
    return emptyLabel
  }
  const labels = selectedValues.map(
    (value) => options.find((option) => option.value === value)?.label ?? value,
  )
  if (labels.length <= 2) {
    return labels.join(", ")
  }
  return `${labels.length} ta tanlangan`
}

type MultiSelectSearchDropdownProps = {
  label: string
  selectedValues: string[]
  options: FilterOption[]
  emptyLabel: string
  searchPlaceholder: string
  onChange: (values: string[]) => void
}

function MultiSelectSearchDropdown({
  label,
  selectedValues,
  options,
  emptyLabel,
  searchPlaceholder,
  onChange,
}: MultiSelectSearchDropdownProps) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState("")

  const filteredOptions = useMemo(() => {
    const query = normalizeSearchValue(search)
    if (!query) {
      return options
    }
    return options.filter((option) => normalizeSearchValue(option.label).includes(query))
  }, [options, search])

  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      <DropdownMenu
        open={open}
        onOpenChange={(nextOpen) => {
          setOpen(nextOpen)
          if (!nextOpen) {
            setSearch("")
          }
        }}
      >
        <DropdownMenuTrigger asChild>
          <Button variant="outline" className="h-11 w-full min-w-[220px] justify-between gap-2 rounded-xl">
            <span className="truncate">{selectionLabel(selectedValues, options, emptyLabel)}</span>
            <ChevronDown className="size-4 shrink-0 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-[min(480px,calc(100vw-2rem))] rounded-xl p-0" align="start">
          <div className="border-b p-2" onKeyDown={(event) => event.stopPropagation()}>
            <div className="relative">
              <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder={searchPlaceholder}
                className="h-9 pl-8"
                autoFocus
              />
            </div>
          </div>
          <DropdownMenuGroup className="max-h-72 overflow-y-auto p-1">
            <DropdownMenuItem
              className={cn(selectedValues.length === 0 && "bg-primary/10")}
              onSelect={(event) => event.preventDefault()}
              onClick={() => onChange([])}
            >
              {emptyLabel}
            </DropdownMenuItem>
            {filteredOptions.length === 0 ? (
              <div className="px-3 py-4 text-center text-sm text-muted-foreground">Topilmadi</div>
            ) : (
              filteredOptions.map((option) => {
                const checked = selectedValues.includes(option.value)
                return (
                  <DropdownMenuItem
                    key={option.value}
                    className={cn(checked && "bg-primary/10")}
                    onSelect={(event) => event.preventDefault()}
                    onClick={() => onChange(toggleValue(selectedValues, option.value))}
                  >
                    <span className="flex items-center gap-2">
                      <input type="checkbox" readOnly checked={checked} className="size-4 rounded border" />
                      <span className="truncate">{option.label}</span>
                    </span>
                  </DropdownMenuItem>
                )
              })
            )}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}

export default function ProductionPlanReportPage() {
  const [dateFrom, setDateFrom] = useState(todayDateInput())
  const [dateTo, setDateTo] = useState(todayDateInput())
  const [rows, setRows] = useState<PlanItemRow[]>([])
  const [loading, setLoading] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [tableExporting, setTableExporting] = useState(false)
  const [selectedArtikuls, setSelectedArtikuls] = useState<string[]>([])
  const [selectedOdooCodes, setSelectedOdooCodes] = useState<string[]>([])
  const [selectedKorxonaKodlari, setSelectedKorxonaKodlari] = useState<string[]>([])
  const [selectedLineIds, setSelectedLineIds] = useState<string[]>([])

  const load = useCallback(async () => {
    setLoading(true)
    const result = await Backend_Request<PlanItemRow[]>(
      { date_from: dateFrom, date_to: dateTo },
      "/api/production/plan/report",
    )
    setLoading(false)
    if (result.result === "ok") {
      setRows(result.data ?? [])
    } else {
      setRows([])
      ShowErrorToast(result.error || "Hisobot yuklanmadi")
    }
  }, [dateFrom, dateTo])

  useEffect(() => {
    void load()
  }, [load])

  const artikulOptions = useMemo(
    () => uniqueOptionsFromRows(rows, (row) => row.artikul_raqami),
    [rows],
  )

  const odooCodeOptions = useMemo(
    () => uniqueOptionsFromRows(rows, (row) => row.odoo_code),
    [rows],
  )

  const korxonaKodiOptions = useMemo(
    () => uniqueOptionsFromRows(rows, (row) => row.label),
    [rows],
  )

  const lineOptions = useMemo(() => uniqueLineOptionsFromRows(rows), [rows])

  const filteredRows = useMemo(() => {
    const odooCodeSet = new Set(selectedOdooCodes)
    const artikulSet = new Set(selectedArtikuls)
    const korxonaKodiSet = new Set(selectedKorxonaKodlari)
    const lineIdSet = new Set(selectedLineIds.map((value) => Number(value)))

    return rows.filter((row) => {
      const artikul = String(row.artikul_raqami ?? "").trim()
      const odooCode = String(row.odoo_code ?? "").trim()
      const korxonaKodi = String(row.label ?? "").trim()

      const artikulOk =
        artikulSet.size === 0 || (artikul !== "" && artikulSet.has(artikul))
      const odooCodeOk =
        odooCodeSet.size === 0 || (odooCode !== "" && odooCodeSet.has(odooCode))
      const korxonaKodiOk =
        korxonaKodiSet.size === 0 || (korxonaKodi !== "" && korxonaKodiSet.has(korxonaKodi))
      const lineOk =
        lineIdSet.size === 0 || (row.line_id > 0 && lineIdSet.has(row.line_id))

      return artikulOk && odooCodeOk && korxonaKodiOk && lineOk
    })
  }, [rows, selectedArtikuls, selectedOdooCodes, selectedKorxonaKodlari, selectedLineIds])

  async function exportMonth() {
    const ym = dateFrom.slice(0, 7) || currentYearMonth()
    setExporting(true)
    try {
      await downloadPlanExport(ym, true)
      ShowOKToast("Excel yuklandi")
    } catch (error) {
      ShowErrorToast(error instanceof Error ? error.message : "Export xatolik")
    } finally {
      setExporting(false)
    }
  }

  async function exportTableToExcel() {
    if (filteredRows.length === 0) {
      ShowErrorToast("Excel uchun ma'lumot topilmadi")
      return
    }

    setTableExporting(true)
    try {
      const XLSX = await import("xlsx")
      const exportData = filteredRows.map((row) => ({
        Sana: row.plan_date,
        Liniya: row.line_name || row.line_id,
        Smena: `${row.shift_no || 1}-sm`,
        "Korxonda kodi": row.label,
        Artikul: row.artikul_raqami,
        "ODOO code": row.odoo_code,
        Reja: row.planned_qty,
        Fakt: row.actual_qty,
        Qoldiq: row.remaining_qty,
        "%": row.completion_pct,
      }))
      const worksheet = XLSX.utils.json_to_sheet(exportData)
      const workbook = XLSX.utils.book_new()
      XLSX.utils.book_append_sheet(workbook, worksheet, "Jadval")
      const excelBuffer = XLSX.write(workbook, { bookType: "xlsx", type: "array" })
      saveAs(
        new Blob([excelBuffer], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
        `reja-bajarilish-${dateFrom}-${dateTo}.xlsx`,
      )
      ShowOKToast("XLSX yuklandi")
    } catch (error) {
      ShowErrorToast(error instanceof Error ? error.message : "Export xatolik")
    } finally {
      setTableExporting(false)
    }
  }

  return (
    <PageContainer
      title="Reja va bajarilish"
      description="Reja va fakt taqqoslash"
      fullWidth
      actions={
        <Button variant="outline" asChild>
          <Link to="/production/plan">Reja boshqaruvi</Link>
        </Button>
      }
    >
      <Panel title="Filtr">
        <div className="grid gap-4 xl:grid-cols-[1fr_auto]">
          <div className="grid gap-4 lg:grid-cols-[minmax(280px,360px)_1fr]">
            <div className="rounded-2xl border border-border/60 bg-muted/20 p-3">
              <div className="mb-2 flex items-center gap-2 text-sm font-medium">
                <CalendarDays className="size-4 text-primary" />
                Sana oralig&apos;i
              </div>
              <div className="grid gap-2 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label>Dan</Label>
                  <Input
                    type="date"
                    value={dateFrom}
                    onChange={(e) => setDateFrom(e.target.value)}
                    className="h-11 rounded-xl"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Gacha</Label>
                  <Input
                    type="date"
                    value={dateTo}
                    onChange={(e) => setDateTo(e.target.value)}
                    className="h-11 rounded-xl"
                  />
                </div>
              </div>
              <div className="mt-2 flex flex-wrap gap-2">
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  className="rounded-full"
                  onClick={() => {
                    const next = applyDatePreset("month")
                    setDateFrom(next.date1)
                    setDateTo(next.date2)
                  }}
                >
                  Oy boshidan
                </Button>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  className="rounded-full"
                  onClick={() => {
                    const next = applyDatePreset("year")
                    setDateFrom(next.date1)
                    setDateTo(next.date2)
                  }}
                >
                  Yil boshidan
                </Button>
              </div>
            </div>

            <div className="space-y-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <MultiSelectSearchDropdown
                  label="Artikul"
                  selectedValues={selectedArtikuls}
                  options={artikulOptions}
                  emptyLabel="Barcha artikullar"
                  searchPlaceholder="Artikul qidirish..."
                  onChange={setSelectedArtikuls}
                />

                <MultiSelectSearchDropdown
                  label="Odoo Code"
                  selectedValues={selectedOdooCodes}
                  options={odooCodeOptions}
                  emptyLabel="Barcha Odoo Code"
                  searchPlaceholder="Odoo Code qidirish..."
                  onChange={setSelectedOdooCodes}
                />
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <MultiSelectSearchDropdown
                  label="Korxona kodi"
                  selectedValues={selectedKorxonaKodlari}
                  options={korxonaKodiOptions}
                  emptyLabel="Barcha kodlar"
                  searchPlaceholder="Korxona kodi qidirish..."
                  onChange={setSelectedKorxonaKodlari}
                />

                <MultiSelectSearchDropdown
                  label="Liniya"
                  selectedValues={selectedLineIds}
                  options={lineOptions}
                  emptyLabel="Barcha liniyalar"
                  searchPlaceholder="Liniya qidirish..."
                  onChange={setSelectedLineIds}
                />
              </div>
            </div>
          </div>

          <div className="flex flex-wrap items-end gap-2">
            <Button onClick={() => void load()} disabled={loading} className="h-11 rounded-xl px-6">
              {loading ? "Yuklanmoqda..." : "Yangilash"}
            </Button>
            <Button
              variant="outline"
              disabled={exporting}
              onClick={() => void exportMonth()}
              className="h-11 rounded-xl px-6"
            >
              <FileDown className="size-4" />
              Excel
            </Button>
          </div>
        </div>
      </Panel>

      <Panel
        title="Jadval"
        className="mt-4"
        noPadding
        action={
          <Button
            variant="outline"
            className="gap-2 rounded-xl"
            disabled={tableExporting || filteredRows.length === 0}
            onClick={() => void exportTableToExcel()}
          >
            <FileDown className="size-4" />
            {tableExporting ? "Saqlanmoqda..." : "XLSX"}
          </Button>
        }
      >
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Sana</TableHead>
                <TableHead>Liniya</TableHead>
                <TableHead>Smena</TableHead>
                <TableHead>Korxonda kodi</TableHead>
                <TableHead>Artikul</TableHead>
                <TableHead>ODOO code</TableHead>
                <TableHead className="text-right">Reja</TableHead>
                <TableHead className="text-right">Fakt</TableHead>
                <TableHead className="text-right">Qoldiq</TableHead>
                <TableHead className="text-right">%</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredRows.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="tabular-nums">{row.plan_date}</TableCell>
                  <TableCell>{row.line_name || row.line_id}</TableCell>
                  <TableCell className="tabular-nums">{row.shift_no || 1}-sm</TableCell>
                  <TableCell className="font-medium">{row.label}</TableCell>
                  <TableCell className="tabular-nums">{row.artikul_raqami}</TableCell>
                  <TableCell className="tabular-nums">{row.odoo_code}</TableCell>
                  <TableCell className="text-right tabular-nums">{row.planned_qty}</TableCell>
                  <TableCell className="text-right tabular-nums">{row.actual_qty}</TableCell>
                  <TableCell className="text-right tabular-nums">{row.remaining_qty}</TableCell>
                  <TableCell
                    className={cn(
                      "text-right tabular-nums",
                      !row.allow_overplan && row.actual_qty > row.planned_qty && "text-destructive font-semibold",
                    )}
                  >
                    {row.completion_pct}%
                  </TableCell>
                </TableRow>
              ))}
              {!loading && filteredRows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={10} className="h-20 text-center text-muted-foreground">
                    Ma&apos;lumot yo&apos;q
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </div>
      </Panel>
    </PageContainer>
  )
}
