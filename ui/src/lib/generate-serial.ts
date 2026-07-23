export function generateSerial(prefix: string, count: number): string {
  const now = new Date()
  const year = String.fromCharCode("A".charCodeAt(0) + (now.getFullYear() - 2023))

  const monthValue = now.getMonth() + 1
  const month =
    monthValue < 10
      ? String(monthValue)
      : String.fromCharCode("A".charCodeAt(0) + (monthValue - 10))

  const dayValue = now.getDate()
  const day =
    dayValue < 10
      ? String(dayValue)
      : String.fromCharCode("A".charCodeAt(0) + (dayValue - 10))

  const counter = String(Math.max(0, count)).padStart(4, "0")
  const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
  const randomLetter = letters[Math.floor(Math.random() * letters.length)]

  return `${prefix}${year}${month}${day}${counter}${randomLetter}`
}
