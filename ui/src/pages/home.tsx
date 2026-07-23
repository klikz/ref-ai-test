import { Link } from "react-router-dom"
import { PageContainer } from "@/components/layout/page-container"
import { homeQuickLinks } from "@/lib/nav-config"
import { Global_Data } from "@/config/config"
import { ArrowRight, Sparkles } from "lucide-react"
import { cn } from "@/lib/utils"

export default function Home() {
  const name = localStorage.getItem("name") || Global_Data.getLogin()

  return (
    <PageContainer
      title={`Xush kelibsiz${name ? `, ${name}` : ""}`}
      description="Modellar, komponentlar, hisobot va dashboardga tezkor kirish"
    >
      <div className="glass-card flex items-start gap-4 border-primary/20 bg-primary/5 p-5">
        <div className="rounded-xl bg-primary/15 p-2.5 text-primary">
          <Sparkles className="size-5" />
        </div>
        <div>
          <p className="font-medium text-foreground">Bugungi ish</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Quyidagi bo‘limlardan kerakli sahifani tanlang. Dashboard ekranida real
            vaqt rejasi ko‘rsatiladi.
          </p>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {homeQuickLinks.map((item) => {
          const Icon = item.icon
          return (
            <Link
              key={item.href}
              to={item.href}
              className={cn(
                "group relative overflow-hidden rounded-2xl border border-border/60 bg-card p-5",
                "shadow-sm transition-all duration-300 hover:border-primary/30 hover:shadow-lg hover:shadow-primary/10",
              )}
            >
              <div className="flex items-start justify-between">
                <div className="rounded-xl bg-muted p-2.5 text-primary transition-colors group-hover:bg-primary/15">
                  <Icon className="size-5" />
                </div>
                <ArrowRight className="size-4 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-primary" />
              </div>
              <h3 className="mt-4 font-semibold text-foreground">{item.title}</h3>
              {item.description ? (
                <p className="mt-1 text-sm text-muted-foreground">{item.description}</p>
              ) : null}
            </Link>
          )
        })}
      </div>
    </PageContainer>
  )
}
