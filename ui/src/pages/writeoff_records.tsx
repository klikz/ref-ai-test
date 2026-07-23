import { useCallback, useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { Search } from "lucide-react"
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
import { type WriteoffRecord } from "@/pages/writeoff_shared"
import { formatQty } from "@/lib/ware-quantity"

export default function WriteoffRecordsPage() {
  const [records, setRecords] = useState<WriteoffRecord[]>([])
  const [search, setSearch] = useState("")

  const loadRecords = useCallback(async () => {
    const result = await Backend_Request<WriteoffRecord[]>({ limit: 500 }, "/api/writeoff/records/list")
    if (result.result === "ok") {
      setRecords(result.data ?? [])
    } else {
      ShowErrorToast(result.error || "Yozuvlar yuklanmadi")
      setRecords([])
    }
  }, [])

  useEffect(() => {
    void loadRecords()
  }, [loadRecords])

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) {
      return records
    }
    return records.filter((row) =>
      [
        row.document_id,
        row.line_name,
        row.item_label,
        row.serial,
        row.written_off_by_name,
        row.written_off_at,
        row.comment,
      ]
        .map((v) => String(v).toLowerCase())
        .join(" ")
        .includes(q),
    )
  }, [records, search])

  return (
    <PageContainer
      title="Hisobdan chiqarish arxivi"
      description="Tasdiqlangan hisobdan chiqarish yozuvlari."
      actions={
        <Button variant="outline" asChild>
          <Link to="/writeoff">Hujjatlar</Link>
        </Button>
      }
    >
      <Panel title={`${filtered.length} ta yozuv`}>
        <div className="mb-4">
          <div className="relative max-w-md">
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
                <TableHead>Hujjat</TableHead>
                <TableHead>Liniya</TableHead>
                <TableHead>Pozitsiya</TableHead>
                <TableHead>Serial</TableHead>
                <TableHead>Miqdor</TableHead>
                <TableHead>Balans oldin/keyin</TableHead>
                <TableHead>Mas'ul</TableHead>
                <TableHead>Vaqt</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.length ? (
                filtered.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>
                      <Link className="text-primary hover:underline" to={`/writeoff/${row.document_id}`}>
                        #{row.document_id}
                      </Link>
                    </TableCell>
                    <TableCell>{row.line_name}</TableCell>
                    <TableCell>{row.item_label}</TableCell>
                    <TableCell>{row.serial || "—"}</TableCell>
                    <TableCell>{formatQty(row.quantity)}</TableCell>
                    <TableCell>
                      {row.item_type === "component"
                        ? `${formatQty(row.balance_before)} → ${formatQty(row.balance_after)}`
                        : "—"}
                    </TableCell>
                    <TableCell>{row.written_off_by_name}</TableCell>
                    <TableCell>{row.written_off_at}</TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={8} className="h-24 text-center text-muted-foreground">
                    Yozuvlar topilmadi
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
