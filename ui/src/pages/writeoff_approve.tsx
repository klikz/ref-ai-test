import { useCallback, useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { Check, CheckCircle2, Clock, X } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { Global_Data } from "@/config/config"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
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
import {
  type WriteoffApproveResult,
  type WriteoffDocument,
  type WriteoffDocumentApproval,
  type WriteoffDocumentItem,
  writeoffStatusLabel,
} from "@/pages/writeoff_shared"
import { formatQty } from "@/lib/ware-quantity"
import { cn } from "@/lib/utils"

export default function WriteoffApprovePage() {
  const currentUserId = Global_Data.id
  const [documents, setDocuments] = useState<WriteoffDocument[]>([])
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [selectedDocument, setSelectedDocument] = useState<WriteoffDocument | null>(null)
  const [items, setItems] = useState<WriteoffDocumentItem[]>([])
  const [approvals, setApprovals] = useState<WriteoffDocumentApproval[]>([])
  const [rejectOpen, setRejectOpen] = useState(false)
  const [rejectComment, setRejectComment] = useState("")
  const [busy, setBusy] = useState(false)

  const loadPending = useCallback(async () => {
    const result = await Backend_Request<WriteoffDocument[]>(
      { status: "pending", limit: 100 },
      "/api/writeoff/documents/list",
    )
    if (result.result === "ok") {
      setDocuments(result.data ?? [])
    } else {
      ShowErrorToast(result.error || "Hujjatlar yuklanmadi")
    }
  }, [])

  const loadDocumentDetails = useCallback(async (documentId: number) => {
    const result = await Backend_Request<{
      document: WriteoffDocument
      items: WriteoffDocumentItem[]
      approvals: WriteoffDocumentApproval[]
    }>({ document_id: documentId }, "/api/writeoff/documents/get")
    if (result.result === "ok" && result.data) {
      setSelectedDocument(result.data.document)
      setItems(result.data.items)
      setApprovals(result.data.approvals ?? [])
    }
  }, [])

  useEffect(() => {
    void loadPending()
  }, [loadPending])

  useEffect(() => {
    if (selectedId) {
      void loadDocumentDetails(selectedId)
    } else {
      setSelectedDocument(null)
      setItems([])
      setApprovals([])
    }
  }, [selectedId, loadDocumentDetails])

  const currentUserApproval = useMemo(
    () => approvals.find((row) => row.user_id === currentUserId) ?? null,
    [approvals, currentUserId],
  )

  const canApprove = Boolean(currentUserApproval && !currentUserApproval.is_approved)

  async function approve(documentId: number) {
    setBusy(true)
    const result = await Backend_Request<WriteoffApproveResult>(
      { document_id: documentId },
      "/api/writeoff/documents/approve",
    )
    setBusy(false)
    if (result.result !== "ok" || !result.data) {
      ShowErrorToast(result.error || "Tasdiqlash amalga oshmadi")
      return
    }
    if (result.data.all_approved) {
      ShowOKToast("Barcha mas'ullar tasdiqladi — hujjat yakunlandi")
      setSelectedId(null)
      await loadPending()
      return
    }
    ShowOKToast(
      `Tasdiqlashingiz qabul qilindi (${result.data.approval_done}/${result.data.approval_required})`,
    )
    await loadDocumentDetails(documentId)
    await loadPending()
  }

  async function reject() {
    if (!selectedId) {
      return
    }
    setBusy(true)
    const result = await Backend_Request(
      { document_id: selectedId, comment: rejectComment },
      "/api/writeoff/documents/reject",
    )
    setBusy(false)
    if (result.result === "ok") {
      ShowOKToast("Rad etildi")
      setRejectOpen(false)
      setRejectComment("")
      setSelectedId(null)
      await loadPending()
    } else {
      ShowErrorToast(result.error || "Rad etish amalga oshmadi")
    }
  }

  return (
    <PageContainer
      title="Hisobdan chiqarish tasdiqlash"
      description="Barcha mas'ullar tasdiqlagach, hujjat yakunlanadi."
      actions={
        <Button variant="outline" asChild>
          <Link to="/writeoff">Hujjatlar ro'yxati</Link>
        </Button>
      }
    >
      <div className="grid gap-4 lg:grid-cols-2">
        <Panel title={`${documents.length} ta kutilmoqda`}>
          <div className="overflow-auto rounded-xl border border-border/60">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>№</TableHead>
                  <TableHead>Muallif</TableHead>
                  <TableHead>Tasdiqlash</TableHead>
                  <TableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {documents.length ? (
                  documents.map((doc) => (
                    <TableRow key={doc.id} data-state={selectedId === doc.id ? "selected" : undefined}>
                      <TableCell>#{doc.id}</TableCell>
                      <TableCell>{doc.created_by_name}</TableCell>
                      <TableCell>
                        {doc.approval_required
                          ? `${doc.approval_done ?? 0}/${doc.approval_required}`
                          : "—"}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          size="sm"
                          variant={selectedId === doc.id ? "default" : "outline"}
                          onClick={() => setSelectedId(doc.id)}
                        >
                          Ko'rish
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={4} className="h-24 text-center text-muted-foreground">
                      Kutilayotgan hujjatlar yo'q
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </Panel>

        <Panel title={selectedId ? `Hujjat #${selectedId}` : "Tafsilotlar"}>
          {selectedId ? (
            <>
              <div className="mb-4 flex flex-wrap items-center gap-2">
                <Button className="gap-2" disabled={busy || !canApprove} onClick={() => void approve(selectedId)}>
                  <Check className="size-4" />
                  {currentUserApproval?.is_approved ? "Siz tasdiqlagansiz" : "Tasdiqlash"}
                </Button>
                <Button variant="outline" className="gap-2" disabled={busy} onClick={() => setRejectOpen(true)}>
                  <X className="size-4" />
                  Rad etish
                </Button>
                <Button variant="ghost" asChild>
                  <Link to={`/writeoff/${selectedId}`}>To'liq ochish</Link>
                </Button>
                {selectedDocument?.approval_required ? (
                  <span className="text-sm text-muted-foreground">
                    {selectedDocument.approval_done ?? 0}/{selectedDocument.approval_required} tasdiqlangan
                  </span>
                ) : null}
              </div>

              <div className="mb-4 overflow-auto rounded-xl border border-border/60">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Mas'ul</TableHead>
                      <TableHead>Holat</TableHead>
                      <TableHead>Vaqt</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {approvals.length ? (
                      approvals.map((row) => (
                        <TableRow key={row.id}>
                          <TableCell>
                            <div>
                              <p className="font-medium">{row.user_name || row.user_login}</p>
                              {row.user_name ? (
                                <p className="text-xs text-muted-foreground">{row.user_login}</p>
                              ) : null}
                            </div>
                          </TableCell>
                          <TableCell>
                            <span
                              className={cn(
                                "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium",
                                row.is_approved
                                  ? "bg-emerald-100 text-emerald-800"
                                  : "bg-amber-100 text-amber-800",
                              )}
                            >
                              {row.is_approved ? (
                                <CheckCircle2 className="size-3.5" />
                              ) : (
                                <Clock className="size-3.5" />
                              )}
                              {row.is_approved ? "Tasdiqlangan" : "Kutilmoqda"}
                            </span>
                          </TableCell>
                          <TableCell>{row.approved_at || "—"}</TableCell>
                        </TableRow>
                      ))
                    ) : (
                      <TableRow>
                        <TableCell colSpan={3} className="h-16 text-center text-muted-foreground">
                          Mas'ullar ro'yxati bo'sh
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>

              <div className="overflow-auto rounded-xl border border-border/60">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Liniya</TableHead>
                      <TableHead>Pozitsiya</TableHead>
                      <TableHead>Serial</TableHead>
                      <TableHead>Miqdor</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {items.map((item) => (
                      <TableRow key={item.id}>
                        <TableCell>{item.line_name}</TableCell>
                        <TableCell>{item.item_label}</TableCell>
                        <TableCell>{item.serial || "—"}</TableCell>
                        <TableCell>{formatQty(item.quantity)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </>
          ) : (
            <p className="text-sm text-muted-foreground">Chapdan hujjatni tanlang.</p>
          )}
        </Panel>
      </div>

      <Dialog open={rejectOpen} onOpenChange={setRejectOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rad etish</DialogTitle>
            <DialogDescription>
              Hujjat #{selectedId} — {writeoffStatusLabel("pending")}
            </DialogDescription>
          </DialogHeader>
          <Input
            value={rejectComment}
            onChange={(e) => setRejectComment(e.target.value)}
            placeholder="Izoh (ixtiyoriy)"
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setRejectOpen(false)}>
              Bekor qilish
            </Button>
            <Button variant="destructive" disabled={busy} onClick={() => void reject()}>
              Rad etish
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PageContainer>
  )
}
