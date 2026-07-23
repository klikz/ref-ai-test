import { Backend_Request } from "@/services/backend";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input"

import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuGroup,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

import {
    useReactTable,
    getCoreRowModel,
    flexRender,
    createColumnHelper,
} from "@tanstack/react-table";

import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { X } from "lucide-react"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"

export default function PrintersPage() {

    const [printers, setPrinters] = useState([])

    const [lines, setLines] = useState([])
    const [selectedLine, setSelectedLine] = useState(null)

    const [sendEnable, setSendEnable] = useState(false)

    function showOkToast(text: string) {
        toast(text, {
            // description: "Sunday, December 03, 2023 at 9:00 AM",
            style: {
                backgroundColor: "rgba(8, 113, 8, 0.5)",
                color: "white",
            },
            position: 'top-right'
            //   action: {
            //     label: "Undo",
            //     onClick: () => console.log("Undo"),
            //   },
        })
    }

    function showErrorToast(text: string) {
        toast(text, {
            // description: "Sunday, December 03, 2023 at 9:00 AM",
            style: {
                backgroundColor: "rgba(31, 41, 55, 0.9)",
                color: "white",
                justifyContent: 'center',
            },
            position: 'top-center'
            //   action: {
            //     label: "Undo",
            //     onClick: () => console.log("Undo"),
            //   },
        })
    }

    async function printersGetAll() {
        let result = await Backend_Request({}, "/api/lines/printers/all")
        console.log("printers: ", result)
        if (result.result === "ok") {
            console.log(result.data)
            setPrinters(result.data)
        } else {
            showErrorToast(result.error);
        }
    }

    async function linesGetAll() {
        let result = await Backend_Request({}, "/api/lines/all")
        if (result.result === "ok") {
            setLines(result.data)
        } else {
            showErrorToast(result.error);
        }
    }

    async function deletePrinter(id) {
        let data = {
            id: Number(id)
        }
        let result = await Backend_Request(data, "/api/tech/printers/delete")
        if (result.result === "ok") {
            showOkToast("Ma'lumot o'chirildi");
            printersGetAll()
        } else {
            showErrorToast(result.error);
        }
    }

    async function addPrinter() {
        let data = {
            line_id: Number(selectedLine.line_id),
            address: printerAddressRef.current.value,
            printer_name: printerNameRef.current.value
        }
        console.log("data: ",data)
        let result = await Backend_Request(data, "/api/tech/printers/add")
        if (result.result === "ok") {
            showOkToast("Ma'lumot qo'shildi");
            printerAddressRef.current.value = ""
            printerNameRef.current.value = ""
            setSelectedLine(null)
            setSendEnable(false)
            printersGetAll()
        } else {
            showErrorToast(result.error);
        }
    }

    function dropDownPrinters() {
        return (
            <DropdownMenu>
                <DropdownMenuTrigger asChild>
                    <Button style={{padding: 10, height: 35, borderWidth: 1.5, width: 200}} variant="outline">{selectedLine?.name || "Liniyani tanlang"}</Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent className="w-50" align="start">
                    <DropdownMenuGroup>
                        {lines.reduce((acc, line) => {
                            acc.push(
                                <DropdownMenuItem key={line.line_id} onClick={()=>{
                                    setSelectedLine(line)
                                    setSendEnable(true)
                                }}>
                                    {line.name}
                                </DropdownMenuItem>
                            )
                            return acc
                        }, [] as React.ReactNode[])}

                    </DropdownMenuGroup>
                </DropdownMenuContent>
            </DropdownMenu>
        )
    }

    const columnHelper = createColumnHelper<any>();

    const columns = [
    columnHelper.accessor("line_name", {
      header: "Liniya nomi",
    }),
    columnHelper.accessor("printer_name", {
      header: "Printer Nomi",
    }),
    columnHelper.accessor("address", {
      header: "Address",
    }),
    columnHelper.accessor("folder_name", {
      header: "Folder Name",
    }),
    columnHelper.display({
      id: "actions",
      header: "Actions",
      cell: ({ row }) => {
        return (
          <Button variant="outline" size="icon" onClick={() => deletePrinter(row.original.id)}>
            <X style={{color: 'red'}}/>
          </Button>
        );
      },
    }),
  ];

  const table = useReactTable({
    data: printers,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

    function printersTable(){
        return(
            <Table style={{}}>

      {/* HEADER */}
      <TableHeader>
        {table.getHeaderGroups().map((hg) => (
          <TableRow key={hg.id}>
            {hg.headers.map((header) => (
              <TableHead
                key={header.id}
                className=""
              >
                {flexRender(
                  header.column.columnDef.header,
                  header.getContext()
                )}

              </TableHead>
            ))}
          </TableRow>
        ))}
      </TableHeader>

      {/* BODY */}
      <TableBody>
        {table.getRowModel().rows.map((row) => (
          <TableRow
            key={row.id}
          >
            {row.getVisibleCells().map((cell) => (
              <TableCell key={cell.id}>
                {flexRender(cell.column.columnDef.cell, cell.getContext())}
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>

    </Table>
        )
    }

    const printerNameRef = useRef(null)
    const printerAddressRef = useRef(null)

    useEffect(() => {
        printersGetAll()
        linesGetAll()
    }, []);

    return (
        <PageContainer title="Printerlar" description="Liniya printerlari sozlamalari">
            <Panel title="Ro‘yxat" noPadding>
                <div className="overflow-hidden rounded-b-2xl">{printersTable()}</div>
            </Panel>
            <Panel title="Yangi printer">
                <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
                    {dropDownPrinters()}
                    <Input ref={printerNameRef} placeholder="Printer nomi" className="sm:max-w-xs" />
                    <Input ref={printerAddressRef} placeholder="Printer address" className="sm:max-w-xs" />
                    <Button disabled={!sendEnable} onClick={addPrinter}>Qo‘shish</Button>
                </div>
            </Panel>
        </PageContainer>
    )
}