import type { LabelDataSource } from "@/lib/label-types"
import { isDerivedBinding, resolveDerivedBinding } from "@/lib/label-derived-bindings"
import { MODEL_FIELD_LABELS, modelFieldLabel } from "@/lib/model-field-labels"

export type LabelSampleData = Record<string, unknown>

const MODEL_SAMPLE_VALUES: Record<string, string> = {
  seriya_raqami: "A1FA01",
  modeli: "PRM-211TFDF/W",
  sovutgich_turi: "Defrost",
  qisqa_nomi: "211",
  odoo_code: "103012111001",
  brend: "Premier",
  gs1_ean13: "4780092900049",
  gost: "GOST ISO 62552-2013",
  rangi: "Oq",
  brutto: "44",
  netto: "39",
  manzil: "Oxangaron",
  korxon_nomi: "Premier LLC",
  ishlab_chiqaruvchi_mamlakat: "O'zbekiston",
  taminot_kuchlanishi_v: "220В-240В/50Гц",
  umumiy_hajmi_l: "211",
  nominal_tok_quvvati_w: "62",
  qadoq_hajmi: "578x600x1475",
  freon: "R600a",
  xladagent_miqdori_g: "45",
  compressor_serial: "*",
  door_code: "",
  comment: "",
}

function buildLabelSampleModel(): Record<string, string> {
  const model: Record<string, string> = {}
  for (const key of Object.keys(MODEL_FIELD_LABELS)) {
    if (key === "id") {
      continue
    }
    model[key] = MODEL_SAMPLE_VALUES[key] ?? `(${modelFieldLabel(key)})`
  }
  return model
}

export const LABEL_SAMPLE_DATA: LabelSampleData = {
  serial: "ABC123456789",
  acc_serial: "ACC001234",
  seriya_raqami: "A1FA01",
  radiator_counter: 1,
  klapan_counter: 1,
  eshik_counter: 1,
  index_1: "IDX-1",
  index_2: "IDX-2",
  radiator: {
    index1: "R-IDX-1",
    index2: "R-IDX-2",
  },
  klapan: {
    index1: "K-IDX-1",
    index2: "K-IDX-2",
  },
  eshik: {
    index1: "E-IDX-1",
    index2: "E-IDX-2",
  },
  count: 50,
  gscode: {
    data: "010478009290004921ABC123456789012345678901234567890",
  },
  model: buildLabelSampleModel(),
}

export type LabelPreviewOverrides = {
  serial?: string
  acc_serial?: string
}

export function buildLabelPreviewData(overrides?: LabelPreviewOverrides): LabelSampleData {
  return {
    ...LABEL_SAMPLE_DATA,
    serial: overrides?.serial ?? LABEL_SAMPLE_DATA.serial,
    acc_serial: overrides?.acc_serial ?? LABEL_SAMPLE_DATA.acc_serial,
  }
}

export function resolveBinding(data: LabelSampleData, binding?: string): string {
  if (!binding) {
    return ""
  }
  if (isDerivedBinding(binding)) {
    return resolveDerivedBinding(data, binding)
  }
  const parts = binding.split(".")
  let current: unknown = data
  for (const part of parts) {
    if (current == null || typeof current !== "object") {
      return ""
    }
    current = (current as Record<string, unknown>)[part]
  }
  if (current == null) {
    return ""
  }
  return String(current)
}

export function resolveElementText(
  data: LabelSampleData,
  dataSource: LabelDataSource,
  staticText?: string,
  binding?: string,
  prefix?: string,
  dateFormat?: string,
): string {
  if (dataSource === "backend" && binding) {
    const value = isDerivedBinding(binding)
      ? resolveDerivedBinding(data, binding, dateFormat)
      : resolveBinding(data, binding)
    if (value) {
      return `${prefix ?? ""}${value}`
    }
    return prefix ?? ""
  }
  return staticText ?? ""
}
