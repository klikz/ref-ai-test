import ExcelJS from "exceljs"

export type ConsumptionNormExportItem = {
  group_level: number
  factory_code?: string
  odoo_code?: string
  standard_name_uz?: string
  quantity: number
  consume_line_name?: string
  receive_line_name?: string
}

export type ConsumptionNormExportLine = {
  line_id: number
  name: string
}

export type ConsumptionNormExportModel = {
  id: number
  model_nomi?: string
  modeli?: string
  qisqa_nomi?: string
}

const COLORS = {
  headerBg: "FF0891B2",
  headerText: "FFFFFFFF",
  metaLabelBg: "FFE0F2FE",
  metaValueBg: "FFF0F9FF",
  idCellBg: "FFF1F5F9",
  metaText: "FF0F172A",
  border: "FFCBD5E1",
  linesHeaderBg: "FF475569",
  linesCellBg: "FFF1F5F9",
  level: {
    0: "FFF8FAFC",
    1: "FFFCE7F3",
    2: "FFDBEAFE",
    3: "FFBFDBFE",
    4: "FFFFEDD5",
    5: "FFFEF3C7",
  } as Record<number, string>,
}

const BOM_SHEET = "BOM list"
const LINES_SHEET = "Liniyalar"
const DATA_START_ROW = 4
const DROPDOWN_EXTRA_ROWS = 200

const HEADERS = [
  "",
  "Factory product code",
  "Miqdori\nQuantity",
  "O`lchov birligi\nUnit of measurement",
  "ishlatilish joyi",
  "i/ch joyi",
]

function applyBorder(cell: ExcelJS.Cell) {
  cell.border = {
    top: { style: "thin", color: { argb: COLORS.border } },
    left: { style: "thin", color: { argb: COLORS.border } },
    bottom: { style: "thin", color: { argb: COLORS.border } },
    right: { style: "thin", color: { argb: COLORS.border } },
  }
}

function levelColor(level: number) {
  return COLORS.level[Math.min(Math.max(level, 0), 5)] ?? COLORS.level[0]
}

function styleMetaCell(cell: ExcelJS.Cell, isLabel: boolean, readOnly = false) {
  applyBorder(cell)
  cell.font = { bold: isLabel || readOnly, color: { argb: COLORS.metaText }, size: 11 }
  cell.alignment = { vertical: "middle", horizontal: isLabel ? "right" : "left" }
  cell.fill = {
    type: "pattern",
    pattern: "solid",
    fgColor: { argb: isLabel ? COLORS.metaLabelBg : readOnly ? COLORS.idCellBg : COLORS.metaValueBg },
  }
}

function styleHeaderCell(cell: ExcelJS.Cell, isLines = false) {
  applyBorder(cell)
  cell.font = { bold: true, color: { argb: COLORS.headerText }, size: 10 }
  cell.alignment = { vertical: "middle", horizontal: "center", wrapText: true }
  cell.fill = {
    type: "pattern",
    pattern: "solid",
    fgColor: { argb: isLines ? COLORS.linesHeaderBg : COLORS.headerBg },
  }
}

function styleDataCell(cell: ExcelJS.Cell, level: number, col: number) {
  applyBorder(cell)
  const isNumber = col === 3
  const isLevel = col === 1
  cell.font = {
    bold: level <= 1 && col <= 2,
    color: { argb: "FF1E293B" },
    size: 10,
  }
  cell.alignment = {
    vertical: "middle",
    horizontal: isNumber || isLevel ? "center" : "left",
    wrapText: col === 2,
    indent: col === 2 ? Math.min(level, 4) : 0,
  }
  cell.fill = {
    type: "pattern",
    pattern: "solid",
    fgColor: { argb: levelColor(level) },
  }
  if (isNumber) {
    cell.numFmt = "0.##########"
  }
}

function writeLinesSheet(workbook: ExcelJS.Workbook, lines: ConsumptionNormExportLine[]) {
  const sheet = workbook.addWorksheet(LINES_SHEET, {
    views: [{ state: "frozen", ySplit: 1 }],
  })

  const idHeader = sheet.getCell("A1")
  idHeader.value = "line_id"
  styleHeaderCell(idHeader, true)

  const nameHeader = sheet.getCell("B1")
  nameHeader.value = "liniya nomi"
  styleHeaderCell(nameHeader, true)

  lines.forEach((line, index) => {
    const row = index + 2
    const idCell = sheet.getCell(row, 1)
    idCell.value = line.line_id
    applyBorder(idCell)
    idCell.font = { size: 10, color: { argb: "FF64748B" } }
    idCell.alignment = { vertical: "middle", horizontal: "center" }
    idCell.fill = {
      type: "pattern",
      pattern: "solid",
      fgColor: { argb: COLORS.linesCellBg },
    }

    const nameCell = sheet.getCell(row, 2)
    nameCell.value = line.name
    applyBorder(nameCell)
    nameCell.font = { size: 10, color: { argb: "FF334155" } }
    nameCell.alignment = { vertical: "middle", horizontal: "left" }
    nameCell.fill = {
      type: "pattern",
      pattern: "solid",
      fgColor: { argb: COLORS.linesCellBg },
    }
  })

  sheet.getColumn(1).width = 10
  sheet.getColumn(2).width = 28
  sheet.getRow(1).height = 22

  return sheet
}

function applyLineDropdowns(
  sheet: ExcelJS.Worksheet,
  lineCount: number,
  toRow: number,
) {
  if (lineCount < 1 || toRow < DATA_START_ROW) return

  const listRange = `'${LINES_SHEET}'!$B$2:$B$${lineCount + 1}`
  const validation: ExcelJS.DataValidation = {
    type: "list",
    allowBlank: true,
    formulae: [listRange],
    showErrorMessage: true,
    errorStyle: "error",
    errorTitle: "Noto'g'ri qiymat",
    error: "Ro'yxatdan liniya tanlang",
    showInputMessage: true,
    promptTitle: "Liniya",
    prompt: "Liniyalar sheetidan tanlang",
  }

  // exceljs runtime supports range add; types omit it
  const validations = (sheet as ExcelJS.Worksheet & {
    dataValidations: { add: (address: string, validation: ExcelJS.DataValidation) => void }
  }).dataValidations

  validations.add(`E${DATA_START_ROW}:E${toRow}`, validation)
  validations.add(`F${DATA_START_ROW}:F${toRow}`, { ...validation })
}

export async function buildConsumptionNormWorkbook(
  model: ConsumptionNormExportModel,
  items: ConsumptionNormExportItem[],
  lines: ConsumptionNormExportLine[],
) {
  const workbook = new ExcelJS.Workbook()
  const sheet = workbook.addWorksheet(BOM_SHEET, {
    views: [{ state: "frozen", ySplit: 2, xSplit: 0 }],
  })
  sheet.properties.outlineProperties = { summaryBelow: false, summaryRight: false }

  writeLinesSheet(workbook, lines)

  styleMetaCell(sheet.getCell("A1"), true)
  sheet.getCell("A1").value = "Modeli"
  styleMetaCell(sheet.getCell("B1"), false)
  sheet.getCell("B1").value = model.modeli || model.qisqa_nomi || model.model_nomi || ""
  styleMetaCell(sheet.getCell("C1"), true)
  sheet.getCell("C1").value = "ID"
  styleMetaCell(sheet.getCell("D1"), false, true)
  sheet.getCell("D1").value = model.id

  HEADERS.forEach((header, index) => {
    const cell = sheet.getCell(2, index + 1)
    cell.value = header
    styleHeaderCell(cell)
  })

  items.forEach((item, index) => {
    const rowNumber = DATA_START_ROW + index
    const row = sheet.getRow(rowNumber)
    row.height = 20
    row.outlineLevel = Math.max(0, Math.min(item.group_level, 7))

    const values = [
      item.group_level,
      item.factory_code || item.odoo_code || "",
      item.quantity,
      "",
      item.consume_line_name || "",
      item.receive_line_name || "",
    ]

    values.forEach((value, colIndex) => {
      const cell = row.getCell(colIndex + 1)
      cell.value = value
      styleDataCell(cell, item.group_level, colIndex + 1)
    })
  })

  const widths = [8, 28, 14, 18, 18, 18]
  widths.forEach((width, index) => {
    sheet.getColumn(index + 1).width = width
  })

  sheet.getRow(1).height = 22
  sheet.getRow(2).height = 36

  const dropdownToRow = Math.max(DATA_START_ROW + items.length - 1, DATA_START_ROW) + DROPDOWN_EXTRA_ROWS
  applyLineDropdowns(sheet, lines.length, dropdownToRow)

  return workbook
}
