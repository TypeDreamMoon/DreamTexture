# 后端说明（M1）

Go 单二进制：托管 ComfyUI、调度生成任务、产出标准材质套装，并把同一套能力
同时暴露为 REST、WebSocket 和 MCP。

## 跑起来

```bash
go build -o bin/dreamtexture.exe ./cmd/dreamtexture
```

```bash
./bin/dreamtexture.exe -config configs/dreamtexture.json
```

配置见 [configs/dreamtexture.json](../configs/dreamtexture.json)。两种模式：

- `managed`：后端拉起并看护 ComfyUI 子进程（需要 `python` 与 `main_py` 路径）。这是交付形态。
- `attach`：连接用户自己开着的实例，只做健康巡检，不管生命周期。

开发时用 [configs/dev.json](../configs/dev.json)（attach 模式）：自己开一个常驻的
ComfyUI，后端重启就是秒级。managed 模式每次重启都要等 ComfyUI 重新安定，
本机实测 1~5 分钟，改一行代码等这么久没法干活。

```bash
./bin/dreamtexture.exe -config configs/dev.json
```

## 包结构

| 包 | 职责 |
|---|---|
| `internal/config` | 配置加载与校验 |
| `internal/comfy` | ComfyUI 的 HTTP/WS 客户端；子进程监管（含 Windows Job Object） |
| `internal/workflow` | 模板注册表；按 `_meta.title` 的参数注入 |
| `internal/job` | 任务队列、状态机、事件总线 |
| `internal/material` | 材质套装落盘、manifest、平铺预览 |
| `internal/store` | SQLite：任务持久化 + 素材索引（FTS5，不可用时降级 LIKE） |
| `internal/api` | REST + WebSocket |
| `internal/model` | 模型盘点、缺失检测、下载队列、令牌（见 [models.md](models.md)） |
| `internal/imagen` | 云端底图来源、亮度场压平（见 [cloud-source.md](cloud-source.md)） |
| `internal/deploy` | 一键部署独立 ComfyUI 运行时（见 [deploy.md](deploy.md)） |
| `internal/settings` | 界面上可改的配置，落回配置文件 |
| `internal/logbuf` | 日志环形缓冲，供控制台页实时显示 |
| `internal/mcpsrv` | MCP 工具，与 REST 共用同一套 service 层 |
| `internal/web` | 内嵌前端（见 [frontend.md](frontend.md)） |

## 关键设计

**单 GPU 单 worker。** 一张卡上并发跑多个任务只会互相挤爆显存，所以排队在后端做，
ComfyUI 那边始终只有一个在执行。

**成败判定不看 ComfyUI 的状态字段。** 见 [comfyui-notes.md](comfyui-notes.md) 的坑 1：
必须检查 `/prompt` 响应的 `node_errors`，并比对期望的输出节点集合与实际 `outputs`。
这条逻辑在 `job.missingOutputs`。

**每个任务用唯一输出前缀** `dreamtexture/<material_id>/`。既防任务间覆盖，也让
ComfyUI 的节点缓存不会把旧任务的文件名当成新任务的产物报回来（坑 2）。

**进度走 WS，完成判定走 history。** WebSocket 事件会因断线或缓冲丢弃而漏，
`/history` 才是唯一可靠的事实来源。

**半成品会被清掉。** 落盘中途失败就删掉整个目录，素材库里不留残缺套装。

**种子限制在 2^53-1 以内。** 种子要穿过入库、出库、manifest、前端、MCP 好几层 JSON，
JSON 没有整数类型，超过 2^53 的值被解成 float64 会被悄悄改写，manifest 里记的种子
就复现不出原图。

**manifest 只记这一次真的发生的事。** `tileable` 按次判定而非抄模板的静态值，
`generator.checkpoint` 取自实际提交的节点图而非模板声明的依赖——条件接线会整段
删掉支路，两者会不一致（见 [manifest-v1.md](manifest-v1.md)）。

**外部底图同步取，不预取。** 云端接口按量计费，用户取消排队中的任务时不该已经
把钱花掉了。代价是这段时间 GPU 闲着，对单人单卡的本地工具这个取舍更划算
（见 [cloud-source.md](cloud-source.md)）。

**ComfyUI 起不来时后端照常启动。** 全新安装的人还没有 ComfyUI，若后端因此拒绝
启动，那个"帮你把它装起来"的页面就永远打不开。任务侧会排队等它回来，健康探针
和自检都如实报告，所以降级运行是安全的（见 [deploy.md](deploy.md)）。

**日志同时留一份在内存里。** `internal/logbuf` 是个定长环形缓冲，slog 与 ComfyUI
的 stdout 都往里抄一份，控制台页据此实时显示。定长是刻意的：ComfyUI 跑起来能刷
出成千上万行，无上限地留着迟早把进程撑爆；看历史该去翻日志文件。

## 接口

### REST

| 方法与路径 | 说明 |
|---|---|
| `GET /api/workflows` | 模板列表与参数 schema，前端据此渲染表单 |
| `POST /api/workflows/{id}/open-in-comfy` · `/import` · `/reload` · `GET /{id}/template` | 工作流往返编辑（见 [workflow-editing.md](workflow-editing.md)） |
| `POST /api/uploads` | 参考图上传（见 [reference-image.md](reference-image.md)） |
| `GET /api/checks` | 环境自检：ComfyUI 连接、节点注册、模型齐全、目录可写、云端底图可达 |
| `GET /api/imagen/providers` | 云端底图来源与可用模型（见 [cloud-source.md](cloud-source.md)） |
| `POST /api/generate` | 提交生成，body `{workflow_id, params, variants, name}` |
| `GET /api/jobs` · `/api/jobs/{id}` | 任务列表与详情 |
| `POST /api/jobs/{id}/cancel` | 取消（运行中会连带打断 ComfyUI） |
| `GET /api/materials?q=&style=&fav=` | 素材库搜索 |
| `GET /api/materials/{id}` | manifest + 索引记录 |
| `POST /api/materials/{id}/favorite` | 收藏 |
| `GET /api/materials/{id}/files/{name}` | 取贴图与预览 |
| `GET /api/models[?refresh=1]` | 模型清单、占用、需求状态、下载队列 |
| `POST /api/models/downloads` · `GET` · `POST .../{id}/cancel` | 下载缺失模型 |
| `GET /api/settings` · `POST /api/settings/tokens` · `/api/settings/endpoint` | 令牌与接口地址（都只写不读）、环境信息 |
| `GET /api/config` · `POST /api/config` | 读写运行时配置，返回哪些项需重启（见 [deploy.md](deploy.md)） |
| `GET /api/deploy` · `POST /api/deploy` · `/cancel` · `/apply` | 一键部署 |
| `GET /api/logs?since=&limit=` | 增量拉日志，供控制台页 |
| `GET /api/comfy/status` · `POST /api/comfy/restart` · `/start` · `/stop` | ComfyUI 状态与生命周期 |
| `GET /api/ws` | 事件流（只推不收，所有操作走 REST） |
| `/api/ue/*` · `/api/materials/{id}/send-to-ue` | Phase B 占位，当前返回 501 |

示例：

```bash
curl -X POST http://127.0.0.1:8777/api/generate -H "Content-Type: application/json" -d '{"workflow_id":"realistic-chord-v1","variants":4,"params":{"prompt":"mossy river rocks"}}'
```

### WebSocket 事件

`job.queued` / `job.progress` / `job.done` / `job.failed` / `comfy.status`。
连上先推一条 `comfy.status`，前端不必额外拉一次。

### MCP

挂在 `/mcp`（streamable HTTP，stateless）。工具：`list_workflows`、`generate_material`、
`get_job`、`wait_job`、`search_materials`、`comfy_status`、`comfy_restart`。

Claude 与 Web 前端是对等客户端，共用同一套 service 层，看到的是同一份世界。

## 实测耗时（RTX 4070 Ti SUPER 16GB，1024×1024）

| 环节 | 耗时 |
|---|---|
| 后端启动到可接活（含等 ComfyUI 安定） | 约 80 秒 |
| 写实管线 `realistic-chord-v1` | 约 35–60 秒 |
| 手绘管线 `stylized-derive-v1` | 约 10–30 秒 |
| 后端自身开销（提交、取回、落盘、建索引） | 1–2 秒 |
