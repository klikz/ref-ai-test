import type { LucideIcon } from "lucide-react"
import {
  FileSpreadsheet,
  Factory,
  FlaskConical,
  Home,
  ClipboardList,
  Layers,
  Monitor,
  Package,
  Printer,
  QrCode,
  RefreshCw,
  Scale,
  ScanLine,
  ScanSearch,
  Tag,
  Target,
  Clock,
  Trash2,
  Truck,
  UserCog,
  Users,
  UserCheck,
  Warehouse,
} from "lucide-react"

export type NavItem = {
  title: string
  href: string
  icon: LucideIcon
  description?: string
}

export type NavSection = {
  label: string
  items: NavItem[]
  adminOnly?: boolean
}

export const navSections: NavSection[] = [
  {
    label: "Asosiy",
    items: [
      { title: "Bosh sahifa", href: "/home", icon: Home, description: "Tezkor kirish" },
      { title: "Dashboard", href: "/dashboard", icon: Monitor, description: "Liniya monitori" },
      { title: "Serial Info", href: "/serial-info", icon: ScanSearch, description: "Serial/kompressor ma'lumot va skan surati" },
      { title: "Laboratoriya", href: "/lab", icon: FlaskConical, description: "VTM laboratoriyasi test natijalari" },
    ],
  },
  {
    label: "Ombor",
    items: [
      { title: "Kirim", href: "/ombor/kirim", icon: Warehouse, description: "Komponent kirimi" },
      { title: "Buyurtma", href: "/ombor/buyurtma", icon: ClipboardList, description: "Sarf normasi buyurtmasi" },
      { title: "Chiqim", href: "/ombor/chiqim", icon: Package, description: "Nakladnomalar ro'yxati" },
      { title: "Hisobdan chiqarish", href: "/writeoff", icon: Trash2, description: "Komponent va mahsulot hisobdan chiqarish" },
      { title: "Hisobdan chiqarish tasdiqlash", href: "/writeoff/approve", icon: UserCheck, description: "Barcha mas'ullar tasdiqlashi kerak" },
      { title: "Hisobdan chiqarish arxivi", href: "/writeoff/records", icon: FileSpreadsheet, description: "Tasdiqlangan yozuvlar" },
    ],
  },
  {
    label: "Liniyalar",
    items: [
      { title: "Yi'g'ish liniyasi", href: "/yigish", icon: Factory, description: "Rejadagi modeldan serial chop etish" },
      { title: "Eshik liniyasi", href: "/eshik", icon: Factory, description: "Rejadagi komponentdan serial chop etish" },
      { title: "Qadoqlash Liniyasi", href: "/qadoqlash", icon: Factory, description: "Lab + acc/eshik/product skan va chop" },
      { title: "Brigadir", href: "/brigadir", icon: UserCheck, description: "Liniya komponentlarini qabul qilish" },
    ],
  },
  {
    label: "Production",
    items: [
      { title: "Modellar", href: "/models", icon: Package },
      { title: "Components", href: "/production/components", icon: Layers },
      { title: "Eshik komponentlari", href: "/production/eshik", icon: Layers, description: "Eshik liniyasi komponentlari va serial prefix" },
      { title: "Sarf normasi", href: "/production/consumption-norm", icon: ClipboardList },
      { title: "Liniya mas'ullari", href: "/production/line-responsibles", icon: UserCog },
      { title: "Hisobdan chiqarish mas'ullari", href: "/writeoff/responsibles", icon: UserCog, description: "Barcha mas'ullar tasdiqlashi shart" },
      { title: "GS Code", href: "/gscode", icon: QrCode },
      { title: "Printerlar", href: "/printers-v2", icon: Printer, description: "Liniya printerlari" },
      { title: "Etiketkalar", href: "/label-templates", icon: Tag, description: "Etiketka shablonlari" },
      { title: "Ishlab chiqarish rejasi", href: "/production/plan", icon: Target, description: "Kunlik reja va Excel" },
      { title: "Smena vaqtlari", href: "/production/shifts", icon: Clock, description: "1-sm va 2-sm boshlanish/tugash" },
      { title: "Reja dashboard", href: "/production/plan/dashboard", icon: Monitor, description: "Reja bajarilishi" },
      { title: "Qayta chop", href: "/reprint", icon: RefreshCw },
    ],
  },
  {
    label: "Hisobot",
    items: [
      { title: "Yordamchi Liniyalar Balansi", href: "/production/balance", icon: Scale },
      { title: "Mahsulot Liniyalari Balansi", href: "/production/product-balance", icon: Truck },
      { title: "Kirim", href: "/ombor/kirim/report", icon: Warehouse },
      { title: "Reja va fakt", href: "/production/plan/report", icon: FileSpreadsheet },
      { title: "GP Mahsulot", href: "/report/gp-products", icon: ScanLine },
      { title: "Ishlab chiqarish", href: "/report", icon: FileSpreadsheet },
      { title: "GS Code", href: "/gscode/report", icon: QrCode },
    ],
  },
  {
    label: "Admin",
    adminOnly: true,
    items: [
      { title: "Foydalanuvchilar", href: "/users", icon: Users },
    ],
  },
]

export const homeQuickLinks: NavItem[] = [
  { title: "Yi'g'ish liniyasi", href: "/yigish", icon: Factory, description: "Serial chop etish" },
  { title: "Modellar", href: "/models", icon: Package, description: "Katalog" },
  { title: "Components", href: "/production/components", icon: Layers, description: "Ishlab chiqarish komponentlari" },
  { title: "GS Code", href: "/gscode", icon: QrCode, description: "Yuklash" },
  { title: "Hisobot", href: "/report", icon: FileSpreadsheet, description: "Ishlab chiqarish" },
  { title: "Dashboard", href: "/dashboard", icon: Monitor, description: "Real vaqt" },
  { title: "Reja", href: "/production/plan", icon: Target, description: "Kunlik ishlab chiqarish rejasi" },
]
