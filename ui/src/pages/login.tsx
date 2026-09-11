import { Global_Data } from "@/config/config"
import { Backend_Request } from "@/services/backend"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ShowErrorToast } from "@/components/showToast"
import { Snowflake, Loader2 } from "lucide-react"
import { useState } from "react"

export default function LoginPage() {
  const [login, setLogin] = useState("")
  const [password, setPassword] = useState("")
  const [loading, setLoading] = useState(false)

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    const result = await Backend_Request({ login, password }, "/user/login")
    setLoading(false)
    if (result?.result === "ok" && result.data) {
      const d = result.data as {
        token: string
        role: string
        id: number
        login: string
        name: string
        role_id: number
      }
      Global_Data.setUserData(true, d.token, d.role, d.id, d.login, d.name, d.role_id)
      Global_Data.loadLocalData()
      window.location.href = "/home"
    } else {
      ShowErrorToast(result.error || "Login xatolik")
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden p-4">
      <div className="pointer-events-none absolute inset-0 app-mesh" />
      <div className="pointer-events-none absolute -left-32 top-1/4 size-96 rounded-full bg-primary/20 blur-3xl" />
      <div className="pointer-events-none absolute -right-32 bottom-1/4 size-96 rounded-full bg-chart-2/15 blur-3xl" />

      <div className="relative w-full max-w-md animate-in fade-in zoom-in-95 duration-500">
        <div className="glass-card p-8 sm:p-10">
          <div className="mb-8 flex flex-col items-center text-center">
            <div className="mb-4 flex size-14 items-center justify-center rounded-2xl bg-primary/15 text-primary shadow-lg shadow-primary/20">
              <Snowflake className="size-7" />
            </div>
            <h1 className="text-2xl font-bold tracking-tight">Premier REF</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              Ishlab chiqarish boshqaruv tizimi
            </p>
          </div>

          <form onSubmit={handleLogin} className="space-y-5">
            <div className="space-y-2">
              <Label htmlFor="login">Login</Label>
              <Input
                id="login"
                placeholder="login"
                value={login}
                onChange={(e) => setLogin(e.target.value)}
                autoComplete="username"
                className="h-11"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Parol</Label>
              <Input
                id="password"
                type="password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="current-password"
                className="h-11"
              />
            </div>
            <Button type="submit" className="h-11 w-full text-base font-semibold" disabled={loading}>
              {loading ? (
                <>
                  <Loader2 className="mr-2 size-4 animate-spin" />
                  Kirilmoqda...
                </>
              ) : (
                "Kirish"
              )}
            </Button>
          </form>
        </div>
        <p className="mt-6 text-center text-xs text-muted-foreground">
          © Premier — sovutgich ishlab chiqarish
        </p>
      </div>
    </div>
  )
}
