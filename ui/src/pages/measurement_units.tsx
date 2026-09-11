import { useDeferredValue, useEffect, useMemo, useState } from "react"
import { Search } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
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

type MeasurementUnit = {
  id: number
  name: string
}

function normalizeSearchValue(value: unknown) {
  return String(value ?? "").trim().toLocaleLowerCase()
}

export default function MeasurementUnitsPage() {
  const [items, setItems] = useState<MeasurementUnit[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState("")
  const deferredSearch = useDeferredValue(search)

  useEffect(() => {
    void (async () => {
      setLoading(true)
      const result = await Backend_Request<MeasurementUnit[]>({}, "/api/production/components/units")
      setLoading(false)
      if (result.result === "ok") {
        setItems(result.data ?? [])
      } else {
        setItems([])
        ShowErrorToast(result.error || "O'lchov birliklari yuklanmadi")
      }
    })()
  }, [])

  const filteredItems = useMemo(() => {
    const query = normalizeSearchValue(deferredSearch)
    if (!query) {
      return items
    }
    return items.filter((item) => {
      return (
        normalizeSearchValue(item.id).includes(query) ||
        normalizeSearchValue(item.name).includes(query)
      )
    })
  }, [deferredSearch, items])

  return (
    <PageContainer
      title="O'lchov birliklari"
      description="production.measurement_units — faqat ko'rish"
      fullWidth
    >
      <Panel title="Ro'yxat">
        <div className="mb-4 max-w-md">
          <div className="relative">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="ID yoki nom bo'yicha qidirish..."
              className="h-11 rounded-xl pl-9"
            />
          </div>
        </div>

        <div className="overflow-x-auto rounded-xl border border-border/60">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-24">ID</TableHead>
                <TableHead>Nomi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredItems.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className="tabular-nums">{item.id}</TableCell>
                  <TableCell className="font-medium">{item.name}</TableCell>
                </TableRow>
              ))}
              {!loading && filteredItems.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={2} className="h-20 text-center text-muted-foreground">
                    Ma&apos;lumot yo&apos;q
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </div>
      </Panel>
    </PageContainer>
  )
}
