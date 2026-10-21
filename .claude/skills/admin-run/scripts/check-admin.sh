#!/usr/bin/env bash
# 一键体检：后端在不在、admin 在不在、API 链路打没打穿。
# 判据都取自 SKILL.md，别各自发明——尤其 IPv6 那条。
set -u

API_BASE="http://localhost:3000"
BACKEND_PORT=8080
ADMIN_PORT=3000

pass=0
fail=0

ok()   { echo "  ✅ $1"; pass=$((pass+1)); }
bad()  { echo "  ❌ $1"; fail=$((fail+1)); }

echo "== 1. Go 后端 :$BACKEND_PORT =="
if netstat -ano 2>/dev/null | grep -E ":${BACKEND_PORT}\s" | grep -q LISTENING; then
  ok "端口 $BACKEND_PORT 有 LISTENING"
else
  bad "端口 $BACKEND_PORT 无监听 —— cd server && make run"
fi

echo
echo "== 2. Nuxt admin :$ADMIN_PORT =="
# Nuxt 只绑 [::1]:3000（IPv6 loopback），grep 0.0.0.0 会误判成没起
if netstat -ano 2>/dev/null | grep -E "\[::1\]:${ADMIN_PORT}\s|:${ADMIN_PORT}\s" | grep -q LISTENING; then
  ok "端口 $ADMIN_PORT 有 LISTENING（注意：是 [::1] 不是 0.0.0.0）"
else
  bad "端口 $ADMIN_PORT 无监听 —— cd admin && npm run dev"
fi

echo
echo "== 3. admin/.env =="
ENVFILE="$(dirname "$0")/../../../../admin/.env"
if [ -f "$ENVFILE" ]; then
  ok "admin/.env 存在"
else
  bad "admin/.env 缺失（只有 .env.example）。补一行 NUXT_PUBLIC_API_BASE=http://localhost:8080/api/v1"
fi

echo
echo "== 4. API 链路（经 Vite 代理 → Gin）=="
echo "   curl POST $API_BASE/api/v1/auth/login/password"
body_file="$(mktemp)"
code=$(curl -sS -o "$body_file" -w "%{http_code}" --max-time 8 \
  -X POST "$API_BASE/api/v1/auth/login/password" \
  -H "Content-Type: application/json" -d '{}' 2>/dev/null)
body=$(cat "$body_file" 2>/dev/null)
rm -f "$body_file"

echo "   HTTP $code"
echo "   $body"

if [ "$code" = "422" ] && echo "$body" | grep -q '"request_id"'; then
  ok "链路打穿：422 + request_id，请求确实到了 Gin"
elif echo "$body" | grep -q '<html\|<!DOCTYPE'; then
  bad "返回的是 HTML —— 代理没转发到后端，或 8080 挂了"
elif [ "$code" = "000" ] || [ -z "$code" ]; then
  bad "连不上 $API_BASE —— Nuxt 没起，或用了 127.0.0.1（只绑 IPv6，要用 localhost）"
else
  bad "链路存疑：HTTP $code，响应里没有 request_id。先查 8080 后端是否健在"
fi

echo
echo "== 5. 浏览器驱动方式 =="
echo "   只用 Playwright MCP（playwright_navigate / playwright_screenshot /"
echo "   playwright_get_visible_text / playwright_console_logs / playwright_evaluate）"
echo "   截图后必须用 Read 打开 PNG，空白帧 = 启动失败。"

echo
echo "──────────────────────────────"
echo "  通过 $pass 项，失败 $fail 项"
[ "$fail" -eq 0 ] && echo "  可以交付给用户了" || echo "  先把 ❌ 处理掉"
echo "──────────────────────────────"
exit "$fail"
