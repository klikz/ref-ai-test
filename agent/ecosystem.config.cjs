# Agent worker (PM2)
# From repo root (…/app on server):
#   cd agent && npm install
#   copy .env.example .env  (CURSOR_API_KEY + CONN_STRING)
#   pm2 start ecosystem.config.cjs

const path = require("path")

module.exports = {
  apps: [
    {
      name: "ref-ai-agent-worker",
      cwd: __dirname,
      script: path.join(__dirname, "src", "worker.js"),
      interpreter: "node",
      autorestart: true,
      watch: false,
      max_restarts: 20,
      env: {
        NODE_ENV: "production",
      },
    },
  ],
}
