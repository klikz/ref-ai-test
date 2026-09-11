// PM2 config for ref-ai-prod. Copy to D:\ref-main\ref-ai\ref-ai-prod\ecosystem.config.cjs if missing.
// Adjust PORT / env in that folder's .env — process inherits cwd.
module.exports = {
  apps: [
    {
      name: "ref-ai-prod",
      script: "./bin/api_v3.exe",
      cwd: "D:\\ref-main\\ref-ai\\ref-ai-prod",
      instances: 1,
      autorestart: true,
      watch: false,
      max_memory_restart: "1G",
      env: {
        NODE_ENV: "production",
      },
    },
  ],
};
