import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { saveAs } from "file-saver"
import { useEffect, useMemo, useState } from "react"
import { Backend_Request } from "@/services/backend"
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuGroup,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
    createColumnHelper,
    flexRender,
    getCoreRowModel,
    useReactTable,
} from "@tanstack/react-table"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table"
import { ChevronDown, FileDown, RotateCcw, Search } from "lucide-react"
import { cn } from "@/lib/utils"
import {
    PLAN_AUX_LINE_IDS,
    PLAN_PRODUCT_LINE_IDS,
    type PlanItemRow,
} from "./production_plan_shared"

type Line = {
    line_id: number
    name: string
}

type Model = {
    id: number
    seriya_raqami: string
    modeli: string
    model_nomi: string
    qisqa_nomi?: string
    rangi: string
    gs1_ean13: string
    odoo_code?: string
}

type SummaryRow = {
    line_id?: number
    line_name: string
    model_id?: number
    model_name: string
    seriya_raqami: string
    odoo_code: string
    count: number
}

type MainPlanSummaryRow = {
    line_id: number
    line_name: string
    model_id: number
    label: string
    odoo_code: string
    planned_qty: number
    actual_qty: number
}

type MainMergedRow = {
    line_id: number
    line_name: string
    model_id: number
    model_name: string
    seriya_raqami: string
    odoo_code: string
    count: number
    planned_qty: number
    actual_qty: number
}

type AuxSummaryRow = {
    line_id: number
    line_name: string
    component_id: number
    factory_code: string
    component_name: string
    odoo_code: string
    received: number
    expended: number
    balance_end: number
}

type AuxMergedRow = {
    line_id: number
    line_name: string
    component_id: number
    factory_code: string
    component_name: string
    odoo_code: string
    received: number
    expended: number
    balance_end: number
    planned_qty: number
    actual_qty: number
}

type AuxPlanSummaryRow = {
    line_id: number
    line_name: string
    component_id: number
    label: string
    factory_code: string
    odoo_code: string
    planned_qty: number
    actual_qty: number
    remaining_qty: number
    completion_pct: number
    allow_overplan: boolean
}

type ReportResponse = {
    detailed: any[]
    count: number
    short_table: SummaryRow[]
    total_pages?: number
}

type AuxReportResponse = {
    detailed: any[]
    count: number
    short_table: AuxSummaryRow[]
    total_pages?: number
}

function toDateInputValue(date: Date) {
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, "0")
    const day = String(date.getDate()).padStart(2, "0")
    return `${year}-${month}-${day}`
}

function pad2(value: number) {
    return String(value).padStart(2, "0")
}

/** datetime-local value: YYYY-MM-DDTHH:mm */
function toDateTimeLocalValue(date: Date) {
    return `${toDateInputValue(date)}T${pad2(date.getHours())}:${pad2(date.getMinutes())}`
}

function startOfDayLocal(date: Date) {
    return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 0, 0, 0, 0)
}

function endOfDayLocal(date: Date) {
    return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 23, 59, 0, 0)
}

/** Reja API uchun faqat sana (YYYY-MM-DD) */
function toPlanDateValue(dateTimeLocal: string) {
    return dateTimeLocal.slice(0, 10)
}

function normalizeSearchValue(value: unknown) {
    return String(value ?? "").trim().toLocaleLowerCase()
}

function modelDropdownLabel(model: Model) {
    const parts = [model.modeli, model.seriya_raqami, model.rangi].filter(Boolean)
    return parts.length > 0 ? parts.join(" — ") : `ID ${model.id}`
}

const filterCardClassName =
    "flex flex-col gap-2 rounded-2xl border border-border/60 bg-card/80 p-3 shadow-sm"

const MAIN_REPORT_LINE_IDS = new Set(PLAN_PRODUCT_LINE_IDS)
const AUX_REPORT_LINE_IDS = new Set(PLAN_AUX_LINE_IDS)

function formatBalanceQty(value: number) {
    return Number(value).toLocaleString(undefined, { maximumFractionDigits: 4 })
}

function aggregateAuxPlanRows(rows: PlanItemRow[]): AuxPlanSummaryRow[] {
    const map = new Map<string, AuxPlanSummaryRow>()
    for (const row of rows) {
        const componentId = row.component_id || row.model_id || row.item_key
        if (componentId <= 0) {
            continue
        }
        const key = `${row.line_id}-${componentId}`
        let item = map.get(key)
        if (!item) {
            item = {
                line_id: row.line_id,
                line_name: row.line_name ?? String(row.line_id),
                component_id: componentId,
                label: row.label,
                factory_code: row.artikul_raqami || row.label || "",
                odoo_code: row.odoo_code ?? "",
                planned_qty: 0,
                actual_qty: 0,
                remaining_qty: 0,
                completion_pct: 0,
                allow_overplan: row.allow_overplan,
            }
            map.set(key, item)
        }
        item.planned_qty += Number(row.planned_qty) || 0
        item.actual_qty += Number(row.actual_qty) || 0
        if (!item.label && row.label) {
            item.label = row.label
        }
        if (!item.odoo_code && row.odoo_code) {
            item.odoo_code = row.odoo_code
        }
        if (!item.factory_code && (row.artikul_raqami || row.label)) {
            item.factory_code = row.artikul_raqami || row.label
        }
        if (row.allow_overplan) {
            item.allow_overplan = true
        }
    }
    return Array.from(map.values())
        .map((item) => {
            item.remaining_qty = item.planned_qty - item.actual_qty
            item.completion_pct =
                item.planned_qty > 0 ? Math.round((item.actual_qty * 100) / item.planned_qty) : 0
            return item
        })
        .sort((a, b) => a.line_name.localeCompare(b.line_name) || a.label.localeCompare(b.label))
}

function aggregateMainPlanRows(rows: PlanItemRow[], modelIds: number[]): MainPlanSummaryRow[] {
    const modelFilter = modelIds.length > 0 ? new Set(modelIds) : null
    const map = new Map<string, MainPlanSummaryRow>()
    for (const row of rows) {
        if (row.model_id <= 0) {
            continue
        }
        if (modelFilter && !modelFilter.has(row.model_id)) {
            continue
        }
        const key = `${row.line_id}-${row.model_id}`
        let item = map.get(key)
        if (!item) {
            item = {
                line_id: row.line_id,
                line_name: row.line_name ?? String(row.line_id),
                model_id: row.model_id,
                label: row.label,
                odoo_code: row.odoo_code ?? "",
                planned_qty: 0,
                actual_qty: 0,
            }
            map.set(key, item)
        }
        item.planned_qty += Number(row.planned_qty) || 0
        item.actual_qty += Number(row.actual_qty) || 0
        if (!item.odoo_code && row.odoo_code) {
            item.odoo_code = row.odoo_code
        }
    }
    return Array.from(map.values()).sort(
        (a, b) =>
            a.line_name.localeCompare(b.line_name) ||
            a.label.localeCompare(b.label),
    )
}

function mergeMainReportRows(
    production: SummaryRow[],
    plan: MainPlanSummaryRow[],
    models: Model[],
): MainMergedRow[] {
    const modelById = new Map(models.map((model) => [model.id, model]))
    const map = new Map<string, MainMergedRow>()

    for (const row of production) {
        const lineId = Number(row.line_id) || 0
        const modelId = Number(row.model_id) || 0
        const key = `${lineId}-${modelId}`
        map.set(key, {
            line_id: lineId,
            line_name: row.line_name,
            model_id: modelId,
            model_name: row.model_name,
            seriya_raqami: row.seriya_raqami,
            odoo_code: row.odoo_code,
            count: row.count,
            planned_qty: 0,
            actual_qty: 0,
        })
    }

    for (const row of plan) {
        const key = `${row.line_id}-${row.model_id}`
        const existing = map.get(key)
        if (existing) {
            existing.planned_qty = row.planned_qty
            existing.actual_qty = row.actual_qty
            if (!existing.odoo_code && row.odoo_code) {
                existing.odoo_code = row.odoo_code
            }
            continue
        }
        const model = modelById.get(row.model_id)
        map.set(key, {
            line_id: row.line_id,
            line_name: row.line_name,
            model_id: row.model_id,
            model_name: model?.modeli ?? row.label,
            seriya_raqami: model?.seriya_raqami ?? "",
            odoo_code: row.odoo_code || model?.odoo_code || "",
            count: 0,
            planned_qty: row.planned_qty,
            actual_qty: row.actual_qty,
        })
    }

    return Array.from(map.values()).sort(
        (a, b) =>
            a.line_name.localeCompare(b.line_name) ||
            a.model_name.localeCompare(b.model_name),
    )
}

function mergeAuxReportRows(balance: AuxSummaryRow[], plan: AuxPlanSummaryRow[]): AuxMergedRow[] {
    const map = new Map<string, AuxMergedRow>()

    for (const row of balance) {
        const key = `${row.line_id}-${row.component_id}`
        map.set(key, {
            line_id: row.line_id,
            line_name: row.line_name,
            component_id: row.component_id,
            factory_code: row.factory_code,
            component_name: row.component_name,
            odoo_code: row.odoo_code,
            received: row.received,
            expended: row.expended,
            balance_end: row.balance_end,
            planned_qty: 0,
            actual_qty: 0,
        })
    }

    for (const row of plan) {
        const key = `${row.line_id}-${row.component_id}`
        const existing = map.get(key)
        if (existing) {
            existing.planned_qty = row.planned_qty
            existing.actual_qty = row.actual_qty
            if (!existing.component_name) {
                existing.component_name = row.label
            }
            if (!existing.odoo_code && row.odoo_code) {
                existing.odoo_code = row.odoo_code
            }
            if (!existing.factory_code && row.factory_code) {
                existing.factory_code = row.factory_code
            }
            continue
        }
        map.set(key, {
            line_id: row.line_id,
            line_name: row.line_name,
            component_id: row.component_id,
            factory_code: row.factory_code || "",
            component_name: row.label,
            odoo_code: row.odoo_code ?? "",
            received: 0,
            expended: 0,
            balance_end: 0,
            planned_qty: row.planned_qty,
            actual_qty: row.actual_qty,
        })
    }

    return Array.from(map.values()).sort(
        (a, b) =>
            a.line_name.localeCompare(b.line_name) ||
            a.factory_code.localeCompare(b.factory_code) ||
            a.component_name.localeCompare(b.component_name),
    )
}

function toggleId(list: number[], id: number): number[] {
    return list.includes(id) ? list.filter((item) => item !== id) : [...list, id]
}

function selectionLabel(items: string[], emptyLabel: string) {
    if (items.length === 0) {
        return emptyLabel
    }
    if (items.length <= 2) {
        return items.join(", ")
    }
    return `${items.length} ta tanlangan`
}

function applyDatePreset(type: "today" | "yesterday" | "month" | "year") {
    const now = new Date()
    if (type === "today") {
        return {
            date1: toDateTimeLocalValue(startOfDayLocal(now)),
            date2: toDateTimeLocalValue(endOfDayLocal(now)),
        }
    }
    if (type === "yesterday") {
        const yesterday = new Date(now)
        yesterday.setDate(now.getDate() - 1)
        return {
            date1: toDateTimeLocalValue(startOfDayLocal(yesterday)),
            date2: toDateTimeLocalValue(endOfDayLocal(yesterday)),
        }
    }
    if (type === "year") {
        const start = new Date(now.getFullYear(), 0, 1, 0, 0, 0, 0)
        return {
            date1: toDateTimeLocalValue(start),
            date2: toDateTimeLocalValue(endOfDayLocal(now)),
        }
    }
    const start = new Date(now.getFullYear(), now.getMonth(), 1, 0, 0, 0, 0)
    return {
        date1: toDateTimeLocalValue(start),
        date2: toDateTimeLocalValue(endOfDayLocal(now)),
    }
}

type LineDropdownProps = {
    lines: Line[]
    selectedLineIds: number[]
    setSelectedLineIds: React.Dispatch<React.SetStateAction<number[]>>
    lineSearch: string
    setLineSearch: React.Dispatch<React.SetStateAction<string>>
    lineMenuOpen: boolean
    setLineMenuOpen: React.Dispatch<React.SetStateAction<boolean>>
    emptyLabel?: string
}

function LineDropdown({
    lines,
    selectedLineIds,
    setSelectedLineIds,
    lineSearch,
    setLineSearch,
    lineMenuOpen,
    setLineMenuOpen,
    emptyLabel = "Barcha liniyalar",
}: LineDropdownProps) {
    const filteredLines = useMemo(() => {
        const query = normalizeSearchValue(lineSearch)
        if (!query) {
            return lines
        }
        return lines.filter((line) => normalizeSearchValue(line.name).includes(query))
    }, [lines, lineSearch])

    const selectedNames = lines
        .filter((line) => selectedLineIds.includes(line.line_id))
        .map((line) => line.name)

    return (
        <DropdownMenu
            open={lineMenuOpen}
            onOpenChange={(open) => {
                setLineMenuOpen(open)
                if (!open) {
                    setLineSearch("")
                }
            }}
        >
            <DropdownMenuTrigger asChild>
                <Button variant="outline" className="h-11 w-full justify-between gap-2 rounded-xl">
                    <span className="truncate">{selectionLabel(selectedNames, emptyLabel)}</span>
                    <ChevronDown className="size-4 shrink-0 opacity-50" />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent className="w-72 rounded-xl p-0" align="start">
                <div className="border-b p-2" onKeyDown={(event) => event.stopPropagation()}>
                    <div className="relative">
                        <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
                        <Input
                            value={lineSearch}
                            onChange={(event) => setLineSearch(event.target.value)}
                            placeholder="Liniya qidirish..."
                            className="h-9 pl-8"
                            autoFocus
                        />
                    </div>
                </div>
                <DropdownMenuGroup className="max-h-64 overflow-y-auto p-1">
                    <DropdownMenuItem
                        onSelect={(event) => event.preventDefault()}
                        onClick={() => setSelectedLineIds([])}
                    >
                        {emptyLabel}
                    </DropdownMenuItem>
                    {filteredLines.length === 0 ? (
                        <div className="px-3 py-4 text-center text-sm text-muted-foreground">
                            Liniya topilmadi
                        </div>
                    ) : (
                        filteredLines.map((line) => {
                            const checked = selectedLineIds.includes(line.line_id)
                            return (
                                <DropdownMenuItem
                                    key={line.line_id}
                                    className={cn(checked && "bg-primary/10")}
                                    onSelect={(event) => event.preventDefault()}
                                    onClick={() =>
                                        setSelectedLineIds((prev) => toggleId(prev, line.line_id))
                                    }
                                >
                                    <span className="flex items-center gap-2">
                                        <input type="checkbox" readOnly checked={checked} className="size-4 rounded border" />
                                        {line.name}
                                    </span>
                                </DropdownMenuItem>
                            )
                        })
                    )}
                </DropdownMenuGroup>
            </DropdownMenuContent>
        </DropdownMenu>
    )
}

type DateFilterBlockProps = {
    date1: string
    date2: string
    onDate1Change: (value: string) => void
    onDate2Change: (value: string) => void
    onPreset: (type: "today" | "yesterday" | "month" | "year") => void
}

function DateFilterBlock({ date1, date2, onDate1Change, onDate2Change, onPreset }: DateFilterBlockProps) {
    return (
        <div className={filterCardClassName}>
            <div className="grid gap-2 sm:grid-cols-2">
                <div className="space-y-1">
                    <Label className="text-xs text-muted-foreground">Dan</Label>
                    <Input
                        type="datetime-local"
                        value={date1}
                        onChange={(e) => onDate1Change(e.target.value)}
                        className="h-11 rounded-xl"
                    />
                </div>
                <div className="space-y-1">
                    <Label className="text-xs text-muted-foreground">Gacha</Label>
                    <Input
                        type="datetime-local"
                        value={date2}
                        onChange={(e) => onDate2Change(e.target.value)}
                        className="h-11 rounded-xl"
                    />
                </div>
            </div>
            <div className="flex h-11 items-center gap-2 overflow-x-auto">
                {[
                    ["today", "Bugun"],
                    ["yesterday", "Kecha"],
                    ["month", "Oy boshidan"],
                    ["year", "Yil boshidan"],
                ].map(([key, label]) => (
                    <Button
                        key={key}
                        variant="secondary"
                        size="sm"
                        className="shrink-0 rounded-full"
                        onClick={() => onPreset(key as "today" | "yesterday" | "month" | "year")}
                    >
                        {label}
                    </Button>
                ))}
            </div>
        </div>
    )
}

export default function LinesReport() {
    const [date1, setDate1] = useState(() => toDateTimeLocalValue(startOfDayLocal(new Date())))
    const [date2, setDate2] = useState(() => toDateTimeLocalValue(endOfDayLocal(new Date())))
    const [lines, setLines] = useState<Line[]>([])
    const [selectedLineIds, setSelectedLineIds] = useState<number[]>([])
    const [models, setModels] = useState<Model[]>([])
    const [selectedModelIds, setSelectedModelIds] = useState<number[]>([])
    const [count, setCount] = useState(0)
    const [shortInfo, setShortInfo] = useState<SummaryRow[]>([])
    const [mainPlanRows, setMainPlanRows] = useState<MainPlanSummaryRow[]>([])
    const [reportLoaded, setReportLoaded] = useState(false)
    const [loading, setLoading] = useState(false)
    const [exporting, setExporting] = useState(false)
    const [lineSearch, setLineSearch] = useState("")
    const [modelSearch, setModelSearch] = useState("")
    const [lineMenuOpen, setLineMenuOpen] = useState(false)
    const [modelMenuOpen, setModelMenuOpen] = useState(false)

    const [auxDate1, setAuxDate1] = useState(() => toDateTimeLocalValue(startOfDayLocal(new Date())))
    const [auxDate2, setAuxDate2] = useState(() => toDateTimeLocalValue(endOfDayLocal(new Date())))
    const [auxLines, setAuxLines] = useState<Line[]>([])
    const [auxSelectedLineIds, setAuxSelectedLineIds] = useState<number[]>([])
    const [auxCount, setAuxCount] = useState(0)
    const [auxShortInfo, setAuxShortInfo] = useState<AuxSummaryRow[]>([])
    const [auxPlanRows, setAuxPlanRows] = useState<AuxPlanSummaryRow[]>([])
    const [auxReportLoaded, setAuxReportLoaded] = useState(false)
    const [auxLoading, setAuxLoading] = useState(false)
    const [auxExporting, setAuxExporting] = useState(false)
    const [auxLineSearch, setAuxLineSearch] = useState("")
    const [auxLineMenuOpen, setAuxLineMenuOpen] = useState(false)

    async function linesGetAll() {
        const result = await Backend_Request<Line[]>({}, "/api/lines/all")
        if (result.result === "ok") {
            const allLines = result.data ?? []
            setLines(allLines.filter((line) => MAIN_REPORT_LINE_IDS.has(line.line_id)))
            setAuxLines(allLines.filter((line) => AUX_REPORT_LINE_IDS.has(line.line_id)))
            setSelectedLineIds((prev) => prev.filter((id) => MAIN_REPORT_LINE_IDS.has(id)))
            setAuxSelectedLineIds((prev) => prev.filter((id) => AUX_REPORT_LINE_IDS.has(id)))
        } else {
            ShowErrorToast(result.error || "Xatolik")
        }
    }

    async function modelsGetAll() {
        const result = await Backend_Request<Model[]>({}, "/api/tech/models/all")
        if (result.result === "ok") {
            setModels(result.data ?? [])
        } else {
            ShowErrorToast(result.error || "Xatolik")
        }
    }

    const filteredModels = useMemo(() => {
        const query = normalizeSearchValue(modelSearch)
        if (!query) {
            return models
        }
        return models.filter((model) =>
            [model.modeli, model.qisqa_nomi, model.rangi, model.gs1_ean13, model.seriya_raqami]
                .map(normalizeSearchValue)
                .join(" ")
                .includes(query),
        )
    }, [models, modelSearch])

    function dropDownModels() {
        const selectedNames = models
            .filter((model) => selectedModelIds.includes(model.id))
            .map((model) => modelDropdownLabel(model))

        return (
            <DropdownMenu
                open={modelMenuOpen}
                onOpenChange={(open) => {
                    setModelMenuOpen(open)
                    if (!open) {
                        setModelSearch("")
                    }
                }}
            >
                <DropdownMenuTrigger asChild>
                    <Button variant="outline" className="h-11 w-full justify-between gap-2 rounded-xl">
                        <span className="truncate">
                            {selectionLabel(selectedNames, "Barcha modellar")}
                        </span>
                        <ChevronDown className="size-4 shrink-0 opacity-50" />
                    </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent
                    className="w-[min(560px,calc(100vw-2rem))] rounded-xl p-0"
                    align="start"
                >
                    <div className="border-b p-2" onKeyDown={(event) => event.stopPropagation()}>
                        <div className="relative">
                            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
                            <Input
                                value={modelSearch}
                                onChange={(event) => setModelSearch(event.target.value)}
                                placeholder="Model qidirish..."
                                className="h-9 pl-8"
                                autoFocus
                            />
                        </div>
                    </div>
                    <DropdownMenuGroup className="max-h-80 overflow-y-auto p-1">
                        <DropdownMenuItem
                            onSelect={(event) => event.preventDefault()}
                            onClick={() => setSelectedModelIds([])}
                        >
                            Barcha modellar
                        </DropdownMenuItem>
                        {filteredModels.length === 0 ? (
                            <div className="px-3 py-4 text-center text-sm text-muted-foreground">
                                Model topilmadi
                            </div>
                        ) : (
                            filteredModels.map((model) => {
                                const checked = selectedModelIds.includes(model.id)
                                return (
                                    <DropdownMenuItem
                                        key={model.id}
                                        className={cn(checked && "bg-primary/10")}
                                        onSelect={(event) => event.preventDefault()}
                                        onClick={() =>
                                            setSelectedModelIds((prev) => toggleId(prev, model.id))
                                        }
                                    >
                                        <span className="flex items-center gap-2">
                                            <input type="checkbox" readOnly checked={checked} className="size-4 rounded border" />
                                            {modelDropdownLabel(model)}
                                        </span>
                                    </DropdownMenuItem>
                                )
                            })
                        )}
                    </DropdownMenuGroup>
                </DropdownMenuContent>
            </DropdownMenu>
        )
    }

    function reportPayload(exportAll = false) {
        return {
            date1,
            date2,
            line_ids: selectedLineIds,
            model_ids: selectedModelIds,
            page: 1,
            page_size: 1,
            export_all: exportAll,
        }
    }

    function auxReportPayload(exportAll = false) {
        return {
            date1: auxDate1,
            date2: auxDate2,
            line_ids: auxSelectedLineIds,
            page: 1,
            page_size: 1,
            export_all: exportAll,
        }
    }

    async function getReport() {
        setLoading(true)
        const lineIds =
            selectedLineIds.length > 0 ? selectedLineIds : [...PLAN_PRODUCT_LINE_IDS]
        const [result, planResult] = await Promise.all([
            Backend_Request<ReportResponse>(reportPayload(false), "/api/lines/report"),
            Backend_Request<PlanItemRow[]>(
                { date_from: toPlanDateValue(date1), date_to: toPlanDateValue(date2), line_ids: lineIds },
                "/api/production/plan/report",
            ),
        ])
        setLoading(false)
        if (result.result === "ok" && result.data) {
            const d = result.data
            setCount(d.count)
            setShortInfo(
                (d.short_table ?? []).map((row) => ({
                    line_id: Number(row.line_id) || 0,
                    line_name: row.line_name ?? "",
                    model_id: Number(row.model_id) || 0,
                    model_name: row.model_name ?? "",
                    seriya_raqami: row.seriya_raqami ?? "",
                    odoo_code: row.odoo_code ?? "",
                    count: Number(row.count) || 0,
                })),
            )
            if (planResult.result === "ok") {
                setMainPlanRows(aggregateMainPlanRows(planResult.data ?? [], selectedModelIds))
            } else {
                setMainPlanRows([])
                ShowErrorToast(planResult.error || "Reja hisoboti yuklanmadi")
            }
            setReportLoaded(true)
        } else {
            ShowErrorToast(result.error || "Xatolik")
            setReportLoaded(false)
            setMainPlanRows([])
        }
    }

    async function getAuxReport() {
        setAuxLoading(true)
        const lineIds =
            auxSelectedLineIds.length > 0 ? auxSelectedLineIds : [...PLAN_AUX_LINE_IDS]
        const [result, planResult] = await Promise.all([
            Backend_Request<AuxReportResponse>(auxReportPayload(false), "/api/lines/auxiliary/report"),
            Backend_Request<PlanItemRow[]>(
                { date_from: toPlanDateValue(auxDate1), date_to: toPlanDateValue(auxDate2), line_ids: lineIds },
                "/api/production/plan/report",
            ),
        ])
        setAuxLoading(false)
        if (result.result === "ok" && result.data) {
            const d = result.data
            setAuxCount(d.count)
            setAuxShortInfo(
                (d.short_table ?? []).map((row) => ({
                    line_id: Number(row.line_id) || 0,
                    line_name: row.line_name ?? "",
                    component_id: Number(row.component_id) || 0,
                    factory_code: row.factory_code ?? "",
                    component_name: row.component_name ?? "",
                    odoo_code: row.odoo_code ?? "",
                    received: Number(row.received) || 0,
                    expended: Number(row.expended) || 0,
                    balance_end: Number(row.balance_end) || 0,
                })),
            )
            if (planResult.result === "ok") {
                setAuxPlanRows(aggregateAuxPlanRows(planResult.data ?? []))
            } else {
                setAuxPlanRows([])
                ShowErrorToast(planResult.error || "Reja hisoboti yuklanmadi")
            }
            setAuxReportLoaded(true)
        } else {
            ShowErrorToast(result.error || "Xatolik")
            setAuxReportLoaded(false)
            setAuxPlanRows([])
        }
    }

    async function loadAllForExport() {
        const result = await Backend_Request<ReportResponse>(reportPayload(true), "/api/lines/report")
        if (result.result === "ok" && result.data) {
            return result.data.detailed
        }
        ShowErrorToast(result.error || "Xatolik")
        return null
    }

    async function loadAuxAllForExport() {
        const result = await Backend_Request<AuxReportResponse>(
            auxReportPayload(true),
            "/api/lines/auxiliary/report",
        )
        if (result.result === "ok" && result.data) {
            return result.data.detailed
        }
        ShowErrorToast(result.error || "Xatolik")
        return null
    }

    const columnHelper2 = createColumnHelper<MainMergedRow>()
    const auxColumnHelper = createColumnHelper<AuxMergedRow>()

    const columns2 = [
        columnHelper2.accessor("line_name", { header: "Liniya Nomi" }),
        columnHelper2.accessor("model_name", { header: "Modeli" }),
        columnHelper2.accessor("seriya_raqami", { header: "Seriya raqami" }),
        columnHelper2.accessor("odoo_code", { header: "ODOO code" }),
        columnHelper2.accessor("count", {
            header: "Soni",
            cell: ({ getValue }) => <span className="tabular-nums">{getValue()}</span>,
        }),
        columnHelper2.accessor("planned_qty", {
            header: "Reja",
            cell: ({ getValue }) => <span className="tabular-nums">{getValue()}</span>,
        }),
        columnHelper2.accessor("actual_qty", {
            header: "Fakt",
            cell: ({ getValue }) => <span className="tabular-nums">{getValue()}</span>,
        }),
    ]

    const auxColumns = [
        auxColumnHelper.accessor("line_name", { header: "Liniya Nomi" }),
        auxColumnHelper.accessor("factory_code", { header: "Korxona kodi" }),
        auxColumnHelper.accessor("component_name", { header: "Komponent" }),
        auxColumnHelper.accessor("odoo_code", { header: "ODOO code" }),
        auxColumnHelper.accessor("received", {
            header: "Kirim",
            cell: ({ getValue }) => (
                <span className="tabular-nums">{formatBalanceQty(getValue())}</span>
            ),
        }),
        auxColumnHelper.accessor("expended", {
            header: "Chiqim",
            cell: ({ getValue }) => (
                <span className="tabular-nums">{formatBalanceQty(getValue())}</span>
            ),
        }),
        auxColumnHelper.accessor("balance_end", {
            header: "Qoldiq",
            cell: ({ getValue }) => (
                <span className="font-medium tabular-nums">{formatBalanceQty(getValue())}</span>
            ),
        }),
        auxColumnHelper.accessor("planned_qty", {
            header: "Reja",
            cell: ({ getValue }) => <span className="tabular-nums">{getValue()}</span>,
        }),
        auxColumnHelper.accessor("actual_qty", {
            header: "Fakt",
            cell: ({ getValue }) => <span className="tabular-nums">{getValue()}</span>,
        }),
    ]

    async function exportToExcel(fileName = "asosiy_liniyalar.xlsx") {
        setExporting(true)
        const rows = await loadAllForExport()
        setExporting(false)
        if (!rows || rows.length === 0) {
            ShowErrorToast("Excel uchun ma'lumot topilmadi")
            return
        }
        const XLSX = await import("xlsx")
        const exportData = rows.map((item: any) => ({
            Serial: item.serial,
            Modeli: item.model,
            ModelNomi: item.qisqa_nomi,
            LiniyaNomi: item.line_name,
            OdooCode: item.odoo_code,
            Vaqt: item.time,
            GS1: item.gs1,
            GsCode: item.gs_code,
        }))
        const worksheet = XLSX.utils.json_to_sheet(exportData)
        const workbook = XLSX.utils.book_new()
        XLSX.utils.book_append_sheet(workbook, worksheet, "Report")
        const excelBuffer = XLSX.write(workbook, { bookType: "xlsx", type: "array" })
        saveAs(
            new Blob([excelBuffer], {
                type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
            }),
            fileName,
        )
        ShowOKToast("XLSX файл сақланди")
    }

    async function exportAuxToExcel(fileName = "yordamchi_liniyalar.xlsx") {
        setAuxExporting(true)
        const rows = await loadAuxAllForExport()
        setAuxExporting(false)
        if (!rows || rows.length === 0) {
            ShowErrorToast("Excel uchun ma'lumot topilmadi")
            return
        }
        const XLSX = await import("xlsx")
        const exportData = rows.map((item: any) => ({
            Vaqt: item.time,
            LiniyaNomi: item.line_name,
            KorxonaKodi: item.factory_code,
            Komponent: item.component_name,
            OdooCode: item.odoo_code,
            Ozgarish: item.quantity_change,
            BalansKeyin: item.quantity_after,
            Manba: item.source,
            Foydalanuvchi: item.user_name,
            Izoh: item.comment,
        }))
        const worksheet = XLSX.utils.json_to_sheet(exportData)
        const workbook = XLSX.utils.book_new()
        XLSX.utils.book_append_sheet(workbook, worksheet, "Report")
        const excelBuffer = XLSX.write(workbook, { bookType: "xlsx", type: "array" })
        saveAs(
            new Blob([excelBuffer], {
                type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
            }),
            fileName,
        )
        ShowOKToast("XLSX файл сақланди")
    }

    function resetFilters() {
        const today = toDateInputValue(new Date())
        setDate1(today)
        setDate2(today)
        setSelectedLineIds([])
        setSelectedModelIds([])
        setLineSearch("")
        setModelSearch("")
        setReportLoaded(false)
        setCount(0)
        setShortInfo([])
        setMainPlanRows([])
    }

    function resetAuxFilters() {
        const today = toDateInputValue(new Date())
        setAuxDate1(today)
        setAuxDate2(today)
        setAuxSelectedLineIds([])
        setAuxLineSearch("")
        setAuxReportLoaded(false)
        setAuxCount(0)
        setAuxShortInfo([])
        setAuxPlanRows([])
    }

    const filterSummary = useMemo(() => {
        const lineNames = lines
            .filter((line) => selectedLineIds.includes(line.line_id))
            .map((line) => line.name)
        const modelNames = models
            .filter((model) => selectedModelIds.includes(model.id))
            .map((model) => model.modeli)
        const line = selectionLabel(lineNames, "Barcha liniyalar")
        const model = selectionLabel(modelNames, "Barcha modellar")
        return `${date1} → ${date2} • ${line} • ${model}`
    }, [date1, date2, lines, models, selectedLineIds, selectedModelIds])

    const auxFilterSummary = useMemo(() => {
        const lineNames = auxLines
            .filter((line) => auxSelectedLineIds.includes(line.line_id))
            .map((line) => line.name)
        const line = selectionLabel(lineNames, "Barcha liniyalar")
        return `${auxDate1} → ${auxDate2} • ${line}`
    }, [auxDate1, auxDate2, auxLines, auxSelectedLineIds])

    const mainMergedRows = useMemo(
        () => mergeMainReportRows(shortInfo, mainPlanRows, models),
        [shortInfo, mainPlanRows, models],
    )

    const mainPlanTotals = useMemo(
        () =>
            mainMergedRows.reduce(
                (acc, row) => ({
                    planned: acc.planned + row.planned_qty,
                    actual: acc.actual + row.actual_qty,
                }),
                { planned: 0, actual: 0 },
            ),
        [mainMergedRows],
    )

    const table2 = useReactTable({
        data: mainMergedRows,
        columns: columns2,
        getCoreRowModel: getCoreRowModel(),
    })

    const auxMergedRows = useMemo(
        () => mergeAuxReportRows(auxShortInfo, auxPlanRows),
        [auxShortInfo, auxPlanRows],
    )

    const auxPlanTotals = useMemo(
        () =>
            auxMergedRows.reduce(
                (acc, row) => ({
                    planned: acc.planned + row.planned_qty,
                    actual: acc.actual + row.actual_qty,
                }),
                { planned: 0, actual: 0 },
            ),
        [auxMergedRows],
    )

    const auxTable = useReactTable({
        data: auxMergedRows,
        columns: auxColumns,
        getCoreRowModel: getCoreRowModel(),
    })

    function renderTable(
        table:
            | ReturnType<typeof useReactTable<MainMergedRow>>
            | ReturnType<typeof useReactTable<AuxMergedRow>>,
    ) {
        return (
            <Table>
                <TableHeader className="bg-muted/70">
                    {table.getHeaderGroups().map((hg) => (
                        <TableRow key={hg.id}>
                            {hg.headers.map((header) => (
                                <TableHead
                                    key={header.id}
                                    onClick={header.column.getToggleSortingHandler()}
                                    className="h-11 cursor-pointer select-none font-semibold"
                                >
                                    {flexRender(header.column.columnDef.header, header.getContext())}
                                </TableHead>
                            ))}
                        </TableRow>
                    ))}
                </TableHeader>
                <TableBody>
                    {table.getRowModel().rows.length === 0 ? (
                        <TableRow>
                            <TableCell
                                colSpan={table.getAllColumns().length}
                                className="h-24 text-center text-muted-foreground"
                            >
                                Ma'lumot topilmadi
                            </TableCell>
                        </TableRow>
                    ) : (
                        table.getRowModel().rows.map((row) => (
                            <TableRow key={row.id}>
                                {row.getVisibleCells().map((cell) => (
                                    <TableCell key={cell.id} className="py-3">
                                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                    </TableCell>
                                ))}
                            </TableRow>
                        ))
                    )}
                </TableBody>
            </Table>
        )
    }

    useEffect(() => {
        linesGetAll()
        modelsGetAll()
    }, [])

    return (
        <PageContainer
            title="Ishlab chiqarish hisoboti"
            description="Asosiy va yordamchi liniyalar bo‘yicha qisqa hisobot"
            scrollable
        >
            <Panel
                title="Asosiy Liniyalar"
                description="T1, T2, T3, Ichki — model bo‘yicha qisqa hisobot, reja va fakt, batafsil ro‘yxat XLSX da"
                noPadding
                action={
                    reportLoaded ? (
                        <Button
                            onClick={() => exportToExcel()}
                            disabled={exporting}
                            variant="outline"
                            className="gap-2 rounded-xl"
                        >
                            <FileDown className="size-4" />
                            {exporting ? "Saqlanmoqda..." : "XLSX"}
                        </Button>
                    ) : undefined
                }
            >
                <div className="border-b p-4">
                    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                        <DateFilterBlock
                            date1={date1}
                            date2={date2}
                            onDate1Change={setDate1}
                            onDate2Change={setDate2}
                            onPreset={(type) => {
                                const next = applyDatePreset(type)
                                setDate1(next.date1)
                                setDate2(next.date2)
                            }}
                        />
                        <div className={filterCardClassName}>
                            <div className="flex h-11 items-center text-sm font-medium text-muted-foreground">
                                Liniya
                            </div>
                            <div className="h-11">
                                <LineDropdown
                                    lines={lines}
                                    selectedLineIds={selectedLineIds}
                                    setSelectedLineIds={setSelectedLineIds}
                                    lineSearch={lineSearch}
                                    setLineSearch={setLineSearch}
                                    lineMenuOpen={lineMenuOpen}
                                    setLineMenuOpen={setLineMenuOpen}
                                />
                            </div>
                        </div>
                        <div className={filterCardClassName}>
                            <div className="flex h-11 items-center text-sm font-medium text-muted-foreground">
                                Model
                            </div>
                            <div className="h-11">{dropDownModels()}</div>
                        </div>
                        <div className={filterCardClassName}>
                            <Button onClick={resetFilters} variant="outline" className="h-11 rounded-xl px-5">
                                <RotateCcw className="size-4" />
                                Tozalash
                            </Button>
                            <Button onClick={() => getReport()} disabled={loading} className="h-11 rounded-xl px-5">
                                <Search className="size-4" />
                                {loading ? "Yuklanmoqda..." : "Hisobot"}
                            </Button>
                        </div>
                    </div>
                </div>

                {reportLoaded && (
                    <>
                        <div className="grid gap-3 border-b px-4 py-3 md:grid-cols-[1fr_auto_auto_auto]">
                            <div>
                                <div className="text-xs uppercase tracking-wide text-muted-foreground">
                                    Tanlangan filtr
                                </div>
                                <div className="mt-1 text-sm font-medium">{filterSummary}</div>
                            </div>
                            <div className="rounded-2xl border border-primary/20 bg-primary/10 px-4 py-3 text-primary">
                                <div className="text-xs uppercase tracking-wide">Jami</div>
                                <div className="text-2xl font-bold tabular-nums">{count}</div>
                            </div>
                            <div className="rounded-2xl border border-border/60 bg-muted/40 px-4 py-3">
                                <div className="text-xs uppercase tracking-wide text-muted-foreground">Reja</div>
                                <div className="text-2xl font-bold tabular-nums">{mainPlanTotals.planned}</div>
                            </div>
                            <div className="rounded-2xl border border-border/60 bg-muted/40 px-4 py-3">
                                <div className="text-xs uppercase tracking-wide text-muted-foreground">Fakt</div>
                                <div className="text-2xl font-bold tabular-nums">{mainPlanTotals.actual}</div>
                            </div>
                        </div>
                        <div className="overflow-hidden rounded-b-xl">{renderTable(table2)}</div>
                    </>
                )}
            </Panel>

            <Panel
                title="Yordamchi liniyalar"
                description="Komponent balansi va ishlab chiqarish rejasi"
                noPadding
                action={
                    auxReportLoaded ? (
                        <Button
                            onClick={() => exportAuxToExcel()}
                            disabled={auxExporting}
                            variant="outline"
                            className="gap-2 rounded-xl"
                        >
                            <FileDown className="size-4" />
                            {auxExporting ? "Saqlanmoqda..." : "XLSX"}
                        </Button>
                    ) : undefined
                }
            >
                <div className="border-b p-4">
                    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                        <DateFilterBlock
                            date1={auxDate1}
                            date2={auxDate2}
                            onDate1Change={setAuxDate1}
                            onDate2Change={setAuxDate2}
                            onPreset={(type) => {
                                const next = applyDatePreset(type)
                                setAuxDate1(next.date1)
                                setAuxDate2(next.date2)
                            }}
                        />
                        <div className={filterCardClassName}>
                            <div className="flex h-11 items-center text-sm font-medium text-muted-foreground">
                                Liniya
                            </div>
                            <div className="h-11">
                                <LineDropdown
                                    lines={auxLines}
                                    selectedLineIds={auxSelectedLineIds}
                                    setSelectedLineIds={setAuxSelectedLineIds}
                                    lineSearch={auxLineSearch}
                                    setLineSearch={setAuxLineSearch}
                                    lineMenuOpen={auxLineMenuOpen}
                                    setLineMenuOpen={setAuxLineMenuOpen}
                                />
                            </div>
                        </div>
                        <div className={filterCardClassName}>
                            <Button onClick={resetAuxFilters} variant="outline" className="h-11 rounded-xl px-5">
                                <RotateCcw className="size-4" />
                                Tozalash
                            </Button>
                            <Button
                                onClick={() => getAuxReport()}
                                disabled={auxLoading}
                                className="h-11 rounded-xl px-5"
                            >
                                <Search className="size-4" />
                                {auxLoading ? "Yuklanmoqda..." : "Hisobot"}
                            </Button>
                        </div>
                    </div>
                </div>

                {auxReportLoaded && (
                    <>
                        <div className="grid gap-3 border-b px-4 py-3 md:grid-cols-[1fr_auto_auto_auto]">
                            <div>
                                <div className="text-xs uppercase tracking-wide text-muted-foreground">
                                    Tanlangan filtr
                                </div>
                                <div className="mt-1 text-sm font-medium">{auxFilterSummary}</div>
                            </div>
                            <div className="rounded-2xl border border-primary/20 bg-primary/10 px-4 py-3 text-primary">
                                <div className="text-xs uppercase tracking-wide">Tranzaksiyalar</div>
                                <div className="text-2xl font-bold tabular-nums">{auxCount}</div>
                            </div>
                            <div className="rounded-2xl border border-border/60 bg-muted/40 px-4 py-3">
                                <div className="text-xs uppercase tracking-wide text-muted-foreground">Reja</div>
                                <div className="text-2xl font-bold tabular-nums">{auxPlanTotals.planned}</div>
                            </div>
                            <div className="rounded-2xl border border-border/60 bg-muted/40 px-4 py-3">
                                <div className="text-xs uppercase tracking-wide text-muted-foreground">Fakt</div>
                                <div className="text-2xl font-bold tabular-nums">{auxPlanTotals.actual}</div>
                            </div>
                        </div>
                        <div className="overflow-hidden rounded-b-xl">{renderTable(auxTable)}</div>
                    </>
                )}
            </Panel>
        </PageContainer>
    )
}
