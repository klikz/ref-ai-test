import { useCallback, useEffect, useMemo, useState } from "react"
import { useParams } from "react-router-dom"
import { useDropzone } from "react-dropzone"
import { saveAs } from "file-saver"
import { ArrowLeft, Download, Save, Upload } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import { formatAccSerialStorage, hasAccSerialValues, parseAccSerialValues } from "@/lib/acc-serial"
import { cn } from "@/lib/utils"

type ModelInfo = Record<string, string | number | boolean | null | undefined>

type FieldDef = {
  key: string
  label: string
  span?: string
  numeric?: boolean
}

type FieldGroup = {
  id: string
  title: string
  description: string
  fields: FieldDef[]
  layout?: "grid" | "serial" | "wide"
}

const multiValueKeys = new Set(["acc_serial", "compressor_serial", "freeze_door_code", "ref_door_code"])

const fieldGroups: FieldGroup[] = [
  {
    id: "identity",
    title: "Asosiy",
    description: "Model identifikatsiyasi va savdo ma'lumotlari",
    fields: [
      { key: "seriya_raqami", label: "Seriya raqami" },
      { key: "modeli", label: "Modeli" },
      { key: "qisqa_nomi", label: "Qisqa nomi" },
      { key: "sovutgich_turi", label: "Sovutgich turi" },
      { key: "rangi", label: "Rangi" },
      { key: "eshik_rangi", label: "Eshik rangi" },
      { key: "rangi_eng", label: "Rangi (eng)" },
      { key: "korpus_rangi_shortname", label: "Ko'rpus rangi (Shortname)" },
      { key: "eshik_rangi_shortname", label: "Eshik rangi (Shortname)" },
      { key: "rangi_kodi", label: "Rangi (kodi)", numeric: true },
      { key: "sotuv_turi", label: "Sotuv turi" },
      { key: "brend", label: "Brend" },
      { key: "local_export", label: "Local/Export" },
      { key: "odoo_code", label: "ODOO code" },
      { key: "gs1_ean13", label: "GS1 EAN13" },
    ],
  },
  {
    id: "serials",
    title: "Serial qoidalari",
    description:
      "Bir nechta qiymat mumkin — har birini alohida qatorga yozing (vergul yoki ; ham bo'ladi). * — har qanday qiymat qabul",
    layout: "serial",
    fields: [
      {
        key: "acc_serial",
        label: "acc_serial",
      },
      {
        key: "compressor_serial",
        label: "kompressor_serial",
      },
      {
        key: "freeze_door_code",
        label: "freeze_door_code",
      },
      {
        key: "ref_door_code",
        label: "ref_door_code",
      },
    ],
  },
  {
    id: "electrical",
    title: "Elektr va energiya",
    description: "Kuchlanish, quvvat va energiya ko‘rsatkichlari",
    fields: [
      { key: "taminot_kuchlanishi_v", label: "Ta'minot kuchlanishi (V)" },
      { key: "elektr_toki_kuchlanishi_va_turi", label: "Elektr toki kuchlanishi va turi" },
      { key: "nominal_tok_quvvati_w", label: "Nominal tok quvvati (W)" },
      { key: "nominal_tok_kuchi_a", label: "Nominal tok kuchi (A)" },
      { key: "yoritgich_lampaning_quvvati_vt", label: "Yoritgich lampaning quvvati (Vt)" },
      { key: "energiya_samaradorlik_sarfi", label: "Energiya samaradorlik sarfi (A+)" },
      { key: "shovqin_darajasi_db", label: "Shovqin darajasi ichki blok dB" },
    ],
  },
  {
    id: "volumes",
    title: "Hajm va og‘irlik",
    description: "Kamera hajmlari, netto/brutto va qadoq",
    fields: [
      { key: "umumiy_hajmi_l", label: "Umumiy hajmi (L)" },
      { key: "sovutgich_kamera_hajmi_l", label: "Sovutgich kamera hajmi (L)" },
      { key: "muzlatgich_kamera_hajmi_l", label: "Muzlatgich kamera hajmi (L)" },
      { key: "muzlatish_quvvati", label: "Muzlatish quvvati" },
      { key: "netto", label: "Netto" },
      { key: "brutto", label: "Brutto" },
      { key: "qadoq_hajmi", label: "Qadoq hajmi" },
      { key: "mahsulot_hajmi", label: "Mahsulot hajmi" },
    ],
  },
  {
    id: "tech",
    title: "Texnik",
    description: "Kompressor, freon va iqlim",
    fields: [
      { key: "kompressor_nomi", label: "Kompressor nomi" },
      { key: "xladagent_miqdori_g", label: "Xladagent miqdori (g)" },
      { key: "freon", label: "Freon" },
      { key: "iqlim_sharoitlari", label: "Iqlim sharoitlari" },
      { key: "gost", label: "GOST" },
    ],
  },
  {
    id: "certs",
    title: "Sertifikatlar va ishlab chiqaruvchi",
    description: "Hujjatlar, mamlakat va manzil",
    fields: [
      { key: "maxalliy_sertifikat", label: "Maxalliy sertifikat" },
      { key: "eac_sertifikati", label: "EAC sertifikati" },
      { key: "ce_sertifikat", label: "CE sertifikat" },
      { key: "ishlab_chiqaruvchi_mamlakat", label: "Ishlab chiqaruvchi mamlakat" },
      { key: "korxon_nomi", label: "Korxona nomi" },
      { key: "manzil", label: "Manzil", span: "md:col-span-2 xl:col-span-3" },
      { key: "manzil_ru", label: "Manzil (ru)", span: "md:col-span-2 xl:col-span-3" },
    ],
  },
  {
    id: "comment",
    title: "Izoh",
    description: "Qo‘shimcha eslatma",
    layout: "wide",
    fields: [{ key: "comment", label: "comment" }],
  },
]

const modelExcelFields = [
  ...fieldGroups.flatMap((group) => group.fields.map((field) => [field.key, field.label] as const)),
].filter(([key], index, all) => all.findIndex((item) => item[0] === key) === index)

const serialPlaceholders: Record<string, string> = {
  acc_serial: "ACC001\nACC002\n*",
  compressor_serial: "CMP001\nCMP002\n*",
  door_code: "DR01\nDR02\n*",
  freeze_door_code: "FR01\n*",
  ref_door_code: "RF01\n*",
}

const textareaClassName = cn(
  "flex min-h-[7.5rem] w-full resize-y rounded-xl border border-input bg-transparent px-3 py-2.5 font-mono text-sm leading-relaxed shadow-xs",
  "placeholder:font-sans placeholder:text-muted-foreground",
  "focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none",
)

function normalizeNullableValue(value: ModelInfo[string]) {
  return value === null || value === undefined ? " " : value
}

function normalizeModel(model: ModelInfo) {
  const entries = Object.entries(model).map(([key, value]) => {
    const normalized = normalizeNullableValue(value)
    if (multiValueKeys.has(key) && typeof normalized === "string") {
      return [key, formatAccSerialStorage(normalized)]
    }
    return [key, normalized]
  })
  return Object.fromEntries(entries) as ModelInfo
}

function MultiValueField({
  label,
  value,
  onChange,
  placeholder,
}: {
  label: string
  value: string
  onChange: (value: string) => void
  placeholder: string
}) {
  const count = parseAccSerialValues(value).length

  return (
    <div className="flex h-full min-h-0 flex-col gap-1.5 rounded-2xl border border-border/60 bg-muted/15 p-3">
      <div className="flex items-center justify-between gap-2">
        <Label className="font-medium">{label}</Label>
        <span className="rounded-full bg-background px-2 py-0.5 text-[11px] text-muted-foreground ring-1 ring-border/60">
          {count} ta
        </span>
      </div>
      <textarea
        value={value}
        onChange={(event) => onChange(event.target.value)}
        rows={5}
        placeholder={placeholder}
        className={cn(textareaClassName, "flex-1")}
      />
    </div>
  )
}

export default function ModelsIdPage() {
  const { id } = useParams()
  const [model, setModel] = useState<ModelInfo | null>(null)
  const [saving, setSaving] = useState(false)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)

  const modelID = Number(id)

  const onDrop = useCallback((acceptedFiles: File[]) => {
    setSelectedFile(acceptedFiles[0] ?? null)
  }, [])

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    multiple: false,
  })

  async function loadModel() {
    const result = await Backend_Request<ModelInfo[]>({}, "/api/tech/models/all")
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Model yuklanmadi")
      return
    }
    const current = result.data?.find((item) => Number(item.id) === modelID)
    if (!current) {
      ShowErrorToast("Model topilmadi")
      return
    }
    setModel(normalizeModel(current))
  }

  useEffect(() => {
    loadModel()
  }, [modelID])

  const title = useMemo(() => {
    if (!model) {
      return "Model"
    }
    return String(model.qisqa_nomi || model.modeli || `Model #${model.id}`)
  }, [model])

  function updateField(key: string, value: string | boolean) {
    setModel((current) => (current ? { ...current, [key]: value } : current))
  }

  async function saveModel() {
    if (!model) {
      return
    }
    if (!hasAccSerialValues(String(model.acc_serial ?? ""))) {
      ShowErrorToast("acc_serial bo'sh bo'lishi mumkin emas")
      return
    }
    setSaving(true)
    const result = await Backend_Request({ model: normalizeModel(model) }, "/api/tech/models/update")
    setSaving(false)
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Model saqlanmadi")
      return
    }
    ShowOKToast("Model saqlandi")
    loadModel()
  }

  function uploadExcel() {
    if (!selectedFile) {
      return
    }
    const reader = new FileReader()
    reader.readAsDataURL(selectedFile)
    reader.onload = async () => {
      const result = await Backend_Request({ file64: reader.result, id: modelID }, "/api/tech/models/update")
      if (result.result !== "ok") {
        ShowErrorToast(result.error || "Excel orqali yangilanmadi")
        return
      }
      ShowOKToast("Excel orqali yangilandi")
      setSelectedFile(null)
      loadModel()
    }
    reader.onerror = () => ShowErrorToast("Fayl o'qilmadi")
  }

  async function downloadExcel() {
    if (!model) {
      return
    }

    const XLSX = await import("xlsx")
    const rows = [
      modelExcelFields.map(([key]) => key),
      modelExcelFields.map(([key]) => normalizeNullableValue(model[key])),
    ]
    const worksheet = XLSX.utils.aoa_to_sheet(rows)
    worksheet["!cols"] = modelExcelFields.map(([, label]) => ({
      wch: Math.max(16, label.length + 2),
    }))

    const workbook = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(workbook, worksheet, "Model")
    const excelBuffer = XLSX.write(workbook, { bookType: "xlsx", type: "array" })
    const fileData = new Blob([excelBuffer], {
      type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    })
    saveAs(fileData, `model_${model.id || modelID}.xlsx`)
  }

  return (
    <PageContainer
      title={title}
      description="Model ma'lumotlarini qo'lda tahrirlash yoki Excel orqali yangilash"
      scrollable
      actions={
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" onClick={() => history.back()} className="rounded-xl">
            <ArrowLeft className="size-4" />
            Back
          </Button>
          <Button onClick={saveModel} disabled={!model || saving} className="rounded-xl">
            <Save className="size-4" />
            {saving ? "Saving..." : "Save"}
          </Button>
        </div>
      }
    >
      <Panel title="Excel update" description="Model ma'lumotlarini XLSX orqali yuklab olish va yangilash">
        <div className="grid gap-3 lg:grid-cols-[1fr_auto_auto]">
          <div
            {...getRootProps()}
            className={cn(
              "cursor-pointer rounded-2xl border border-dashed border-border bg-muted/20 px-4 py-5 text-sm transition",
              isDragActive && "border-primary bg-primary/10",
            )}
          >
            <input {...getInputProps()} />
            <div className="flex items-center gap-2 font-medium">
              <Upload className="size-4 text-primary" />
              {selectedFile ? selectedFile.name : "XLSX faylni tanlang yoki shu yerga tashlang"}
            </div>
          </div>
          <Button
            onClick={downloadExcel}
            disabled={!model}
            variant="outline"
            className="h-full min-h-12 rounded-xl"
          >
            <Download className="size-4" />
            Download XLSX
          </Button>
          <Button onClick={uploadExcel} disabled={!selectedFile} className="h-full min-h-12 rounded-xl">
            <Upload className="size-4" />
            Upload XLSX
          </Button>
        </div>
      </Panel>

      {!model ? (
        <Panel title="Model fields">
          <div className="py-10 text-center text-muted-foreground">Loading...</div>
        </Panel>
      ) : (
        <>
          <Panel title="Meta" description="O‘zgarmas identifikatorlar">
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <div className="space-y-1.5">
                <Label>ID</Label>
                <Input value={String(model.id ?? "")} disabled className="rounded-xl" />
              </div>
              <div className="space-y-1.5">
                <Label>GS code count</Label>
                <Input value={String(model.gscode_count ?? 0)} disabled className="rounded-xl" />
              </div>
              <label className="flex h-10 items-center gap-2 self-end rounded-xl border px-3 text-sm sm:col-span-2 xl:col-span-2">
                <input
                  type="checkbox"
                  checked={Boolean(model.status)}
                  onChange={(event) => updateField("status", event.target.checked)}
                />
                Status active
              </label>
            </div>
          </Panel>

          {fieldGroups.map((group) => (
            <Panel key={group.id} title={group.title} description={group.description}>
              {group.layout === "serial" ? (
                <div className="grid gap-3 lg:grid-cols-3">
                  {group.fields.map((field) => (
                    <MultiValueField
                      key={field.key}
                      label={field.label}
                      value={String(model[field.key] ?? "")}
                      onChange={(value) => updateField(field.key, value)}
                      placeholder={serialPlaceholders[field.key] ?? ""}
                    />
                  ))}
                </div>
              ) : group.layout === "wide" ? (
                <div className="space-y-1.5">
                  <Label>{group.fields[0]?.label}</Label>
                  <textarea
                    value={String(model[group.fields[0]?.key ?? ""] ?? "")}
                    onChange={(event) => updateField(group.fields[0].key, event.target.value)}
                    rows={3}
                    className={cn(textareaClassName, "min-h-20 font-sans")}
                  />
                </div>
              ) : (
                <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                  {group.fields.map((field) => (
                    <div key={field.key} className={cn("space-y-1.5", field.span)}>
                      <Label>{field.label}</Label>
                      <Input
                        value={String(model[field.key] ?? "")}
                        onChange={(event) =>
                          updateField(
                            field.key,
                            field.numeric ? event.target.value.replace(/\D/g, "") : event.target.value,
                          )
                        }
                        inputMode={field.numeric ? "numeric" : undefined}
                        placeholder={field.numeric ? "faqat son" : undefined}
                        className="rounded-xl"
                      />
                    </div>
                  ))}
                </div>
              )}
            </Panel>
          ))}
        </>
      )}
    </PageContainer>
  )
}
