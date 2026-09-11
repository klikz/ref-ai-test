import { MODEL_FIELD_LABELS, modelFieldLabel } from "@/lib/model-field-labels"

export type LabelBinding = {
  key: string
  label: string
  group: string
}

const MODEL_BINDING_EXCLUDE = new Set(["id"])

function buildModelBindings(): LabelBinding[] {
  return Object.keys(MODEL_FIELD_LABELS)
    .filter((key) => !MODEL_BINDING_EXCLUDE.has(key))
    .sort((a, b) => modelFieldLabel(a).localeCompare(modelFieldLabel(b), "uz"))
    .map((key) => ({
      key: `model.${key}`,
      label: modelFieldLabel(key),
      group: "Model",
    }))
}

export const LABEL_BINDINGS: LabelBinding[] = [
  { key: "today", label: "Bugungi sana", group: "Sana" },
  { key: "now", label: "Sana + soat (YYYY-MM-DD HH24:MI)", group: "Sana" },
  { key: "serial", label: "Serial", group: "Mahsulot" },
  { key: "acc_serial", label: "Aksessuar nomer (skaner)", group: "Mahsulot" },
  { key: "brand_logo", label: "Brand logotipi (image)", group: "Brand" },
  { key: "eshik.serial", label: "Eshik Liniya serial", group: "Eshik liniya" },
  { key: "index_1", label: "index_1", group: "Eshik liniya" },
  { key: "index_2", label: "index_2", group: "Eshik liniya" },
  { key: "door_code", label: "door_code (eshik part type)", group: "Eshik liniya" },
  { key: "freeze_door_code", label: "freeze_door_code (model)", group: "Model" },
  { key: "ref_door_code", label: "ref_door_code (model)", group: "Model" },
  { key: "freeze_door_serial", label: "freeze_door_serial (skan)", group: "Mahsulot" },
  { key: "ref_door_serial", label: "ref_door_serial (skan)", group: "Mahsulot" },
  { key: "model_name", label: "Eshik model nomi", group: "Eshik liniya" },
  { key: "gscode.data", label: "GS1 DataMatrix (asl belgisi)", group: "GS Code" },
  { key: "gscode.data38", label: "GS1 Data (38 belgi)", group: "GS Code" },
  ...buildModelBindings(),
]

export function getBindingLabel(key: string): string {
  return LABEL_BINDINGS.find((b) => b.key === key)?.label ?? key
}
