import { Link, useLocation } from "react-router-dom"
import { ChevronDown, LogOut, Menu, Moon, Snowflake, Sun } from "lucide-react"
import { cn } from "@/lib/utils"
import { Global_Data } from "@/config/config"
import { navSections } from "@/lib/nav-config"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useTheme } from "@/components/theme-provider"

export function AppTopNav() {
  const location = useLocation()
  const login = Global_Data.getLogin()
  const name = localStorage.getItem("name") || ""
  const { theme, setTheme } = useTheme()
  const visibleNavSections = Global_Data.visibleNavSections(navSections)

  async function handleLogout() {
    localStorage.removeItem("token")
    await Global_Data.clearUserData()
    window.location.href = "/login"
  }

  function isActivePath(href: string) {
    return location.pathname === href || (href !== "/home" && location.pathname.startsWith(href + "/"))
  }

  return (
    <header className="sticky top-0 z-50 border-b border-border/60 bg-background/75 backdrop-blur-2xl">
      <div className="mx-auto flex h-14 w-full min-w-0 max-w-[1800px] items-center gap-2 px-3 sm:px-4 lg:px-5">
        <Link to="/home" className="group flex shrink-0 items-center gap-3">
          <div className="relative flex size-9 items-center justify-center overflow-hidden rounded-xl bg-primary text-primary-foreground shadow-lg shadow-primary/25">
            <Snowflake className="size-4 transition-transform duration-300 group-hover:rotate-45" />
            <div className="absolute inset-0 bg-gradient-to-br from-white/20 to-transparent" />
          </div>
          <div className="hidden sm:block">
            <p className="text-sm font-bold leading-none tracking-tight">Premier AC</p>
            <p className="mt-1 text-xs text-muted-foreground">Ishlab chiqarish</p>
          </div>
        </Link>

        <nav className="hidden min-w-0 flex-1 items-center justify-center gap-1 lg:flex">
          {visibleNavSections.map((section) => {
            const sectionActive = section.items.some((item) => isActivePath(item.href))
            return (
              <DropdownMenu key={section.label} modal={false}>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant={sectionActive ? "secondary" : "ghost"}
                    className={cn(
                      "h-9 gap-2 rounded-full px-3 text-sm font-semibold",
                      sectionActive && "bg-primary/10 text-primary hover:bg-primary/15",
                    )}
                  >
                    {section.label}
                    <ChevronDown className="size-4 opacity-60" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  align="center"
                  sideOffset={12}
                  className="w-64 rounded-2xl border-border/60 bg-popover/95 p-2 shadow-2xl shadow-black/10 backdrop-blur-xl"
                >
                  <DropdownMenuLabel className="px-3 py-2">{section.label}</DropdownMenuLabel>
                  <DropdownMenuGroup>
                    {section.items.map((item) => {
                      const Icon = item.icon
                      const active = isActivePath(item.href)
                      return (
                        <DropdownMenuItem key={item.href} asChild>
                          <Link
                            to={item.href}
                            className={cn(
                              "flex cursor-pointer items-center gap-3 rounded-xl px-3 py-2.5",
                              active && "bg-primary/10 text-primary",
                            )}
                          >
                            <span className="flex size-8 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                              <Icon className="size-4" />
                            </span>
                            <span className="min-w-0">
                              <span className="block truncate text-sm font-medium">{item.title}</span>
                              {item.description ? (
                                <span className="block truncate text-xs text-muted-foreground">
                                  {item.description}
                                </span>
                              ) : null}
                            </span>
                          </Link>
                        </DropdownMenuItem>
                      )
                    })}
                  </DropdownMenuGroup>
                </DropdownMenuContent>
              </DropdownMenu>
            )
          })}
        </nav>

        <div className="ml-auto flex items-center gap-2">
          <DropdownMenu modal={false}>
            <DropdownMenuTrigger asChild>
            <Button variant="outline" size="icon" className="size-9 lg:hidden">
                <Menu className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="end"
              sideOffset={12}
              className="w-72 rounded-2xl border-border/60 bg-popover/95 p-2 shadow-2xl backdrop-blur-xl"
            >
              {visibleNavSections.map((section) => (
                <div key={section.label}>
                  <DropdownMenuLabel>{section.label}</DropdownMenuLabel>
                  {section.items.map((item) => {
                    const Icon = item.icon
                    return (
                      <DropdownMenuItem key={item.href} asChild>
                        <Link to={item.href} className="flex cursor-pointer items-center gap-2 rounded-lg px-2 py-2">
                          <Icon className="size-4" />
                          {item.title}
                        </Link>
                      </DropdownMenuItem>
                    )
                  })}
                  <DropdownMenuSeparator />
                </div>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>

          <Button
            variant="ghost"
            size="icon"
            className="size-9 rounded-full"
            onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
            title="Theme"
          >
            <Sun className="size-4 dark:hidden" />
            <Moon className="hidden size-4 dark:block" />
          </Button>

          <div className="hidden h-8 w-px bg-border sm:block" />

          <DropdownMenu modal={false}>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-9 rounded-full px-2 sm:px-3">
                <span className="flex size-7 items-center justify-center rounded-full bg-primary/10 text-xs font-bold text-primary">
                  {(name || login || "U").slice(0, 1).toUpperCase()}
                </span>
                <span className="hidden max-w-36 truncate text-sm font-medium sm:inline">
                  {name || login || "User"}
                </span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" sideOffset={12} className="w-56 rounded-2xl p-2">
              <DropdownMenuLabel>
                <span className="block truncate">{name || "Foydalanuvchi"}</span>
                {login ? <span className="block truncate text-xs font-normal">{login}</span> : null}
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={handleLogout} variant="destructive" className="cursor-pointer">
                <LogOut className="size-4" />
                Chiqish
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </header>
  )
}
