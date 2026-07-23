import { useState } from "react"
import { Link } from "react-router-dom"
import { FileDown, FileUp } from "lucide-react"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ProductionPlanMonthGrid } from "@/components/production_plan_month_grid"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import {
  currentYearMonth,
  downloadPlanExport,
  uploadPlanImport,
} from "./production_plan_shared"

export default function ProductionPlanPage() {
  const [yearMonth, setYearMonth] = useState(currentYearMonth())
  const [exporting, setExporting] = useState(false)
  const [importing, setImporting] = useState(false)
  const [gridRefreshKey, setGridRefreshKey] = useState(0)

  async function handleExport(includeActual: boolean) {
    setExporting(true)
    try {
      await downloadPlanExport(yearMonth, includeActual)
      ShowOKToast("Excel yuklandi")
    } catch (error) {
      ShowErrorToast(error instanceof Error ? error.message : "Export xatolik")
    } finally {
      setExporting(false)
    }
  }

  async function handleImport(file: File | null) {
    if (!file) {
      return
    }
    setImporting(true)
    const result = await uploadPlanImport(yearMonth, file)
    setImporting(false)
    if (result.result !== "ok") {
      const details = result.data?.errors?.map((e) => `${e.sheet} #${e.row}: ${e.message}`).join("\n")
      ShowErrorToast(details ? `${result.error}\n${details}` : result.error || "Import xatolik")
      return
    }
    ShowOKToast(`Import: ${result.data?.imported_rows ?? 0} ta`)
    setGridRefreshKey((k) => k + 1)
  }

  return (
    <PageContainer
      title="Ishlab chiqarish rejasi"
      description="Oylik reja — Excel ko'rinishida tahrirlash yoki import/export."
      fullWidth
      actions={
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" asChild>
            <Link to="/production/shifts">Smena vaqtlari</Link>
          </Button>
          <Button variant="outline" asChild>
            <Link to="/production/plan/dashboard">Dashboard</Link>
          </Button>
          <Button variant="outline" asChild>
            <Link to="/production/plan/report">Reja va bajarilish</Link>
          </Button>
        </div>
      }
    >
      <Panel title="Excel">
        <div className="flex flex-wrap items-end gap-4">
          <div className="space-y-2">
            <Label htmlFor="year-month">Oy</Label>
            <Input
              id="year-month"
              type="month"
              value={yearMonth}
              onChange={(e) => setYearMonth(e.target.value)}
              className="w-44"
            />
          </div>
          <Button variant="outline" disabled={exporting} onClick={() => void handleExport(false)}>
            <FileDown className="size-4" />
            Shablon / reja
          </Button>
          <Button variant="outline" disabled={exporting} onClick={() => void handleExport(true)}>
            <FileDown className="size-4" />
            Reja + fakt
          </Button>
          <label className="inline-flex cursor-pointer items-center gap-2">
            <Button variant="default" disabled={importing} asChild>
              <span>
                <FileUp className="size-4" />
                {importing ? "Import..." : "Import"}
              </span>
            </Button>
            <input
              type="file"
              accept=".xlsx"
              className="hidden"
              onChange={(e) => void handleImport(e.target.files?.[0] ?? null)}
            />
          </label>
        </div>
      </Panel>

      <Panel title="Oylik reja" className="mt-4">
        <ProductionPlanMonthGrid key={`${yearMonth}-${gridRefreshKey}`} yearMonth={yearMonth} />
      </Panel>
    </PageContainer>
  )
}
