# Cursor SDK agent worker

UI dan **Server** origin bilan vazifa yaratiladi → `queued` → shu worker olib Cursor SDK da bajaradi → `ready_for_test`.

## Setup (server)

```powershell
cd D:\ref-main\ref-ai\ref-ai-test\app\agent
npm install
Copy-Item .env.example .env
notepad .env
```

`.env`:
- `CURSOR_API_KEY` — Cursor dashboard API key
- `CONN_STRING` — test DB (`ref-test`), API bilan bir xil
- `AGENT_CWD` — odatda `D:\ref-main\ref-ai\ref-ai-test\app` (bo‘sh qoldirilsa parent)

```powershell
pm2 start ecosystem.config.cjs
pm2 save
pm2 logs ref-ai-agent-worker
```

## One-shot test

```powershell
npm run once
```

## Flow

1. `/agent` → origin **Server** → vazifa yuborish
2. Worker log: `running task #N`
3. Status `ready_for_test`
4. UI dan **Testga** / **Prodga**
