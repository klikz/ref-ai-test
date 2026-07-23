import { useEffect, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
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
import { formatQty } from "@/lib/ware-quantity"

type DeliveryNoteRow = {
  id: number
  model_id: number
  model_name: string
  model_quantity: number
  user_id: number
  user_name: string
  created_at: string
  item_count: number
}

function normalize(value: unknown) {
  return String(value ?? "").trim().toLocaleLowerCase()
}

export default function WareOutcomePage() {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [notes, setNotes] = useState<DeliveryNoteRow[]>([])
  const [search, setSearch] = useState("")

  useEffect(() => {
    void loadNotes()
  }, [])

  async function loadNotes() {
    setLoading(true)
    const result = await Backend_Request<DeliveryNoteRow[]>({ limit: 200 }, "/api/ware/delivery-notes")
    setLoading(false)
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Nakladnomalar yuklanmadi")
      return
    }
    setNotes(result.data ?? [])
  }

  const visibleNotes = useMemo(() => notes.filter((n) => n.item_count > 0), [notes])

  const filteredNotes = useMemo(() => {
    const q = normalize(search)
    if (!q) {
      return visibleNotes
    }
    return visibleNotes.filter((n) =>
      [n.id, n.model_name, n.user_name, n.created_at]
        .map(normalize)
        .join(" ")
        .includes(q),
    )
  }, [visibleNotes, search])

  function openNote(note: DeliveryNoteRow) {
    const params = new URLSearchParams({
      id: String(note.id),
      model_name: note.model_name || "",
      model_quantity: formatQty(note.model_quantity),
      created_at: note.created_at || "",
      user_name: note.user_name || "",
    })
    navigate(`/ombor/chiqim/detail?${params.toString()}`)
  }

  return (
    <PageContainer
      title="Ombor chiqim"
      description="Yaratilgan nakladnomalar ro'yxati"
      scrollable
    >
      <Panel>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div className="w-full sm:max-w-md">
            <div className="text-sm font-medium text-muted-foreground">Qidirish</div>
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="ID / model / foydalanuvchi / sana..."
              className="mt-2 h-11 rounded-xl"
            />
          </div>

          <Button type="button" className="h-11 rounded-xl" disabled={loading} onClick={() => void loadNotes()}>
            {loading ? "Yuklanmoqda..." : "Yangilash"}
          </Button>
        </div>
      </Panel>

      <Panel
        title="Nakladnomalar"
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
                    {loading ? "Yuklanmoqda..." : "Nakladnoma topilmadi"}
                  </TableCell>
                </TableRow>
              ) : (
                filteredNotes.map((note, idx) => (
                  <TableRow
                    key={note.id}
                    className={cn(
                      "cursor-pointer border-border/40 hover:bg-muted/30",
                      idx % 2 === 0 ? "bg-transparent" : "bg-muted/10",
                    )}
                    onClick={() => openNote(note)}
                  >
                    <TableCell className="px-4 py-3 text-sm font-medium tabular-nums">
                      #{note.id}
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
    </PageContainer>
  )
}
