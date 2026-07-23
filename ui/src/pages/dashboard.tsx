import { Backend_Request } from "@/services/backend"
import { type CSSProperties, useEffect, useState } from "react"
import { ShowErrorToast } from "@/components/showToast"

type DashboardData = {
  t3: { model_name: string; count: number }[]
  plan: number
  done: number
  progress: number
  cycle_time: string
  is_behind_plan: boolean
  plan_status_text: string
  work_time_text: string
  current_shift_no?: number
  current_shift_label?: string
  plan_date?: string
}

const DASHBOARD_BG = "#06111f"

function applyDashboardFullscreen() {
  const targets = [
    document.documentElement,
    document.body,
    document.getElementById("root"),
  ].filter((el): el is HTMLElement => el instanceof HTMLElement)

  for (const el of targets) {
    el.style.setProperty("margin", "0", "important")
    el.style.setProperty("padding", "0", "important")
    el.style.setProperty("overflow", "auto", "important")
    el.style.setProperty("background-color", DASHBOARD_BG, "important")
    el.style.setProperty("background", DASHBOARD_BG, "important")
    el.style.setProperty("width", "100%", "important")
    el.style.setProperty("min-height", "100vh", "important")
    el.style.removeProperty("height")
  }
}

function clearDashboardFullscreen() {
  const targets = [
    document.documentElement,
    document.body,
    document.getElementById("root"),
  ].filter((el): el is HTMLElement => el instanceof HTMLElement)

  const props = [
    "margin",
    "padding",
    "overflow",
    "background-color",
    "background",
    "width",
    "height",
    "min-height",
  ]

  for (const el of targets) {
    for (const prop of props) {
      el.style.removeProperty(prop)
    }
  }
}

export default function Dashboard() {
  const [dashboard, setDashboard] = useState<DashboardData>({
    t3: [],
    plan: 0,
    done: 0,
    progress: 0,
    cycle_time: "00:00",
    is_behind_plan: false,
    plan_status_text: "олдинда 0",
    work_time_text: "08:00 - 19:00",
  })

  async function getAllData() {
    const result = await Backend_Request<DashboardData>({}, "/api/lines/dashboard")
    if (result.result === "ok" && result.data) {
      setDashboard(result.data)
    } else {
      ShowErrorToast(result.error || "Xatolik")
    }
  }

  useEffect(() => {
    const html = document.documentElement
    html.classList.add("dashboard-fullscreen")
    applyDashboardFullscreen()

    const handleResize = () => applyDashboardFullscreen()
    window.addEventListener("resize", handleResize)

    return () => {
      html.classList.remove("dashboard-fullscreen")
      clearDashboardFullscreen()
      window.removeEventListener("resize", handleResize)
    }
  }, [])

  useEffect(() => {
    getAllData()
    const interval = setInterval(() => {
      getAllData()
    }, 3000)
    return () => clearInterval(interval)
  }, [])

  return (
    <div style={styles.shell}>
    <div style={styles.page}>
      <header style={styles.header}>
        <h1 style={styles.title}>
          Кондиционер ишлаб чиқариш линияси
        </h1>
        {dashboard.current_shift_label ? (
          <p style={styles.shiftBadge}>
            {dashboard.current_shift_label}
            {dashboard.work_time_text ? ` · ${dashboard.work_time_text}` : ""}
          </p>
        ) : null}
      </header>

      <div style={styles.stats}>
        <div style={styles.statCard}>
          <div style={styles.statLabel}>Режа</div>
          <div style={styles.statValue}>{dashboard.plan}</div>
        </div>
        <div
          style={{
            ...styles.statCard,
            ...(dashboard.is_behind_plan ? styles.statCardDanger : styles.statCardSuccess),
          }}
        >
          <div style={styles.statLabel}>Бажарилди</div>
          <div style={styles.statValue}>{dashboard.done}</div>
          {dashboard.plan > 0 ? (
            <div style={dashboard.is_behind_plan ? styles.statHintDanger : styles.statHint}>
              {dashboard.progress}% Бажарилди · {dashboard.plan_status_text}
            </div>
          ) : null}
        </div>
      </div>

      {dashboard.plan > 0 && (
        <div style={styles.progressShell}>
          <div style={styles.planInfo}>
            <span>Cycle time: {dashboard.cycle_time}</span>
            <span>
              {dashboard.current_shift_label
                ? `${dashboard.current_shift_label} · ${dashboard.work_time_text}`
                : dashboard.work_time_text}
            </span>
          </div>
          <div style={styles.progressTrack}>
            <div
              style={{
                ...styles.progressFill,
                ...(dashboard.is_behind_plan ? styles.progressFillDanger : {}),
                width: `${dashboard.progress}%`,
              }}
            />
          </div>
        </div>
      )}

      <section style={styles.panel}>
        <div style={styles.panelHeader}>
          <h2 style={styles.panelTitle}>
              Ишлаб чиқарилган моделлар
          </h2>
          <span style={styles.modelCount}>
              {dashboard.t3.length} модель
          </span>
        </div>
        <div style={styles.tableWrap}>
          <table style={styles.table}>
            <thead>
              <tr style={styles.tableHeadRow}>
                <th style={styles.tableHead}>Модель номи</th>
                <th style={styles.tableHead}>Сони</th>
              </tr>
            </thead>
            <tbody>
              {dashboard.t3.map((row, index) => (
                <tr
                  key={`${row.model_name}-${index}`}
                  style={index % 2 === 0 ? styles.tableRowEven : styles.tableRowOdd}
                >
                  <td style={styles.tableCell}>{row.model_name}</td>
                  <td style={{ ...styles.tableCell, ...styles.countCell }}>{row.count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
    </div>
  )
}

const styles: Record<string, CSSProperties> = {
  shell: {
    width: "100%",
    minHeight: "100vh",
    margin: 0,
    padding: 0,
    backgroundColor: DASHBOARD_BG,
  },
  page: {
    width: "100%",
    minHeight: "100vh",
    margin: 0,
    boxSizing: "border-box",
    display: "flex",
    flexDirection: "column",
    padding: "22px 22px 32px",
    fontFamily: "Arial, Helvetica, sans-serif",
    color: "#eaf2ff",
    backgroundColor: DASHBOARD_BG,
    background:
      "radial-gradient(circle at 50% 0%, rgba(14, 165, 233, 0.28), transparent 38%), linear-gradient(135deg, #06111f 0%, #0f172a 48%, #111827 100%)",
  },
  header: {
    textAlign: "center",
    flexShrink: 0,
    marginBottom: 16,
  },
  badge: {
    display: "inline-block",
    border: "1px solid rgba(56, 189, 248, 0.45)",
    borderRadius: 999,
    padding: "8px 18px",
    color: "#7dd3fc",
    background: "rgba(14, 165, 233, 0.14)",
    fontSize: 18,
    fontWeight: 700,
  },
  title: {
    margin: "12px 0 0",
    fontSize: "clamp(34px, 4vw, 64px)",
    lineHeight: 1.08,
    fontWeight: 900,
    letterSpacing: "-0.04em",
  },
  shiftBadge: {
    margin: "10px 0 0",
    color: "#7dd3fc",
    fontSize: "clamp(18px, 2vw, 28px)",
    fontWeight: 800,
  },
  stats: {
    width: "100%",
    maxWidth: 1180,
    margin: "0 auto",
    display: "grid",
    gridTemplateColumns: "1fr 1fr",
    gap: 16,
    flexShrink: 0,
  },
  statCard: {
    minHeight: 132,
    borderRadius: 28,
    border: "1px solid rgba(148, 163, 184, 0.22)",
    padding: "20px 24px",
    background: "rgba(15, 23, 42, 0.78)",
    boxShadow: "0 22px 70px rgba(0, 0, 0, 0.25)",
  },
  statCardAccent: {
    borderColor: "rgba(34, 211, 238, 0.38)",
    background: "linear-gradient(135deg, rgba(8, 145, 178, 0.35), rgba(15, 23, 42, 0.82))",
  },
  statCardSuccess: {
    borderColor: "rgba(74, 222, 128, 0.5)",
    background: "linear-gradient(135deg, rgba(22, 163, 74, 0.48), rgba(15, 23, 42, 0.82))",
  },
  statCardDanger: {
    borderColor: "rgba(248, 113, 113, 0.55)",
    background: "linear-gradient(135deg, rgba(185, 28, 28, 0.55), rgba(15, 23, 42, 0.82))",
  },
  statLabel: {
    color: "#93c5fd",
    fontSize: 18,
    fontWeight: 800,
    textTransform: "uppercase",
    letterSpacing: "0.08em",
  },
  statValue: {
    marginTop: 6,
    fontSize: "clamp(72px, 10vw, 160px)",
    lineHeight: 1,
    fontWeight: 900,
  },
  statHint: {
    marginTop: 8,
    color: "#bae6fd",
    fontSize: 28,
    fontWeight: 700,
  },
  statHintDanger: {
    marginTop: 8,
    color: "#fecaca",
    fontSize: 28,
    fontWeight: 800,
  },
  progressShell: {
    width: "100%",
    maxWidth: 1180,
    margin: "16px auto 0",
    flexShrink: 0,
  },
  planInfo: {
    display: "flex",
    justifyContent: "space-between",
    gap: 16,
    marginBottom: 8,
    color: "#cbd5e1",
    fontSize: 18,
    fontWeight: 800,
  },
  progressTrack: {
    height: 14,
    overflow: "hidden",
    borderRadius: 999,
    background: "rgba(148, 163, 184, 0.22)",
  },
  progressFill: {
    height: "100%",
    borderRadius: 999,
    background: "linear-gradient(90deg, #06b6d4, #22c55e)",
    transition: "width 700ms ease",
  },
  progressFillDanger: {
    background: "linear-gradient(90deg, #ef4444, #f97316)",
  },
  panel: {
    width: "100%",
    maxWidth: 1180,
    display: "flex",
    flexDirection: "column",
    margin: "16px auto 0",
    borderRadius: 28,
    border: "1px solid rgba(148, 163, 184, 0.24)",
    background: "rgba(15, 23, 42, 0.82)",
    boxShadow: "0 22px 70px rgba(0, 0, 0, 0.28)",
  },
  panelHeader: {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 16,
    padding: "16px 22px",
    borderBottom: "1px solid rgba(148, 163, 184, 0.2)",
    flexShrink: 0,
  },
  panelTitle: {
    margin: 0,
    fontSize: 45,
    fontWeight: 900,
  },
  modelCount: {
    borderRadius: 999,
    padding: "8px 16px",
    color: "#7dd3fc",
    background: "rgba(14, 165, 233, 0.14)",
    fontSize: 38,
    fontWeight: 800,
  },
  tableWrap: {
    overflowX: "auto",
  },
  table: {
    width: "100%",
    borderCollapse: "collapse",
  },
  tableHeadRow: {
    background: "linear-gradient(90deg, #0891b2, #2563eb)",
  },
  tableHead: {
    position: "sticky",
    top: 0,
    zIndex: 1,
    padding: "14px 18px",
    color: "#ffffff",
    background: "linear-gradient(90deg, #0891b2, #2563eb)",
    textAlign: "center",
    fontSize: 40,
    fontWeight: 900,
  },
  tableRowEven: {
    background: "rgba(148, 163, 184, 0.10)",
  },
  tableRowOdd: {
    background: "rgba(15, 23, 42, 0.35)",
  },
  tableCell: {
    padding: "14px 18px",
    borderBottom: "1px solid rgba(148, 163, 184, 0.16)",
    textAlign: "center",
    fontSize: "clamp(24px, 2.6vw, 42px)",
    lineHeight: 1.18,
    fontWeight: 800,
  },
  countCell: {
    color: "#7dd3fc",
    fontWeight: 900,
    width: "28%",
  },
}
