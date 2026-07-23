import { Backend_Request } from "@/services/backend"
import { useEffect, useRef, useState } from "react"
import { useNavigate } from "react-router-dom"
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  createColumnHelper,
  type SortingState,
} from "@tanstack/react-table"
import { Button } from "@/components/ui/button"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { DataTable } from "@/components/layout/data-table"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"

import { Field, FieldGroup } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
type Person = {
  id: number
  login: string
  name: string
  role: string
  role_id: number
  status: boolean
}

export default function UsersPage() {

  const [users, setUsers] = useState([]);
  const navigate = useNavigate();

  const newUserNameRef = useRef(null)
  const newUserLoginRef = useRef(null)
  const newUserPasswordRef = useRef(null)
  const newUserPassword2Ref = useRef(null)
  const [newUserAccountType, setNewUserAccountType] = useState("user")

  const columnHelper = createColumnHelper<Person>();

  const columns = [
    columnHelper.accessor("login", {
      header: "Login",
    }),
    columnHelper.accessor("name", {
      header: "Name",
    }),
    columnHelper.accessor("role", {
      header: "Account type",
    }),
  ];

  useEffect(() => {
    fetchUsers();
  }, []);

  async function fetchUsers() {
    let result = await Backend_Request("", "/api/user/getall");
    if (result.result === "ok") {
      setUsers(result.data);
    } else {
      // showErrorToast(result.error);
    }
  }

  const [sorting, setSorting] = useState<SortingState>([]);
  const table = useReactTable({
    data: users,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });


  async function addUser() {
    console.log("newuserName: ", newUserNameRef.current?.value);
    console.log("newuserLogin: ", newUserLoginRef.current?.value);
    console.log("newuserPassword: ", newUserPasswordRef.current?.value);
    console.log("newuserPassword2: ", newUserPassword2Ref.current?.value);
    if (!newUserNameRef.current?.value || !newUserLoginRef.current?.value || !newUserPasswordRef.current?.value || !newUserPassword2Ref.current?.value) {
      ShowErrorToast("Ma'lumotlarni to'liq kiriting");
      return;
    }
    if (newUserPasswordRef.current?.value !== newUserPassword2Ref.current?.value) {
      ShowErrorToast("Password mos kelmadi");
      return;
    }

    if (/^[A-Za-z][A-Za-z0-9]*$/.test(newUserLoginRef.current?.value) === false) {
      ShowErrorToast("login da faqat xarflar kiritilishi shart a-z A-Z");
      return
    }

    let data = {
      login: newUserLoginRef.current.value,
      name: newUserNameRef.current.value,
      password: newUserPasswordRef.current.value,
      account_type: newUserAccountType,
    }
    let result = await Backend_Request(data, "/api/user/create");
    console.log(result);
    if (result.result === "ok") {
      ShowOKToast("Kiritildi")
      setOpen(false)
      fetchUsers();
      return
    } else {
      ShowErrorToast(result.error);
      return
    }

  }
    const [open, setOpen] = useState(false)

  function addUserDialog() {
    return (
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          <Button variant="outline">Yangi foydalanuvchi</Button>
        </DialogTrigger>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Yangi foydalanuvchi kiritish</DialogTitle>
            <DialogDescription>
              Foydalanuvchi ma'lumotlarini kiriting. Saqlash tugmasini bosing.
            </DialogDescription>
          </DialogHeader>
          <FieldGroup>
            <Field>
              <Label htmlFor="name-1">Login</Label>
              <Input ref={newUserLoginRef} id="name-1" name="name" defaultValue="" />
            </Field>
            <Field>
              <Label htmlFor="username-1">Ismi</Label>
              <Input ref={newUserNameRef} id="username-1" name="username" defaultValue="" />
            </Field>
            <Field>
              <Label htmlFor="password-1">Password</Label>
              <Input ref={newUserPasswordRef} id="password-1" name="password" defaultValue="" />
            </Field>
            <Field>
              <Label htmlFor="password-2">Password qayta kiriting</Label>
              <Input ref={newUserPassword2Ref} id="password-2" name="password-2" defaultValue="" />
            </Field>
            <Field>
              <Label htmlFor="account-type">Account type</Label>
              <select
                id="account-type"
                value={newUserAccountType}
                onChange={(event) => setNewUserAccountType(event.target.value)}
                className="h-10 rounded-xl border border-input bg-background px-3 text-sm"
              >
                <option value="user">Oddiy user</option>
                <option value="admin">Admin</option>
              </select>
            </Field>
          </FieldGroup>

          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Bekor qilish</Button>
            </DialogClose>
            <Button onClick={addUser}>Kiritish</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    )
  }

  return (
    <PageContainer
      title="Foydalanuvchilar"
      description="Tizim foydalanuvchilari va huquqlari"
      actions={addUserDialog()}
    >
      <Panel noPadding>
        <div className="overflow-x-auto rounded-b-xl">
          <DataTable
            table={table}
            onRowClick={(row) => navigate(`/user/${row.id}`)}
          />
        </div>
      </Panel>
    </PageContainer>
  )
}