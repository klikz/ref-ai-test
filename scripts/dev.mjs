import { execSync, spawn } from "child_process"
import path from "path"
import readline from "readline"
import { fileURLToPath } from "url"

const root = path.join(path.dirname(fileURLToPath(import.meta.url)), "..")
const uiDir = path.join(root, "ui")
const isWin = process.platform === "win32"
const API_PORT = process.env.API_PORT || "8001"

let apiProcess = null
let uiProcess = null
let apiRestarting = false
let shuttingDown = false

function log(tag, message) {
  process.stdout.write(`[${tag}] ${message}\n`)
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function pipeOutput(child, tag) {
  const write = (chunk) => {
    for (const line of chunk.toString().split(/\r?\n/)) {
      if (line.length > 0) {
        log(tag, line)
      }
    }
  }
  child.stdout?.on("data", write)
  child.stderr?.on("data", write)
}

function killProcessTree(child) {
  if (!child?.pid) {
    return Promise.resolve()
  }

  return new Promise((resolve) => {
    if (isWin) {
      const killer = spawn("taskkill", ["/PID", String(child.pid), "/T", "/F"], {
        stdio: "ignore",
        windowsHide: true,
      })
      killer.on("exit", () => resolve())
      killer.on("error", () => resolve())
      setTimeout(resolve, 3000)
      return
    }

    let settled = false
    const done = () => {
      if (settled) {
        return
      }
      settled = true
      resolve()
    }

    child.once("exit", done)
    child.kill("SIGTERM")
    setTimeout(() => {
      try {
        child.kill("SIGKILL")
      } catch {
        // already dead
      }
      done()
    }, 3000)
  })
}

function killApiPortListeners() {
  if (!isWin) {
    return
  }

  try {
    const output = execSync(`netstat -ano | findstr :${API_PORT}`, {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"],
    })

    const pids = new Set()
    for (const line of output.split(/\r?\n/)) {
      if (!line.includes("LISTENING")) {
        continue
      }
      const parts = line.trim().split(/\s+/)
      const pid = parts[parts.length - 1]
      if (pid && /^\d+$/.test(pid) && pid !== "0") {
        pids.add(pid)
      }
    }

    for (const pid of pids) {
      try {
        execSync(`taskkill /PID ${pid} /T /F`, { stdio: "ignore" })
      } catch {
        // process already gone
      }
    }
  } catch {
    // nothing listening
  }
}

function startApi() {
  apiProcess = spawn("go", ["run", "-v", "./cmd/api_v3"], {
    cwd: root,
    stdio: ["ignore", "pipe", "pipe"],
    env: process.env,
    windowsHide: true,
  })

  pipeOutput(apiProcess, "api")

  apiProcess.on("exit", (code, signal) => {
    if (shuttingDown || apiRestarting) {
      return
    }
    if (signal) {
      log("api", `stopped (${signal})`)
      return
    }
    if (code !== 0 && code !== null) {
      log("api", `exited with code ${code}`)
    }
  })
}

async function stopApi() {
  if (apiProcess) {
    await killProcessTree(apiProcess)
    apiProcess = null
  }
  killApiPortListeners()
  await sleep(400)
}

async function restartApi() {
  if (shuttingDown || apiRestarting) {
    return
  }

  apiRestarting = true
  log("dev", "Restarting backend...")

  try {
    await stopApi()
    if (!shuttingDown) {
      startApi()
      log("dev", "Backend restarted (R — restart again)")
    }
  } finally {
    apiRestarting = false
  }
}

function startUi() {
  // Windows: .cmd files require shell, otherwise spawn throws EINVAL
  if (isWin) {
    uiProcess = spawn("npm run dev", {
      cwd: uiDir,
      shell: true,
      stdio: ["ignore", "pipe", "pipe"],
      env: process.env,
      windowsHide: true,
    })
  } else {
    uiProcess = spawn("npm", ["run", "dev"], {
      cwd: uiDir,
      stdio: ["ignore", "pipe", "pipe"],
      env: process.env,
    })
  }

  pipeOutput(uiProcess, "ui")

  uiProcess.on("exit", (code) => {
    if (!shuttingDown && code !== 0 && code !== null) {
      log("ui", `exited with code ${code}`)
    }
  })
}

async function shutdown() {
  if (shuttingDown) {
    return
  }
  shuttingDown = true
  log("dev", "Stopping...")
  await stopApi()
  if (uiProcess) {
    await killProcessTree(uiProcess)
    uiProcess = null
  }
  process.exit(0)
}

function bindRestartHotkey() {
  if (!process.stdin.isTTY) {
    log("dev", "TTY yo'q — R bilan restart ishlamaydi")
    return
  }

  readline.emitKeypressEvents(process.stdin)
  process.stdin.setRawMode(true)

  process.stdin.on("keypress", (_str, key) => {
    if (!key) {
      return
    }
    if (key.ctrl && key.name === "c") {
      void shutdown()
      return
    }
    if (key.name === "r") {
      void restartApi()
    }
  })
}

log("dev", "api + ui ishga tushmoqda...")
log("dev", "R — backend qayta ishga tushirish, Ctrl+C — to'xtatish")

startApi()
startUi()
bindRestartHotkey()

process.on("SIGINT", () => void shutdown())
process.on("SIGTERM", () => void shutdown())
