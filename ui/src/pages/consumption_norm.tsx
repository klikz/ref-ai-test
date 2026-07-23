import { useEffect, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import {
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  type SortingState,
  useReactTable,
} from "@tanstack/react-table"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { DataTable } from "@/components/layout/data-table"
import { PageSearchInput } from "@/components/layout/page-search-input"
import { ShowErrorToast } from "@/components/showToast"

type ModelRow = {
  id: number
  model_nomi?: string
  modeli?: string
  seriya_raqami?: string
  brend?: string
  umumiy_hajmi_l?: string
  item_count?: number
}

function normalizeSearchValue(value: unknown) {
  return String(value ?? "").trim().toLocaleLowerCase()
}

const columnHelper = createColumnHelper<ModelRow>()

export default function ConsumptionNormPage() {
  const navigate = useNavigate()
  const [models, setModels] = useState<ModelRow[]>([])
  const [globalFilter, setGlobalFilter] = useState("")
  const [sorting, setSorting] = useState<SortingState>([])

  async function loadModels() {
    const result = await Backend_Request<ModelRow[]>({}, "/api/production/consumption-norm/models")
    if (result.result === "ok") {
      setModels(result.data ?? [])
    } else {
      ShowErrorToast(result.error || "Modellar yuklanmadi")
    }
  }

  useEffect(() => {
    void loadModels()
  }, [])

  const columns = useMemo(
    () => [
      columnHelper.accessor("modeli", { header: "Modeli" }),
      columnHelper.accessor("seriya_raqami", { header: "Seriya raqami" }),
      columnHelper.accessor("brend", { header: "Brend" }),
      columnHelper.accessor("umumiy_hajmi_l", { header: "Hajmi (L)" }),
      columnHelper.accessor("item_count", {
        header: "Pozitsiyalar",
        cell: ({ getValue }) => getValue() ?? 0,
      }),
    ],
    [],
  )

  const table = useReactTable({
    data: models,
    columns,
    state: { sorting, globalFilter },
    onSortingChange: setSorting,
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    globalFilterFn: (row, _columnId, filterValue) => {
      const query = normalizeSearchValue(filterValue)
      if (!query) {
        return true
      }
      const item = row.original
      return [item.model_nomi, item.modeli, item.seriya_raqami, item.brend, item.umumiy_hajmi_l, item.item_count]
        .map(normalizeSearchValue)
        .join(" ")
        .includes(query)
    },
  })

  return (
    <PageContainer
      title="Sarf normasi"
      description="Modelni tanlang — sarf normasini ko'rish va yuklash"
      fullWidth
      center={<PageSearchInput value={globalFilter} onChange={setGlobalFilter} />}
    >
      <Panel noPadding>
        <div className="overflow-x-auto rounded-b-2xl pb-3">
          <DataTable
            table={table}
            relaxed
            onRowClick={(row) => navigate(`/production/consumption-norm/${row.id}`)}
            emptyMessage={models.length ? "Qidiruv bo'yicha model topilmadi" : "Modellar topilmadi"}
          />
        </div>
      </Panel>
    </PageContainer>
  )
}
