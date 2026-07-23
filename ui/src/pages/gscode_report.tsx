import { useMemo, useState } from "react"
import {
    createColumnHelper,
    flexRender,
    getCoreRowModel,
    getSortedRowModel,
    type SortingState,
    useReactTable,
} from "@tanstack/react-table"
import { saveAs } from "file-saver"
import { CalendarDays, FileSpreadsheet, PackageCheck, RotateCcw, Search, Warehouse } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
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
import { cn } from "@/lib/utils"

type GsCodeReportRow = {
    yuklangan: number
    ishlatilgan: number
    qoldiq: number
    model_id: number
    modeli: string
    seriya_raqami: string
    artikul_raqami: string
    odoo_code: string
}

function toDateInputValue(date: Date) {
    return date.toISOString().slice(0, 10)
}

export default function GsCodesReport() {
    const [date1, setDate1] = useState(toDateInputValue(new Date()))
    const [date2, setDate2] = useState(toDateInputValue(new Date()))
    const [reportData, setReportData] = useState<GsCodeReportRow[]>([])
    const [sorting, setSorting] = useState<SortingState>([])
    const [loading, setLoading] = useState(false)
    const [exporting, setExporting] = useState(false)
    const [loaded, setLoaded] = useState(false)

    async function getReport() {
        const data = {
            date1,
            date2
        }

        setLoading(true)
        const result = await Backend_Request<GsCodeReportRow[]>(data, "/api/tech/gscode/report")
        setLoading(false)
        if (result.result === "ok") {
            setReportData(result.data ?? [])
            setLoaded(true)
        } else {
            ShowErrorToast(result.error || "Xatolik")
        }
    }

    function setPreset(type: "today" | "yesterday" | "month" | "year") {
        const now = new Date()
        if (type === "today") {
            const value = toDateInputValue(now)
            setDate1(value)
            setDate2(value)
            return
        }
        if (type === "yesterday") {
            const yesterday = new Date(now)
            yesterday.setDate(now.getDate() - 1)
            const value = toDateInputValue(yesterday)
            setDate1(value)
            setDate2(value)
            return
        }
        if (type === "year") {
            setDate1(toDateInputValue(new Date(now.getFullYear(), 0, 1)))
            setDate2(toDateInputValue(now))
            return
        }
        setDate1(toDateInputValue(new Date(now.getFullYear(), now.getMonth(), 1)))
        setDate2(toDateInputValue(now))
    }

    function resetFilters() {
        const today = toDateInputValue(new Date())
        setDate1(today)
        setDate2(today)
        setReportData([])
        setLoaded(false)
    }

    const totals = useMemo(() => {
        return reportData.reduce(
            (acc, row) => {
                acc.yuklangan += Number(row.yuklangan || 0)
                acc.ishlatilgan += Number(row.ishlatilgan || 0)
                acc.qoldiq += Number(row.qoldiq || 0)
                return acc
            },
            { yuklangan: 0, ishlatilgan: 0, qoldiq: 0 },
        )
    }, [reportData])

    const columnHelper = createColumnHelper<GsCodeReportRow>();

    const columns = [
        columnHelper.accessor("modeli", {
            header: "Modeli",
            cell: ({ row }) => (
                <div>
                    <div className="font-medium">{row.original.modeli}</div>
                    <div className="text-xs text-muted-foreground">ID: {row.original.model_id}</div>
                </div>
            ),
        }),
        columnHelper.accessor("seriya_raqami", { header: "Seriya raqami" }),
        columnHelper.accessor("artikul_raqami", { header: "Artikul raqami" }),
        columnHelper.accessor("odoo_code", { header: "ODOO code" }),
        columnHelper.accessor("yuklangan", {
            header: "Yuklangan",
            cell: ({ getValue }) => (
                <span className="font-semibold tabular-nums">{getValue()}</span>
            ),
        }),
        columnHelper.accessor("ishlatilgan", {
            header: "Ishlatilgan",
            cell: ({ getValue }) => (
                <span className="font-semibold tabular-nums text-primary">{getValue()}</span>
            ),
        }),
        columnHelper.accessor("qoldiq", {
            header: "Qoldiq",
            cell: ({ getValue }) => (
                <span className="font-semibold tabular-nums">{getValue()}</span>
            ),
        }),
    ];



    function reportTable() {
        return (
            <Table>
                <TableHeader className="sticky top-0 z-10 bg-muted/90 backdrop-blur">
                    {table.getHeaderGroups().map((hg) => (
                        <TableRow key={hg.id}>
                            {hg.headers.map((header) => (
                                <TableHead
                                    key={header.id}
                                    onClick={header.column.getToggleSortingHandler()}
                                    className="h-11 cursor-pointer select-none font-semibold"
                                >
                                    {flexRender(
                                        header.column.columnDef.header,
                                        header.getContext()
                                    )}
                                    {{
                                        asc: " ↑",
                                        desc: " ↓",
                                    }[header.column.getIsSorted() as string] ?? null}
                                </TableHead>
                            ))}
                        </TableRow>
                    ))}
                </TableHeader>
                <TableBody>
                    {table.getRowModel().rows.length === 0 ? (
                        <TableRow>
                            <TableCell colSpan={7} className="h-28 text-center text-muted-foreground">
                                {loaded ? "Ma'lumot topilmadi" : "Hisobotni yuklash uchun Tasdiqlash tugmasini bosing"}
                            </TableCell>
                        </TableRow>
                    ) : table.getRowModel().rows.map((row, index) => (
                        <TableRow
                            key={row.id}
                            className={cn(index % 2 === 0 ? "bg-transparent" : "bg-muted/20")}
                        >
                            {row.getVisibleCells().map((cell) => (
                                <TableCell key={cell.id} className="px-4 py-3">
                                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                </TableCell>
                            ))}
                        </TableRow>
                    ))}
                </TableBody>
            </Table>
        )
    }

    const table = useReactTable({
        data: reportData,
        columns,
        state: { sorting },
        onSortingChange: setSorting,
        getCoreRowModel: getCoreRowModel(),
        getSortedRowModel: getSortedRowModel(),
    });

    async function exportToExcel(fileName = `gscode-report-${date1}-${date2}.xlsx`) {
        const rows = table.getSortedRowModel().rows.map((row) => row.original)
        if (rows.length === 0) {
            ShowErrorToast("Excel uchun ma'lumot topilmadi")
            return
        }

        setExporting(true)
        const XLSX = await import("xlsx")
        const exportData = rows.map((row) => ({
            Modeli: row.modeli,
            "Seriya raqami": row.seriya_raqami,
            "Artikul raqami": row.artikul_raqami,
            "ODOO code": row.odoo_code,
            Yuklangan: row.yuklangan,
            Ishlatilgan: row.ishlatilgan,
            Qoldiq: row.qoldiq,
        }))
        const worksheet = XLSX.utils.json_to_sheet(exportData)
        const workbook = XLSX.utils.book_new()
        XLSX.utils.book_append_sheet(workbook, worksheet, "GS Code")

        const excelBuffer = XLSX.write(workbook, {
            bookType: "xlsx",
            type: "array",
        })
        const fileData = new Blob([excelBuffer], {
            type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        })
        saveAs(fileData, fileName)
        setExporting(false)
        ShowOKToast("XLSX файл сақланди")
    }

    return (
        <PageContainer
            title="GS Code hisobot"
            description="Yuklangan, ishlatilgan va qoldiq GS kodlar bo‘yicha hisobot"
            scrollable
        >
            <Panel title="Filtrlar" description="Sana oralig‘ini tanlang">
                <div className="grid gap-3 lg:grid-cols-[1fr_auto]">
                    <div className="rounded-2xl border border-border/60 bg-muted/20 p-3">
                        <div className="mb-2 flex items-center gap-2 text-sm font-medium">
                            <CalendarDays className="size-4 text-primary" />
                            Sana oralig‘i
                        </div>
                        <div className="grid gap-2 sm:grid-cols-2">
                            <Input
                                type="date"
                                value={date1}
                                onChange={(e) => setDate1(e.target.value)}
                                className="h-11 rounded-xl"
                            />
                            <Input
                                type="date"
                                value={date2}
                                onChange={(e) => setDate2(e.target.value)}
                                className="h-11 rounded-xl"
                            />
                        </div>
                        <div className="mt-2 flex flex-wrap gap-2">
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
                                    className="rounded-full"
                                    onClick={() =>
                                        setPreset(key as "today" | "yesterday" | "month" | "year")
                                    }
                                >
                                    {label}
                                </Button>
                            ))}
                        </div>
                    </div>

                    <div className="flex flex-col justify-end gap-2">
                        <Button onClick={getReport} disabled={loading} className="h-11 rounded-xl px-6">
                            <Search className="size-4" />
                            {loading ? "Yuklanmoqda..." : "Tasdiqlash"}
                        </Button>
                        <Button onClick={resetFilters} variant="outline" className="h-11 rounded-xl px-6">
                            <RotateCcw className="size-4" />
                            Tozalash
                        </Button>
                        <Button
                            onClick={() => exportToExcel()}
                            disabled={exporting || reportData.length === 0}
                            variant="outline"
                            className="h-11 rounded-xl px-6"
                        >
                            <FileSpreadsheet className="size-4" />
                            {exporting ? "Saqlanmoqda..." : "XLSX"}
                        </Button>
                    </div>
                </div>
            </Panel>

            <div className="grid gap-3 sm:grid-cols-3">
                <div className="rounded-2xl border border-border/60 bg-card/80 p-4">
                    <div className="flex items-center justify-between">
                        <div className="text-xs uppercase tracking-wide text-muted-foreground">Yuklangan</div>
                        <PackageCheck className="size-4 text-primary" />
                    </div>
                    <div className="mt-2 text-3xl font-bold tabular-nums">{totals.yuklangan}</div>
                </div>
                <div className="rounded-2xl border border-primary/20 bg-primary/10 p-4 text-primary">
                    <div className="flex items-center justify-between">
                        <div className="text-xs uppercase tracking-wide">Ishlatilgan</div>
                        <Search className="size-4" />
                    </div>
                    <div className="mt-2 text-3xl font-bold tabular-nums">{totals.ishlatilgan}</div>
                </div>
                <div className="rounded-2xl border border-border/60 bg-card/80 p-4">
                    <div className="flex items-center justify-between">
                        <div className="text-xs uppercase tracking-wide text-muted-foreground">Qoldiq</div>
                        <Warehouse className="size-4 text-muted-foreground" />
                    </div>
                    <div className="mt-2 text-3xl font-bold tabular-nums">{totals.qoldiq}</div>
                </div>
            </div>

            <Panel title="Model bo‘yicha GS Code hisobot" noPadding>
                <div className="overflow-x-auto rounded-b-xl">
                    {reportTable()}
                </div>
            </Panel>
        </PageContainer>
    )
}