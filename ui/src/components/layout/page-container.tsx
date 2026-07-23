import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

import { PageHeader } from "./page-header"



type PageContainerProps = {

  title: string

  titleAddon?: ReactNode

  description?: string

  center?: ReactNode

  actions?: ReactNode

  children: ReactNode

  className?: string

  contentClassName?: string

  fullWidth?: boolean

  /** @deprecated Page scroll is handled by the document; this prop is ignored. */

  scrollable?: boolean

}



export function PageContainer({

  title,

  titleAddon,

  description,

  center,

  actions,

  children,

  className,

  contentClassName,

  fullWidth,

}: PageContainerProps) {

  return (

    <div

      className={cn(

        "mx-auto flex w-full min-w-0 max-w-full flex-col animate-in fade-in slide-in-from-bottom-2 duration-500",

        fullWidth ? "max-w-none" : "max-w-[1600px]",

        className,

      )}

    >

      <PageHeader

        title={title}

        titleAddon={titleAddon}

        description={description}

        center={center}

        actions={actions}

      />

      <div className={cn("mt-3 space-y-3", contentClassName)}>{children}</div>

    </div>

  )

}


