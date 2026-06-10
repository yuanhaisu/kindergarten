@echo off
chcp 65001 >nul
title Redis自动启动和配置脚本

echo 正在启动Redis服务器...
start "Redis Server" cmd /k "cd /d E:\software\Redis && redis-server.exe redis.windows.conf"

echo 等待3秒让Redis服务器启动...
timeout /t 3 /nobreak >nul

echo 正在启动Redis客户端并设置密码...
start "Redis Client" cmd /k "cd /d E:\software\Redis && redis-cli -h 127.0.0.1 -p 6379 CONFIG SET requirepass 123456qwe!@"

echo 操作完成！
echo Redis服务器已在第一个窗口启动
echo Redis客户端已在第二个窗口打开，密码已设置为: 123456qwe!@
pause