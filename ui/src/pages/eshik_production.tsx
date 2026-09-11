import { useEffect, useMemo, useRef, useState } from "react"
import { ChevronsUpDown, Download, FileSpreadsheet, Plus, Search, Trash2, Upload } from "lucide-react"
import { saveAs } from "file-saver"
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  type SortingState,
  useReactTable,
} from "@tanstack/react-table"
import { Backend_Request, Backend_Request_Blob } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
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
import { cn } from "@/lib/utils"

type ProductionComponent = {
  id?: number
  factory_code?: string
  manufacturer_code: string
  full_name_uz?: string
  standard_name_uz: string
  odoo_code: string
  comment?: string
  type?: string
  unit?: string
  photo_path?: string
}

type EshikPart = {
  id?: number
  door_code?: string
  component_id: number
  factory_code?: string
  full_name_uz?: string
  comment?: string
  index1: string
  index2: string
  seriya_raqami: string
}

type EshikModel = {
  id: number
  model_name: string
  freeze: EshikPart
  ref: EshikPart
  c_time: string
}

type EshikImportError = {
  row: number
  column: string
  message: string
}

type EshikImportResult = {
  imported_rows?: number
  inserted?: number
  updated?: number
  errors?: EshikImportError[]
}

type PartDraft = {
  component_id: number
  seriya_raqami: string
  index1: string
  index2: string
}

const emptyPart = (): PartDraft => ({
  component_id: 0,
  seriya_raqami: "",
  index1: "",
  index2: "",
})

const columnHelper = createColumnHelper<EshikModel>()

function componentFactoryCode(component: Pick<ProductionComponent, "factory_code" | "manufacturer_code">) {
  return component.factory_code?.trim() || component.manufacturer_code?.trim() || ""
}

function componentLabel(component: ProductionComponent) {
  const code = componentFactoryCode(component)
  const comment = component.comment?.trim() || ""
  if (code && comment) {
    return `${code} — ${comment}`
  }
  return code || comment || "Komponent"
}

function partSummary(part?: EshikPart) {
  if (!part?.component_id) return "—"
  const code = part.factory_code?.trim() || ""
  const prefix = part.seriya_raqami?.trim() || "—"
  return `${code || `ID ${part.component_id}`} · ${prefix}`
}

function normalizeSearchValue(value: unknown) {
  return String(value ?? "").trim().toLocaleLowerCase()
}

function ComponentPicker({
  label,
  selected,
  components,
  usedIds,
  onSelect,
}: {
  label: string
  selected: ProductionComponent | null
  components: ProductionComponent[]
  usedIds: Set<number>
  onSelect: (component: ProductionComponent) => void
}) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState("")

  const available = useMemo(() => {
    const query = normalizeSearchValue(search)
    return components
      .filter((component) => component.id && (!usedIds.has(component.id) || component.id === selected?.id))
      .filter((component) => {
        if (!query) return true
        return [
          component.factory_code,
          component.manufacturer_code,
          component.full_name_uz,
          component.standard_name_uz,
          component.odoo_code,
          component.comment,
        ]
          .map(normalizeSearchValue)
          .join(" ")
          .includes(query)
      })
  }, [components, usedIds, search, selected?.id])

  return (
    <div className="relative min-w-0 space-y-2">
      <Label>{label}</Label>
      <Button
        type="button"
        variant="outline"
        className="h-10 w-full min-w-0 justify-between gap-2 px-3 font-normal"
        onClick={() => setOpen((value) => !value)}
      >
        <span className="min-w-0 flex-1 truncate text-left">
          {selected ? componentLabel(selected) : "Komponentni tanlang"}
        </span>
        <ChevronsUpDown className="size-4 shrink-0 opacity-50" />
      </Button>
      {open ? (
        <div className="absolute top-full right-0 left-0 z-50 mt-1 max-w-full overflow-hidden rounded-lg bg-popover text-sm text-popover-foreground shadow-md ring-1 ring-foreground/10">
          <div className="border-b p-2">
            <div className="relative min-w-0">
              <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder="Factory code, comment..."
                className="h-9 min-w-0 pl-8"
                autoFocus
                onKeyDown={(event) => {
                  if (event.key === "Escape") {
                    setOpen(false)
                    setSearch("")
                  }
                }}
              />
            </div>
          </div>
          <div className="max-h-56 overflow-x-hidden overflow-y-auto overscroll-contain p-1" onWheel={(event) => event.stopPropagation()}>
            {available.length ? (
              available.map((component) => (
                <button
                  key={component.id}
                  type="button"
                  className={cn(
                    "block w-full min-w-0 overflow-hidden rounded-md px-3 py-2 text-left text-sm hover:bg-muted",
                    selected?.id === component.id && "bg-muted font-medium",
                  )}
                  onClick={() => {
                    onSelect(component)
                    setSearch("")
                    setOpen(false)
                  }}
                >
                  <div className="truncate font-medium">{componentFactoryCode(component) || "—"}</div>
                  <div className="truncate text-xs text-muted-foreground">{component.comment?.trim() || "—"}</div>
                </button>
              ))
            ) : (
              <div className="px-3 py-6 text-center text-sm text-muted-foreground">Komponent topilmadi</div>
            )}
          </div>
        </div>
      ) : null}
    </div>
  )
}

export default function EshikProductionPage() {
  const [items, setItems] = useState<EshikModel[]>([])
  const [allComponents, setAllComponents] = useState<ProductionComponent[]>([])
  const [globalFilter, setGlobalFilter] = useState("")
  const [sorting, setSorting] = useState<SortingState>([])
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState(0)
  const [modelName, setModelName] = useState("")
  const [freeze, setFreeze] = useState<PartDraft>(emptyPart())
  const [refPart, setRefPart] = useState<PartDraft>(emptyPart())
  const [saving, setSaving] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const importInputRef = useRef<HTMLInputElement>(null)

  async function loadItems() {
    const result = await Backend_Request<EshikModel[]>({}, "/api/production/eshik/all")
    if (result.result === "ok") {
      setItems(result.data ?? [])
    } else {
      ShowErrorToast(result.error || "Xatolik")
    }
  }

  async function loadComponents() {
    const result = await Backend_Request<ProductionComponent[]>({}, "/api/production/components/all")
    if (result.result === "ok") {
      setAllComponents(result.data ?? [])
    } else {
      ShowErrorToast(result.error || "Xatolik")
    }
  }

  useEffect(() => {
    void loadItems()
    void loadComponents()
  }, [])

  function findComponent(id: number) {
    return allComponents.find((component) => component.id === id) ?? null
  }

  function openAddDialog() {
    setEditingId(0)
    setModelName("")
    setFreeze(emptyPart())
    setRefPart(emptyPart())
    setDialogOpen(true)
  }

  function openEditDialog(item: EshikModel) {
    setEditingId(item.id)
    setModelName(item.model_name ?? "")
    setFreeze({
      component_id: item.freeze?.component_id ?? 0,
      seriya_raqami: item.freeze?.seriya_raqami ?? "",
      index1: item.freeze?.index1 ?? "",
      index2: item.freeze?.index2 ?? "",
    })
    setRefPart({
      component_id: item.ref?.component_id ?? 0,
      seriya_raqami: item.ref?.seriya_raqami ?? "",
      index1: item.ref?.index1 ?? "",
      index2: item.ref?.index2 ?? "",
    })
    setDialogOpen(true)
  }

  async function deleteItem(id: number) {
    const result = await Backend_Request({ id }, "/api/production/eshik/delete")
    if (result.result === "ok") {
      ShowOKToast("O'chirildi")
      await loadItems()
    } else {
      ShowErrorToast(result.error || "O'chirish xatolik")
    }
  }

  async function saveItem() {
    if (!modelName.trim()) {
      ShowErrorToast("Model nomini kiriting")
      return
    }
    if (!freeze.component_id || !refPart.component_id) {
      ShowErrorToast("Freeze va Ref komponentlarini tanlang")
      return
    }
    if (!freeze.seriya_raqami.trim() || !refPart.seriya_raqami.trim()) {
      ShowErrorToast("Har ikkala prefixni kiriting")
      return
    }

    setSaving(true)
    const payload = {
      id: editingId || undefined,
      model_name: modelName.trim(),
      freeze: {
        component_id: freeze.component_id,
        seriya_raqami: freeze.seriya_raqami.trim(),
        index1: freeze.index1.trim(),
        index2: freeze.index2.trim(),
      },
      ref: {
        component_id: refPart.component_id,
        seriya_raqami: refPart.seriya_raqami.trim(),
        index1: refPart.index1.trim(),
        index2: refPart.index2.trim(),
      },
    }
    const result = await Backend_Request(
      payload,
      editingId ? "/api/production/eshik/update" : "/api/production/eshik/add",
    )
    setSaving(false)
    if (result.result === "ok") {
      ShowOKToast(editingId ? "Saqlandi" : "Qo'shildi")
      setDialogOpen(false)
      await loadItems()
    } else {
      ShowErrorToast(result.error || "Saqlash xatolik")
    }
  }

  async function downloadExport() {
    setExporting(true)
    const result = await Backend_Request_Blob({}, "/api/production/eshik/export")
    setExporting(false)
    if (result.result !== "ok" || !result.data) {
      ShowErrorToast(result.error || "Export xatolik")
      return
    }
    saveAs(result.data, "eshik_models.xlsx")
  }

  async function downloadTemplate() {
    const result = await Backend_Request_Blob({}, "/api/production/eshik/template")
    if (result.result !== "ok" || !result.data) {
      ShowErrorToast(result.error || "Shablon yuklanmadi")
      return
    }
    saveAs(result.data, "eshik_models_template.xlsx")
  }

  async function uploadXlsx() {
    if (!selectedFile) {
      ShowErrorToast("Avval XLSX fayl tanlang")
      return
    }
    setUploading(true)
    const reader = new FileReader()
    reader.readAsDataURL(selectedFile)
    reader.onload = async () => {
      const result = await Backend_Request<EshikImportResult>(
        { file64: reader.result },
        "/api/production/eshik/import",
      )
      setUploading(false)
      if (result.result !== "ok") {
        const errors = result.data?.errors ?? []
        if (errors.length > 0) {
          const first = errors[0]
          ShowErrorToast(`Qator ${first.row}: ${first.message}`)
        } else {
          ShowErrorToast(result.error || "Import xatolik")
        }
        return
      }
      const data = result.data
      const errCount = data?.errors?.length ?? 0
      ShowOKToast(
        `Import: qo'shildi ${data?.inserted ?? 0}, yangilandi ${data?.updated ?? 0}` +
          (errCount > 0 ? `, xato ${errCount}` : ""),
      )
      if (errCount > 0 && data?.errors?.[0]) {
        ShowErrorToast(`Qator ${data.errors[0].row}: ${data.errors[0].message}`)
      }
      setSelectedFile(null)
      if (importInputRef.current) {
        importInputRef.current.value = ""
      }
      await loadItems()
    }
    reader.onerror = () => {
      setUploading(false)
      ShowErrorToast("Fayl o'qilmadi")
    }
  }

  const columns = useMemo(
    () => [
      columnHelper.accessor("model_name", {
        header: "Model nomi",
        cell: ({ row }) => (
          <button type="button" className="font-medium hover:underline" onClick={() => openEditDialog(row.original)}>
            {row.original.model_name}
          </button>
        ),
      }),
      columnHelper.display({
        id: "freeze",
        header: "Freeze",
        cell: ({ row }) => (
          <div className="space-y-0.5 text-sm">
            <div className="font-mono">{partSummary(row.original.freeze)}</div>
            <div className="text-xs text-muted-foreground">
              idx {row.original.freeze?.index1 || "—"} / {row.original.freeze?.index2 || "—"}
            </div>
          </div>
        ),
      }),
      columnHelper.display({
        id: "ref",
        header: "Ref",
        cell: ({ row }) => (
          <div className="space-y-0.5 text-sm">
            <div className="font-mono">{partSummary(row.original.ref)}</div>
            <div className="text-xs text-muted-foreground">
              idx {row.original.ref?.index1 || "—"} / {row.original.ref?.index2 || "—"}
            </div>
          </div>
        ),
      }),
      columnHelper.display({
        id: "actions",
        header: "",
        cell: ({ row }) => (
          <Button variant="ghost" size="icon" className="text-destructive hover:text-destructive" onClick={() => deleteItem(row.original.id)}>
            <Trash2 className="size-4" />
          </Button>
        ),
      }),
    ],
    [],
  )

  const table = useReactTable({
    data: items,
    columns,
    state: { sorting, globalFilter },
    onSortingChange: setSorting,
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
  })

  // Same component may be reused across models; only freeze≠ref within one model.
  const freezeUsed = useMemo(() => {
    const ids = new Set<number>()
    if (refPart.component_id) ids.add(refPart.component_id)
    return ids
  }, [refPart.component_id])

  const refUsed = useMemo(() => {
    const ids = new Set<number>()
    if (freeze.component_id) ids.add(freeze.component_id)
    return ids
  }, [freeze.component_id])

  return (
    <PageContainer
      title="Eshik modellari"
      description="Model nomi + freeze/ref: prefix, index_1/index_2 va balans komponenti"
      actions={
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" className="rounded-xl" onClick={() => void downloadExport()} disabled={exporting}>
            <Download className="size-4" />
            {exporting ? "Export..." : "Export XLSX"}
          </Button>
          <Button variant="outline" className="rounded-xl" onClick={() => void downloadTemplate()}>
            <Download className="size-4" />
            Shablon
          </Button>
          <Button className="rounded-xl" onClick={openAddDialog}>
            <Plus className="size-4" />
            Model qo'shish
          </Button>
        </div>
      }
      fullWidth
    >
      <Panel title={`${table.getFilteredRowModel().rows.length} ta model`} noPadding>
        <div className="flex flex-wrap items-center gap-2 border-b px-4 py-3">
          <div className="relative min-w-[14rem] flex-1 max-w-md">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={globalFilter}
              onChange={(event) => setGlobalFilter(event.target.value)}
              placeholder="Qidirish..."
              className="h-10 pl-9"
            />
          </div>
          <input
            ref={importInputRef}
            type="file"
            accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
            className="hidden"
            onChange={(event) => setSelectedFile(event.target.files?.[0] ?? null)}
          />
          <Button
            type="button"
            variant="outline"
            className="h-10 rounded-xl"
            disabled={uploading}
            onClick={() => importInputRef.current?.click()}
          >
            <FileSpreadsheet className="size-4" />
            {selectedFile ? selectedFile.name.slice(0, 18) : "Import"}
          </Button>
          <Button
            className="h-10 rounded-xl bg-emerald-600 text-white hover:bg-emerald-700"
            disabled={!selectedFile || uploading}
            onClick={() => void uploadXlsx()}
          >
            <Upload className="size-4" />
            {uploading ? "Yuklanmoqda..." : "Upload"}
          </Button>
        </div>
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead key={header.id}>
                      {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows.length ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={columns.length} className="h-24 text-center text-muted-foreground">
                    Modellar yo'q
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </Panel>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="w-full max-w-3xl overflow-hidden">
          <DialogHeader>
            <DialogTitle>{editingId ? "Modelni tahrirlash" : "Model qo'shish"}</DialogTitle>
            <DialogDescription>Freeze va Ref uchun alohida komponent, prefix va indekslar</DialogDescription>
          </DialogHeader>
          <div className="min-w-0 space-y-5">
            <div className="space-y-2">
              <Label>Model nomi</Label>
              <Input value={modelName} onChange={(event) => setModelName(event.target.value)} placeholder="Masalan: RF-340" />
            </div>

            {(["freeze", "ref"] as const).map((door) => {
              const draft = door === "freeze" ? freeze : refPart
              const setDraft = door === "freeze" ? setFreeze : setRefPart
              const used = door === "freeze" ? freezeUsed : refUsed
              const title = door === "freeze" ? "Freeze door" : "Ref door"
              return (
                <div key={door} className="space-y-3 rounded-xl border p-4">
                  <div className="text-sm font-semibold">{title}</div>
                  <ComponentPicker
                    label="Komponent (balans)"
                    selected={findComponent(draft.component_id)}
                    components={allComponents}
                    usedIds={used}
                    onSelect={(component) =>
                      setDraft((current) => ({ ...current, component_id: component.id ?? 0 }))
                    }
                  />
                  <div className="grid gap-3 sm:grid-cols-3">
                    <div className="space-y-2">
                      <Label>Serial prefix</Label>
                      <Input
                        value={draft.seriya_raqami}
                        onChange={(event) => setDraft((current) => ({ ...current, seriya_raqami: event.target.value }))}
                        className="font-mono"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Index 1</Label>
                      <Input
                        value={draft.index1}
                        onChange={(event) => setDraft((current) => ({ ...current, index1: event.target.value }))}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Index 2</Label>
                      <Input
                        value={draft.index2}
                        onChange={(event) => setDraft((current) => ({ ...current, index2: event.target.value }))}
                      />
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Bekor
            </Button>
            <Button onClick={() => void saveItem()} disabled={saving}>
              {saving ? "Saqlanmoqda..." : editingId ? "Saqlash" : "Qo'shish"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PageContainer>
  )
}
