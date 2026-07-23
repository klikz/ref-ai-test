import { AppTopNav } from "@/components/layout/app-top-nav"
import { Outlet } from "react-router-dom"

export default function MainLayout() {
  return (
    <div className="app-mesh min-h-svh w-full max-w-full min-w-0 bg-background">
      <AppTopNav />
      <main className="mx-auto w-full min-w-0 max-w-[1800px] p-2 pb-6 sm:p-3 sm:pb-8 lg:p-4 lg:pb-10">
        <Outlet />
      </main>
    </div>
  )
}
