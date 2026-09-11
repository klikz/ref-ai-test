import { useEffect, useState } from "react"
import { Bot, Loader2, RefreshCw, Send } from "lucide-react"
import { useNavigate } from "react-router-dom"
import { Backend_Request } from "@/services/backend"
import { Global_Data } from "@/config/config"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"

type AgentTask = {
  id: number
  prompt: string
  origin: string
  status: string
  created_by: number
  git_sha?: string
  log_text?: string
  error_text?: string
  created_at: string
  updated_at: string
}

export default function AgentPage() {
  const navigate = useNavigate()
  const [allowed, setAllowed] = useState<boolean | null>(null)
  const [prompt, setPrompt] = useState("")
  const [origin, setOrigin] = useState<"server" | "pc">("server")
  const [tasks, setTasks] = useState<AgentTask[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)

  async function loadAccess() {
    const result = await Backend_Request<{ allowed: boolean; user_id: number }>({}, "/api/agent/access")
    if (result.result !== "ok") {
      setAllowed(false)
      Global_Data.setAgentAllowed(false)
      return false
    }
    const ok = Boolean(result.data?.allowed)
    setAllowed(ok)
    Global_Data.setAgentAllowed(ok)
    return ok
  }

  async function loadTasks() {
    setLoading(true)
    const result = await Backend_Request<AgentTask[]>({}, "/api/agent/tasks")
    setLoading(false)
    if (result.result === "ok") {
      setTasks(result.data ?? [])
    } else {
      setTasks([])
      ShowErrorToast(result.error || "Vazifalar yuklanmadi")
    }
  }

  useEffect(() => {
    void (async () => {
      const ok = await loadAccess()
      if (!ok) {
        ShowErrorToast("Agent moduli faqat egasi uchun")
        navigate("/home", { replace: true })
        return
      }
      await loadTasks()
    })()
  }, [navigate])

  useEffect(() => {
    const active = tasks.some((t) => t.status === "queued" || t.status === "running")
    if (!active || allowed !== true) return
    const timer = window.setInterval(() => {
      void loadTasks()
    }, 3000)
    return () => window.clearInterval(timer)
  }, [tasks, allowed])

  async function handleCreate() {
    if (!prompt.trim()) {
      ShowErrorToast("Vazifa matnini yozing")
      return
    }
    setSubmitting(true)
    const result = await Backend_Request<AgentTask>(
      { prompt: prompt.trim(), origin },
      "/api/agent/tasks/create",
    )
    setSubmitting(false)
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Vazifa yaratilmadi")
      return
    }
    ShowOKToast("Vazifa yaratildi")
    setPrompt("")
    await loadTasks()
  }

  async function handleApproveTest(id: number) {
    const result = await Backend_Request<AgentTask>({ id }, "/api/agent/tasks/approve-test")
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Test tasdiqlanmadi")
      return
    }
    ShowOKToast("Testga chiqarish tasdiqlandi")
    await loadTasks()
  }

  async function handleApproveProd(id: number) {
    const result = await Backend_Request<AgentTask>({ id }, "/api/agent/tasks/approve-prod")
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Prod tasdiqlanmadi")
      return
    }
    ShowOKToast("Prodga chiqarish tasdiqlandi")
    await loadTasks()
  }

  async function handleReject(id: number) {
    const result = await Backend_Request<AgentTask>(
      { id, reason: "Owner rejected" },
      "/api/agent/tasks/reject",
    )
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Rad etilmadi")
      return
    }
    ShowOKToast("Vazifa rad etildi")
    await loadTasks()
  }

  if (allowed !== true) {
    return (
      <PageContainer title="Agent" description="Tekshirilmoqda...">
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          Ruxsat tekshirilmoqda
        </div>
      </PageContainer>
    )
  }

  return (
    <PageContainer
      title="Agent vazifalar"
      description="Faqat egasi — PC yoki serverdan galma-gal vazifa. Deploy tasdiqlari shu yerda."
      fullWidth
      actions={
        <Button variant="outline" className="h-10 rounded-xl" onClick={() => void loadTasks()}>
          <RefreshCw className="size-4" />
          Yangilash
        </Button>
      }
    >
      <div className="grid gap-4 lg:grid-cols-[1fr_1.2fr]">
        <Panel title="Yangi vazifa">
          <div className="space-y-3">
            <div className="flex gap-2">
              <Button
                type="button"
                variant={origin === "server" ? "default" : "outline"}
                className="h-9 rounded-xl"
                onClick={() => setOrigin("server")}
              >
                Server
              </Button>
              <Button
                type="button"
                variant={origin === "pc" ? "default" : "outline"}
                className="h-9 rounded-xl"
                onClick={() => setOrigin("pc")}
              >
                PC (Cursor)
              </Button>
            </div>
            <textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Masalan: yangi liniya qo'shilsin, report filter qo'shilsin..."
              className="min-h-36 w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
            />
            <Button
              className="h-10 rounded-xl"
              disabled={submitting}
              onClick={() => void handleCreate()}
            >
              {submitting ? <Loader2 className="size-4 animate-spin" /> : <Send className="size-4" />}
              Vazifa yuborish
            </Button>
            <p className="text-xs text-muted-foreground">
              <b>Server</b> origin: Cursor SDK worker (`ref-ai-agent-worker`) avtomatik bajaradi.
              <b> PC</b>: Cursor IDE da qo&apos;lda. Bir vaqtda bitta faol (queued/running) vazifa.
            </p>
          </div>
        </Panel>

        <Panel title="Oxirgi vazifalar">
          {loading ? (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 className="size-4 animate-spin" />
              Yuklanmoqda...
            </div>
          ) : tasks.length === 0 ? (
            <p className="text-sm text-muted-foreground">Hali vazifa yo'q.</p>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-border/60">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>Origin</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Prompt</TableHead>
                    <TableHead>Amallar</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tasks.map((task) => (
                    <TableRow key={task.id}>
                      <TableCell className="font-mono text-xs">{task.id}</TableCell>
                      <TableCell>{task.origin}</TableCell>
                      <TableCell>
                        <span className="rounded-md bg-muted px-2 py-1 text-xs">{task.status}</span>
                      </TableCell>
                      <TableCell className="max-w-[280px]">
                        <div className="truncate text-sm" title={task.prompt}>
                          {task.prompt}
                        </div>
                        {task.log_text ? (
                          <div className="mt-1 line-clamp-2 text-xs text-muted-foreground">
                            {task.log_text}
                          </div>
                        ) : null}
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          {(task.status === "ready_for_test" || task.status === "failed") && (
                            <Button
                              size="sm"
                              className="h-8 rounded-lg"
                              onClick={() => void handleApproveTest(task.id)}
                            >
                              Testga
                            </Button>
                          )}
                          {task.status === "ready_for_prod" && (
                            <Button
                              size="sm"
                              className="h-8 rounded-lg"
                              onClick={() => void handleApproveProd(task.id)}
                            >
                              Prodga
                            </Button>
                          )}
                          {!["promoted", "rejected"].includes(task.status) && (
                            <Button
                              size="sm"
                              variant="outline"
                              className="h-8 rounded-lg"
                              onClick={() => void handleReject(task.id)}
                            >
                              Rad
                            </Button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </Panel>
      </div>

      <div className="mt-4">
        <Panel title="Qisqa oqim">
          <ol className="list-decimal space-y-1 pl-5 text-sm text-muted-foreground">
            <li>Vazifa yozing (PC yoki Server).</li>
            <li>Kodni gitga push qiling (Cursor / worker).</li>
            <li>
              <Bot className="mr-1 inline size-3.5" />
              Testga — serverda pull/build/restart qiling.
            </li>
            <li>Test OK bo'lsa — Prodga tasdiqlang.</li>
          </ol>
        </Panel>
      </div>
    </PageContainer>
  )
}
