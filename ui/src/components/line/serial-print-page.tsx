import { useEffect, useRef, useState } from "react"
import { Printer } from "lucide-react"
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  type SortingState,
  useReactTable,
} from "@tanstack/react-table"
import { Backend_Request } from "@/services/backend"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { LAST_RECORDS_TITLE } from "@/lib/last-records"
import { cn } from "@/lib/utils"

type PrinterInfo = {
  id: number
  line_id: number
  printer_name: string
}

type LastProduct = {
  serial: string
  acc_serial?: string
  model: string
  model_nomi: string
}

type SerialPrintPageProps = {
  title: string
  description: string
  lineID: number
  printURL: string
  reprintURL?: string
  requireAccSerial?: boolean
}

const columnHelper = createColumnHelper<LastProduct>()

export function SerialPrintPage({
  title,
  lineID,
  printURL,
  reprintURL,
  requireAccSerial = false,
}: SerialPrintPageProps) {
  const serialRef = useRef<HTMLInputElement>(null)
  const accSerialRef = useRef<HTMLInputElement>(null)
  const submittingRef = useRef(false)
  const [lastProducts, setLastProducts] = useState<LastProduct[]>([])
  const [printers, setPrinters] = useState<PrinterInfo[]>([])
  const [selectedPrinter, setSelectedPrinter] = useState<PrinterInfo | null>(null)
  const [sorting, setSorting] = useState<SortingState>([])
  const [accSerialDraft, setAccSerialDraft] = useState("")

  async function printersGetAll() {
    const result = await Backend_Request<PrinterInfo[]>({}, "/api/lines/printers/all")
    if (result.result === "ok" && result.data) {
      const filtered = result.data.filter((val) => val.line_id === lineID)
      setPrinters(filtered)
      setSelectedPrinter(filtered[0] ?? null)
    } else {
      ShowErrorToast(result.error || "Printerlar yuklanmadi")
    }
  }

  async function productsGetLast() {
    const result = await Backend_Request<LastProduct[]>({ line_id: lineID }, "/api/lines/last")
    if (result.result === "ok" && result.data) {
      setLastProducts(result.data)
    } else {
      ShowErrorToast(result.error || "Oxirgi mahsulotlar yuklanmadi")
    }
  }

  function focusSerialInput() {
    window.setTimeout(() => serialRef.current?.focus(), 0)
  }

  function readSerialValue() {
    return serialRef.current?.value.trim() ?? ""
  }

  function readAccSerialValue() {
    if (requireAccSerial) {
      return accSerialDraft.trim()
    }
    return accSerialRef.current?.value.trim() ?? ""
  }

  function clearInputs() {
    if (serialRef.current) {
      serialRef.current.value = ""
    }
    if (accSerialRef.current) {
      accSerialRef.current.value = ""
    }
    setAccSerialDraft("")
  }

  function focusAccSerialInput() {
    const serial = readSerialValue()
    if (!serial) {
      ShowErrorToast("Serial nomer bo'sh")
      focusSerialInput()
      return
    }
    window.setTimeout(() => accSerialRef.current?.focus(), 0)
  }

  function submitScanner() {
    const serial = readSerialValue()
    const accSerial = readAccSerialValue()

    if (!serial) {
      ShowErrorToast("Serial nomer bo'sh")
      focusSerialInput()
      return
    }

    if (requireAccSerial && !accSerial) {
      focusAccSerialInput()
      return
    }

    void printLabel(serial, false, accSerial)
  }

  function handleScannerSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    submitScanner()
  }

  async function printLabel(serial: string, reprint: boolean, accSerial = "") {
    if (submittingRef.current) {
      return
    }

    const normalizedSerial = serial.trim()
    const normalizedAccSerial = accSerial.trim()

    if (!normalizedSerial) {
      ShowErrorToast("Serial nomer bo'sh")
      focusSerialInput()
      return
    }

    if (requireAccSerial && !reprint && !normalizedAccSerial) {
      ShowErrorToast("Aksessuar nomer bo'sh")
      accSerialRef.current?.focus()
      return
    }

    if (!selectedPrinter) {
      ShowErrorToast("Printer tanlanmagan")
      return
    }

    const payload: Record<string, unknown> = {
      serial: normalizedSerial,
      printer_id: Number(selectedPrinter.id),
    }
    if (requireAccSerial && !reprint) {
      payload.acc_serial = normalizedAccSerial
    }
    if (!reprintURL && reprint) {
      payload.reprint = true
    }

    submittingRef.current = true
    let result: Awaited<ReturnType<typeof Backend_Request>>
    try {
      result = await Backend_Request(
        payload,
        reprint && reprintURL ? reprintURL : printURL,
      )
    } finally {
      submittingRef.current = false
    }

    if (result.result === "ok") {
      clearInputs()
      focusSerialInput()
      ShowOKToast(`${normalizedSerial}: ${reprint ? "Qayta chiqarildi" : "Chiqarildi"}`)
      productsGetLast()
      return
    }

    ShowErrorToast(`${normalizedSerial}: ${result.error || "Xatolik"}`)
    if (requireAccSerial && !reprint) {
      accSerialRef.current?.focus()
      return
    }
    focusSerialInput()
  }

  const columns = [
    columnHelper.accessor("serial", { header: "Serial" }),
    ...(requireAccSerial
      ? [
          columnHelper.accessor((row) => row.acc_serial ?? "", {
            id: "acc_serial",
            header: "Aksessuar nomer",
            cell: (info) => {
              const value = String(info.getValue() ?? "").trim()
              return value || "—"
            },
          }),
        ]
      : []),
    columnHelper.accessor("model", { header: "Modeli" }),
    columnHelper.accessor("model_nomi", { header: "Model nomi" }),
    columnHelper.display({
      id: "actions",
      header: "",
      cell: ({ row }) => (
        <Button
          variant="outline"
          size="icon"
          className="size-10 rounded-xl"
          onClick={() => printLabel(row.original.serial, true)}
        >
          <Printer className="size-4" />
        </Button>
      ),
    }),
  ]

  const table = useReactTable({
    data: lastProducts,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  })

  useEffect(() => {
    printersGetAll()
    productsGetLast()
    window.setTimeout(() => serialRef.current?.focus(), 100)
  }, [])

  return (
    <PageContainer title={title} fullWidth>
      <div className="grid w-full min-w-0 grid-cols-1 gap-3 short-landscape:grid-cols-1 xl:grid-cols-[minmax(260px,340px)_1fr]">
        <Panel title="Printerni tanlang" className="h-fit">
          <RadioGroup
            value={selectedPrinter?.id ? String(selectedPrinter.id) : undefined}
            className="grid gap-2 sm:grid-cols-2 xl:grid-cols-1 short-landscape:grid-cols-1"
            onValueChange={(value) =>
              setSelectedPrinter(printers.find((printer) => printer.id === Number(value)) ?? null)
            }
          >
            {printers.map((printer) => (
              <Label
                key={printer.id}
                htmlFor={`printer-${lineID}-${printer.id}`}
                className={cn(
                  "flex min-h-10 cursor-pointer items-center gap-2 rounded-xl border p-2.5 text-sm font-medium transition-colors",
                  selectedPrinter?.id === printer.id
                    ? "border-primary bg-primary/10 text-primary"
                    : "border-border bg-background/50 hover:bg-muted/60",
                )}
              >
                <RadioGroupItem
                  value={String(printer.id)}
                  id={`printer-${lineID}-${printer.id}`}
                />
                <span className="truncate">{printer.printer_name}</span>
              </Label>
            ))}
          </RadioGroup>
        </Panel>

        <div className="flex min-w-0 flex-col gap-3">
          <Panel title="Serial skaner">
            <form
              className="flex flex-col gap-2 lg:flex-row lg:items-center short-landscape:flex-col"
              onSubmit={handleScannerSubmit}
            >
              <Input
                ref={serialRef}
                type="text"
                placeholder="Serial nomer"
                className="h-12 w-full min-w-0 text-base lg:max-w-md"
                onKeyDown={(e) => {
                  if (e.key !== "Enter") {
                    return
                  }
                  e.preventDefault()
                  e.stopPropagation()
                  submitScanner()
                }}
              />
              {requireAccSerial ? (
                <Input
                  ref={accSerialRef}
                  type="text"
                  placeholder="Aksessuar nomer"
                  className="h-12 w-full min-w-0 text-base lg:max-w-md"
                  value={accSerialDraft}
                  onChange={(e) => setAccSerialDraft(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key !== "Enter") {
                      return
                    }
                    e.preventDefault()
                    e.stopPropagation()
                    submitScanner()
                  }}
                />
              ) : null}
              <Button
                type="button"
                className="h-12 px-5 text-base font-semibold"
                disabled={!selectedPrinter}
                tabIndex={requireAccSerial && !accSerialDraft.trim() ? -1 : 0}
                onClick={() => {
                  submitScanner()
                }}
              >
                <Printer className="size-5" />
                Print
              </Button>
            </form>
          </Panel>

          <Panel title={LAST_RECORDS_TITLE} noPadding>
            <div className="overflow-x-auto rounded-b-xl">
              <Table>
                <TableHeader>
                  {table.getHeaderGroups().map((hg) => (
                    <TableRow key={hg.id} className="border-b border-border/60 bg-muted/40 hover:bg-muted/40">
                      {hg.headers.map((header) => (
                        <TableHead key={header.id} className="h-10 whitespace-nowrap px-3">
                          {flexRender(header.column.columnDef.header, header.getContext())}
                        </TableHead>
                      ))}
                    </TableRow>
                  ))}
                </TableHeader>
                <TableBody>
                  {table.getRowModel().rows.map((row) => (
                    <TableRow key={row.id}>
                      {row.getVisibleCells().map((cell) => (
                        <TableCell key={cell.id} className="px-3 py-2">
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </Panel>
        </div>
      </div>
    </PageContainer>
  )
}
