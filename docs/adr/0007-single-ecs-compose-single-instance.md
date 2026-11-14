# 0007 — 单台 ECS + Docker Compose + 托管数据服务

- Status: Accepted
- Date: 2026-11-03

## Context

项目规模为 ≤10 所园、≤3000 用户、峰值 ≤50 QPS，且由单人运维。后端进程内包含定时通知、识别队列、课表同步与导出临时文件；K8s / SLB 的成本与复杂度与规模不匹配。原生 systemd 部署虽少一层容器，但环境一致性、回滚和 staging 隔离都更差。

## Decision

单台 ECS 承载宿主机 Nginx 与两套 Docker Compose：生产仅 `api`，预发为 `api + mysql + redis`。生产 MySQL / Redis 使用阿里云托管服务。首期不引入 SLB、K8s、CDN、WAF，后端按单实例运行。

## Consequences

- ECS 是单点故障；本期接受，不做水平扩展。
- 进程内任务与导出临时文件按单实例设计；将来多实例必须先改为共享存储 / 粘性路由。
- 每次发布有短暂重启；优雅停机是上线前置。
- 监控、备份与恢复演练是单点风险的补偿措施，不是可选项。

## Alternatives considered

- **ACK / K8s**：成本与运维复杂度远超当前规模。
- **systemd 原生二进制**：少一层容器，但可复现性、回滚和 staging 隔离更差。
- **多实例 + SLB**：需要先改导出 / 下载与进程内任务，首期不值得。
