import { useEffect, useMemo, useState } from "react"
import { Plus, Search, Trash2, UserCog } from "lucide-react"
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  type SortingState,
  useReactTable,
} from "@tanstack/react-table"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import { cn } from "@/lib/utils"

type UserItem = {
  id: number
  login: string
  name: string
}

type WriteoffResponsible = {
  id: number
  user_id: number
  user_name: string
  user_login: string
  c_time: string
}

const columnHelper = createColumnHelper<WriteoffResponsible>()

function normalizeSearchValue(value: unknown) {
  return String(value ?? "").trim().toLocaleLowerCase()
}

export default function WriteoffResponsiblesPage() {
  const [items, setItems] = useState<WriteoffResponsible[]>([])
  const [users, setUsers] = useState<UserItem[]>([])
  const [globalFilter, setGlobalFilter] = useState("")
  const [sorting, setSorting] = useState<SortingState>([])
  const [dialogOpen, setDialogOpen] = useState(false)
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null)
  const [saving, setSaving] = useState(false)

  async function loadItems() {
    const result = await Backend_Request<WriteoffResponsible[]>({}, "/api/writeoff/responsibles/all")
    if (result.result === "ok") {
      setItems(result.data ?? [])
    } else {
      ShowErrorToast(result.error || "Xatolik")
    }
  }

  async function loadUsers() {
    const result = await Backend_Request<UserItem[]>({}, "/api/user/getall")
    if (result.result === "ok") {
      setUsers(result.data ?? [])
    } else {
      ShowErrorToast(result.error || "Foydalanuvchilar yuklanmadi")
    }
  }

  useEffect(() => {
    void loadItems()
    void loadUsers()
  }, [])

  const assignedUserIds = useMemo(() => new Set(items.map((item) => item.user_id)), [items])

  const columns = useMemo(
    () => [
      columnHelper.display({
        id: "user",
        header: "Mas'ul",
        cell: ({ row }) => (
          <div>
            <p className="font-medium">{row.original.user_name || row.original.user_login}</p>
            {row.original.user_name ? (
              <p className="text-xs text-muted-foreground">{row.original.user_login}</p>
            ) : null}
          </div>
        ),
      }),
      columnHelper.accessor("c_time", { header: "Tayinlangan vaqt" }),
      columnHelper.display({
        id: "actions",
        header: "",
        cell: ({ row }) => (
          <Button
            variant="outline"
            size="icon"
            className="size-9 text-destructive hover:text-destructive"
            onClick={() => void deleteItem(row.original.id)}
          >
            <Trash2 className="size-4" />
          </Button>
        ),
      }),
    ],
    [],
  )

  const table = useReactTable({
    data: items,
    columns,
    state: { sorting, globalFilter },
    onSortingChange: setSorting,
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    globalFilterFn: (row, _columnId, filterValue) => {
      const query = normalizeSearchValue(filterValue)
      if (!query) {
        return true
      }
      const item = row.original
      return [item.user_name, item.user_login, item.user_id]
        .map(normalizeSearchValue)
        .join(" ")
        .includes(query)
    },
  })

  async function saveAssignment() {
    if (!selectedUserId) {
      ShowErrorToast("Foydalanuvchini tanlang")
      return
    }
    setSaving(true)
    const result = await Backend_Request({ user_id: selectedUserId }, "/api/writeoff/responsibles/add")
    setSaving(false)
    if (result.result === "ok") {
      ShowOKToast("Mas'ul qo'shildi")
      setDialogOpen(false)
      await loadItems()
    } else {
      ShowErrorToast(result.error || "Qo'shishda xatolik")
    }
  }

  async function deleteItem(id: number) {
    const result = await Backend_Request({ id }, "/api/writeoff/responsibles/delete")
    if (result.result === "ok") {
      ShowOKToast("O'chirildi")
      await loadItems()
    } else {
      ShowErrorToast(result.error || "O'chirishda xatolik")
    }
  }

  return (
    <PageContainer
      title="Hisobdan chiqarish mas'ullari"
      description="Hujjat yakunlanishi uchun barcha mas'ullar alohida tasdiqlashi kerak."
      actions={
        <Button className="gap-2" onClick={() => setDialogOpen(true)}>
          <Plus className="size-4" />
          Tayinlash
        </Button>
      }
    >
      <Panel title={`${table.getFilteredRowModel().rows.length} ta mas'ul`}>
        <div className="mb-4">
          <div className="relative max-w-md">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={globalFilter}
              onChange={(event) => setGlobalFilter(event.target.value)}
              placeholder="Foydalanuvchi bo'yicha qidirish..."
              className="h-10 pl-9"
            />
          </div>
        </div>

        <div className="overflow-auto rounded-xl border border-border/60">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead key={header.id}>
                      {flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getFilteredRowModel().rows.length ? (
                table.getFilteredRowModel().rows.map((row) => (
                  <TableRow key={row.id}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={columns.length} className="h-24 text-center text-muted-foreground">
                    Mas'ullar topilmadi
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </Panel>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <UserCog className="size-5" />
              Mas'ul tayinlash
            </DialogTitle>
            <DialogDescription>
              Hisobdan chiqarish hujjatlarini tasdiqlash uchun foydalanuvchini tanlang. Barcha mas'ullar
              tasdiqlagach, hujjat yakunlanadi.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-2">
            <Label>Foydalanuvchi</Label>
            <div className="grid max-h-64 gap-2 overflow-auto rounded-xl border p-2">
              {users.map((user) => {
                const assigned = assignedUserIds.has(user.id)
                return (
                  <button
                    key={user.id}
                    type="button"
                    disabled={assigned}
                    onClick={() => setSelectedUserId(user.id)}
                    className={cn(
                      "rounded-lg border px-3 py-2 text-left transition",
                      assigned
                        ? "cursor-not-allowed border-dashed opacity-60"
                        : selectedUserId === user.id
                          ? "border-primary bg-primary/10"
                          : "border-border hover:bg-muted/50",
                    )}
                  >
                    <p className="font-medium">{user.name || user.login}</p>
                    <p className="text-xs text-muted-foreground">{user.login}</p>
                  </button>
                )
              })}
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Bekor qilish
            </Button>
            <Button disabled={saving || !selectedUserId} onClick={() => void saveAssignment()}>
              {saving ? "Saqlanmoqda..." : "Saqlash"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PageContainer>
  )
}
