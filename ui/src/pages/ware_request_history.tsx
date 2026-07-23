import { useCallback, useEffect, useMemo, useState } from "react"
import { ChevronsUpDown, Search } from "lucide-react"
import { useNavigate } from "react-router-dom"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
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
import { formatQty } from "@/lib/ware-quantity"

type NormModel = {
  id: number
  modeli?: string
  model_nomi?: string
  qisqa_nomi?: string
}

type SnapshotRow = {
  id: number
  delivery_note_id: number
  model_id: number
  model_name: string
  model_quantity: number
  user_id: number
  user_name: string
  created_at: string
  item_count: number
}

type SnapshotItemRow = {
  id: number
  component_id: number
  line_id: number
  line_name: string
  component_name: string
  manufacturer_code: string
  factory_code: string
  quantity: number
}

function toDateInputValue(date: Date) {
  return date.toISOString().slice(0, 10)
}

function modelLabel(model: NormModel | null) {
  if (!model) {
    return "Barcha modellar"
  }
  return model.modeli || model.qisqa_nomi || `ID ${model.id}`
}

export default function WareRequestHistoryPage() {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [notes, setNotes] = useState<SnapshotRow[]>([])
  const [models, setModels] = useState<NormModel[]>([])
  const [selectedModel, setSelectedModel] = useState<NormModel | null>(null)
  const [modelPickerOpen, setModelPickerOpen] = useState(false)
  const [modelSearch, setModelSearch] = useState("")
  const [dateFrom, setDateFrom] = useState("")
  const [dateTo, setDateTo] = useState(toDateInputValue(new Date()))
  const [search, setSearch] = useState("")
  const [dialogOpen, setDialogOpen] = useState(false)
  const [selectedNote, setSelectedNote] = useState<SnapshotRow | null>(null)
  const [dialogItems, setDialogItems] = useState<SnapshotItemRow[]>([])
  const [dialogLoading, setDialogLoading] = useState(false)

  useEffect(() => {
    void (async () => {
      const result = await Backend_Request<NormModel[]>({}, "/api/tech/models/all")
      if (result.result === "ok") {
        setModels(result.data ?? [])
      }
    })()
  }, [])

  const loadNotes = useCallback(async () => {
    setLoading(true)
    const payload: Record<string, unknown> = { limit: 500 }
    if (selectedModel?.id) {
      payload.model_id = selectedModel.id
    }
    if (dateFrom) {
      payload.date_from = dateFrom
    }
    if (dateTo) {
      payload.date_to = dateTo
    }

    const result = await Backend_Request<SnapshotRow[]>(payload, "/api/ware/delivery-notes/snapshots")
    setLoading(false)

    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Tarix yuklanmadi")
      return
    }
    setNotes(result.data ?? [])
  }, [selectedModel, dateFrom, dateTo])

  useEffect(() => {
    void loadNotes()
  }, [loadNotes])

  const filteredModels = useMemo(() => {
    const q = modelSearch.trim().toLocaleLowerCase()
    if (!q) {
      return models
    }
    return models.filter((model) =>
      [model.modeli, model.qisqa_nomi, model.id].join(" ").toLocaleLowerCase().includes(q),
    )
  }, [models, modelSearch])

  const filteredNotes = useMemo(() => {
    const q = search.trim().toLocaleLowerCase()
    if (!q) {
      return notes
    }
    return notes.filter((note) =>
      [
        note.delivery_note_id,
        note.model_name,
        note.user_name,
        note.created_at,
        note.item_count,
      ]
        .join(" ")
        .toLocaleLowerCase()
        .includes(q),
    )
  }, [notes, search])

  async function openNoteDialog(note: SnapshotRow) {
    setSelectedNote(note)
    setDialogOpen(true)
    setDialogItems([])
    setDialogLoading(true)

    const result = await Backend_Request<SnapshotItemRow[]>(
      { delivery_note_id: note.delivery_note_id },
      "/api/ware/delivery-notes/snapshots/items",
    )
    setDialogLoading(false)

    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Nakladnoy tarkibi yuklanmadi")
      return
    }
    setDialogItems(result.data ?? [])
  }

  return (
    <PageContainer
      title="Buyurtma tarixi"
      description="Yaratilgan nakladnoylar (dastlabki holat)"
      fullWidth
      actions={
        <Button
          type="button"
          variant="outline"
          className="h-10 rounded-xl"
          onClick={() => navigate("/ombor/buyurtma")}
        >
          Orqaga
        </Button>
      }
    >
      <Panel title="Filtrlar" className="shrink-0">
        <div className="flex flex-col gap-3 lg:flex-row lg:flex-wrap lg:items-end">
          <div className="space-y-2">
            <Label htmlFor="history-date-from">Dan</Label>
            <Input
              id="history-date-from"
              type="date"
              value={dateFrom}
              onChange={(event) => setDateFrom(event.target.value)}
              className="h-11 w-full min-w-[10rem] rounded-xl"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="history-date-to">Gacha</Label>
            <Input
              id="history-date-to"
              type="date"
              value={dateTo}
              onChange={(event) => setDateTo(event.target.value)}
              className="h-11 w-full min-w-[10rem] rounded-xl"
            />
          </div>
          <div className="min-w-[14rem] flex-1 space-y-2">
            <div className="text-sm font-medium text-muted-foreground">Model</div>
            <Popover
              open={modelPickerOpen}
              onOpenChange={(open) => {
                setModelPickerOpen(open)
                if (!open) {
                  setModelSearch("")
                }
              }}
            >
              <PopoverTrigger asChild>
                <Button
                  type="button"
                  variant="outline"
                  className="h-11 w-full justify-between gap-2 px-3 font-normal"
                >
                  <span className="truncate text-left">{modelLabel(selectedModel)}</span>
                  <ChevronsUpDown className="size-4 shrink-0 opacity-50" />
                </Button>
              </PopoverTrigger>
              <PopoverContent
                side="bottom"
                align="start"
                sideOffset={4}
                className="w-[var(--radix-popover-trigger-width)] gap-0 p-0"
              >
                <div className="border-b p-2">
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
                <div className="max-h-56 overflow-y-auto p-1">
                  <button
                    type="button"
                    className={cn(
                      "block w-full rounded-md px-3 py-2 text-left text-sm hover:bg-muted",
                      !selectedModel && "bg-primary/10",
                    )}
                    onClick={() => {
                      setSelectedModel(null)
                      setModelPickerOpen(false)
                    }}
                  >
                    Barcha modellar
                  </button>
                  {filteredModels.map((model) => (
                    <button
                      key={model.id}
                      type="button"
                      className={cn(
                        "block w-full rounded-md px-3 py-2 text-left text-sm hover:bg-muted",
                        selectedModel?.id === model.id && "bg-primary/10",
                      )}
                      onClick={() => {
                        setSelectedModel(model)
                        setModelPickerOpen(false)
                      }}
                    >
                      {model.modeli || model.qisqa_nomi || `ID ${model.id}`}
                    </button>
                  ))}
                </div>
              </PopoverContent>
            </Popover>
          </div>
          <div className="min-w-[14rem] flex-1 space-y-2">
            <div className="text-sm font-medium text-muted-foreground">Qidirish</div>
            <Input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="ID / model / foydalanuvchi..."
              className="h-11 rounded-xl"
            />
          </div>
          <Button type="button" className="h-11 rounded-xl" disabled={loading} onClick={() => void loadNotes()}>
            {loading ? "Yuklanmoqda..." : "Yangilash"}
          </Button>
        </div>
      </Panel>

      <Panel
        title="Nakladnoylar"
        description={`${filteredNotes.length} ta`}
        noPadding
      >
        <div className="overflow-x-auto rounded-b-2xl pb-3">
          <Table>
            <TableHeader className="sticky top-0 z-10 bg-muted/90 backdrop-blur">
              <TableRow className="border-b border-border/60 bg-muted/40 hover:bg-muted/40">
                <TableHead className="px-4 py-3 text-sm font-semibold">ID</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Model</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Miqdor</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Poz.</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Foydalanuvchi</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Sana</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredNotes.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="h-24 text-center text-muted-foreground">
                    {loading ? "Yuklanmoqda..." : "Nakladnoy topilmadi"}
                  </TableCell>
                </TableRow>
              ) : (
                filteredNotes.map((note, index) => (
                  <TableRow
                    key={note.id}
                    className={cn(
                      "cursor-pointer border-border/40 hover:bg-muted/30",
                      index % 2 === 1 && "bg-muted/10",
                    )}
                    onClick={() => void openNoteDialog(note)}
                  >
                    <TableCell className="px-4 py-3 text-sm font-medium tabular-nums">
                      #{note.delivery_note_id}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm">{note.model_name || "—"}</TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums">
                      {formatQty(note.model_quantity)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums">{note.item_count}</TableCell>
                    <TableCell className="px-4 py-3 text-sm">{note.user_name || "—"}</TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums">{note.created_at}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </Panel>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-h-[90svh] overflow-y-auto sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>
              {selectedNote ? `Nakladnoy #${selectedNote.delivery_note_id}` : "Nakladnoy"}
            </DialogTitle>
            <DialogDescription asChild>
              <div className="space-y-1 pt-1 text-sm text-muted-foreground">
                {selectedNote ? (
                  <>
                    <p>Model: {selectedNote.model_name || "—"}</p>
                    <p>Miqdor: {formatQty(selectedNote.model_quantity)}</p>
                    <p>Foydalanuvchi: {selectedNote.user_name || "—"}</p>
                    <p>Sana: {selectedNote.created_at}</p>
                    <p>Pozitsiyalar: {selectedNote.item_count} ta</p>
                  </>
                ) : null}
              </div>
            </DialogDescription>
          </DialogHeader>

          <div className="mt-2 overflow-hidden rounded-xl border border-border/60">
            <Table>
              <TableHeader className="bg-muted/40">
                <TableRow className="border-border/60 hover:bg-transparent">
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Komponent</TableHead>
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Kod</TableHead>
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Liniya</TableHead>
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Miqdor</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {dialogLoading ? (
                  <TableRow>
                    <TableCell colSpan={4} className="h-20 text-center text-muted-foreground">
                      Yuklanmoqda...
                    </TableCell>
                  </TableRow>
                ) : dialogItems.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="h-20 text-center text-muted-foreground">
                      Pozitsiyalar topilmadi
                    </TableCell>
                  </TableRow>
                ) : (
                  dialogItems.map((item, index) => (
                    <TableRow
                      key={item.id}
                      className={cn("border-border/40", index % 2 === 1 && "bg-muted/20")}
                    >
                      <TableCell className="px-4 py-3 text-sm">
                        {item.component_name?.trim() || item.manufacturer_code?.trim() || `ID ${item.component_id}`}
                      </TableCell>
                      <TableCell className="px-4 py-3 text-sm">
                        {item.manufacturer_code?.trim() || item.factory_code?.trim() || "—"}
                      </TableCell>
                      <TableCell className="px-4 py-3 text-sm">{item.line_name?.trim() || "—"}</TableCell>
                      <TableCell className="px-4 py-3 text-sm tabular-nums">{formatQty(item.quantity)}</TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </DialogContent>
      </Dialog>
    </PageContainer>
  )
}
