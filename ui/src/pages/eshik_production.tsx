import { useEffect, useMemo, useRef, useState } from "react"
import { ChevronsUpDown, ImagePlus, Plus, Search, Trash2 } from "lucide-react"
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  type SortingState,
  useReactTable,
} from "@tanstack/react-table"
import { Backend_Request } from "@/services/backend"
import { Global_Data } from "@/config/config"
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

type EshikComponent = {
  id: number
  component_id: number
  factory_code: string
  full_name_uz: string
  odoo_code: string
  comment: string
  type: string
  unit: string
  photo_path?: string
  index1: string
  index2: string
  seriya_raqami: string
  c_time: string
}

type FieldDraft = {
  index1: string
  index2: string
  seriya_raqami: string
}

const columnHelper = createColumnHelper<EshikComponent>()

function componentFactoryCode(component: Pick<ProductionComponent, "factory_code" | "manufacturer_code">) {
  return component.factory_code?.trim() || component.manufacturer_code?.trim() || ""
}

function photoUrl(path?: string) {
  if (!path) return ""
  if (path.startsWith("http")) return path
  return `${Global_Data.server_ip}${path}`
}

function componentLabel(component: ProductionComponent) {
  const code = componentFactoryCode(component)
  const comment = component.comment?.trim() || ""
  if (code && comment) {
    return `${code} — ${comment}`
  }
  return code || comment || "Komponent"
}

function normalizeSearchValue(value: unknown) {
  return String(value ?? "").trim().toLocaleLowerCase()
}

export default function EshikProductionPage() {
  const [items, setItems] = useState<EshikComponent[]>([])
  const [allComponents, setAllComponents] = useState<ProductionComponent[]>([])
  const [globalFilter, setGlobalFilter] = useState("")
  const [sorting, setSorting] = useState<SortingState>([])
  const [dialogOpen, setDialogOpen] = useState(false)
  const [componentPickerOpen, setComponentPickerOpen] = useState(false)
  const [componentSearch, setComponentSearch] = useState("")
  const [selectedComponent, setSelectedComponent] = useState<ProductionComponent | null>(null)
  const [addIndex1, setAddIndex1] = useState("")
  const [addIndex2, setAddIndex2] = useState("")
  const [addSeriya, setAddSeriya] = useState("")
  const [saving, setSaving] = useState(false)

  const itemsRef = useRef<EshikComponent[]>([])
  const savedFieldsRef = useRef<Record<number, FieldDraft>>({})
  const saveTimersRef = useRef<Record<number, number>>({})
  const pendingSaveRef = useRef<Set<number>>(new Set())

  function syncSavedFields(data: EshikComponent[]) {
    const saved: Record<number, FieldDraft> = {}
    for (const item of data) {
      saved[item.id] = {
        index1: item.index1 ?? "",
        index2: item.index2 ?? "",
        seriya_raqami: item.seriya_raqami ?? "",
      }
    }
    savedFieldsRef.current = saved
  }

  async function loadItems() {
    const result = await Backend_Request<EshikComponent[]>({}, "/api/production/eshik/all")
    if (result.result === "ok") {
      const data = result.data ?? []
      setItems(data)
      itemsRef.current = data
      syncSavedFields(data)
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

  useEffect(() => {
    itemsRef.current = items
  }, [items])

  const usedComponentIds = useMemo(
    () => new Set(items.map((item) => item.component_id)),
    [items],
  )

  const availableComponents = useMemo(() => {
    const query = normalizeSearchValue(componentSearch)
    return allComponents
      .filter((component) => component.id && !usedComponentIds.has(component.id))
      .filter((component) => {
        if (!query) return true
        return [
          component.factory_code,
          component.manufacturer_code,
          component.full_name_uz,
          component.standard_name_uz,
          component.odoo_code,
          component.comment,
          component.type,
          component.unit,
        ]
          .map(normalizeSearchValue)
          .join(" ")
          .includes(query)
      })
  }, [allComponents, usedComponentIds, componentSearch])

  function openAddDialog() {
    setSelectedComponent(null)
    setComponentSearch("")
    setComponentPickerOpen(false)
    setAddIndex1("")
    setAddIndex2("")
    setAddSeriya("")
    setDialogOpen(true)
  }

  function handleFieldChange(id: number, field: keyof FieldDraft, value: string) {
    setItems((current) => {
      const next = current.map((row) => (row.id === id ? { ...row, [field]: value } : row))
      itemsRef.current = next
      return next
    })
    scheduleFieldSave(id)
  }

  function scheduleFieldSave(id: number) {
    const existingTimer = saveTimersRef.current[id]
    if (existingTimer) window.clearTimeout(existingTimer)
    saveTimersRef.current[id] = window.setTimeout(() => {
      delete saveTimersRef.current[id]
      void persistFields(id)
    }, 350)
  }

  function flushFieldSave(id: number) {
    const existingTimer = saveTimersRef.current[id]
    if (existingTimer) {
      window.clearTimeout(existingTimer)
      delete saveTimersRef.current[id]
    }
    void persistFields(id, { notify: true })
  }

  async function persistFields(id: number, options?: { notify?: boolean }) {
    const item = itemsRef.current.find((row) => row.id === id)
    if (!item) return

    const nextIndex1 = (item.index1 ?? "").trim()
    const nextIndex2 = (item.index2 ?? "").trim()
    const nextSeriya = (item.seriya_raqami ?? "").trim()
    const saved = savedFieldsRef.current[id] ?? { index1: "", index2: "", seriya_raqami: "" }

    if (
      nextIndex1 === saved.index1.trim() &&
      nextIndex2 === saved.index2.trim() &&
      nextSeriya === saved.seriya_raqami.trim()
    ) {
      return
    }

    if (pendingSaveRef.current.has(id)) {
      scheduleFieldSave(id)
      return
    }

    pendingSaveRef.current.add(id)
    try {
      const result = await Backend_Request(
        { id, index1: nextIndex1, index2: nextIndex2, seriya_raqami: nextSeriya },
        "/api/production/eshik/update",
      )
      if (result.result === "ok") {
        savedFieldsRef.current[id] = { index1: nextIndex1, index2: nextIndex2, seriya_raqami: nextSeriya }
        if (options?.notify) ShowOKToast("Ma'lumotlar saqlandi")
      } else {
        ShowErrorToast(result.error || "Saqlash xatolik")
      }
    } finally {
      pendingSaveRef.current.delete(id)
    }
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

  const columns = useMemo(
    () => [
      columnHelper.display({
        id: "photo",
        header: "Photo",
        cell: ({ row }) =>
          row.original.photo_path ? (
            <img src={photoUrl(row.original.photo_path)} alt={row.original.factory_code} className="size-12 rounded-lg border object-cover" />
          ) : (
            <div className="flex size-12 items-center justify-center rounded-lg border border-dashed bg-muted text-muted-foreground">
              <ImagePlus className="size-5" />
            </div>
          ),
      }),
      columnHelper.accessor("factory_code", { header: "Factory code" }),
      columnHelper.accessor("comment", {
        header: "Comment",
        cell: ({ getValue }) => (
          <span className="block max-w-[18rem] truncate" title={String(getValue() ?? "")}>
            {String(getValue() ?? "") || "—"}
          </span>
        ),
      }),
      columnHelper.display({
        id: "seriya_raqami",
        header: "Serial prefix",
        cell: ({ row }) => (
          <Input
            value={row.original.seriya_raqami ?? ""}
            onChange={(event) => handleFieldChange(row.original.id, "seriya_raqami", event.target.value)}
            onBlur={() => flushFieldSave(row.original.id)}
            className="h-9 min-w-[7rem] rounded-lg font-mono"
          />
        ),
      }),
      columnHelper.display({
        id: "index1",
        header: "Index 1",
        cell: ({ row }) => (
          <Input
            value={row.original.index1 ?? ""}
            onChange={(event) => handleFieldChange(row.original.id, "index1", event.target.value)}
            onBlur={() => flushFieldSave(row.original.id)}
            className="h-9 min-w-[7rem] rounded-lg"
          />
        ),
      }),
      columnHelper.display({
        id: "index2",
        header: "Index 2",
        cell: ({ row }) => (
          <Input
            value={row.original.index2 ?? ""}
            onChange={(event) => handleFieldChange(row.original.id, "index2", event.target.value)}
            onBlur={() => flushFieldSave(row.original.id)}
            className="h-9 min-w-[7rem] rounded-lg"
          />
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

  async function addItem() {
    if (!selectedComponent?.id) {
      ShowErrorToast("Komponentni tanlang")
      return
    }
    setSaving(true)
    const result = await Backend_Request(
      {
        component_id: selectedComponent.id,
        index1: addIndex1.trim(),
        index2: addIndex2.trim(),
        seriya_raqami: addSeriya.trim(),
      },
      "/api/production/eshik/add",
    )
    setSaving(false)
    if (result.result === "ok") {
      ShowOKToast("Qo'shildi")
      setDialogOpen(false)
      await loadItems()
    } else {
      ShowErrorToast(result.error || "Qo'shish xatolik")
    }
  }

  return (
    <PageContainer
      title="Eshik liniyasi komponentlari"
      description="Har bir komponent uchun serial prefix, index_1 va index_2"
      actions={
        <Button className="rounded-xl" onClick={openAddDialog}>
          <Plus className="size-4" />
          Komponent qo'shish
        </Button>
      }
      fullWidth
    >
      <Panel title={`${table.getFilteredRowModel().rows.length} ta komponent`} noPadding>
        <div className="border-b px-4 py-3">
          <div className="relative max-w-md">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={globalFilter}
              onChange={(event) => setGlobalFilter(event.target.value)}
              placeholder="Qidirish..."
              className="h-10 pl-9"
            />
          </div>
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
                    Komponentlar yo'q
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </Panel>

      <Dialog
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open)
          if (!open) {
            setComponentPickerOpen(false)
            setComponentSearch("")
          }
        }}
      >
        <DialogContent className="w-full max-w-2xl overflow-hidden">
          <DialogHeader>
            <DialogTitle>Komponent qo'shish</DialogTitle>
            <DialogDescription>Production katalogidan komponent tanlang va serial parametrlarini kiriting</DialogDescription>
          </DialogHeader>
          <div className="min-w-0 space-y-4">
            <div className="relative min-w-0 space-y-2">
              <Label>Komponent</Label>
              {/* Absolute dropdown (no portal): Dialog scroll-lock blocks wheel on body-portaled Popover. */}
              <Button
                type="button"
                variant="outline"
                className="h-10 w-full min-w-0 justify-between gap-2 px-3 font-normal"
                onClick={() => setComponentPickerOpen((open) => !open)}
              >
                <span className="min-w-0 flex-1 truncate text-left">
                  {selectedComponent ? componentLabel(selectedComponent) : "Komponentni tanlang"}
                </span>
                <ChevronsUpDown className="size-4 shrink-0 opacity-50" />
              </Button>
              {componentPickerOpen ? (
                <div className="absolute top-full right-0 left-0 z-50 mt-1 max-w-full overflow-hidden rounded-lg bg-popover text-sm text-popover-foreground shadow-md ring-1 ring-foreground/10">
                  <div className="border-b p-2">
                    <div className="relative min-w-0">
                      <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
                      <Input
                        value={componentSearch}
                        onChange={(event) => setComponentSearch(event.target.value)}
                        placeholder="Factory code, comment..."
                        className="h-9 min-w-0 pl-8"
                        autoFocus
                        onKeyDown={(event) => {
                          if (event.key === "Escape") {
                            setComponentPickerOpen(false)
                            setComponentSearch("")
                          }
                        }}
                      />
                    </div>
                  </div>
                  <div
                    className="max-h-56 overflow-x-hidden overflow-y-auto overscroll-contain p-1"
                    onWheel={(event) => event.stopPropagation()}
                  >
                    {availableComponents.length ? (
                      availableComponents.map((component) => (
                        <button
                          key={component.id}
                          type="button"
                          className={cn(
                            "block w-full min-w-0 overflow-hidden rounded-md px-3 py-2 text-left text-sm hover:bg-muted",
                            selectedComponent?.id === component.id && "bg-muted font-medium",
                          )}
                          onClick={() => {
                            setSelectedComponent(component)
                            setComponentSearch("")
                            setComponentPickerOpen(false)
                          }}
                        >
                          <div className="truncate font-medium">{componentFactoryCode(component) || "—"}</div>
                          <div className="truncate text-xs text-muted-foreground">
                            {component.comment?.trim() || "—"}
                          </div>
                        </button>
                      ))
                    ) : (
                      <div className="px-3 py-6 text-center text-sm text-muted-foreground">
                        {allComponents.length === 0
                          ? "Komponentlar yuklanmadi"
                          : componentSearch.trim()
                            ? "Qidiruv bo'yicha komponent topilmadi"
                            : "Qo'shish uchun bo'sh komponent qolmadi"}
                      </div>
                    )}
                  </div>
                </div>
              ) : null}
            </div>
            <div className="grid gap-3 sm:grid-cols-3">
              <div className="space-y-2">
                <Label>Serial prefix</Label>
                <Input value={addSeriya} onChange={(e) => setAddSeriya(e.target.value)} className="font-mono" />
              </div>
              <div className="space-y-2">
                <Label>Index 1</Label>
                <Input value={addIndex1} onChange={(e) => setAddIndex1(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>Index 2</Label>
                <Input value={addIndex2} onChange={(e) => setAddIndex2(e.target.value)} />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>Bekor</Button>
            <Button onClick={() => void addItem()} disabled={saving || !selectedComponent}>
              {saving ? "Saqlanmoqda..." : "Qo'shish"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PageContainer>
  )
}
