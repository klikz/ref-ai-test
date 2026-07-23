import { Global_Data } from "@/config/config"

export function labelAssetUrl(path?: string): string {
  if (!path) {
    return ""
  }
  if (path.startsWith("http") || path.startsWith("data:")) {
    return path
  }
  return `${Global_Data.server_ip}${path}`
}

export function fileToDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result))
    reader.onerror = () => reject(new Error("Faylni o'qib bo'lmadi"))
    reader.readAsDataURL(file)
  })
}

export function getImageDimensions(src: string): Promise<{ width: number; height: number }> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve({ width: img.naturalWidth, height: img.naturalHeight })
    img.onerror = () => reject(new Error("Rasm yuklanmadi"))
    img.src = src
  })
}

export function calcImageSizeMm(
  naturalWidth: number,
  naturalHeight: number,
  maxWidthMm = 35,
  maxHeightMm = 35,
): { width: number; height: number } {
  if (naturalWidth <= 0 || naturalHeight <= 0) {
    return { width: 20, height: 10 }
  }

  const ratio = naturalHeight / naturalWidth
  let width = maxWidthMm
  let height = Math.round(width * ratio * 10) / 10

  if (height > maxHeightMm) {
    height = maxHeightMm
    width = Math.round((height / ratio) * 10) / 10
  }

  return { width, height }
}

export function isImageFile(file: File): boolean {
  return file.type.startsWith("image/")
}

export function pickImageFile(files: FileList | null): File | null {
  if (!files?.length) {
    return null
  }
  for (const file of Array.from(files)) {
    if (isImageFile(file)) {
      return file
    }
  }
  return null
}
