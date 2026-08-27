# 节点管理与模型库

不用离开 DreamTexture 就能装节点、找模型。界面在 `#/nodes` 和 `#/models` 的「浏览下载」档。

## 节点：代理 ComfyUI-Manager

不自己实现 `git clone` + `pip install`。Manager 已经把安装、依赖解析（含 pip 覆盖、
uv 统一编译、跨节点冲突检测）、版本、快照、风险等级做得很完整，重写一遍只会多一套
要维护的坑。更要命的是**定位 Python 解释器**——整合包和 Desktop 用的都是内嵌 Python，
自己 pip 极易装到系统 Python 里去。

我们做的是把它的能力搬进自己的界面：7757 个节点包可搜（名称、作者、描述），
已安装的排前面，一键安装 / 卸载 / 启停 / 更新。

### 两代接口必须运行时探测

Manager 在 V4 做了破坏性改动：

| | V3 | V4 |
|---|---|---|
| 路径 | 裸路径 `/manager/queue/install` | `/v2/` 前缀 |
| 操作端点 | 五个逐操作端点 | 合并成一个 `/v2/manager/queue/task`，用 `kind` 区分 |
| 节点目录 | `/customnode/getlist` | **默认 UI 模式下没有这个端点**（前端直接查 Registry），需 `--enable-manager-legacy-ui` |

所以不能写死路径。`internal/nodes/capability.go` 启动时先试 `/api/v2/manager/version`，
不通再试 `/manager/version`，据此选路。**不按版本号字符串猜能力**——发行版会带自己的
版本号（本机秋叶整合包报的是 V3.39.2），直接试端点通不通才靠谱。

### 三个必踩的坑

**入队后必须显式 start。** Manager 的 worker 不会自己醒来，`queue/task` 之后不调
`queue/start` 的话任务会一直排着不动。两代都是如此。

**判定完成不能看 `total_count > 0`。** Manager 干完活会把计数清零，「不在处理且
total>0」这个条件永远等不到，界面会一直转圈。正确做法是「见过在处理、之后不在处理了」
就算完，另加一个轮询次数上限兜住任务瞬间结束的情况。

**装完必须重启 ComfyUI** 节点才会注册进来。界面上做成显式的「立即重启」按钮，
不偷偷重启——用户可能正在跑图。

安全级别：Manager 的 `security_level` 会拦掉安装类操作。返回 403 时我们直接告诉用户
去 `ComfyUI/user/__manager/config.ini` 把它调成 `normal` 或 `weak`。

## 模型库：两个来源互补

| 来源 | 数量 | 特点 |
|---|---|---|
| **精选清单** | 562 | Manager 人工维护，**目标目录是核对过的**，一键下载最稳，但覆盖窄 |
| **Civitai** | 社区库 | 量大，带预览图、触发词、底模、下载量；目标目录靠启发式判断 |

精选的排前面——它的目录是人工核对过的，比启发式可靠。

### 目标目录是可改的

Civitai 的 `type` 和 ComfyUI 的目录并非一一对应，启发式会猜错，而**猜错会让模型静默
失效**（ComfyUI 只是扫不到它，不会报错）。所以界面上目标目录是个可改的下拉框，
候选项从 `GET /api/models` 动态取——不硬编码，因为用户很可能通过
`extra_model_paths.yaml` 把模型挂在别的盘上（本机就挂在 `G:\Workflow\ComfyUIModels`）。

同一类目录有多个搜索路径时，优先写用户额外挂的那个，理由同模型盘点：ComfyUI 自带的
`models/` 往往和程序同盘、余量紧张。

### 下载走我们自己的下载器

不用 Manager 的 `install_model`：那是个黑盒，不支持断点续传控制、SHA256 校验、
自定义落盘路径。模型动辄几个 GB，这些都是刚需。复用 M3 已经写好并测试过的下载器
（`.part` 临时文件、Range 续传、拒绝网页响应、401 给可操作提示）。

### 接口不接受任意 URL

节点操作只接受包 id，模型下载只接受来源 + id——具体的仓库地址和下载链从我们自己的
检索结果里重新取。不让客户端指定地址，免得这两个接口变成任人指使的下载器。

## 接口

| 方法与路径 | 说明 |
|---|---|
| `GET /api/nodes/manager` | Manager 可用性、版本、接口世代、有无节点目录 |
| `GET /api/nodes?q=&state=&limit=` | 搜索节点包 |
| `POST /api/nodes/action` | body `{id, action}`，action 取 install/uninstall/enable/disable/update |
| `GET /api/nodes/queue` | Manager 的任务队列状态 |
| `GET /api/catalog/models?q=&kind=&source=` | 检索模型库 |
| `GET /api/catalog/dirs` | ComfyUI 认得的模型目录，供目标目录选择 |
| `POST /api/catalog/download` | body `{source, id, query, kind, dir}` |

## 实测记录（2026-08-27，本机）

- Manager V3.39.2，节点目录 7757 个，首次拉取 4.7MB / 约 13 秒，之后走 10 分钟缓存
- 真实安装了 `ComfyUI-Universal-Seamless-Tiles` 验证闭环，确认落进 `custom_nodes/`
- 精选清单检索 0.9 秒；Civitai 检索 1.1 秒（带预览图与触发词）
- 从精选清单下载 TAESD 解码器（4.9MB）落到 `models/vae_approx/`，校验字节数一致
