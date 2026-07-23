import { Loader2 } from "lucide-react"

export function PageLoader() {
  return (
    <div className="flex min-h-[40vh] flex-col items-center justify-center gap-3 text-muted-foreground">
      <Loader2 className="size-8 animate-spin text-primary" />
      <p className="text-sm font-medium">Yuklanmoqda...</p>
    </div>
  )
}
