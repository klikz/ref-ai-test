import { useEffect, type FocusEvent, type RefObject } from "react"

const KEYBOARD_SCROLL_DELAY_MS = 350
const VIEWPORT_MARGIN_PX = 16

/** Scroll focused input above the on-screen keyboard (mobile / tablet). */
export function scrollFocusedInputIntoView(element: HTMLElement) {
  window.setTimeout(() => {
    const viewport = window.visualViewport
    if (!viewport) {
      element.scrollIntoView({ block: "center", behavior: "smooth" })
      return
    }

    const rect = element.getBoundingClientRect()
    const visibleTop = viewport.offsetTop + VIEWPORT_MARGIN_PX
    const visibleBottom = viewport.offsetTop + viewport.height - VIEWPORT_MARGIN_PX

    if (rect.bottom > visibleBottom) {
      window.scrollBy({ top: rect.bottom - visibleBottom, behavior: "smooth" })
      return
    }
    if (rect.top < visibleTop) {
      window.scrollBy({ top: rect.top - visibleTop, behavior: "smooth" })
    }
  }, KEYBOARD_SCROLL_DELAY_MS)
}

export function onInputFocusForKeyboard(event: FocusEvent<HTMLInputElement>) {
  scrollFocusedInputIntoView(event.currentTarget)
}

export function useKeyboardSafeInput(ref: RefObject<HTMLInputElement | null>) {
  useEffect(() => {
    const input = ref.current
    if (!input) {
      return
    }

    const onViewportResize = () => {
      if (document.activeElement === input) {
        scrollFocusedInputIntoView(input)
      }
    }

    const viewport = window.visualViewport
    viewport?.addEventListener("resize", onViewportResize)
    return () => viewport?.removeEventListener("resize", onViewportResize)
  }, [ref])
}
