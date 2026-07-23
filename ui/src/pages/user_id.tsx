import { Backend_Request } from "@/services/backend"
import { useEffect, useMemo, useState } from "react"
import { useNavigate, useParams } from "react-router-dom"
import {
  ArrowLeft,
  KeyRound,
  Save,
  Search,
  ShieldBan,
  ShieldCheck,
  Trash2,
  UserRound,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
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
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import { cn } from "@/lib/utils"

type Permission = {
  id: number
  route: string
  comment: string
  is_flag: boolean
}

type UserResponse = {
  permissions: Permission[]
  user_info: {
    id: number
    login: string
    name: string
    role?: string
    status?: boolean
  }
}

type PermissionFilter = "all" | "granted" | "denied"

type PermissionGroup = {
  id: string
  title: string
  description: string
  items: Permission[]
}

const PERMISSION_GROUPS: Array<{
  id: string
  title: string
  description: string
  match: (route: string) => boolean
}> = [
  {
    id: "users",
    title: "Foydalanuvchilar",
    description: "Akkaunt va ruxsatlarni boshqarish",
    match: (route) => route.startsWith("/api/user"),
  },
  {
    id: "serial",
    title: "Serial ma'lumot",
    description: "Serial va kompressor bo'yicha qidiruv",
    match: (route) => route.startsWith("/api/serial"),
  },
  {
    id: "lab",
    title: "Laboratoriya",
    description: "VTM laboratoriyasi test natijalari",
    match: (route) => route.startsWith("/api/lab"),
  },
  {
    id: "models-gscode",
    title: "Modellar va GS Code",
    description: "Model katalogi va GS kodlar",
    match: (route) =>
      route.startsWith("/api/tech/models") || route.startsWith("/api/tech/gscode"),
  },
  {
    id: "printers-labels",
    title: "Printer va etiketkalar",
    description: "Printerlar, shablonlar va brand logolar",
    match: (route) =>
      route.startsWith("/api/tech/printers") ||
      route.startsWith("/api/tech/label-templates") ||
      route.startsWith("/api/tech/brands") ||
      route.startsWith("/api/tech/brand-logos") ||
      route === "/api/tech/lines/add_gp_component",
  },
  {
    id: "production-catalog",
    title: "Production katalog",
    description: "Komponentlar, mas'ullar va sarf normasi",
    match: (route) =>
      route.startsWith("/api/production/components") ||
      route.startsWith("/api/production/line_responsibles") ||
      route.startsWith("/api/production/consumption-norm"),
  },
  {
    id: "production-plan",
    title: "Reja va hisobot",
    description: "Ishlab chiqarish rejasi, smena va umumiy hisobot",
    match: (route) =>
      route.startsWith("/api/production/plan") ||
      route.startsWith("/api/production/shifts") ||
      route === "/api/production/info" ||
      route === "/api/production/report/xlsx",
  },
  {
    id: "lines",
    title: "Liniyalar",
    description: "Yi'g'ish, brigadir, balans va liniya hisobotlari",
    match: (route) => route.startsWith("/api/lines"),
  },
  {
    id: "warehouse",
    title: "Ombor",
    description: "Kirim, chiqim, buyurtma va GP mahsulotlar",
    match: (route) => route.startsWith("/api/ware"),
  },
  {
    id: "writeoff",
    title: "Hisobdan chiqarish",
    description: "Hujjatlar, tasdiqlash va arxiv",
    match: (route) => route.startsWith("/api/writeoff"),
  },
]

function permissionTitle(permission: Permission) {
  const comment = permission.comment?.trim()
  if (comment && comment.toLowerCase() !== "no comment") return comment
  return permission.route
}

function permissionShortRoute(route: string) {
  return route.replace(/^\/api\//, "")
}

function groupPermissions(items: Permission[]): PermissionGroup[] {
  const sorted = [...items].sort((a, b) => {
    const titleCmp = permissionTitle(a).localeCompare(permissionTitle(b), "uz")
    if (titleCmp !== 0) return titleCmp
    return a.route.localeCompare(b.route)
  })

  const used = new Set<number>()
  const groups: PermissionGroup[] = []

  for (const group of PERMISSION_GROUPS) {
    const matched = sorted.filter((item) => group.match(item.route))
    matched.forEach((item) => used.add(item.id))
    if (matched.length > 0) {
      groups.push({
        id: group.id,
        title: group.title,
        description: group.description,
        items: matched,
      })
    }
  }

  const other = sorted.filter((item) => !used.has(item.id))
  if (other.length > 0) {
    groups.push({
      id: "other",
      title: "Boshqa",
      description: "Guruhlanmagan route'lar",
      items: other,
    })
  }

  return groups
}

export default function UserPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [name, setName] = useState("")
  const [login, setLogin] = useState("")
  const [accountType, setAccountType] = useState("user")
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [permissionSearch, setPermissionSearch] = useState("")
  const [permissionFilter, setPermissionFilter] = useState<PermissionFilter>("all")
  const [password, setPassword] = useState("")
  const [repeatPassword, setRepeatPassword] = useState("")
  const [passwordOpen, setPasswordOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [togglingRouteId, setTogglingRouteId] = useState<number | null>(null)

  const filteredPermissions = useMemo(() => {
    const search = permissionSearch.trim().toLowerCase()
    return permissions.filter((permission) => {
      if (permissionFilter === "granted" && !permission.is_flag) return false
      if (permissionFilter === "denied" && permission.is_flag) return false
      if (!search) return true
      return `${permissionTitle(permission)} ${permission.route}`
        .toLowerCase()
        .includes(search)
    })
  }, [permissionFilter, permissionSearch, permissions])

  const permissionGroups = useMemo(
    () => groupPermissions(filteredPermissions),
    [filteredPermissions],
  )

  const grantedCount = useMemo(
    () => permissions.filter((item) => item.is_flag).length,
    [permissions],
  )

  async function changePermission(routeID: number, status: boolean) {
    setTogglingRouteId(routeID)
    const result = await Backend_Request(
      {
        route_id: routeID,
        status: Boolean(status),
      },
      "/api/user/permission/" + id,
    )
    setTogglingRouteId(null)
    if (result.result === "error") {
      ShowErrorToast(result.error || "Ruxsat o'zgarmadi")
      return
    }
    setPermissions((prev) =>
      prev.map((item) => (item.id === routeID ? { ...item, is_flag: status } : item)),
    )
    ShowOKToast(status ? "Ruxsat berildi" : "Ruxsat olib tashlandi")
  }

  async function getUserData() {
    const result = await Backend_Request<UserResponse>("", "/api/user/" + id)
    if (result.result === "ok" && result.data) {
      setPermissions(result.data.permissions)
      setLogin(result.data.user_info.login)
      setName(result.data.user_info.name)
      setAccountType(result.data.user_info.role?.toLowerCase() === "admin" ? "admin" : "user")
    } else {
      ShowErrorToast(result.error || "Foydalanuvchi yuklanmadi")
    }
  }

  async function changePassword() {
    if (!password || password.length < 4) {
      ShowErrorToast("Parol kamida 4 ta belgidan iborat bo'lishi kerak")
      return
    }
    if (password !== repeatPassword) {
      ShowErrorToast("Parollar mos kelmadi")
      return
    }

    const result = await Backend_Request(
      {
        id: Number(id),
        password,
        is_password_update: true,
      },
      "/api/user/update",
    )
    if (result.result === "ok") {
      setPassword("")
      setRepeatPassword("")
      setPasswordOpen(false)
      ShowOKToast("Password changed")
    } else {
      ShowErrorToast(result.error || "Password o'zgarmadi")
    }
  }

  async function saveChanges() {
    if (!login || !name) {
      ShowErrorToast("Ma'lumotlar to'liq kiritilmadi")
      return
    }

    if (/^[A-Za-z][A-Za-z0-9]*$/.test(login) === false) {
      ShowErrorToast("Login faqat harf bilan boshlanib, harf va sonlardan iborat bo'lishi kerak")
      return
    }

    setSaving(true)
    const result = await Backend_Request(
      {
        id: Number(id),
        login,
        name,
        account_type: accountType,
        is_password_update: false,
      },
      "/api/user/update",
    )
    setSaving(false)
    if (result.result === "ok") {
      getUserData()
      ShowOKToast("Ma'lumotlar saqlandi")
    } else {
      ShowErrorToast(result.error || "Ma'lumotlar saqlanmadi")
    }
  }

  async function deleteUser() {
    const result = await Backend_Request({ user_id: Number(id) }, "/api/user/delete")
    if (result.result === "ok") {
      ShowOKToast("Foydalanuvchi o'chirildi")
      navigate("/users")
    } else {
      ShowErrorToast(result.error || "Foydalanuvchi o'chirilmadi")
    }
  }

  useEffect(() => {
    getUserData()
  }, [id])

  return (
    <PageContainer
      title="Foydalanuvchi"
      description="Akkaunt ma'lumotlari, parol va route ruxsatlarini boshqarish"
      scrollable
      actions={
        <Button variant="outline" onClick={() => navigate("/users")} className="rounded-xl">
          <ArrowLeft className="size-4" />
          Orqaga
        </Button>
      }
    >
      <div className="grid gap-4 xl:grid-cols-[420px_1fr]">
        <div className="space-y-4">
          <Panel title="Akkaunt" description={`ID: ${id}`}>
            <div className="mb-4 flex items-center gap-3 rounded-2xl border border-primary/20 bg-primary/10 p-4">
              <div className="flex size-12 items-center justify-center rounded-2xl bg-primary text-primary-foreground">
                <UserRound className="size-6" />
              </div>
              <div className="min-w-0">
                <div className="truncate text-lg font-semibold">{name || "Foydalanuvchi"}</div>
                <div className="truncate text-sm text-muted-foreground">{login || "login"}</div>
              </div>
            </div>

            <div className="space-y-3">
              <div className="space-y-2">
                <Label htmlFor="user-login">Login</Label>
                <Input
                  id="user-login"
                  value={login}
                  onChange={(e) => setLogin(e.target.value)}
                  className="h-11 rounded-xl"
                  placeholder="login"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="user-name">Ism</Label>
                <Input
                  id="user-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="h-11 rounded-xl"
                  placeholder="user name"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="account-type">Account type</Label>
                <select
                  id="account-type"
                  value={accountType}
                  onChange={(event) => setAccountType(event.target.value)}
                  className="h-11 w-full rounded-xl border border-input bg-background px-3 text-sm"
                >
                  <option value="user">Oddiy user</option>
                  <option value="admin">Admin</option>
                </select>
              </div>
              <div className="grid gap-2 sm:grid-cols-2">
                <Button onClick={saveChanges} disabled={saving} className="h-11 rounded-xl">
                  <Save className="size-4" />
                  {saving ? "Saqlanmoqda..." : "Saqlash"}
                </Button>
                <Dialog open={passwordOpen} onOpenChange={setPasswordOpen}>
                  <DialogTrigger asChild>
                    <Button variant="outline" className="h-11 rounded-xl">
                      <KeyRound className="size-4" />
                      Password
                    </Button>
                  </DialogTrigger>
                  <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                      <DialogTitle>Password ni yangilash</DialogTitle>
                      <DialogDescription>
                        Bu tugma aynan ushbu akkaunt parolini o'zgartiradi.
                      </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-3">
                      <div className="space-y-2">
                        <Label htmlFor="new-password">Yangi password</Label>
                        <Input
                          id="new-password"
                          type="password"
                          value={password}
                          onChange={(e) => setPassword(e.target.value)}
                          className="h-11 rounded-xl"
                          placeholder="Password kiriting"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="repeat-password">Qayta kiriting</Label>
                        <Input
                          id="repeat-password"
                          type="password"
                          value={repeatPassword}
                          onChange={(e) => setRepeatPassword(e.target.value)}
                          className="h-11 rounded-xl"
                          placeholder="Password qayta"
                        />
                      </div>
                    </div>
                    <DialogFooter>
                      <DialogClose asChild>
                        <Button variant="outline">Bekor</Button>
                      </DialogClose>
                      <Button onClick={changePassword}>Saqlash</Button>
                    </DialogFooter>
                  </DialogContent>
                </Dialog>
              </div>
            </div>
          </Panel>

          <Panel title="Danger zone" description="Foydalanuvchini o'chirish">
            <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
              <DialogTrigger asChild>
                <Button variant="destructive" className="h-11 w-full rounded-xl">
                  <Trash2 className="size-4" />
                  Delete user
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-md">
                <DialogHeader>
                  <DialogTitle>Foydalanuvchini o'chirish</DialogTitle>
                  <DialogDescription>
                    {login} ni o'chirishni tasdiqlaysizmi? Bu amal akkauntni deaktiv qiladi.
                  </DialogDescription>
                </DialogHeader>
                <DialogFooter>
                  <DialogClose asChild>
                    <Button variant="outline">Bekor</Button>
                  </DialogClose>
                  <Button variant="destructive" onClick={deleteUser}>
                    Tasdiqlash
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </Panel>
        </div>

        <Panel
          title="Route ruxsatlari"
          description={`${grantedCount} / ${permissions.length} ta ruxsat berilgan · bo'limlar bo'yicha tartiblangan`}
          action={
            <div className="flex w-full flex-col gap-2 sm:w-auto sm:min-w-[320px]">
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  value={permissionSearch}
                  onChange={(e) => setPermissionSearch(e.target.value)}
                  placeholder="Nomi yoki route qidirish..."
                  className="h-9 rounded-xl pl-9"
                />
              </div>
              <div className="flex flex-wrap gap-1.5">
                {(
                  [
                    { id: "all", label: "Barchasi" },
                    { id: "granted", label: "Ruxsat bor" },
                    { id: "denied", label: "Yo'q" },
                  ] as const
                ).map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => setPermissionFilter(item.id)}
                    className={cn(
                      "rounded-lg px-2.5 py-1 text-xs font-medium transition-colors",
                      permissionFilter === item.id
                        ? "bg-primary text-primary-foreground"
                        : "bg-muted text-muted-foreground hover:text-foreground",
                    )}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
            </div>
          }
          noPadding
        >
          <div className="max-h-[min(78vh,920px)] overflow-auto">
            {permissionGroups.length === 0 ? (
              <div className="px-4 py-16 text-center text-sm text-muted-foreground">
                Route topilmadi
              </div>
            ) : (
              permissionGroups.map((group) => {
                const groupGranted = group.items.filter((item) => item.is_flag).length
                return (
                  <section key={group.id} className="border-b border-border/50 last:border-b-0">
                    <div className="sticky top-0 z-10 flex items-start justify-between gap-3 border-b border-border/40 bg-muted/95 px-4 py-3 backdrop-blur">
                      <div className="min-w-0">
                        <h3 className="text-sm font-semibold text-foreground">{group.title}</h3>
                        <p className="mt-0.5 text-xs text-muted-foreground">{group.description}</p>
                      </div>
                      <div className="shrink-0 rounded-lg bg-background/80 px-2.5 py-1 text-xs font-medium text-muted-foreground ring-1 ring-border/60">
                        {groupGranted}/{group.items.length}
                      </div>
                    </div>

                    <ul className="divide-y divide-border/40">
                      {group.items.map((permission) => (
                        <li
                          key={permission.id}
                          className="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
                        >
                          <div className="min-w-0 space-y-1.5">
                            <div className="flex flex-wrap items-center gap-2">
                              <span className="text-sm font-medium text-foreground">
                                {permissionTitle(permission)}
                              </span>
                              <span
                                className={cn(
                                  "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold",
                                  permission.is_flag
                                    ? "bg-emerald-500/10 text-emerald-700 dark:text-emerald-400"
                                    : "bg-muted text-muted-foreground",
                                )}
                              >
                                {permission.is_flag ? (
                                  <ShieldCheck className="size-3" />
                                ) : (
                                  <ShieldBan className="size-3" />
                                )}
                                {permission.is_flag ? "Bor" : "Yo'q"}
                              </span>
                            </div>
                            <code className="block truncate text-[11px] text-muted-foreground">
                              {permissionShortRoute(permission.route)}
                            </code>
                          </div>

                          <Button
                            variant={permission.is_flag ? "outline" : "default"}
                            size="sm"
                            className="h-9 shrink-0 rounded-xl sm:min-w-[110px]"
                            disabled={togglingRouteId === permission.id}
                            onClick={() =>
                              changePermission(permission.id, !permission.is_flag)
                            }
                          >
                            {togglingRouteId === permission.id
                              ? "..."
                              : permission.is_flag
                                ? "Olib tashlash"
                                : "Berish"}
                          </Button>
                        </li>
                      ))}
                    </ul>
                  </section>
                )
              })
            )}
          </div>
        </Panel>
      </div>
    </PageContainer>
  )
}
