# 模型管理（M3）

盘点 ComfyUI 的模型、比对各工作流的需求、下载缺失的那些。界面在 `#/models`。

## 清单从哪来

一律经 ComfyUI 的接口取得，不自己扫文件系统：

| 接口 | 用途 |
|---|---|
| `GET /experiment/models` | 全部模型类别及其**绝对路径**（含 extra_model_paths 挂进来的） |
| `GET /experiment/models/{类别}` | 该类别下的文件：名称、所属路径下标、大小、时间 |

这样 attach 模式（用户自己开着 ComfyUI）同样可用，也天然认得用户挂的额外目录——
不必让用户在 DreamTexture 里重复配一遍模型路径。

只扫工作流用得到的类别加上几个常见大件（checkpoints / loras / vae / upscale_models /
controlnet）；ComfyUI 的类别有二十多个，全扫一遍慢且大多与我们无关。

## 缺失检测

每个工作流在 `<id>.params.json` 的 `model_requirements` 里声明所需模型。
盘点时与实际文件比对（Windows 下大小写不敏感，比对统一转小写），得到就位 / 缺失。
同一个文件被多个工作流引用时合并成一条，并列出引用它的工作流。

## 下载

`source` 与 `download_url` 是**两个不同的东西**，必须分开：

- `source` 是给人看的模型主页；
- `download_url` 是能直接 GET 到文件本体的地址。

只有 `download_url` 存在时才允许自动下载。没有直链就老实提示用户手动下——
把模型主页下下来是一张 HTML，存成 `.safetensors` 只会让 ComfyUI 在加载时
报一堆看不懂的错。

其他约定：

- **串行下载**。模型动辄几个 GB，并行既抢带宽又让进度难判断，对机械盘也不友好。
- **断点续传**。先写 `<目标>.part`，完整了再改名——中断留下的半截文件不会被
  ComfyUI 当成可用模型。重启后带 `Range` 头接着下。
- **拒绝网页响应**。有些站点不返 401 而是 200 一张登录页，按 `Content-Type` 拦下来。
- **写入目录的选择**：一类模型常有多个搜索路径，优先用 extra_model_paths 挂上来的
  那个——ComfyUI 自带的 `models/` 往往和程序同盘、余量紧张，而用户额外挂的目录
  通常正是为放模型准备的。
- 下载接口只接受 `file` + `dir` 定位到**已登记的需求**，不接受任意 URL，
  免得这个接口变成任人指使的下载器。

## 访问令牌

有些来源要求登录（HuggingFace 的 gated 模型、Civitai 的部分资源）。
令牌由用户自己在界面上填写，存在本机 `configs/secrets.json`（权限 0600，已在 .gitignore）。

**设计原则是只写不读**：接口只报告某个令牌"有没有设置"，永远不回传内容。
想换就重新填一次。下载时按域名匹配，作为 `Authorization: Bearer` 头带上。

没有令牌也能用——界面会给出来源页链接和目标目录，用户在浏览器里下完放进去即可，
重新盘点就会变成就位。

## 接口

| 方法与路径 | 说明 |
|---|---|
| `GET /api/models[?refresh=1]` | 清单、占用、需求状态、下载队列 |
| `POST /api/models/downloads` | body `{file, dir}`，加入下载队列 |
| `GET /api/models/downloads` | 下载队列 |
| `POST /api/models/downloads/{id}/cancel` | 取消 |
| `GET /api/settings` | 令牌是否已设置、ComfyUI 状态、输出目录 |
| `POST /api/settings/tokens` | body `{provider, token}`，空字符串表示清除 |

进度经 WebSocket 的 `model.download` 事件推送。

MCP 侧对应 `list_models` 与 `download_model` 两个工具。

## 测试

`internal/model/download_test.go` 用 `httptest` 覆盖了下载器的关键行为：
成功落盘并清掉 `.part`、从半截文件续传时正确发 `Range`、拒绝网页响应、
401 给出可操作的提示、没有直链时拒绝入队。
