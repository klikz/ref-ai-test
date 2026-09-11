import { Link, useLocation } from "react-router-dom"
import {
  ChevronLeft,
  ChevronRight,
  LogOut,
  Snowflake,
} from "lucide-react"
import { useState } from "react"
import { cn } from "@/lib/utils"
import { Global_Data } from "@/config/config"
import { navSections } from "@/lib/nav-config"
import { Button } from "@/components/ui/button"

export function AppSidebar() {
  const location = useLocation()
  const login = Global_Data.getLogin()
  const name = localStorage.getItem("name") || ""
  const [collapsed, setCollapsed] = useState(false)
  const visibleNavSections = Global_Data.visibleNavSections(navSections)

  async function handleLogout() {
    localStorage.removeItem("token")
    await Global_Data.clearUserData()
    window.location.href = "/login"
  }

  return (
    <aside
      className={cn(
        "sticky top-0 flex h-screen shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground transition-[width] duration-300 ease-out",
        collapsed ? "w-[72px]" : "w-[260px]",
      )}
    >
      <div className="flex h-16 items-center gap-3 border-b border-sidebar-border px-4">
        <div className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-primary/15 text-primary">
          <Snowflake className="size-5" />
        </div>
        {!collapsed && (
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-semibold tracking-tight">Premier REF</p>
            <p className="truncate text-xs text-muted-foreground">Ishlab chiqarish</p>
          </div>
        )}
      </div>

      <nav className="flex-1 space-y-6 overflow-y-auto px-3 py-4 [scrollbar-width:thin]">
        {visibleNavSections.map((section) => (
          <div key={section.label}>
            {!collapsed && (
              <p className="mb-2 px-2 text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">
                {section.label}
              </p>
            )}
            <ul className="space-y-0.5">
              {section.items.map((item) => {
                const isActive =
                  location.pathname === item.href ||
                  (item.href !== "/home" && location.pathname.startsWith(item.href + "/"))
                const Icon = item.icon
                return (
                  <li key={item.href}>
                    <Link
                      to={item.href}
                      title={collapsed ? item.title : undefined}
                      className={cn(
                        "group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all",
                        isActive
                          ? "bg-sidebar-primary text-sidebar-primary-foreground shadow-md shadow-primary/20"
                          : "text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
                      )}
                    >
                      <Icon
                        className={cn(
                          "size-[18px] shrink-0",
                          isActive ? "opacity-100" : "opacity-70 group-hover:opacity-100",
                        )}
                      />
                      {!collapsed && <span className="truncate">{item.title}</span>}
                    </Link>
                  </li>
                )
              })}
            </ul>
          </div>
        ))}
      </nav>

      <div className="border-t border-sidebar-border p-3 space-y-2">
        {!collapsed && (login || name) && (
          <div className="rounded-xl bg-sidebar-accent/50 px-3 py-2.5">
            <p className="truncate text-xs text-muted-foreground">Foydalanuvchi</p>
            <p className="truncate text-sm font-medium">{name || login}</p>
          </div>
        )}
        <div className={cn("flex gap-1", collapsed ? "flex-col" : "flex-row")}>
          <Button
            variant="ghost"
            size={collapsed ? "icon" : "sm"}
            className={cn("text-sidebar-foreground/80", !collapsed && "flex-1")}
            onClick={() => setCollapsed((c) => !c)}
          >
            {collapsed ? <ChevronRight className="size-4" /> : <ChevronLeft className="size-4" />}
            {!collapsed && <span className="ml-1">Yig‘ish</span>}
          </Button>
          <Button
            variant="ghost"
            size={collapsed ? "icon" : "sm"}
            className="text-sidebar-foreground/80 hover:text-destructive"
            onClick={handleLogout}
            title="Chiqish"
          >
            <LogOut className="size-4" />
            {!collapsed && <span className="ml-1">Chiqish</span>}
          </Button>
        </div>
      </div>
    </aside>
  )
}
