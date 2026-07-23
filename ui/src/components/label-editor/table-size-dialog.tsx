import { useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { TABLE_COLS_MAX, TABLE_ROWS_MAX, clampTableCols, clampTableRows } from "@/lib/label-types"

type TableSizeDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (rows: number, cols: number) => void
}

export function TableSizeDialog({ open, onOpenChange, onConfirm }: TableSizeDialogProps) {
  const [rows, setRows] = useState(2)
  const [cols, setCols] = useState(3)

  useEffect(() => {
    if (open) {
      setRows(2)
      setCols(3)
    }
  }, [open])

  function handleConfirm() {
    onConfirm(clampTableRows(rows), clampTableCols(cols))
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent showCloseButton>
        <DialogHeader>
          <DialogTitle>Jadval o&apos;lchami</DialogTitle>
          <DialogDescription>
            Qator va ustunlar sonini kiriting (maks. {TABLE_ROWS_MAX}×{TABLE_COLS_MAX})
          </DialogDescription>
        </DialogHeader>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label className="text-xs">Qatorlar</Label>
            <Input
              type="number"
              min={1}
              max={TABLE_ROWS_MAX}
              value={rows}
              onChange={(e) => setRows(Number(e.target.value))}
              className="h-9"
              autoFocus
            />
          </div>
          <div className="space-y-1">
            <Label className="text-xs">Ustunlar</Label>
            <Input
              type="number"
              min={1}
              max={TABLE_COLS_MAX}
              value={cols}
              onChange={(e) => setCols(Number(e.target.value))}
              className="h-9"
            />
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Bekor qilish
          </Button>
          <Button type="button" onClick={handleConfirm}>
            Qo&apos;shish
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
