import { useCallback, useEffect, useState } from "react"
import { Clock, Save } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"

type ShiftSettings = {
  shift1_start: string
  shift1_end: string
  shift2_start: string
  shift2_end: string
}

type CurrentShift = {
  shift_no: number
  plan_date: string
  start_time: string
  end_time: string
  label: string
}

type ShiftsResponse = {
  settings: ShiftSettings
  current: CurrentShift
}

const EMPTY_SETTINGS: ShiftSettings = {
  shift1_start: "08:00",
  shift1_end: "20:00",
  shift2_start: "20:00",
  shift2_end: "08:00",
}

export default function ProductionShiftsPage() {
  const [settings, setSettings] = useState<ShiftSettings>(EMPTY_SETTINGS)
  const [current, setCurrent] = useState<CurrentShift | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    const result = await Backend_Request<ShiftsResponse>({}, "/api/production/shifts/get")
    setLoading(false)
    if (result.result === "ok" && result.data) {
      setSettings({
        shift1_start: result.data.settings?.shift1_start || EMPTY_SETTINGS.shift1_start,
        shift1_end: result.data.settings?.shift1_end || EMPTY_SETTINGS.shift1_end,
        shift2_start: result.data.settings?.shift2_start || EMPTY_SETTINGS.shift2_start,
        shift2_end: result.data.settings?.shift2_end || EMPTY_SETTINGS.shift2_end,
      })
      setCurrent(result.data.current ?? null)
    } else {
      ShowErrorToast(result.error || "Smena sozlamalari yuklanmadi")
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  async function handleSave() {
    setSaving(true)
    const result = await Backend_Request<ShiftsResponse>(settings, "/api/production/shifts/save")
    setSaving(false)
    if (result.result === "ok" && result.data) {
      setSettings({
        shift1_start: result.data.settings.shift1_start,
        shift1_end: result.data.settings.shift1_end,
        shift2_start: result.data.settings.shift2_start,
        shift2_end: result.data.settings.shift2_end,
      })
      setCurrent(result.data.current ?? null)
      ShowOKToast("Smena vaqtlari saqlandi")
    } else {
      ShowErrorToast(result.error || "Saqlashda xatolik")
    }
  }

  return (
    <PageContainer
      title="Smena vaqtlari"
      description="Barcha liniyalar uchun 1-sm va 2-sm smena boshlanish/tugash vaqtlari"
    >
      <div className="grid max-w-3xl gap-4">
        {current?.shift_no ? (
          <Panel title="Hozirgi smena">
            <div className="flex items-center gap-3 text-base">
              <Clock className="size-5 text-primary" />
              <div>
                <p className="font-semibold">
                  {current.label || `${current.shift_no}-sm smena`}
                </p>
                <p className="text-sm text-muted-foreground">
                  {current.plan_date} · {current.start_time}–{current.end_time}
                </p>
              </div>
            </div>
          </Panel>
        ) : null}

        <Panel title="Smena sozlamalari">
          <div className="grid gap-6 sm:grid-cols-2">
            <div className="space-y-3 rounded-xl border border-border/60 p-4">
              <h3 className="font-semibold">1-sm smena</h3>
              <div className="space-y-2">
                <Label htmlFor="shift1-start">Boshlanish</Label>
                <Input
                  id="shift1-start"
                  type="time"
                  value={settings.shift1_start}
                  disabled={loading || saving}
                  onChange={(e) => setSettings((s) => ({ ...s, shift1_start: e.target.value }))}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="shift1-end">Tugash</Label>
                <Input
                  id="shift1-end"
                  type="time"
                  value={settings.shift1_end}
                  disabled={loading || saving}
                  onChange={(e) => setSettings((s) => ({ ...s, shift1_end: e.target.value }))}
                />
              </div>
            </div>

            <div className="space-y-3 rounded-xl border border-border/60 p-4">
              <h3 className="font-semibold">2-sm smena</h3>
              <div className="space-y-2">
                <Label htmlFor="shift2-start">Boshlanish</Label>
                <Input
                  id="shift2-start"
                  type="time"
                  value={settings.shift2_start}
                  disabled={loading || saving}
                  onChange={(e) => setSettings((s) => ({ ...s, shift2_start: e.target.value }))}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="shift2-end">Tugash</Label>
                <Input
                  id="shift2-end"
                  type="time"
                  value={settings.shift2_end}
                  disabled={loading || saving}
                  onChange={(e) => setSettings((s) => ({ ...s, shift2_end: e.target.value }))}
                />
              </div>
              <p className="text-xs text-muted-foreground">
                Tungi smena uchun tugash vaqti keyingi kun bo&apos;lishi mumkin (masalan 20:00–08:00).
              </p>
            </div>
          </div>

          <div className="mt-4 flex justify-end">
            <Button disabled={loading || saving} onClick={() => void handleSave()}>
              <Save className="size-4" />
              {saving ? "Saqlanmoqda..." : "Saqlash"}
            </Button>
          </div>
        </Panel>
      </div>
    </PageContainer>
  )
}
