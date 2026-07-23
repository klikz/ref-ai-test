import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Loader2, Copy, Pencil, Plus, Tag, Trash2 } from "lucide-react"
import { toast } from "sonner"

import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { LabelTemplate } from "@/lib/label-types"
import { Backend_Request } from "@/services/backend"

type LineItem = { line_id: number; name: string }

export default function LabelTemplatesPage() {
  const navigate = useNavigate()
  const [templates, setTemplates] = useState<LabelTemplate[]>([])
  const [lines, setLines] = useState<LineItem[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [duplicatingId, setDuplicatingId] = useState<number | null>(null)

  const [name, setName] = useState("")
  const [lineId, setLineId] = useState("")
  const [widthMm, setWidthMm] = useState("100")
  const [heightMm, setHeightMm] = useState("50")
  const [dpi, setDpi] = useState("203")

  async function loadTemplates() {
    setLoading(true)
    const result = await Backend_Request<LabelTemplate[]>({}, "/api/tech/label-templates/all")
    setLoading(false)
    if (result.result === "ok") {
      setTemplates(result.data ?? [])
    } else {
      toast.error(result.error || "Shablonlar yuklanmadi")
    }
  }

  async function loadLines() {
    const result = await Backend_Request<LineItem[]>({}, "/api/lines/all")
    if (result.result === "ok") {
      setLines(result.data ?? [])
    }
  }

  async function createTemplate() {
    if (!name.trim()) {
      toast.error("Shablon nomini kiriting")
      return
    }
    if (!lineId) {
      toast.error("Liniyani tanlang")
      return
    }

    setCreating(true)
    const result = await Backend_Request<LabelTemplate>(
      {
        name: name.trim(),
        line_id: Number(lineId),
        width_mm: Number(widthMm) || 100,
        height_mm: Number(heightMm) || 50,
        dpi: Number(dpi) || 203,
      },
      "/api/tech/label-templates/create",
    )
    setCreating(false)

    if (result.result === "ok" && result.data) {
      toast.success("Shablon yaratildi")
      navigate(`/label-templates/${result.data.id}`)
    } else {
      toast.error(result.error || "Yaratishda xatolik")
    }
  }

  async function duplicateTemplate(template: LabelTemplate) {
    setDuplicatingId(template.id)
    const result = await Backend_Request<LabelTemplate>(
      { id: template.id },
      "/api/tech/label-templates/duplicate",
    )
    setDuplicatingId(null)

    if (result.result === "ok" && result.data) {
      toast.success(`Nusxa yaratildi: ${result.data.name}`)
      navigate(`/label-templates/${result.data.id}`)
    } else {
      toast.error(result.error || "Nusxalashda xatolik")
    }
  }

  async function deleteTemplate(id: number) {
    const result = await Backend_Request({ id }, "/api/tech/label-templates/delete")
    if (result.result === "ok") {
      toast.success("O'chirildi")
      void loadTemplates()
    } else {
      toast.error(result.error || "O'chirishda xatolik")
    }
  }

  useEffect(() => {
    void loadTemplates()
    void loadLines()
  }, [])

  return (
    <PageContainer
      title="Etiketka shablonlari"
      description="Web interfeys orqali etiketka maketlarini yarating va tahrirlang"
      actions={
        <Button variant="outline" onClick={() => void loadTemplates()} disabled={loading}>
          {loading ? <Loader2 className="size-4 animate-spin" /> : "Yangilash"}
        </Button>
      }
    >
      <Panel title="Shablonlar ro'yxati" noPadding>
        {loading ? (
          <div className="flex items-center justify-center gap-2 p-8 text-muted-foreground">
            <Loader2 className="size-5 animate-spin" />
            Yuklanmoqda...
          </div>
        ) : templates.length === 0 ? (
          <div className="flex flex-col items-center gap-2 p-10 text-center text-muted-foreground">
            <Tag className="size-10 opacity-40" />
            <p>Hali shablon yo&apos;q. Quyida yangi shablon yarating.</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nomi</TableHead>
                <TableHead>Liniya</TableHead>
                <TableHead>O&apos;lcham</TableHead>
                <TableHead>DPI</TableHead>
                <TableHead className="text-right">Amallar</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {templates.map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="font-medium">{t.name}</TableCell>
                  <TableCell>{t.line_name || `Liniya ${t.line_id}`}</TableCell>
                  <TableCell>
                    {t.width_mm}×{t.height_mm} mm
                  </TableCell>
                  <TableCell>{t.dpi}</TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button
                        variant="outline"
                        size="icon"
                        className="size-8"
                        onClick={() => navigate(`/label-templates/${t.id}`)}
                      >
                        <Pencil className="size-4" />
                      </Button>
                      <Button
                        variant="outline"
                        size="icon"
                        className="size-8"
                        title="Nusxa olish"
                        disabled={duplicatingId !== null}
                        onClick={() => void duplicateTemplate(t)}
                      >
                        {duplicatingId === t.id ? (
                          <Loader2 className="size-4 animate-spin" />
                        ) : (
                          <Copy className="size-4" />
                        )}
                      </Button>
                      <Button
                        variant="outline"
                        size="icon"
                        className="size-8"
                        onClick={() => void deleteTemplate(t.id)}
                      >
                        <Trash2 className="size-4 text-destructive" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Panel>

      <Panel title="Yangi shablon">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div className="space-y-1">
            <Label>Nomi</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="T3 etiketka" />
          </div>
          <div className="space-y-1">
            <Label>Liniya</Label>
            <select
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={lineId}
              onChange={(e) => setLineId(e.target.value)}
            >
              <option value="">Tanlang</option>
              {lines.map((line) => (
                <option key={line.line_id} value={line.line_id}>
                  {line.name}
                </option>
              ))}
            </select>
          </div>
          <div className="space-y-1">
            <Label>Kenglik (mm)</Label>
            <Input value={widthMm} onChange={(e) => setWidthMm(e.target.value)} type="number" />
          </div>
          <div className="space-y-1">
            <Label>Balandlik (mm)</Label>
            <Input value={heightMm} onChange={(e) => setHeightMm(e.target.value)} type="number" />
          </div>
          <div className="space-y-1">
            <Label>DPI</Label>
            <select
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={dpi}
              onChange={(e) => setDpi(e.target.value)}
            >
              <option value="203">203</option>
              <option value="300">300</option>
            </select>
          </div>
          <div className="flex items-end">
            <Button onClick={() => void createTemplate()} disabled={creating} className="gap-2">
              {creating ? <Loader2 className="size-4 animate-spin" /> : <Plus className="size-4" />}
              Yaratish va tahrirlash
            </Button>
          </div>
        </div>
      </Panel>
    </PageContainer>
  )
}
