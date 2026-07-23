import { useCallback, useEffect, useMemo, useState } from "react"
import { Link, useNavigate } from "react-router-dom"
import { FileSpreadsheet, Plus, Search } from "lucide-react"
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
import { type WriteoffDocument, writeoffStatusLabel } from "@/pages/writeoff_shared"
import { cn } from "@/lib/utils"

const statusFilters = [
  { value: "", label: "Barchasi" },
  { value: "draft", label: "Qoralama" },
  { value: "pending", label: "Kutilmoqda" },
  { value: "approved", label: "Tasdiqlangan" },
  { value: "rejected", label: "Rad etilgan" },
]

function statusClass(status: string) {
  switch (status) {
    case "draft":
      return "bg-slate-100 text-slate-700"
    case "pending":
      return "bg-amber-100 text-amber-800"
    case "approved":
      return "bg-emerald-100 text-emerald-800"
    case "rejected":
      return "bg-rose-100 text-rose-800"
    default:
      return "bg-muted text-muted-foreground"
  }
}

export default function WriteoffPage() {
  const navigate = useNavigate()
  const [items, setItems] = useState<WriteoffDocument[]>([])
  const [status, setStatus] = useState("")
  const [search, setSearch] = useState("")
  const [creating, setCreating] = useState(false)

  const loadItems = useCallback(async () => {
    const result = await Backend_Request<WriteoffDocument[]>(
      { status, limit: 200 },
      "/api/writeoff/documents/list",
    )
    if (result.result === "ok") {
      setItems(result.data ?? [])
    } else {
      ShowErrorToast(result.error || "Hujjatlar yuklanmadi")
      setItems([])
    }
  }, [status])

  useEffect(() => {
    void loadItems()
  }, [loadItems])

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) {
      return items
    }
    return items.filter((item) =>
      [item.id, item.created_by_name, item.status, item.c_time, item.item_count]
        .map((v) => String(v).toLowerCase())
        .join(" ")
        .includes(q),
    )
  }, [items, search])

  async function createDocument() {
    setCreating(true)
    const result = await Backend_Request<{ document_id: number }>({}, "/api/writeoff/documents/create")
    setCreating(false)
    if (result.result === "ok" && result.data?.document_id) {
      navigate(`/writeoff/${result.data.document_id}`)
      return
    }
    ShowErrorToast(result.error || "Hujjat yaratilmadi")
  }

  return (
    <PageContainer
      title="Hisobdan chiqarish"
      description="Komponent va mahsulotlarni hisobdan chiqarish hujjatlari."
      actions={
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" className="gap-2" asChild>
            <Link to="/writeoff/records">
              <FileSpreadsheet className="size-4" />
              Arxiv
            </Link>
          </Button>
          <Button className="gap-2" onClick={() => void createDocument()} disabled={creating}>
            <Plus className="size-4" />
            {creating ? "Yaratilmoqda..." : "Yangi hujjat"}
          </Button>
        </div>
      }
    >
      <Panel title={`${filtered.length} ta hujjat`}>
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-wrap gap-2">
            {statusFilters.map((filter) => (
              <Button
                key={filter.value || "all"}
                size="sm"
                variant={status === filter.value ? "default" : "outline"}
                onClick={() => setStatus(filter.value)}
              >
                {filter.label}
              </Button>
            ))}
          </div>
          <div className="relative max-w-md flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Qidirish..."
              className="h-10 pl-9"
            />
          </div>
        </div>

        <div className="overflow-auto rounded-xl border border-border/60">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>№</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Muallif</TableHead>
                <TableHead>Qatorlar</TableHead>
                <TableHead>Yaratilgan</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.length ? (
                filtered.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell className="font-medium">#{item.id}</TableCell>
                    <TableCell>
                      <span className={cn("rounded-full px-2.5 py-1 text-xs font-medium", statusClass(item.status))}>
                        {writeoffStatusLabel(item.status)}
                      </span>
                    </TableCell>
                    <TableCell>{item.created_by_name}</TableCell>
                    <TableCell>{item.item_count}</TableCell>
                    <TableCell>{item.c_time}</TableCell>
                    <TableCell className="text-right">
                      <Button variant="outline" size="sm" asChild>
                        <Link to={`/writeoff/${item.id}`}>Ochish</Link>
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={6} className="h-24 text-center text-muted-foreground">
                    Hujjatlar topilmadi
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </Panel>
    </PageContainer>
  )
}
