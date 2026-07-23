import { Barcode, Grid3x3, GripVertical, Image, Minus, QrCode, Square, Table, Type } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { LABEL_ELEMENT_DRAG_MIME, type LabelElementType } from "@/lib/label-types"

const ELEMENTS: { type: LabelElementType; label: string; icon: typeof Type }[] = [
  { type: "text", label: "Matn", icon: Type },
  { type: "barcode", label: "Shtrix kod", icon: Barcode },
  { type: "datamatrix", label: "DataMatrix", icon: Grid3x3 },
  { type: "qrcode", label: "QR kod", icon: QrCode },
  { type: "image", label: "Rasm", icon: Image },
  { type: "line", label: "Chiziq", icon: Minus },
  { type: "rect", label: "Ramka", icon: Square },
  { type: "table", label: "Jadval", icon: Table },
]

export function ElementPalette() {
  return (
    <div className="flex flex-col gap-2">
      <p className="text-xs text-muted-foreground">
        Elementni maket ustiga sudrab tashlang. Jadval uchun o&apos;lcham so&apos;raladi.
      </p>
      {ELEMENTS.map(({ type, label, icon: Icon }) => (
        <Button
          key={type}
          type="button"
          variant="outline"
          draggable
          className={cn(
            "cursor-grab justify-start gap-2 active:cursor-grabbing",
            "hover:border-primary/50",
          )}
          onDragStart={(e) => {
            e.dataTransfer.setData(LABEL_ELEMENT_DRAG_MIME, type)
            e.dataTransfer.setData("text/plain", type)
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
