import { Calendar, Database, Pin, Plus } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { groupLabelBindings } from "@/lib/label-binding-groups"
import type { LabelBinding } from "@/lib/label-bindings"
import { cn } from "@/lib/utils"

type LabelDataPanelProps = {
  onAddStaticText: (text: string) => void
  onAddBackendText: (binding: string) => void
  onAddBackendBarcode: (binding: string) => void
  onAddBackendDataMatrix: (binding: string) => void
  onAddTodayDate: () => void
  onAddGS1Data38: () => void
}

export function LabelDataPanel({
  onAddStaticText,
  onAddBackendText,
  onAddBackendBarcode,
  onAddBackendDataMatrix,
  onAddTodayDate,
  onAddGS1Data38,
}: LabelDataPanelProps) {
  const groups = groupLabelBindings()

  return (
    <div className="flex flex-col gap-4">
      <section className="space-y-2">
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          <Pin className="size-3.5" />
          Statik ma&apos;lumot
        </div>
        <p className="text-xs text-muted-foreground">
          Har doim bir xil matn (masalan: &quot;Ishlab chiqaruvchi:&quot;, logotip yonidagi sarlavha).
        </p>
        <StaticTextQuickAdd onAdd={onAddStaticText} />
      </section>

      <section className="space-y-2">
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          <Calendar className="size-3.5" />
          Avtomatik
        </div>
        <p className="text-xs text-muted-foreground">
          Chop etish vaqtida serverda hisoblanadi (masalan, bugungi sana).
        </p>
        <Button type="button" size="sm" variant="outline" className="w-full justify-start gap-2" onClick={onAddTodayDate}>
          <Plus className="size-3.5" />
          Bugungi sana
        </Button>
        <Button type="button" size="sm" variant="outline" className="w-full justify-start gap-2" onClick={onAddGS1Data38}>
          <Plus className="size-3.5" />
          GS1 Data (38 belgi)
        </Button>
      </section>

      <section className="space-y-2">
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          <Database className="size-3.5" />
          Backend maydonlari
        </div>
        <p className="text-xs text-muted-foreground">
          Chop etishda server yuboradi. Nom — UI da, kalit — backend da ishlatiladi.
        </p>
        <div className="max-h-[340px] space-y-3 overflow-y-auto pr-1">
          {groups.map(([group, items]) => (
            <BindingGroup
              key={group}
              group={group}
              items={items}
              onAddText={onAddBackendText}
              onAddBarcode={onAddBackendBarcode}
              onAddDataMatrix={onAddBackendDataMatrix}
            />
          ))}
        </div>
      </section>
    </div>
  )
}

function StaticTextQuickAdd({ onAdd }: { onAdd: (text: string) => void }) {
  return (
    <form
      className="flex gap-2"
      onSubmit={(e) => {
        e.preventDefault()
        const form = e.currentTarget
        const input = form.elements.namedItem("staticText") as HTMLInputElement
        const value = input.value.trim()
        if (!value) return
        onAdd(value)
        input.value = ""
      }}
    >
      <Input name="staticText" className="h-8 text-sm" placeholder="Statik matn..." />
      <Button type="submit" size="sm" variant="outline" className="shrink-0 gap-1">
        <Plus className="size-3.5" />
        Qo&apos;shish
      </Button>
    </form>
  )
}

function BindingGroup({
  group,
  items,
  onAddText,
  onAddBarcode,
  onAddDataMatrix,
}: {
  group: string
  items: LabelBinding[]
  onAddText: (binding: string) => void
  onAddBarcode: (binding: string) => void
  onAddDataMatrix: (binding: string) => void
}) {
  return (
    <div className="rounded-lg border bg-muted/20 p-2">
      <div className="mb-1.5 text-xs font-medium text-foreground">{group}</div>
      <ul className="space-y-1">
        {items.map((item) => (
          <BindingRow
            key={item.key}
            item={item}
            onAddText={() => onAddText(item.key)}
            onAddBarcode={() => onAddBarcode(item.key)}
            onAddDataMatrix={() => onAddDataMatrix(item.key)}
          />
        ))}
      </ul>
    </div>
  )
}

function BindingRow({
  item,
  onAddText,
  onAddBarcode,
  onAddDataMatrix,
}: {
  item: LabelBinding
  onAddText: () => void
  onAddBarcode: () => void
  onAddDataMatrix: () => void
}) {
  const isDataMatrixBinding = item.key === "gscode.data"
  const isGsCodeGroup = item.key.startsWith("gscode.")

  return (
    <li className="rounded-md border border-transparent px-1.5 py-1 hover:border-border hover:bg-background">
      <div className="text-sm font-medium leading-tight">{item.label}</div>
      <code className="text-[10px] text-muted-foreground">{item.key}</code>
      <div className="mt-1 flex flex-wrap gap-1">
        <MiniBtn label="Matn" onClick={onAddText} />
        {!isGsCodeGroup ? <MiniBtn label="Shtrix" onClick={onAddBarcode} /> : null}
        {isDataMatrixBinding ? <MiniBtn label="DM" onClick={onAddDataMatrix} primary /> : null}
      </div>
    </li>
  )
}

function MiniBtn({
  label,
  onClick,
  primary,
}: {
  label: string
  onClick: () => void
  primary?: boolean
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "rounded px-1.5 py-0.5 text-[10px] font-medium transition-colors",
        primary
          ? "bg-primary text-primary-foreground hover:bg-primary/90"
          : "bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground",
      )}
    >
      + {label}
    </button>
  )
}

export function BindingSelect({
  value,
  bindings,
  onChange,
  allowEmpty = true,
  emptyLabel = "— tanlanmagan —",
}: {
  value: string
  bindings: LabelBinding[]
  onChange: (key: string) => void
  allowEmpty?: boolean
  emptyLabel?: string
}) {
  const groups = groupLabelBindings(bindings)

  return (
    <select
      className="h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
      value={value}
      onChange={(e) => onChange(e.target.value)}
    >
      {allowEmpty ? <option value="">{emptyLabel}</option> : null}
      {groups.map(([group, items]) => (
        <optgroup key={group} label={group}>
          {items.map((b) => (
            <option key={b.key} value={b.key}>
              {b.label} ({b.key})
            </option>
          ))}
        </optgroup>
      ))}
    </select>
  )
}
