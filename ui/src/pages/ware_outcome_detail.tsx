import { useCallback, useEffect, useState } from "react"
import { Check, Trash2 } from "lucide-react"
import { useNavigate, useSearchParams } from "react-router-dom"
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
import { formatQty, parseQty } from "@/lib/ware-quantity"

type DeliveryNoteItemRow = {
  id: number
  delivery_note_id: number
  component_id: number
  line_id: number
  line_name: string
  component_name: string
  manufacturer_code: string
  factory_code: string
  quantity: number
  quantity_actual: number
  is_ready: boolean
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

function parsePositiveInt(value: string | null) {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0
}

export default function WareOutcomeDetailPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const noteId = parsePositiveInt(searchParams.get("id"))
  const modelName = searchParams.get("model_name") ?? ""
  const modelQuantity = searchParams.get("model_quantity") ?? ""
  const createdAt = searchParams.get("created_at") ?? ""
  const userName = searchParams.get("user_name") ?? ""

  const [items, setItems] = useState<DeliveryNoteItemRow[]>([])
  const [loading, setLoading] = useState(false)
  const [savingItemId, setSavingItemId] = useState<number | null>(null)
  const [confirmingItemId, setConfirmingItemId] = useState<number | null>(null)
  const [unconfirmingItemId, setUnconfirmingItemId] = useState<number | null>(null)
  const [deletingItemId, setDeletingItemId] = useState<number | null>(null)
  const [confirmingAll, setConfirmingAll] = useState(false)
  const [unconfirmingAll, setUnconfirmingAll] = useState(false)
  const [shortageDialogOpen, setShortageDialogOpen] = useState(false)
  const [shortages, setShortages] = useState<WareStockShortage[]>([])
  const [draftQty, setDraftQty] = useState<Record<number, string>>({})

  const loadItems = useCallback(async () => {
    if (!noteId) {
      return
    }

    setLoading(true)
    const result = await Backend_Request<DeliveryNoteItemRow[]>(
      { delivery_note_id: noteId },
      "/api/ware/delivery-notes/items",
    )
    setLoading(false)

    if (result.result === "ok") {
      const loaded = result.data ?? []
      setItems(loaded)
      setDraftQty(
        Object.fromEntries(
          loaded.map((item) => [item.id, formatQty(item.quantity_actual ?? item.quantity)]),
        ),
      )
      return
    }

    ShowErrorToast(result.error || "Nakladnoy tarkibi yuklanmadi")
    setItems([])
    setDraftQty({})
  }, [noteId])

  useEffect(() => {
    void loadItems()
  }, [loadItems])

  function updateDraftQty(itemId: number, value: string) {
    setDraftQty((current) => ({ ...current, [itemId]: value }))
  }

  async function saveQuantity(item: DeliveryNoteItemRow) {
    if (item.is_ready) {
      return
    }

    const quantityActual = parseQty(draftQty[item.id] ?? "")
    if (quantityActual <= 0) {
      ShowErrorToast("Miqdor 0 dan katta bo'lishi kerak")
      setDraftQty((current) => ({
        ...current,
        [item.id]: formatQty(item.quantity_actual ?? item.quantity),
      }))
      return
    }

    if (quantityActual === (item.quantity_actual ?? item.quantity)) {
      return
    }

    setSavingItemId(item.id)
    const result = await Backend_Request(
      { item_id: item.id, quantity_actual: quantityActual },
      "/api/ware/delivery-notes/items/update",
    )
    setSavingItemId(null)

    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Miqdor saqlanmadi")
      setDraftQty((current) => ({
        ...current,
        [item.id]: formatQty(item.quantity_actual ?? item.quantity),
      }))
      return
    }

    setItems((current) =>
      current.map((row) =>
        row.id === item.id ? { ...row, quantity_actual: quantityActual } : row,
      ),
    )
    ShowOKToast("Miqdor saqlandi")
  }

  async function confirmItem(item: DeliveryNoteItemRow) {
    if (item.is_ready) {
      return
    }

    const quantityActual = parseQty(draftQty[item.id] ?? formatQty(item.quantity_actual ?? item.quantity))
    if (quantityActual <= 0) {
      ShowErrorToast("Miqdor 0 dan katta bo'lishi kerak")
      return
    }

    setConfirmingItemId(item.id)
    const result = await Backend_Request(
      { item_id: item.id, quantity_actual: quantityActual },
      "/api/ware/delivery-notes/items/confirm",
    )
    setConfirmingItemId(null)

    if (result.result !== "ok") {
      const shortage = Array.isArray(result.data) ? (result.data as WareStockShortage[])[0] : null
      if (shortage) {
        ShowErrorToast(
          `Omborda yetarli emas: kerak ${formatQty(shortage.required_quantity)}, bor ${formatQty(shortage.available_quantity)}`,
        )
        return
      }
      ShowErrorToast(result.error || "Tasdiqlash amalga oshmadi")
      return
    }

    setItems((current) =>
      current.map((row) =>
        row.id === item.id
          ? { ...row, quantity_actual: quantityActual, is_ready: true }
          : row,
      ),
    )
    setDraftQty((current) => ({ ...current, [item.id]: formatQty(quantityActual) }))
    ShowOKToast("Komponent tayyor deb belgilandi")
  }

  async function unconfirmItem(item: DeliveryNoteItemRow) {
    if (!item.is_ready) {
      return
    }

    setUnconfirmingItemId(item.id)
    const result = await Backend_Request(
      { item_id: item.id },
      "/api/ware/delivery-notes/items/unconfirm",
    )
    setUnconfirmingItemId(null)

    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Holat o'zgartirilmadi")
      return
    }

    setItems((current) =>
      current.map((row) => (row.id === item.id ? { ...row, is_ready: false } : row)),
    )
    ShowOKToast("Komponent tayyor emas deb belgilandi")
  }

  async function deleteItem(item: DeliveryNoteItemRow) {
    if (item.is_ready) {
      return
    }

    setDeletingItemId(item.id)
    const result = await Backend_Request({ item_id: item.id }, "/api/ware/delivery-notes/items/delete")
    setDeletingItemId(null)

    if (result.result !== "ok") {
      ShowErrorToast(result.error || "O'chirish amalga oshmadi")
      return
    }

    setItems((current) => current.filter((row) => row.id !== item.id))
    setDraftQty((current) => {
      const next = { ...current }
      delete next[item.id]
      return next
    })
    ShowOKToast("Komponent o'chirildi")
  }

  const pendingCount = items.filter((item) => !item.is_ready).length
  const readyCount = items.filter((item) => item.is_ready).length
  const isBulkBusy = confirmingAll || unconfirmingAll

  async function confirmAll() {
    const pendingItems = items.filter((item) => !item.is_ready)
    if (pendingItems.length === 0) {
      ShowErrorToast("Tasdiqlash uchun pozitsiyalar yo'q")
      return
    }

    const payload = pendingItems.map((item) => {
      const quantityActual = parseQty(
        draftQty[item.id] ?? formatQty(item.quantity_actual ?? item.quantity),
      )
      return { item_id: item.id, quantity_actual: quantityActual }
    })

    if (payload.some((item) => item.quantity_actual <= 0)) {
      ShowErrorToast("Barcha miqdorlar 0 dan katta bo'lishi kerak")
      return
    }

    setConfirmingAll(true)
    const result = await Backend_Request<{ confirmed_count: number }>(
      { delivery_note_id: noteId, items: payload },
      "/api/ware/delivery-notes/items/confirm-all",
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

    const qtyById = new Map(payload.map((item) => [item.item_id, item.quantity_actual]))
    setItems((current) =>
      current.map((row) =>
        !row.is_ready
          ? { ...row, quantity_actual: qtyById.get(row.id) ?? row.quantity_actual, is_ready: true }
          : row,
      ),
    )
    setDraftQty((current) => {
      const next = { ...current }
      for (const item of pendingItems) {
        next[item.id] = formatQty(qtyById.get(item.id) ?? item.quantity_actual ?? item.quantity)
      }
      return next
    })
    ShowOKToast(`${result.data?.confirmed_count ?? pendingItems.length} ta pozitsiya tasdiqlandi`)
  }

  async function unconfirmAll() {
    if (readyCount === 0) {
      ShowErrorToast("Bekor qilish uchun tayyor pozitsiyalar yo'q")
      return
    }

    setUnconfirmingAll(true)
    const result = await Backend_Request<{ unconfirmed_count: number }>(
      { delivery_note_id: noteId },
      "/api/ware/delivery-notes/items/unconfirm-all",
    )
    setUnconfirmingAll(false)

    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Barchasini bekor qilish amalga oshmadi")
      return
    }

    setItems((current) => current.map((row) => ({ ...row, is_ready: false })))
    ShowOKToast(`${result.data?.unconfirmed_count ?? readyCount} ta pozitsiya bekor qilindi`)
  }

  const summary = [
    createdAt,
    modelName || "—",
    modelQuantity ? `Miqdor: ${modelQuantity}` : "",
    userName ? `Foydalanuvchi: ${userName}` : "",
  ]
    .filter(Boolean)
    .join(" • ")

  return (
    <PageContainer
      title={noteId ? `Nakladnoy #${noteId}` : "Nakladnoy"}
      description={noteId ? summary : "Nakladnoy parametrlari topilmadi"}
      scrollable
      actions={
        <Button variant="outline" className="h-10" onClick={() => navigate("/ombor/chiqim")}>
          Orqaga
        </Button>
      }
    >
      <Panel
        title="Komponentlar"
        description={loading ? "Yuklanmoqda..." : `${items.length} ta pozitsiya`}
        noPadding
      >
        {!noteId ? (
          <div className="p-6 text-center text-muted-foreground">Nakladnoy ID topilmadi</div>
        ) : (
          <>
            <div className="flex flex-wrap items-center justify-end gap-2 border-b border-border/60 px-4 py-3">
              <Button
                type="button"
                variant="outline"
                className="h-10 rounded-xl"
                disabled={loading || unconfirmingAll || readyCount === 0}
                onClick={() => void unconfirmAll()}
              >
                {unconfirmingAll ? "Bekor qilinmoqda..." : "Barchasini bekor qilish"}
              </Button>
              <Button
                type="button"
                className="h-10 rounded-xl"
                disabled={loading || confirmingAll || pendingCount === 0}
                onClick={() => void confirmAll()}
              >
                {confirmingAll ? "Tasdiqlanmoqda..." : "Barchasini tasdiqlash"}
              </Button>
            </div>

            <div className="overflow-x-auto rounded-b-2xl pb-3">
            <Table>
              <TableHeader className="sticky top-0 z-10 bg-muted/90 backdrop-blur">
                <TableRow className="border-b border-border/60 bg-muted/40 hover:bg-muted/40">
                  <TableHead className="px-4 py-3 text-sm font-semibold">Komponent</TableHead>
                  <TableHead className="px-4 py-3 text-sm font-semibold">Kod</TableHead>
                  <TableHead className="px-4 py-3 text-sm font-semibold">Liniya</TableHead>
                  <TableHead className="px-4 py-3 text-sm font-semibold">Miqdor</TableHead>
                  <TableHead className="px-4 py-3 text-sm font-semibold">Miqdor 2</TableHead>
                  <TableHead className="px-4 py-3 text-sm font-semibold">Holat</TableHead>
                  <TableHead className="w-44 px-4 py-3" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                      Yuklanmoqda...
                    </TableCell>
                  </TableRow>
                ) : items.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                      Pozitsiyalar topilmadi
                    </TableCell>
                  </TableRow>
                ) : (
                  items.map((item, index) => {
                    const isBusy =
                      isBulkBusy ||
                      savingItemId === item.id ||
                      confirmingItemId === item.id ||
                      unconfirmingItemId === item.id ||
                      deletingItemId === item.id

                    return (
                      <TableRow
                        key={item.id}
                        className={cn(
                          "border-border/40",
                          index % 2 === 0 ? "bg-transparent" : "bg-muted/20",
                          item.is_ready && "bg-emerald-50/40 dark:bg-emerald-950/20",
                        )}
                      >
                        <TableCell className="px-4 py-3 text-sm">
                          <div className="font-medium">
                            {item.component_name || `ID ${item.component_id}`}
                          </div>
                        </TableCell>
                        <TableCell className="px-4 py-3 text-sm">
                          <div className="tabular-nums">
                            {item.manufacturer_code || item.factory_code || "—"}
                          </div>
                          {item.factory_code ? (
                            <div className="text-xs text-muted-foreground tabular-nums">
                              {item.factory_code}
                            </div>
                          ) : null}
                        </TableCell>
                        <TableCell className="px-4 py-3 text-sm">{item.line_name || "—"}</TableCell>
                        <TableCell className="px-4 py-3 text-sm tabular-nums">
                          {formatQty(item.quantity)}
                        </TableCell>
                        <TableCell className="px-4 py-3 text-sm">
                          <Input
                            value={draftQty[item.id] ?? formatQty(item.quantity_actual ?? item.quantity)}
                            onChange={(event) => updateDraftQty(item.id, event.target.value)}
                            onBlur={() => void saveQuantity(item)}
                            disabled={item.is_ready || isBusy}
                            inputMode="decimal"
                            className={cn(
                              "h-9 w-28 rounded-lg tabular-nums",
                              item.is_ready && "cursor-not-allowed opacity-60",
                            )}
                          />
                        </TableCell>
                        <TableCell className="px-4 py-3 text-sm">
                          <button
                            type="button"
                            disabled={!item.is_ready || isBusy}
                            onClick={() => void unconfirmItem(item)}
                            className={cn(
                              "inline-flex rounded-full px-2.5 py-1 text-xs font-medium transition-colors",
                              item.is_ready
                                ? "cursor-pointer bg-emerald-100 text-emerald-800 hover:bg-emerald-200 dark:bg-emerald-950 dark:text-emerald-300 dark:hover:bg-emerald-900"
                                : "cursor-default bg-muted text-muted-foreground",
                              isBusy && "pointer-events-none opacity-60",
                            )}
                          >
                            {unconfirmingItemId === item.id
                              ? "..."
                              : item.is_ready
                                ? "Tayyor"
                                : "Tayyor emas"}
                          </button>
                        </TableCell>
                        <TableCell className="px-4 py-3 text-right">
                          <div className="flex justify-end gap-2">
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              className="h-9 rounded-lg"
                              disabled={item.is_ready || isBusy}
                              onClick={() => void confirmItem(item)}
                            >
                              <Check className="size-4" />
                              {confirmingItemId === item.id ? "..." : "Tasdiqlash"}
                            </Button>
                            <Button
                              type="button"
                              size="icon"
                              variant="ghost"
                              className="text-destructive hover:text-destructive"
                              disabled={item.is_ready || isBusy}
                              onClick={() => void deleteItem(item)}
                            >
                              <Trash2 className="size-4" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
            </div>
          </>
        )}
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
