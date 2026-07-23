import { useCallback, useEffect, useMemo, useState } from "react"
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
import { Search } from "lucide-react"

type ModelRow = {
  id: number
  seriya_raqami?: string
  modeli?: string
  model_nomi?: string
  qisqa_nomi?: string
  rangi?: string
  gs1_ean13?: string
  status?: boolean
  item_count?: number
}

type NormItem = {
  id: number
  model_id: number
  sort_order: number
  group_level: number
  component_id: number
  manufacturer_code?: string
  factory_code?: string
  odoo_code?: string
  standard_name_uz?: string
  quantity: number
  consume_line_id?: number
  consume_line_name?: string
  receive_line_id?: number
  receive_line_name?: string
}

type DisplayRow = {
  item: NormItem
  index: number
  level: number
}

type NormGroup = {
  parent: NormItem
  children: NormItem[]
}

type LineRow = {
  line_id: number
  name: string
}

const FIN_PRESS_LINE_ID = 8

function normalizeSearchValue(value: unknown) {
  return String(value ?? "").trim().toLocaleLowerCase()
}

function componentLabel(item: NormItem) {
  const code = item.factory_code?.trim() || item.odoo_code?.trim() || item.manufacturer_code?.trim() || ""
  const name = item.standard_name_uz?.trim() || ""
  if (code && name) {
    return `${code} — ${name}`
  }
  return code || name || `ID ${item.component_id}`
}

function toDisplayRows(items: NormItem[]): DisplayRow[] {
  return items.map((item, index) => ({
    item,
    index,
    level: item.group_level,
  }))
}

function rowHasChildren(rows: DisplayRow[], index: number) {
  if (index >= rows.length - 1) {
    return false
  }
  return rows[index + 1].level > rows[index].level
}

function getDirectChildren(rows: DisplayRow[], parentIndex: number): NormItem[] {
  const level = rows[parentIndex].level
  const children: NormItem[] = []
  for (let childIndex = parentIndex + 1; childIndex < rows.length; childIndex++) {
    if (rows[childIndex].level <= level) {
      break
    }
    if (rows[childIndex].level === level + 1) {
      children.push(rows[childIndex].item)
    }
  }
  return children
}

function buildNormGroups(items: NormItem[]): NormGroup[] {
  const rows = toDisplayRows(items)
  const groups: NormGroup[] = []

  for (let index = 0; index < rows.length; index++) {
    if (!rowHasChildren(rows, index)) {
      continue
    }
    groups.push({
      parent: rows[index].item,
      children: getDirectChildren(rows, index),
    })
  }

  return groups
}

function normItemHasFinPressLine(item: NormItem, finPressLineId: number) {
  return item.receive_line_id === finPressLineId || item.consume_line_id === finPressLineId
}

function filterNormGroups(groups: NormGroup[], finPressLineId: number) {
  return groups
    .map((group) => ({
      parent: group.parent,
      children: group.children.filter((child) => child.consume_line_id === finPressLineId),
    }))
    .filter(
      (group) =>
        group.parent.receive_line_id === finPressLineId ||
        group.children.length > 0,
    )
}

function modelLabel(model: ModelRow) {
  return model.modeli || model.qisqa_nomi || `ID ${model.id}`
}

export default function TestModelsPage() {
  const [models, setModels] = useState<ModelRow[]>([])
  const [loading, setLoading] = useState(false)
  const [search, setSearch] = useState("")
  const [selectedModel, setSelectedModel] = useState<ModelRow | null>(null)
  const [normItems, setNormItems] = useState<NormItem[]>([])
  const [normLoading, setNormLoading] = useState(false)
  const [finPressLineName, setFinPressLineName] = useState("Fin Press")
  const [finPressOnly, setFinPressOnly] = useState(false)
  const [finPressModelIds, setFinPressModelIds] = useState<Set<number>>(() => new Set())
  const [finPressFilterLoading, setFinPressFilterLoading] = useState(false)

  const loadModels = useCallback(async () => {
    setLoading(true)
    const [modelsResult, normResult] = await Promise.all([
      Backend_Request<ModelRow[]>({}, "/api/tech/models/all"),
      Backend_Request<{ id: number; item_count?: number }[]>({}, "/api/production/consumption-norm/models"),
    ])
    setLoading(false)

    if (modelsResult.result !== "ok") {
      ShowErrorToast(modelsResult.error || "Modellar yuklanmadi")
      return
    }

    const countById = new Map(
      (normResult.result === "ok" ? normResult.data ?? [] : []).map(
        (row) => [row.id, row.item_count ?? 0] as const,
      ),
    )

    setModels(
      (modelsResult.data ?? []).map((model) => ({
        ...model,
        item_count: countById.get(model.id) ?? 0,
      })),
    )
  }, [])

  const loadNormItems = useCallback(async (modelId: number) => {
    setNormLoading(true)
    const result = await Backend_Request<NormItem[]>(
      { model_id: modelId },
      "/api/production/consumption-norm/items",
    )
    setNormLoading(false)
    if (result.result === "ok") {
      setNormItems(result.data ?? [])
    } else {
      setNormItems([])
      ShowErrorToast(result.error || "Sarf normasi yuklanmadi")
    }
  }, [])

  const loadFinPressLine = useCallback(async () => {
    const result = await Backend_Request<LineRow[]>({}, "/api/lines/all")
    if (result.result !== "ok") {
      return
    }
    const finPressLine = (result.data ?? []).find((line) => line.line_id === FIN_PRESS_LINE_ID)
    if (finPressLine?.name) {
      setFinPressLineName(finPressLine.name)
    }
  }, [])

  const loadFinPressModelIds = useCallback(async (modelRows: ModelRow[]) => {
    const candidates = modelRows.filter((model) => (model.item_count ?? 0) > 0)
    if (candidates.length === 0) {
      setFinPressModelIds(new Set())
      return
    }

    setFinPressFilterLoading(true)
    const matchedIds = await Promise.all(
      candidates.map(async (model) => {
        const result = await Backend_Request<NormItem[]>(
          { model_id: model.id },
          "/api/production/consumption-norm/items",
        )
        if (result.result !== "ok") {
          return null
        }
        const hasFinPress = (result.data ?? []).some((item) =>
          normItemHasFinPressLine(item, FIN_PRESS_LINE_ID),
        )
        return hasFinPress ? model.id : null
      }),
    )
    setFinPressFilterLoading(false)
    setFinPressModelIds(new Set(matchedIds.filter((id): id is number => id !== null)))
  }, [])

  useEffect(() => {
    void loadModels()
    void loadFinPressLine()
  }, [loadFinPressLine, loadModels])

  useEffect(() => {
    if (!finPressOnly || models.length === 0) {
      return
    }
    void loadFinPressModelIds(models)
  }, [finPressOnly, loadFinPressModelIds, models])

  useEffect(() => {
    if (!selectedModel?.id) {
      setNormItems([])
      return
    }
    void loadNormItems(selectedModel.id)
  }, [loadNormItems, selectedModel?.id])

  const filteredModels = useMemo(() => {
    const query = normalizeSearchValue(search)
    return models.filter((model) => {
      if (finPressOnly && !finPressModelIds.has(model.id)) {
        return false
      }
      if (!query) {
        return true
      }
      return [model.id, model.modeli, model.qisqa_nomi, model.seriya_raqami, model.rangi, model.gs1_ean13, model.item_count]
        .map(normalizeSearchValue)
        .join(" ")
        .includes(query)
    })
  }, [finPressModelIds, finPressOnly, models, search])

  const normGroups = useMemo(() => {
    const groups = buildNormGroups(normItems)
    if (!finPressOnly) {
      return groups
    }
    return filterNormGroups(groups, FIN_PRESS_LINE_ID)
  }, [finPressOnly, normItems])

  return (
    <PageContainer
      title="Test: modellar"
      description="Modellar va sarf normasi tuzilmasi (i/ch joyi → ishlatilish joyi)"
      scrollable
    >
      <Panel title="Modellar" description={loading ? "Yuklanmoqda..." : `${filteredModels.length} ta model`}>
        <div className="mb-4 flex flex-wrap items-center gap-3">
          <div className="relative max-w-md min-w-[220px] flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Qidirish..."
              className="pl-9"
            />
          </div>
          <Button
            type="button"
            variant={finPressOnly ? "default" : "outline"}
            className="h-10 shrink-0"
            disabled={finPressFilterLoading}
            onClick={() => setFinPressOnly((current) => !current)}
          >
            {finPressFilterLoading
              ? "Fin Press..."
              : `${finPressLineName} (id ${FIN_PRESS_LINE_ID})`}
          </Button>
          <span className="text-sm text-muted-foreground">
            {finPressOnly ? `${filteredModels.length} / ${models.length}` : `Jami: ${models.length}`}
          </span>
        </div>

        <div className="overflow-auto rounded-xl border border-border/60">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">ID</TableHead>
                <TableHead>Modeli</TableHead>
                <TableHead>Model nomi</TableHead>
                <TableHead>Seriya</TableHead>
                <TableHead>Rang</TableHead>
                <TableHead className="text-right">Sarf normasi</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredModels.length ? (
                filteredModels.map((model) => {
                  const isSelected = selectedModel?.id === model.id
                  return (
                    <TableRow
                      key={model.id}
                      onClick={() => setSelectedModel(model)}
                      className={cn(
                        "cursor-pointer",
                        isSelected && "bg-primary/10 hover:bg-primary/15",
                      )}
                    >
                      <TableCell className="tabular-nums">{model.id}</TableCell>
                      <TableCell className="font-medium">{model.modeli || "—"}</TableCell>
                      <TableCell>{model.qisqa_nomi || "—"}</TableCell>
                      <TableCell>{model.seriya_raqami || "—"}</TableCell>
                      <TableCell>{model.rangi || "—"}</TableCell>
                      <TableCell className="text-right tabular-nums font-medium">
                        {model.item_count ?? 0}
                      </TableCell>
                      <TableCell>
                        <span
                          className={cn(
                            "rounded-full px-2 py-0.5 text-xs font-medium",
                            model.status === false
                              ? "bg-destructive/10 text-destructive"
                              : "bg-emerald-500/10 text-emerald-700",
                          )}
                        >
                          {model.status === false ? "Nofaol" : "Faol"}
                        </span>
                      </TableCell>
                    </TableRow>
                  )
                })
              ) : (
                <TableRow>
                  <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                    {loading ? "Yuklanmoqda..." : "Model topilmadi"}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </Panel>

      {selectedModel ? (
        <Panel
          title={`Sarf normasi: ${modelLabel(selectedModel)}`}
          description={
            normLoading
              ? "Yuklanmoqda..."
              : `${normItems.length} ta komponent, ${normGroups.length} ta guruh`
          }
        >
          {normLoading ? (
            <p className="text-sm text-muted-foreground">Yuklanmoqda...</p>
          ) : normItems.length === 0 ? (
            <p className="text-sm text-muted-foreground">Bu model uchun sarf normasi yo&apos;q</p>
          ) : normGroups.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              {finPressOnly
                ? `Bu modelda ${finPressLineName} liniyasi bo'yicha guruhlar topilmadi`
                : "Guruhlangan komponentlar topilmadi (faqat bitta darajadagi pozitsiyalar)"}
            </p>
          ) : (
            <div className="space-y-4">
              {normGroups.map((group) => (
                <div
                  key={group.parent.id}
                  className="overflow-hidden rounded-xl border border-border/60"
                >
                  <div className="border-b bg-muted/30 px-4 py-3">
                    <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                      i/ch joyi
                    </p>
                    <div className="mt-1 flex flex-wrap items-baseline justify-between gap-2">
                      <p className="font-medium">{componentLabel(group.parent)}</p>
                      <p className="text-sm text-muted-foreground">
                        Miqdor: <span className="tabular-nums">{formatQty(group.parent.quantity)}</span>
                      </p>
                    </div>
                    <p className="mt-1 text-sm">
                      <span className="text-muted-foreground">Liniya:</span>{" "}
                      <span className="font-medium">
                        {group.parent.receive_line_name?.trim() || "—"}
                      </span>
                    </p>
                  </div>

                  {group.children.length ? (
                    <Table>
                      <TableHeader>
                        <TableRow className="hover:bg-transparent">
                          <TableHead>Komponent (ishlatilish joyi)</TableHead>
                          <TableHead className="w-36">Miqdor</TableHead>
                          <TableHead className="w-48">Ishlatilish joyi</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {group.children.map((child) => (
                          <TableRow key={child.id}>
                            <TableCell className="text-sm">{componentLabel(child)}</TableCell>
                            <TableCell className="tabular-nums text-sm">
                              {formatQty(child.quantity)}
                            </TableCell>
                            <TableCell className="text-sm">
                              {child.consume_line_name?.trim() || "—"}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  ) : (
                    <p className="px-4 py-3 text-sm text-muted-foreground">Podkomponentlar yo&apos;q</p>
                  )}
                </div>
              ))}
            </div>
          )}
        </Panel>
      ) : (
        <Panel>
          <p className="text-sm text-muted-foreground">
            Modelni tanlang — pastda i/ch joyi va ishlatilish joyi bo&apos;yicha tuzilma ko&apos;rsatiladi
          </p>
        </Panel>
      )}
    </PageContainer>
  )
}
