export function parseAccSerialValues(value: string): string[] {
  const seen = new Set<string>()
  const result: string[] = []
  for (const part of value.split(/[\n\r,;]+/)) {
    const trimmed = part.trim()
    if (!trimmed || seen.has(trimmed)) {
      continue
    }
    seen.add(trimmed)
    result.push(trimmed)
  }
  return result
}

export function formatAccSerialDisplay(value: string): string {
  return parseAccSerialValues(value).join(", ")
}

export function formatAccSerialStorage(value: string): string {
  return parseAccSerialValues(value).join("\n")
}

export function hasAccSerialValues(value: string): boolean {
  return parseAccSerialValues(value).length > 0
}
