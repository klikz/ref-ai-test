/**
 * Cursor SDK worker: polls agent.tasks (queued/server) and runs local Agent.prompt.
 *
 * Env:
 *   CURSOR_API_KEY (required)
 *   CONN_STRING (postgres key=value or URL)
 *   AGENT_CWD (repo root; default parent of /agent)
 *   CURSOR_AGENT_MODEL (default composer-2.5)
 *   AGENT_POLL_MS (default 4000)
 *
 * Usage:
 *   npm start
 *   npm run once
 */

import fs from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"
import { spawnSync } from "node:child_process"
import dotenv from "dotenv"
import pg from "pg"
import { Agent, CursorAgentError } from "@cursor/sdk"

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const agentRoot = path.resolve(__dirname, "..")
const defaultCwd = path.resolve(agentRoot, "..")

loadEnvFiles([
  path.join(agentRoot, ".env"),
  path.join(defaultCwd, ".env"),
  path.join(defaultCwd, "..", ".env"),
])

const once = process.argv.includes("--once")
const pollMs = Number(process.env.AGENT_POLL_MS || 4000)
const apiKey = String(process.env.CURSOR_API_KEY || "").trim()
const modelId = String(process.env.CURSOR_AGENT_MODEL || "composer-2.5").trim()
const agentCwd = path.resolve(String(process.env.AGENT_CWD || defaultCwd).trim())
const connString = String(process.env.CONN_STRING || "").trim()

function loadEnvFiles(files) {
  for (const file of files) {
    if (fs.existsSync(file)) {
      dotenv.config({ path: file, override: false })
    }
  }
}

function parseConn(raw) {
  if (!raw) throw new Error("CONN_STRING is required")
  if (raw.startsWith("postgres://") || raw.startsWith("postgresql://")) {
    return { connectionString: raw }
  }
  const cfg = {}
  for (const part of raw.split(/\s+/)) {
    const i = part.indexOf("=")
    if (i <= 0) continue
    const key = part.slice(0, i).toLowerCase()
    const value = part.slice(i + 1)
    if (key === "host") cfg.host = value
    else if (key === "port") cfg.port = Number(value)
    else if (key === "user") cfg.user = value
    else if (key === "password") cfg.password = value
    else if (key === "dbname" || key === "database") cfg.database = value
    else if (key === "sslmode" && value !== "disable") cfg.ssl = { rejectUnauthorized: false }
  }
  if (!cfg.host && !cfg.connectionString) {
    throw new Error("CONN_STRING parse failed")
  }
  return cfg
}

function buildPrompt(task) {
  return [
    "You are an autonomous coding agent working in a Go + React ERP repository.",
    `Task id: ${task.id}`,
    `Origin: ${task.origin}`,
    "",
    "Owner request:",
    task.prompt,
    "",
    "Rules:",
    "- Implement the request with minimal, focused changes.",
    "- Prefer existing project patterns.",
    "- Do not commit secrets (.env).",
    "- If you change code, leave the working tree ready for review.",
    "- After finishing, summarize what changed.",
  ].join("\n")
}

async function claimNext(client) {
  await client.query("BEGIN")
  try {
    const selected = await client.query(
      `SELECT id, prompt, origin, status, created_by, git_sha, log_text, error_text, created_at, updated_at
       FROM agent.tasks
       WHERE status = 'queued' AND origin = 'server'
       ORDER BY id ASC
       FOR UPDATE SKIP LOCKED
       LIMIT 1`,
    )
    if (selected.rowCount === 0) {
      await client.query("ROLLBACK")
      return null
    }
    const id = selected.rows[0].id
    const updated = await client.query(
      `UPDATE agent.tasks
       SET status = 'running',
           log_text = 'Cursor SDK worker ishlamoqda...',
           updated_at = NOW()
       WHERE id = $1
       RETURNING id, prompt, origin, status, created_by, git_sha, log_text, error_text, created_at, updated_at`,
      [id],
    )
    await client.query("COMMIT")
    return updated.rows[0]
  } catch (err) {
    await client.query("ROLLBACK")
    throw err
  }
}

async function updateTask(client, id, status, logText = "", errorText = "", gitSha = "") {
  const res = await client.query(
    `UPDATE agent.tasks
     SET status = $2,
         log_text = CASE WHEN $3 = '' THEN log_text ELSE $3 END,
         error_text = CASE WHEN $4 = '' THEN error_text ELSE $4 END,
         git_sha = CASE WHEN $5 = '' THEN git_sha ELSE $5 END,
         updated_at = NOW()
     WHERE id = $1
     RETURNING id, status`,
    [id, status, logText, errorText, gitSha],
  )
  return res.rows[0]
}

function gitShortSha(cwd) {
  const r = spawnSync("git", ["rev-parse", "--short", "HEAD"], { cwd, encoding: "utf8" })
  if (r.status === 0) return String(r.stdout || "").trim()
  return ""
}

async function runCursor(task) {
  const prompt = buildPrompt(task)
  console.log(`[worker] running task #${task.id} in ${agentCwd}`)
  const result = await Agent.prompt(prompt, {
    apiKey,
    model: { id: modelId },
    local: { cwd: agentCwd },
  })
  return result
}

async function processOne(client) {
  const task = await claimNext(client)
  if (!task) return false

  try {
    const result = await runCursor(task)
    const sha = gitShortSha(agentCwd)
    const summary =
      typeof result?.result === "string"
        ? result.result.slice(0, 4000)
        : JSON.stringify(result?.result ?? result?.status ?? "done").slice(0, 4000)

    if (result?.status === "error") {
      await updateTask(
        client,
        task.id,
        "failed",
        summary || "Cursor run status=error",
        "Cursor agent run failed",
        sha,
      )
      console.error(`[worker] task #${task.id} failed`)
      return true
    }

    await updateTask(
      client,
      task.id,
      "ready_for_test",
      `Cursor tugadi.\n${summary}\nKeyin: git push (kerak bo'lsa) va UI dan Testga tasdiqlang.`,
      "",
      sha,
    )
    console.log(`[worker] task #${task.id} -> ready_for_test`)
  } catch (err) {
    const message = err instanceof CursorAgentError ? err.message : String(err?.message || err)
    await updateTask(client, task.id, "failed", "", message.slice(0, 2000), "")
    console.error(`[worker] task #${task.id} error:`, message)
  }
  return true
}

async function main() {
  if (!apiKey) {
    console.error("CURSOR_API_KEY required")
    process.exit(1)
  }
  const client = new pg.Client(parseConn(connString))
  await client.connect()
  console.log(`[worker] connected; cwd=${agentCwd}; model=${modelId}; once=${once}`)

  try {
    if (once) {
      const did = await processOne(client)
      if (!did) console.log("[worker] no queued tasks")
      return
    }
    for (;;) {
      try {
        const did = await processOne(client)
        if (!did) await new Promise((r) => setTimeout(r, pollMs))
      } catch (err) {
        console.error("[worker] loop error:", err)
        await new Promise((r) => setTimeout(r, pollMs))
      }
    }
  } finally {
    await client.end()
  }
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
