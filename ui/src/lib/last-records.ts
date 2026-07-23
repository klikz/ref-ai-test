export const LAST_RECORDS_LIMIT = 10
export const LAST_RECORDS_TITLE = `Oxirgi ${LAST_RECORDS_LIMIT} ta`

/** How many session rows are visible before scrolling (data loads up to LAST_RECORDS_LIMIT). */
export const LAST_RECORDS_VISIBLE_ROWS = 5

// Sticky header (45px) + LAST_RECORDS_VISIBLE_ROWS × 64px action rows.
export const LAST_RECORDS_TABLE_VIEWPORT_CLASS = "h-[calc(45px+20rem)]"

export const LAST_RECORDS_PANEL_CLASS = "flex flex-col overflow-hidden"

export const LAST_RECORDS_SESSION_GRID =
  "minmax(0, 1.3fr) minmax(0, 0.9fr) minmax(0, 0.9fr) minmax(0, 1.1fr) 3.5rem"
