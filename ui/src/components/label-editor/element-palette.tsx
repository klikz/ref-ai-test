import { Barcode, Grid3x3, GripVertical, Image, Minus, QrCode, SeparatorVertical, Square, Table, Type } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { LABEL_ELEMENT_DRAG_MIME, type LabelElementType } from "@/lib/label-types"

type PaletteItem = {
  dragType: LabelElementType | "vline"
  label: string
  icon: typeof Type
}

const ELEMENTS: PaletteItem[] = [
  { dragType: "text", label: "Matn", icon: Type },
  { dragType: "barcode", label: "Shtrix kod", icon: Barcode },
  { dragType: "datamatrix", label: "DataMatrix", icon: Grid3x3 },
  { dragType: "qrcode", label: "QR kod", icon: QrCode },
  { dragType: "image", label: "Rasm", icon: Image },
  { dragType: "line", label: "Chiziq", icon: Minus },
  { dragType: "vline", label: "Vertikal chiziq", icon: SeparatorVertical },
  { dragType: "rect", label: "Ramka", icon: Square },
  { dragType: "table", label: "Jadval", icon: Table },
]

export function ElementPalette() {
  return (
    <div className="flex flex-col gap-2">
      <p className="text-xs text-muted-foreground">
        Elementni maket ustiga sudrab tashlang. Jadval uchun o&apos;lcham so&apos;raladi.
      </p>
      {ELEMENTS.map(({ dragType, label, icon: Icon }) => (
        <Button
          key={dragType}
          type="button"
          variant="outline"
          draggable
          className={cn(
            "cursor-grab justify-start gap-2 active:cursor-grabbing",
            "hover:border-primary/50",
          )}
          onDragStart={(e) => {
            e.dataTransfer.setData(LABEL_ELEMENT_DRAG_MIME, dragType)
            e.dataTransfer.setData("text/plain", dragType)
            e.dataTransfer.effectAllowed = "copy"
          }}
        >
          <GripVertical className="size-3.5 shrink-0 text-muted-foreground" />
          <Icon className="size-4" />
          {label}
        </Button>
      ))}
    </div>
  )
}
