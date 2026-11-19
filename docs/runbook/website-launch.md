# 官网发布 Runbook（单独发布与公开上线）

官网是纯静态站，不依赖后端。`Website Release` workflow 只构建、检查、打包并部署静态产物，
不构建镜像、不跑迁移、不访问 RDS/Redis/ACR，也不调用快照脚本。

- 配套：[部署 Runbook](deployment.md)（全栈发布）、[部署架构](../deployment-architecture.md)、ADR-0017。
- 状态交接：[官网交付与未完成项](../handoffs/2026-11-07-website-pending.md)。
- 服务器 IP / SSH 入口不入库；本文用 `<staging-public-ip>`、`<prod-host>` 等占位符，真实值见运维私密记录。

## 0. 组成与边界

| 层 | 位置 | 作用 |
|---|---|---|
| 内容门禁 | `website/scripts/check-content.mjs` | 检查 `website/.output/public`；`--production` 追加法务 preflight |
| 发布脚本 | `deploy/scripts/deploy-website.sh` | `deploy-website.sh <prod|staging> <version> <website-tar>` |
| 发布 workflow | `.github/workflows/website-release.yml` | 手动触发；输入 `environment` = `staging` / `production` |
| nginx | `deploy/nginx/kindergarten.conf.tmpl` | 生产 `www` 块、apex 301、预发 `test-www` 块 |

- 发布只切换 `/opt/kindergarten/<env>/website/current` 这个 symlink；脚本**不 reload nginx**，
  静态内容切换即可生效。
- 版本标签由 workflow 生成：`website-<run>-<attempt>-<sha7>`（run number + attempt + 7 位 SHA）。
  版本目录不可变，重复版本脚本直接拒绝（exit 2）。
- 同一目标串行发布（`cancel-in-progress: false`）；重跑/重试会得到新标签，不会覆盖在线版本。
- 失败语义：extraction 或 symlink 切换失败会清理临时目录并恢复旧链接；symlink 切换是最后一次
  写操作，切换前被取消时旧版本继续在线，切换后完成则是有效的新 release。
- production 只允许从 `main` 或 `v*` tag 触发；staging 接受任意 ref（演练用 `main`）。

## 1. 上线前置（缺一不可）

| # | 前置 | 验收方式 |
|---|---|---|
| 1 | ICP 备案 | 已通过（2026-11-13），备案号 `湘ICP备2026054245号-1`；首页与页脚展示并链接工信部 |
| 2 | DNS | apex `nurtureplanet.cn` 与 `www.nurtureplanet.cn` 的 A 记录指向生产 ECS；解析线路为「默认」或加「境外」 |
| 3 | 证书 SAN | 同一张证书覆盖 apex 与 `www`（同期上线的 `api` / `admin` 一并加入）；缺一个主机名对应 server block 握手失败 |
| 4 | 生产静态目录 | `/opt/kindergarten/prod/website/releases` 存在，deploy 用户可写、nginx 可读 |
| 5 | nginx 生产块 | 模板已渲染：`www` server 块、apex 301、`root /opt/kindergarten/prod/website/current`；`nginx -t` 通过 |
| 6 | GitHub Environment `production` | Secrets `ECS_HOST` / `ECS_USER` / `ECS_SSH_KEY` 已配置；required reviewer 已开启 |
| 7 | 法务 | 两份法务页去掉起草标记、填写生效日期，`legalDraft` 为 `false`；`--production` 门禁通过（见 §4） |

### 1.1 DNS 与证书

- DNS 与证书是人工/外部操作；本 runbook 只列验收点，签发/续期沿用
  [deployment.md](deployment.md) §1.0 步骤 7 与 §6，把 apex 与 `www` 加入 `-d` 列表。
- `ACME_USE_IPV4_REQUESTS=1` 必须保留（服务器 IPv6 出口不通）。
- 验收：

```bash
dig +short nurtureplanet.cn
dig +short www.nurtureplanet.cn
openssl s_client -connect www.nurtureplanet.cn:443 -servername www.nurtureplanet.cn </dev/null 2>/dev/null \
  | openssl x509 -noout -ext subjectAltName
```

### 1.2 生产静态目录与 nginx

```bash
# 1) 目录：deploy 用户可写，nginx worker 可读
sudo mkdir -p /opt/kindergarten/prod/website/releases
sudo chown -R deploy:deploy /opt/kindergarten/prod/website
sudo chmod -R o+rx /opt/kindergarten/prod/website

# 2) 线上 conf 先备份，再渲染模板；其余生产初始化步骤见 deployment.md §1
sudo cp -a /etc/nginx/conf.d/kindergarten.conf \
  /etc/nginx/conf.d/kindergarten.conf.bak-$(date +%Y%m%d-%H%M)
sudo env DOMAIN=nurtureplanet.cn envsubst '$DOMAIN' < deploy/nginx/kindergarten.conf.tmpl \
  | sudo tee /etc/nginx/conf.d/kindergarten.conf >/dev/null
sudo nginx -t && sudo systemctl reload nginx
```

> 生产 `www` 块不设访问门禁；`test-www` 的 Basic Auth 只存在于预发块（见 deployment.md §1.1）。
> 首次发布前 `current` 还不存在，`/` 会返回 nginx 404，属正常；发布后自然恢复。

## 2. 本地预检

```bash
cd website
npm ci
npm test                 # 门禁自身的 fixture 测试，不需要先构建
npm run generate         # 产出 website/.output/public
npm run check:content    # staging 模式
npm run check:content -- --production   # 生产 preflight；法务未跟进时应失败
```

- 默认产物目录是 `website/.output/public`，相对项目解析，不随调用目录漂移。
- CI 的 `website` job 跑同样的 `npm test` 与 `npm run check:content`；发布前确认 CI 绿。

## 3. staging 实跑演练（合并到 main 之后）

> `workflow_dispatch` 只认默认分支上的 workflow 定义；**合并到 `main` 之前无法手动触发**。
> 因此实跑演练是合入后的动作，不在实现 ticket 的分支上执行。

1. 触发：

```bash
gh workflow run website-release.yml --ref main -f environment=staging
gh run list --workflow=website-release.yml --limit 3
gh run watch <run-id> --exit-status
```

2. 记录版本标签：

```bash
gh run view <run-id> --log | grep -E 'deploy target:|website deploy ok'
# deploy target: staging -> staging (website-<run>-<attempt>-<sha7>)
# ==> website deploy ok: staging website-<run>-<attempt>-<sha7>
```

3. 确认 symlink 指向本次版本（不能只看 Last-Modified）：

```bash
ssh <ssh-user>@<staging-public-ip> 'readlink -f /opt/kindergarten/staging/website/current'
# 期望：/opt/kindergarten/staging/website/releases/website-<run>-<attempt>-<sha7>
```

4. 确认 `test-www` 提供新 HTML（staging 口令读取见 deployment.md §1.1.4，不要贴进聊天或 ticket）：

```bash
curl -sI -u "staging:$PW" https://test-www.nurtureplanet.cn/ | grep -iE '^(HTTP/|last-modified)'
curl -fsS -u "staging:$PW" https://test-www.nurtureplanet.cn/ | grep -F '湘ICP备2026054245号-1'
for p in /privacy /terms /404.html; do
  curl -s -o /dev/null -w "$p %{http_code}\n" "https://test-www.nurtureplanet.cn$p"
done   # 期望全部 200
```

`/` 匿名仍是 401（Basic Auth 门禁），这是预期。

5. 重跑一次，证明标签不碰撞：

```bash
gh workflow run website-release.yml --ref main -f environment=staging
```

期望得到新的 `website-<run>-<attempt>-<sha7>`；symlink 指向第二次的版本；第一次的 release
目录保留不删。

6. 把两次版本标签、`website/current` symlink 目标、`Last-Modified` 与 200 结果记录到 ticket
   #131 作为证据。

## 4. 法务跟进（生产发布前必须完成）

法务复核通过后，以下三处必须**一起**改，缺一不可：

1. `website/pages/privacy.vue` 与 `website/pages/terms.vue` 顶部的硬编码 meta 行：
   - 现状：`适用主体：{{ entity }}　·　生效日期：待定　·　版本：草案`
   - 改为真实生效日期与版本，例如：`适用主体：{{ entity }}　·　生效日期：2026-XX-XX　·　版本：v1.0`
2. `website/content/site.ts` 的 `legalDraft: true` → `false`（页脚草案提示消失）。
3. 本地复核生产门禁：

```bash
cd website
npm run generate
npm run check:content -- --production
# 期望：[check-content] OK ...（生产模式）
```

- **只改 `legalDraft` 不够**：顶部 meta 行是硬编码的；`--production` 检查构建产物里的可见文本，
  `版本：草案` 与 `生效日期：待定` 任一存在都会失败。
- 门禁只是技术绊线，不能替代律师复核；复核完成前不要发布。
- 班内共享文案继续钉住当前行为（「默认开启」「可随时撤回 / 可随时关闭」）；**不要**提前改成
  「没有家长开关」。等 #88 的后端与家长端改动落地后，网站文案与门禁在同一次变更里一起更新。

## 5. 生产发布

1. 发布前记录回滚点：

```bash
ssh <ssh-user>@<prod-host> \
  'readlink -f /opt/kindergarten/prod/website/current; ls -1t /opt/kindergarten/prod/website/releases | head -5'
```

记录当前版本标签；首次发布没有 `current`，记为「无（首次发布）」。

2. 触发（只能是 `main` 或 `v*` tag）：

```bash
gh workflow run website-release.yml --ref main -f environment=production
```

3. 在 GitHub Actions 中批准 `production` 环境：
   - run 会停在 `Waiting for approval`；打开 run 页面 → **Review deployments** → 批准。
   - 若没有暂停，说明 `production` 环境未配置 required reviewer——这是配置缺陷，先补
     Settings → Environments → `production`。

4. 批准后 workflow 依次执行：构建 tokens 与官网 → `npm test` →
   `npm run check:content -- --production` → 打包 → scp → ssh 运行
   `deploy-website.sh prod <version> /tmp/<version>.tar.gz`。
   - 内容门禁未通过时在打包前失败，草稿不会上传。
   - 记录日志中的 `deploy target: production -> prod (...)` 与
     `website deploy ok: prod <version>`。

5. 记录新的 symlink 目标（同 §3 命令，把 `staging` 换成 `prod`），然后执行 §6 验收。

## 6. 上线验收

```bash
BASE=https://www.nurtureplanet.cn

# 首页 200
curl -fsSI "$BASE/" | head -1

# apex 301 -> www
curl -sI https://nurtureplanet.cn/ | head -2

# 政策页与品牌 404
curl -fsS -o /dev/null -w '/privacy %{http_code}\n' "$BASE/privacy"
curl -fsS -o /dev/null -w '/terms %{http_code}\n' "$BASE/terms"
curl -s -o /dev/null -w '/no-such-page %{http_code}\n' "$BASE/no-such-page"
curl -fsS "$BASE/no-such-page" | grep -F '养成星球'

# 带斜杠地址反向 301 到无斜杠（与 canonical / sitemap 一致）
curl -sI "$BASE/privacy/" | head -2

# sitemap / robots
curl -fsS "$BASE/sitemap.xml"
curl -fsS "$BASE/robots.txt"

# canonical / OG / ICP 页脚
curl -fsS "$BASE/" | grep -o 'rel="canonical" href="[^"]*"' | head -1
curl -fsS "$BASE/" | grep -o 'property="og:url" content="[^"]*"' | head -1
curl -fsS "$BASE/" | grep -F '湘ICP备2026054245号-1'
curl -fsS "$BASE/privacy" | grep -F '湘ICP备2026054245号-1'
curl -fsS "$BASE/terms" | grep -F '湘ICP备2026054245号-1'

# 法务草案标记必须消失（期望无输出）
curl -fsS "$BASE/privacy" | grep -F '生效日期：待定'
curl -fsS "$BASE/terms" | grep -F '版本：草案'
```

浏览器补充走查：桌面与移动首屏、导航、联系邮箱、页脚备案链接指向工信部备案系统；家长端
web-view 匿名打开 `/privacy` 与 `/terms` 版面正常。

## 7. 回滚

静态站回滚 = 把 symlink 指回上一个 release，秒级生效，**不需要 reload nginx**。

1. 查当前与历史：

```bash
ssh <ssh-user>@<prod-host> \
  'readlink -f /opt/kindergarten/prod/website/current; ls -1t /opt/kindergarten/prod/website/releases | head -10'
```

2. 切换（用 §5 记录的发布前版本标签，不要按时间猜）：

```bash
ssh <ssh-user>@<prod-host> \
  'ln -sfn /opt/kindergarten/prod/website/releases/<previous-label> /opt/kindergarten/prod/website/current'
```

3. 复核：

```bash
ssh <ssh-user>@<prod-host> 'readlink -f /opt/kindergarten/prod/website/current'
curl -fsSI https://www.nurtureplanet.cn/ | head -1
```

- **保留旧 release 目录**：版本目录不可变，回滚依赖它们存在；不要 `rm -rf`。
- 首次生产发布没有上一个版本；此时回滚等于撤下官网，只在未公开时可用。
- 发布失败但未切 symlink 时，旧版本继续在线，无需操作；脚本在 extraction / switch 失败时
  已自动恢复旧链接。
- 回滚后仍需复查 `/privacy`、`/terms`、品牌 404 与 ICP 页脚（§6）。

## 8. 常见故障

| 现象 | 排查 |
|---|---|
| production 触发即报 ref 错误 | 只能从 `main` 或 `v*` tag 触发 |
| run 停在 approval | `production` 环境 required reviewer；在 run 页面 Review deployments 批准 |
| 门禁报 `版本：草案` / `生效日期：待定` | 法务跟进未完成（§4） |
| `version already exists`（exit 2） | 版本标签重复；workflow 用 run + attempt 保证唯一，人工执行需换标签 |
| 发布后仍是旧页面 | `readlink -f .../website/current` 是否指向新 release；查 run 日志有无 `website switch failed` |
| `test-www` 首页 401 | staging Basic Auth 是预期；生产 `www` 不设门禁 |
| nginx 403 / 404 | 目录权限或 `current` 不存在；检查 releases 与 symlink |
| 证书告警 | SAN 缺 apex 或 `www`；按 deployment.md §6 重签并 reload |

## 9. 边界与不做的事

- 不自动化 DNS 解析与证书签发；两者是人工前置。
- 不部署后端、不跑迁移、不创建 RDS/Redis/ACR/EIP。
- 不修改班内共享文案，直到 #88 产品变更落地。
- 不删除历史 release 目录。
