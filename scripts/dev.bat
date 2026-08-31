@echo off
rem Local dev launcher (Windows) - Linux server monitor platform
rem Usage: run dev.bat, then open http://localhost:5173

echo [dev] Building and starting Go backend (:8080)...
start "monitor-backend" cmd /k "cd /d %~dp0..\backend && go run ./cmd/server"

echo [dev] Starting frontend Vite dev server (:5173, proxying /api /ws to :8080)...
cd /d %~dp0..\frontend
if not exist node_modules (
  echo [dev] First run, installing frontend dependencies...
  call npm install
)
call npm run dev
