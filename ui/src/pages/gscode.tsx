import { useCallback, useEffect, useMemo, useState } from "react"
import { FileSpreadsheet, Upload } from "lucide-react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"
import { Backend_Request } from "@/services/backend"
import { useDropzone } from "react-dropzone"
import { Button } from "@/components/ui/button"
import { PageSearchInput } from "@/components/layout/page-search-input"
import { cn } from "@/lib/utils"

import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuGroup,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

import {
    useReactTable,
    getCoreRowModel,
    getSortedRowModel,
    flexRender,
    createColumnHelper,
    type SortingState,
    getFilteredRowModel,
} from "@tanstack/react-table";

import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"

export default function GsCodePgae() {
    const navigate = useNavigate()
    const [gscodeInfo, setGsCodeInfo] = useState([])

    const [models, setModels] = useState([])
    const [selectedModel, setSelectedModel] = useState(null)
    const [selectedModelInfo, setSelectedModelInfo] = useState([])
    const [showInfo, setShowInfo] = useState(false)

    const [errorCodesString, setErrorCodesString] = useState(null)

    const [selectedModelName, setSelectedModelName] = useState("Modelni tanlang")

    function showOkToast(text: string) {
        toast(text, {
            // description: "Sunday, December 03, 2023 at 9:00 AM",
            style: {
                backgroundColor: "rgba(8, 113, 8, 0.5)",
                color: "white",
            },
            position: 'top-right'
            //   action: {
            //     label: "Undo",
            //     onClick: () => console.log("Undo"),
            //   },
        })
    }

    function showErrorToast(text: string) {
        toast(text, {
            // description: "Sunday, December 03, 2023 at 9:00 AM",
            style: {
                backgroundColor: "rgba(31, 41, 55, 0.9)",
                color: "white",
                justifyContent: 'center',
            },
            position: 'top-center'
            //   action: {
            //     label: "Undo",
            //     onClick: () => console.log("Undo"),
            //   },
        })
    }

    async function modelsGetAll() {
        const result = await Backend_Request({}, "/api/tech/models/all")
        if (result.result !== "ok" || !Array.isArray(result.data)) {
            showErrorToast(result.error || "Modellar yuklanmadi")
            setModels([])
            return
        }
        const activeModels = result.data
            .filter((model: { status?: boolean }) => model.status !== false)
            .sort((a: { modeli?: string }, b: { modeli?: string }) =>
                String(a.modeli || "").localeCompare(String(b.modeli || ""), "uz"),
            )
        setModels(activeModels)
    }

    async function gsCodeCount() {
        const result = await Backend_Request({}, "/api/tech/gscode/count")
        if (result.result === "ok" && Array.isArray(result.data)) {
            setGsCodeInfo(result.data)
        } else {
            showErrorToast(result.error || "GS Code soni yuklanmadi")
        }
    }

    const onDrop = useCallback((acceptedFiles: File[]) => {
        if (acceptedFiles.length > 0) {
            setSelectedFile(acceptedFiles[0]);
        }
    }, []);

    const [selectedFile, setSelectedFile] = useState<File | null>(null)
    const [uploading, setUploading] = useState(false)

    const {
        getRootProps,
        getInputProps,
        isDragActive,
        open,
    } = useDropzone({
        onDrop,
        multiple: false,
        noClick: true,
    })

    const handleSend = () => {
        if (!selectedFile || !selectedModel) {
            return
        }

        setUploading(true)
        const reader = new FileReader()
        reader.readAsDataURL(selectedFile)

        reader.onload = async () => {
            const result = await Backend_Request(
                { file64: reader.result, model_id: selectedModel.id },
                "/api/tech/gscode/upload",
            )
            setUploading(false)
            if (result.result === "ok") {
                const data = result.data as { inserted?: number; duplicates?: number } | null
                const inserted = data && typeof data === "object" ? (data.inserted ?? 0) : 0
                const duplicates = data && typeof data === "object" ? (data.duplicates ?? 0) : 0
                if (inserted > 0) {
                    showOkToast(
                        duplicates > 0
                            ? `Qo'shildi: ${inserted} ta kod (${duplicates} ta dublikat o'tkazib yuborildi)`
                            : `Qo'shildi: ${inserted} ta kod`,
                    )
                    gsCodeCount()
                    setSelectedFile(null)
                } else {
                    showErrorToast("Yangi kod qo'shilmadi")
                }
            } else {
                showErrorToast("Yuklashda muammo: " + result.error)
                if (Array.isArray(result.data)) {
                    arrayToString(result.data)
                }
            }
        }
        reader.onerror = (error) => {
            setUploading(false)
            showErrorToast(String(error))
        }
    }

    function arrayToString(array: string[]) {
        const temp = array.join("\n");
        setErrorCodesString(temp);
        // let temp = ""
        // for (let i = 0; i < array.length; i++) {
        //     temp += array[i] + "\n"
        // }
        // setErrorCodesString(temp)
    }

    function selectModel(model: any) {
        setSelectedModelName(model.modeli)
        setSelectedModelInfo([model])
        setShowInfo(true)
        setSelectedModel(model)
    }

    function selectModelFromTableRow(row: any) {
        const full =
            models.find((m: any) => m.id === row.model_id) ??
            models.find((m: any) => m.seriya_raqami === row.seriya_raqami)

        if (full) {
            selectModel(full)
            return
        }

        selectModel({
            id: row.model_id,
            modeli: row.modeli,
            model_nomi: row.qisqa_nomi,
            seriya_raqami: row.seriya_raqami,
            gs1_ean13: row.gs1_ean13,
            brend: row.brand,
        })
    }

    function dropDownModels() {
        return (
            <DropdownMenu>
                <DropdownMenuTrigger asChild>
                    <Button
                        variant="outline"
                        className="h-10 max-w-[220px] shrink-0 truncate rounded-xl px-3 font-semibold shadow-sm"
                    >
                        {selectedModelName}
                    </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent className="w-[500px]" align="start">
                    <DropdownMenuGroup>
                        {models.reduce((acc, model) => {
                            acc.push(
                                <DropdownMenuItem
                                    key={model.id}
                                    onClick={() => selectModel(model)}
                                >
                                    {model.modeli} - {model.qisqa_nomi} - {model.rangi} - {model.gs1_ean13}
                                </DropdownMenuItem>
                            )
                            return acc
                        }, [] as React.ReactNode[])}

                    </DropdownMenuGroup>
                </DropdownMenuContent>
            </DropdownMenu>
        )
    }

    const columnHelper = createColumnHelper<any>();

    const columns = [
        columnHelper.accessor("brand", {
            header: "Brand",
        }),
        columnHelper.accessor("modeli", {
            header: "Modeli",
        }),
        columnHelper.accessor("model_nomi", {
            header: "Model Nomi",
        }),
        columnHelper.accessor("seriya_raqami", {
            header: "Seriya raqami",
        }),
        columnHelper.accessor("gs1_ean13", {
            header: "GS1 Code",
        }),
        columnHelper.accessor("count", {
            header: "Soni",
        }),
    ];

    const modelInfoColumns = [
        columnHelper.accessor("brend", {
            header: "Brand",
        }),
        columnHelper.accessor("modeli", {
            header: "Modeli",
        }),
        columnHelper.accessor("model_nomi", {
            header: "Model Nomi",
        }),
        columnHelper.accessor("seriya_raqami", {
            header: "Seriya raqami",
        }),
        columnHelper.accessor("gs1_ean13", {
            header: "GS1 Code",
        }),
        columnHelper.accessor("odoo_code", {
            header: "Odoo code",
        }),
    ];

    const [globalFilter, setGlobalFilter] = useState("");

    const tableData = useMemo(() => {
        const rows = [...gscodeInfo]
        const seenIds = new Set(rows.map((row: any) => row.model_id))

        for (const model of models) {
            if (!seenIds.has(model.id)) {
                rows.push({
                    model_id: model.id,
                    brand: model.brend,
                    modeli: model.modeli,
                    model_nomi: model.qisqa_nomi,
                    seriya_raqami: model.seriya_raqami,
                    gs1_ean13: model.gs1_ean13,
                    count: 0,
                })
            }
        }

        return rows.sort((a: any, b: any) => {
            const aZero = a.count === 0
            const bZero = b.count === 0
            if (aZero !== bZero) {
                return aZero ? 1 : -1
            }

            const brandCmp = String(a.brand || "").localeCompare(String(b.brand || ""))
            if (brandCmp !== 0) {
                return brandCmp
            }
            return String(a.seriya_raqami || "").localeCompare(String(b.seriya_raqami || ""))
        })
    }, [gscodeInfo, models])

    const [sorting, setSorting] = useState<SortingState>([]);
    const table = useReactTable({
        data: tableData,
        columns,

        state: {
            sorting,
            globalFilter,
        },

        onSortingChange: setSorting,
        onGlobalFilterChange: setGlobalFilter,

        getCoreRowModel: getCoreRowModel(),
        getSortedRowModel: getSortedRowModel(),
        getFilteredRowModel: getFilteredRowModel(),

        globalFilterFn: (row, _columnId, filterValue) => {
            const search = String(filterValue).toLowerCase();
            const modeli = String(row.original.modeli || "").toLowerCase();
            const modelNomi = String(row.original.model_nomi || "").toLowerCase();
            const seriya = String(row.original.seriya_raqami || "").toLowerCase();
            const brand = String(row.original.brand || "").toLowerCase();

            return (
                modeli.includes(search) ||
                modelNomi.includes(search) ||
                seriya.includes(search) ||
                brand.includes(search)
            );
        },
    });

    const table2 = useReactTable({
        data: selectedModelInfo,
        columns: modelInfoColumns,
        getCoreRowModel: getCoreRowModel(),
    });

    function gscodeCountTable() {
        return (
            <Table>
            <TableHeader className="sticky top-0 z-10 bg-muted/90 backdrop-blur">
                {table.getHeaderGroups().map((hg) => (
                    <TableRow key={hg.id} className="border-b border-border/60 bg-muted/40 hover:bg-muted/40">
                        {hg.headers.map((header) => (
                            <TableHead
                                key={header.id}
                                onClick={header.column.getToggleSortingHandler()}
                                className="sticky top-0 z-10 cursor-pointer select-none bg-muted/90 font-semibold backdrop-blur"
                            >
                                {flexRender(
                                    header.column.columnDef.header,
                                    header.getContext()
                                )}

                                {/* sorting indicator */}
                                {{
                                    asc: " 🔼",
                                    desc: " 🔽",
                                }[header.column.getIsSorted() as string] ?? null}
                            </TableHead>
                        ))}
                    </TableRow>
                ))}
            </TableHeader>

            {/* BODY */}
            <TableBody>
                {table.getRowModel().rows.map((row) => (
                    <TableRow
                        key={row.id}
                        onClick={() => selectModelFromTableRow(row.original)}
                        className={cn(
                            "cursor-pointer",
                            selectedModel?.id === row.original.model_id && "bg-primary/10 hover:bg-primary/15",
                        )}
                    >
                        {row.getVisibleCells().map((cell) => (
                            <TableCell key={cell.id}>
                                {flexRender(cell.column.columnDef.cell, cell.getContext())}
                            </TableCell>
                        ))}
                    </TableRow>
                ))}
            </TableBody>
        </Table>
        )
    }

    function modelInfoTable() {
        return <div style={{borderWidth: 1.5, borderRadius: 10}}>
            <Table >

            {/* HEADER */}
            <TableHeader>
                {table2.getHeaderGroups().map((hg) => (
                    <TableRow key={hg.id}>
                        {hg.headers.map((header) => (
                            <TableHead
                                key={header.id}
                                onClick={header.column.getToggleSortingHandler()}
                                className="cursor-pointer select-none"
                            >
                                {flexRender(
                                    header.column.columnDef.header,
                                    header.getContext()
                                )}

                                {/* sorting indicator */}
                                {{
                                    asc: " 🔼",
                                    desc: " 🔽",
                                }[header.column.getIsSorted() as string] ?? null}
                            </TableHead>
                        ))}
                    </TableRow>
                ))}
            </TableHeader>

            {/* BODY */}
            <TableBody>
                {table2.getRowModel().rows.map((row) => (
                    <TableRow
                        key={row.id}
                        // onClick={() => navigate(`/t1/${row.original.id}`)}
                    // className="cursor-pointer"
                    >
                        {row.getVisibleCells().map((cell) => (
                            <TableCell key={cell.id}>
                                {flexRender(cell.column.columnDef.cell, cell.getContext())}
                            </TableCell>
                        ))}
                    </TableRow>
                ))}
            </TableBody>
        </Table></div>
    }

    useEffect(() => {
        modelsGetAll()
        gsCodeCount()
    }, [])

    return (
        <PageContainer
            title="GS Code"
            description="GS kodlarni yuklash va boshqarish"
            fullWidth
            center={
                <PageSearchInput value={globalFilter} onChange={setGlobalFilter} />
            }
            actions={
                <div
                    {...getRootProps()}
                    className={cn(
                        "flex shrink-0 flex-nowrap items-center gap-2",
                        isDragActive && "rounded-xl ring-2 ring-primary/40",
                    )}
                >
                    <input {...getInputProps()} />
                    {dropDownModels()}
                    <Button
                        type="button"
                        variant="outline"
                        className="h-10 shrink-0 rounded-xl px-3 font-semibold shadow-sm"
                        disabled={uploading}
                        onClick={() => open()}
                    >
                        <FileSpreadsheet className="size-4" />
                        {selectedFile ? selectedFile.name.slice(0, 18) : "Import GsCode"}
                    </Button>
                    <Button
                        type="button"
                        onClick={handleSend}
                        disabled={!selectedFile || !selectedModel || uploading}
                        className="h-10 shrink-0 rounded-xl bg-emerald-600 px-4 font-semibold text-white shadow-sm hover:bg-emerald-700"
                    >
                        <Upload className="size-4" />
                        {uploading ? "Yuklanmoqda..." : "Yuklash"}
                    </Button>
                    <Button
                        type="button"
                        variant="outline"
                        className="ml-5 h-10 shrink-0 rounded-xl px-3 font-semibold shadow-sm"
                        onClick={() => navigate("/gscode/report")}
                    >
                        Hisobot
                    </Button>
                </div>
            }
        >
            {showInfo && (
                <Panel title="Tanlangan model" className="shrink-0">
                    {modelInfoTable()}
                </Panel>
            )}
            <Panel
                title="Yuklangan GS Code ro‘yxati"
                noPadding
            >
                <div className="overflow-x-auto rounded-b-2xl pb-3">
                    {gscodeCountTable()}
                </div>
            </Panel>
            {errorCodesString && (
                <Panel title="Muammoli kodlar" className="border-destructive/30">
                    <pre className="max-h-40 overflow-auto text-xs whitespace-pre-wrap text-muted-foreground">
                        {errorCodesString}
                    </pre>
                </Panel>
            )}
        </PageContainer>
    )
}