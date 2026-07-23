import { useCallback, useEffect, useMemo, useState } from "react"
import { Check } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
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
import { formatQty } from "@/lib/ware-quantity"

type BrigadirItemRow = {
  id: number
  delivery_note_id: number
  model_name: string
  note_created_at: string
  component_id: number
  component_name: string
  manufacturer_code: string
  factory_code: string
  line_id: number
  line_name: string
  quantity_actual: number
  is_received: boolean
}

type WareStockShortage = {
  component_id: number
  manufacturer_code?: string
  component_name?: string
  line_name?: string
  required_quantity: number
  available_quantity: number
  missing_quantity: number
}

function normalize(value: unknown) {
  return String(value ?? "").trim().toLocaleLowerCase()
}

export default function BrigadirPage() {
  const [loading, setLoading] = useState(false)
  const [items, setItems] = useState<BrigadirItemRow[]>([])
  const [search, setSearch] = useState("")
  const [confirmingItemId, setConfirmingItemId] = useState<number | null>(null)
  const [confirmingAll, setConfirmingAll] = useState(false)
  const [shortageDialogOpen, setShortageDialogOpen] = useState(false)
  const [shortages, setShortages] = useState<WareStockShortage[]>([])

  const loadItems = useCallback(async () => {
    setLoading(true)
    const result = await Backend_Request<BrigadirItemRow[]>({}, "/api/lines/brigadir/items")
    setLoading(false)

    if (result.result === "ok") {
      setItems(result.data ?? [])
      return
    }

    ShowErrorToast(result.error || "Komponentlar yuklanmadi")
    setItems([])
  }, [])

  useEffect(() => {
    void loadItems()
  }, [loadItems])

  const filteredItems = useMemo(() => {
    const q = normalize(search)
    if (!q) {
      return items
    }
    return items.filter((item) =>
      [
        item.delivery_note_id,
        item.model_name,
        item.line_name,
        item.component_name,
        item.manufacturer_code,
        item.factory_code,
        item.note_created_at,
      ]
        .map(normalize)
        .join(" ")
        .includes(q),
    )
  }, [items, search])

  async function confirmItem(item: BrigadirItemRow) {
    setConfirmingItemId(item.id)
    const result = await Backend_Request({ item_id: item.id }, "/api/lines/brigadir/confirm")
    setConfirmingItemId(null)

    if (result.result !== "ok") {
      const shortage = Array.isArray(result.data) ? (result.data as WareStockShortage[])[0] : null
      if (shortage) {
        ShowErrorToast(
          `Omborda yetarli emas: kerak ${formatQty(shortage.required_quantity)}, bor ${formatQty(shortage.available_quantity)}`,
        )
        return
      }
      ShowErrorToast(result.error || "Qabul tasdiqlanmadi")
      return
    }

    setItems((current) => current.filter((row) => row.id !== item.id))
    ShowOKToast("Komponent qabul qilindi")
  }

  async function confirmAll() {
    if (items.length === 0) {
      ShowErrorToast("Tasdiqlash uchun pozitsiyalar yo'q")
      return
    }

    setConfirmingAll(true)
    const result = await Backend_Request<{ confirmed_count: number }>(
      {},
      "/api/lines/brigadir/confirm-all",
    )
    setConfirmingAll(false)

    if (result.result !== "ok") {
      const maybeShortages = Array.isArray(result.data) ? (result.data as WareStockShortage[]) : []
      if (maybeShortages.length > 0) {
        setShortages(maybeShortages)
        setShortageDialogOpen(true)
        return
      }
      ShowErrorToast(result.error || "Barchasini tasdiqlash amalga oshmadi")
      return
    }

    setItems([])
    ShowOKToast(`${result.data?.confirmed_count ?? items.length} ta pozitsiya qabul qilindi`)
  }

  return (
    <PageContainer
      title="Brigadir"
      description="Ombordan tayyorlangan komponentlarni liniyaga qabul qilish"
      scrollable
    >
      <Panel>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div className="w-full sm:max-w-md">
            <div className="text-sm font-medium text-muted-foreground">Qidirish</div>
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Nakladnoma / liniya / komponent..."
              className="mt-2 h-11 rounded-xl"
            />
          </div>

          <Button type="button" className="h-11 rounded-xl" disabled={loading} onClick={() => void loadItems()}>
            {loading ? "Yuklanmoqda..." : "Yangilash"}
          </Button>
        </div>
      </Panel>

      <Panel
        title="Qabul qilish uchun komponentlar"
        description={loading ? "Yuklanmoqda..." : `${filteredItems.length} ta`}
        noPadding
      >
        <div className="flex justify-end border-b border-border/60 px-4 py-3">
          <Button
            type="button"
            className="h-10 rounded-xl"
            disabled={loading || confirmingAll || items.length === 0}
            onClick={() => void confirmAll()}
          >
            {confirmingAll ? "Tasdiqlanmoqda..." : "Barchasini tasdiqlash"}
          </Button>
        </div>

        <div className="overflow-x-auto rounded-b-2xl pb-3">
          <Table>
            <TableHeader className="sticky top-0 z-10 bg-muted/90 backdrop-blur">
              <TableRow className="border-b border-border/60 bg-muted/40 hover:bg-muted/40">
                <TableHead className="px-4 py-3 text-sm font-semibold">Nakladnoma</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Sana</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Model</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Liniya</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Komponent</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Kod</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Miqdor</TableHead>
                <TableHead className="w-36 px-4 py-3" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredItems.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="h-24 text-center text-muted-foreground">
                    {loading ? "Yuklanmoqda..." : "Qabul qilish uchun komponentlar yo'q"}
                  </TableCell>
                </TableRow>
              ) : (
                filteredItems.map((item, index) => (
                  <TableRow
                    key={item.id}
                    className={cn(
                      "border-border/40",
                      index % 2 === 0 ? "bg-transparent" : "bg-muted/10",
                    )}
                  >
                    <TableCell className="px-4 py-3 text-sm font-medium tabular-nums">
                      #{item.delivery_note_id}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums">{item.note_created_at}</TableCell>
                    <TableCell className="px-4 py-3 text-sm">{item.model_name || "—"}</TableCell>
                    <TableCell className="px-4 py-3 text-sm">{item.line_name || "—"}</TableCell>
                    <TableCell className="px-4 py-3 text-sm">
                      <div className="font-medium">
                        {item.component_name || `ID ${item.component_id}`}
                      </div>
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums">
                      {item.manufacturer_code || item.factory_code || "—"}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums">
                      {formatQty(item.quantity_actual)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-right">
                      <Button
                        type="button"
                        size="sm"
                        className="h-9 rounded-lg"
                        disabled={confirmingItemId === item.id || confirmingAll}
                        onClick={() => void confirmItem(item)}
                      >
                        <Check className="size-4" />
                        {confirmingItemId === item.id ? "..." : "Tasdiqlash"}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </Panel>

      <Dialog open={shortageDialogOpen} onOpenChange={setShortageDialogOpen}>
        <DialogContent className="max-h-[85svh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>Omborda komponent yetarli emas</DialogTitle>
            <DialogDescription>
              Quyidagi komponentlar bo'yicha qoldiq yetmaydi. Barchasini tasdiqlab bo'lmaydi.
            </DialogDescription>
          </DialogHeader>

          <div className="mt-2 overflow-hidden rounded-xl border border-border/60">
            <Table>
              <TableHeader className="bg-muted/40">
                <TableRow className="border-border/60 hover:bg-transparent">
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Komponent</TableHead>
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Liniya</TableHead>
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Omborda</TableHead>
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Kerak</TableHead>
                  <TableHead className="px-4 py-2.5 text-sm font-semibold text-destructive">
                    Yetmaydi
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {shortages.map((item, index) => (
                  <TableRow key={`${item.component_id}-${item.line_name ?? index}`} className="border-border/40">
                    <TableCell className="px-4 py-3 text-sm">
                      <div className="font-medium">
                        {item.component_name?.trim() || item.manufacturer_code?.trim() || `ID ${item.component_id}`}
                      </div>
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm">{item.line_name?.trim() || "—"}</TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums">
                      {formatQty(item.available_quantity)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums">
                      {formatQty(item.required_quantity)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums text-destructive">
                      {formatQty(item.missing_quantity)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          <DialogFooter className="mt-4">
            <Button type="button" className="rounded-xl" onClick={() => setShortageDialogOpen(false)}>
              Yopish
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PageContainer>
  )
}
