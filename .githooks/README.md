# Git hooks

`pre-commit` 在提交前扫暂存区的新增行，命中疑似凭据（明文口令、`sk-` 开头的 key、
带口令的 MySQL 连接串、AK/SK）或本地配置文件（`.env` / `app.local.ini`）时阻断提交。

## 启用

Git 不会自动使用仓库里的钩子目录，**每个克隆各做一次**：

```bash
git config core.hooksPath .githooks
```

确认生效：

```bash
git config --get core.hooksPath   # 应输出 .githooks
```

## 跳过

确认是误报时 `git commit --no-verify`。

## 为什么不是只靠 .gitignore

`.gitignore` 按**路径**挡，`git add -f` 或目录改名都能绕过去。这个仓库的历史里
有明文 RDS 连接串和 DeepSeek key（见 CLAUDE.md「Known Issues」1），而真正有效的
补救是**轮换凭据**——hook 只是让「再欠一笔」更难发生。
