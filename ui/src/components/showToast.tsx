import { toast } from "sonner"

export function ShowOKToast(text: string) {
  toast.success(text, { position: "top-right" })
}

export function ShowWarningToast(text: string) {
  toast.warning(text, { position: "top-right" })
}

export function ShowErrorToast(text: string) {
  toast.error(text, { position: "top-center" })
}
