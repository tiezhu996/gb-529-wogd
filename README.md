# LNG 蒸发气物理质量平衡

面向 LNG 储运技术人员的离线工程分析工作台。系统将储罐参数、不可变计量快照和期间物理转移固化为可重放证据，计算液相质量平衡、BOG / 未解释项及合成不确定度，并通过独立复核形成审计链。

系统不连接 DCS、SIS、阀门或现场仪表，不下发控制动作，不用于贸易结算、库存台账或财务对账。未解释差异不能直接认定为泄漏或安全事件。

## Docker 快速启动

```bash
cp .env.example .env
# 修改 .env 中的 DB_PASSWORD 和 JWT_SECRET
docker compose up -d --build
docker compose ps
```

访问地址：

- 前端工作台：<http://127.0.0.1:18529>
- 后端健康检查：<http://127.0.0.1:19529/healthz>
- 后端就绪检查：<http://127.0.0.1:19529/readyz>
- PostgreSQL：`127.0.0.1:57529`

种子账号密码均为 `LngBalance!2026`：

| 角色 | 邮箱 | 权限 |
| --- | --- | --- |
| 工艺分析员 | `analyst@lng.local` | 储罐、快照和转移维护，运行平衡并提交复核 |
| 独立复核员 | `reviewer@lng.local` | 查询证据、接受或驳回待复核运行、查看审计 |
| 管理员 | `admin@lng.local` | 全部权限，并可作废非终态运行 |

停止并删除本项目专用数据卷：

```bash
docker compose down -v --remove-orphans
docker compose ps
```

## 主要功能

- 储罐参数：维护名义容积、有效液位、参考密度、温度膨胀系数和多项式罐容曲线；更新使用 `version` 乐观锁。
- 计量快照：记录液位、液温、汽相压力、密度、不确定度和质量标记；写入时计算罐容、修正密度及液相质量，原值不可覆盖。
- 物理转移：记录实际流入/流出、时间段、计量质量和物理参考；同一储罐的未取消时间段不得重叠。
- 平衡运行：选择期初和期末有效快照，仅汇总**完全落入**期间的已确认转移（`start_at >= 期间起点` 且 `end_at <= 期间终点`）；同一储罐任一已确认转移只要在期间内有质量交换却跨越期间起点或终点，整次计算即被拒绝，错误返回转移编号与越界时刻，不会静默剔除或按比例折算；草稿和已取消转移一律不参与。计算在单个数据库事务内锁定储罐并重新读取快照和转移，确认/取消与计算并发时取事务内最新状态，失败请求不产生任何平衡记录或审计。
- 独立复核：`queued -> calculating -> pending_review -> accepted | rejected | invalidated`，接受/驳回只允许复核员或管理员。
- 审计追踪：参数、快照、转移、运行、提交和复核均保存 request ID、操作者及前后摘要。
- 横切能力：JWT、RBAC、全局错误、结构化访问日志、request ID、panic recovery、本地令牌桶限流和优雅停机。

## 计算方法与单位

质量单位为 `kg`，体积为 `m³`，密度为 `kg/m³`，液位为 `m`，温度为 `°C`，压力为 `kPa`。

1. 罐容曲线：`V(h) = a0 + a1*h + a2*h² + ...`。保存参数时按 20 个液位点校验单调性和名义容积边界。
2. 温度修正密度：`rho_t = rho_input / (1 + alpha * (T - T_ref))`。
3. 液相质量：`M = V(h) * rho_t`。
4. 物理质量平衡：`M_open + M_in - M_out - M_close = M_BOG/unexplained`。
5. 不确定度：`U = sqrt(sum((M_i * u_i / 100)²))`。
6. `|deviation| <= U` 为 `within_uncertainty`；`U < |deviation| <= 2U` 为 `watch`；超过 `2U` 为 `investigate`；无效边界为 `invalid`。

模型不包含组分、分层、压力-温度平衡、管线存量或现场仪表系统误差，不能替代经批准的工艺与安全程序。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 前端 | React 18、TypeScript、Vite 8、Ant Design、Zustand、ECharts、Lucide |
| 后端 | Go 1.22、Gin、GORM、validator/v10、JWT、bcrypt、slog |
| 正式数据库 | PostgreSQL 16 |
| 自包含验证 | GORM SQLite 内存数据库，仅用于测试和 runtime smoke |
| 部署 | Docker Compose、Nginx 多阶段构建与 `/api` 反向代理 |

## 项目结构

```text
.
├── backend/
│   ├── cmd/server/                 # 服务入口和优雅停机
│   ├── internal/config/            # 环境、双数据库驱动、迁移与种子
│   ├── internal/constants/         # 状态、差异级别、质量和角色
│   ├── internal/model/             # 四个核心 GORM 实体
│   ├── internal/dto/               # HTTP 输入与证据输出
│   ├── internal/repository/        # 事务、重叠检查、版本更新与审计
│   ├── internal/service/           # 领域校验、状态机和计算编排
│   ├── internal/handler/           # HTTP 参数与统一响应
│   ├── internal/router/            # /api/v1 路由与权限边界
│   ├── internal/middleware/        # request ID、认证、RBAC、日志、恢复、限流
│   ├── internal/balance/           # 单位、罐容、质量方程、不确定度
│   └── pkg/api/                    # 统一响应和错误模型
├── frontend/src/
│   ├── api/                        # 按实体拆分的真实 API 客户端
│   ├── stores/                     # 认证与四个领域 Zustand store
│   ├── types/                      # 前后端一致的领域类型
│   ├── components/common/          # 质量徽标、瀑布图、证据面板
│   ├── hooks/                      # useAuth、useBalanceRun
│   ├── pages/                      # 储罐、计量、转移、平衡、审计
│   ├── router/                     # JWT/RBAC 路由守卫
│   └── utils/                      # 单位、数字和时间格式
├── docker-compose.yml
├── go.work
├── runtime_smoke.json
└── output/                         # 验收报告与内置 Browser 截图
```

## API 清单

业务 API 使用 `/api/v1`；除登录外均要求 Bearer Token。成功响应为 `{ data, request_id, meta? }`，错误响应为 `{ error: { code, message, details? }, request_id }`。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/auth/login` | 登录 |
| `GET` | `/api/v1/auth/me` | 当前身份 |
| `GET/POST` | `/api/v1/tanks` | 储罐列表与登记 |
| `GET/PUT` | `/api/v1/tanks/:id` | 详情与版本化参数更新 |
| `GET` | `/api/v1/tanks/:id/measurement-quality` | 快照质量汇总 |
| `GET/POST` | `/api/v1/measurements` | 不可变计量快照 |
| `GET` | `/api/v1/measurements/:id` | 快照详情 |
| `GET/POST` | `/api/v1/transfers` | 物理转移列表与登记 |
| `POST` | `/api/v1/transfers/:id/status` | 确认或取消转移 |
| `GET` | `/api/v1/balances` | 平衡运行历史 |
| `POST` | `/api/v1/balances/run` | 运行质量平衡 |
| `POST` | `/api/v1/balances/:id/submit` | 提交独立复核 |
| `POST` | `/api/v1/balances/:id/review` | 接受或驳回 |
| `POST` | `/api/v1/balances/:id/invalidate` | 管理员作废 |
| `GET` | `/api/v1/balances/:id/uncertainty` | 不确定度分解 |
| `GET` | `/api/v1/audits` | 复核员/管理员查询审计 |

## 共享枚举出现位置

`BalanceStatus = queued | calculating | pending_review | accepted | rejected | invalidated`：

- 数据库/model：`backend/internal/model/balance_run.go`
- 后端常量、DTO、repository、service、handler、router：`backend/internal/constants/balance.go`、`backend/internal/dto/balance_run.go`、`backend/internal/repository/balance_run.go`、`backend/internal/service/balance_run.go`、`backend/internal/handler/balance_run.go`、`backend/internal/router/router.go`
- 前端类型、API、store、hook、组件和页面：`frontend/src/types/balance.ts`、`frontend/src/api/balances.ts`、`frontend/src/stores/balanceStore.ts`、`frontend/src/hooks/useBalanceRun.ts`、`frontend/src/components/common/EvidenceBreakdownPanel.tsx`、`frontend/src/pages/BalancesPage.tsx`

`DeviationLevel = within_uncertainty | watch | investigate | invalid`：

- 数据库/model：`backend/internal/model/balance_run.go`
- 后端常量、算法与 service：`backend/internal/constants/deviation.go`、`backend/internal/balance/uncertainty.go`、`backend/internal/service/balance_run.go`
- 前端类型、共享组件和页面：`frontend/src/types/deviation.ts`、`frontend/src/types/balance.ts`、`frontend/src/components/common/EvidenceBreakdownPanel.tsx`、`frontend/src/components/common/MassBalanceWaterfall.tsx`、`frontend/src/pages/BalancesPage.tsx`

## 环境变量与端口

| 变量 | 默认示例 | 说明 |
| --- | --- | --- |
| `COMPOSE_PROJECT_NAME` | `lng-boiloff-gas-balance` | 固定英文 Compose 名 |
| `FRONTEND_PORT` | `18529` | 前端宿主端口 |
| `BACKEND_PORT` | `19529` | 后端宿主端口 |
| `DB_PORT` | `57529` | PostgreSQL 宿主端口 |
| `DB_NAME/DB_USER/DB_PASSWORD` | 见 `.env.example` | PostgreSQL 配置 |
| `JWT_SECRET` | 至少 32 字符 | JWT HMAC 密钥 |
| `CORS_ORIGINS` | 两个本地地址 | 逗号分隔的允许来源 |
| `DEFAULT_UNCERTAINTY_PCT` | `0.35` | 默认工程不确定度配置边界 |
| `LOG_LEVEL` | `info` | `debug/info/warn/error` |

数据库使用命名卷 `postgres_data`，不绑定中文宿主路径。Nginx 保留 `/api/v1` 路径，并提供 SPA fallback、`/api/healthz` 和 `/api/readyz`。

## 本地开发与质量检查

后端可用 SQLite 文件启动：

```bash
export DB_DRIVER=sqlite
export DB_DSN='file:local-dev.db?_foreign_keys=on'
export JWT_SECRET='local-development-secret-at-least-32-bytes'
export PORT=19529
go run ./backend/cmd/server
```

另开终端启动前端：

```bash
npm --prefix frontend ci
npm --prefix frontend run dev
```

完整检查：

```bash
go work sync
go build ./backend/...
go vet ./backend/...
go test ./backend/...
npm --prefix frontend run build
node scripts/api-smoke.mjs


```

## 常见问题

- `OPENING_SNAPSHOT_MISSING`：期间开始时点前没有 `good` 或 `suspect` 快照。
- `CLOSING_SNAPSHOT_MISSING`：期间内没有晚于期初的有效期末快照。
- `TRANSFER_TIME_OVERLAP`：同一储罐已有时间重叠且未取消的物理转移。
- `TRANSFER_CROSSES_PERIOD_BOUNDARY`：已确认物理转移跨越平衡期间起点或终点；错误详情返回 `transfer_id`（转移编号）、`crossed_boundary`（`period_start`/`period_end`）、`crossed_at`（越界时刻）和转移起止时间。必须先取消该转移或调整期间后重算，系统不会按比例折算或静默剔除。
- `TANK_VERSION_CONFLICT` / `BALANCE_VERSION_CONFLICT`：数据被其他请求更新，刷新后使用新版本重试。
- 后端未 healthy：运行 `docker compose logs backend`，检查 JWT、数据库配置和 PostgreSQL 健康状态。
- 前端 API 失败：确认 Nginx 的 `/api/` 使用无尾斜杠的 `proxy_pass http://backend:8080`，避免剥离 `/api`。

## License

MIT License。该软件仅用于离线工程演示与分析支持，不构成设备控制、贸易计量或现场安全处置建议。
