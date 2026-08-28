# ComfyUI 集成笔记

M0 阶段实跑得出，直接作为 M1（Go 后端 `internal/comfy`、`internal/workflow`、`internal/job`）的实现依据。

## 本机环境

| 项 | 值 |
|---|---|
| ComfyUI | `I:\ComfyUI\ComfyUI-aki-v3`（秋叶整合包 v3），内核 **0.9.2** |
| Python | 嵌入式 3.13.11，torch 2.9.1+cu130 |
| 启动 | `python.exe -s ComfyUI\main.py --listen 127.0.0.1 --port 8188 --preview-method none` |
| GPU | RTX 4070 Ti SUPER 16GB / 64GB 内存 |
| 模型目录 | `G:\Workflow\ComfyUIModels`，经 `ComfyUI\extra_model_paths.yaml`（键 `dreamtexture`）挂载 |

I: 盘余量紧张，所以模型不放整合包内部。后端做环境自检时要同时检查两处路径。

### 节点包

| 包 | 来源 | 许可 |
|---|---|---|
| ComfyUI-Chord | `ubisoft/ComfyUI-Chord` | Ubisoft ML，**research-only** |
| ComfyUI-seamless-tiling | `spinagon/ComfyUI-seamless-tiling` | GPL-3.0 |
| ComfyUI-TextureAlchemy | `amtarr/ComfyUI-TextureAlchemy` | **Apache 2.0**（已核实 LICENSE） |
| ComfyUI-DreamTexture | 本仓库 `comfy_nodes/` | 自有 |

自研包用目录联接挂进去，改代码后重启 ComfyUI 即生效：

```bash
mklink /J "I:\ComfyUI\ComfyUI-aki-v3\ComfyUI\custom_nodes\ComfyUI-DreamTexture" "G:\Workflow\DreamTexture\comfy_nodes"
```

## 实测耗时（1024×1024，30 步）

| 管线 | 耗时 |
|---|---|
| `stylized-derive-v1` | 约 10 秒 |
| `realistic-chord-v1`（含 CHORD 分解） | 约 18 秒 |

16GB 显存全程富余。参照项目在 12GB 卡上跑更重的链路是 85 秒，我们的余量很足。

## 必须处理的行为坑

### -1. 显存被占着时不要跑重的分解——同一条管线 650 秒 vs 31 秒

16GB 卡上，如果显存已经被上一条管线的模型占着（底模 + LoRA 等），再跑 CHORD 分解
会慢一个数量级。日志里的特征是 `Unloaded partially: ... MB freed, ... MB remains loaded`，
显存逼近上限时 Python 循环密集的泊松求解器（400 个子区域）会被显存分配器拖死。

| 场景 | 写实管线耗时 |
|---|---|
| 先跑一次手绘（占住显存）再跑写实 | **650 秒** |
| 同上，但切换前先调 `/free` 卸载模型 | **31 秒** |
| 进程上的第一个任务（显存本来就干净） | 30~60 秒 |

后端的应对（`internal/job` 的 `freeIfSwitching`）：检测到工作流切换就先调
`POST /free {unload_models, free_memory}`。同一个工作流连着跑不受影响，只在切换时腾一次，
代价是重新加载模型的几秒钟。

> 修正记录：这条最早被误判成下面第 0 条的一部分。第一次遇到 878 秒时 ComfyUI-Manager
> 恰好也在忙，就把两件事算成了一件。后来在 Manager 早已跑完的干净进程上又复现了 650 秒，
> 才分离出这个独立原因。两条都真实存在，也都要处理。

### 0. 启动期不要派活——同一张图 878 秒 vs 34.7 秒

ComfyUI 能应答 `/system_stats` 之后，自定义节点的启动任务还会在后台跑很久
（这台机器上是 ComfyUI-Manager 抓取节点 registry，实测持续 **1~5 分钟**，
且抓取常因网络失败而拖长）。落在这个窗口里的生成任务会慢一个数量级：

| 提交时机 | 同一条写实管线的 ComfyUI 执行耗时 |
|---|---|
| ComfyUI 刚应答、Manager 仍在抓取 | **14 分 38 秒** |
| Manager 启动任务完成后 | **34.75 秒** |

慢的是 Python 侧——大量后台网络与 JSON 解析抢占 GIL，而工作流里恰好有
Python 循环密集的节点（CHORD 的泊松求解器要跑 400 个子区域）。

**注意 HTTP 探针发现不了这件事**：实测整个窗口内 `/system_stats` 始终是
3 毫秒左右，aiohttp 的事件循环照常响应。所以"能连上"不等于"能好好干活"，
按探针延迟判断就绪是行不通的（这条是试出来的，不是推出来的）。

后端的应对（`internal/comfy` 的 `waitSettled`）：managed 模式下 ComfyUI 的日志
归我们管，就等日志连续 `settle_quiet`（默认 12 秒）没有新输出再放行，上限
`settle_timeout`。判据只看日志静默，不认任何具体插件，因此不与 Manager 耦合；
没有额外启动任务的环境日志立刻安静，几乎不产生等待。设为 0 可关闭。

**根治办法：把 ComfyUI-Manager 的 `network_mode` 改成 `private`**（配置文件在
`ComfyUI/user/__manager/config.ini`，只读文件，没有环境变量或命令行开关）。

实测 2026-08-27：**启动 5~6 分钟 → 42 秒，而节点目录一个没少。**

| | public | private | offline |
|---|---|---|---|
| 启动 | 5~6 分钟 | **42 秒** | 最快 |
| 5 个清单文件 | 每次刷新 | **每次刷新** | 只读缓存，永不更新 |
| ComfyRegistry（178 页） | 抓 | 跳过 | 跳过 |
| 节点目录条数 | 7757 | **7760** | 4022（自带快照，2026-01-18） |
| 精选模型清单 | 562 | **562** | 缓存 |
| 装/卸/启停节点 | 可用 | 可用 | 可用（源码里没有拦截安装的分支） |

关键在于源码的这段分支（`manager_server.py`）：

```python
if get_config()['network_mode'] != 'offline':
    ...拉取 5 个清单文件...              # private 也走这里
    if get_config()['network_mode'] == 'private':
        logging.info("private comfyregistry is not yet supported")   # 跳过
    else:
        await core.unified_manager.reload('remote', dont_wait=False)  # ← 慢的就是它
        await core.unified_manager.get_custom_nodes(channel_url, 'remote')
```

**别选 offline。** 直觉上"最彻底"，实际代价大得多：`manager_core.py` 里 offline
分支永远读缓存，连界面上点刷新都不会去联网；没有缓存时回退到 Manager 自带的快照
（本机这份是 2026-01-18 的，只有 4022 个包，不到在线的六成）。

private 之所以几乎无损，是因为 `/customnode/getlist` 读的是刚刷新过的
`custom-node-list.json` 加本地 DB，而那 178 页的 ComfyRegistry 抓取是给
`unified_manager` 的另一条路用的——对我们的节点页没有贡献，纯粹是白等。

**已知的漏网情形：`waitSettled` 会落进"暴风雨前的平静"。** 2026-08-27 实测到一次：

```
17:19:15  ComfyUI 应答（Manager 自行重启后）
17:19:39  日志最后一次增长
17:19:51  判定「已安定」（连续 12 秒无输出），放行
17:20:01  日志出现 FETCH ComfyRegistry Data: 30/178   ← 抓取这时才真正开始
17:25:24  抓取结束（前后约 6 分钟）
```

安静窗口出现在 Manager **开始**抓取之前，不是结束之后。任何固定的静默阈值都能被
一段足够长的前置停顿骗过去，把阈值调大只是把这个洞挪远一点。

试过的替代信号：`/object_info` 要在 Python 侧序列化一千多个节点类，理论上比
`/system_stats` 更能反映 GIL 争用。实测抓取期 260~400ms、空闲期约 180ms——只有
两倍出头，噪声里分不干净，做不了可靠的就绪判据。**所以这条目前没有好的通用解**，
根治办法仍是把 `network_mode` 改成 `offline`；`waitSettled` 是尽力而为的兜底，
不是保证。

### 0.5 自定义节点可能超前于 ComfyUI 本体

`ComfyUI-TextureAlchemy` 的 `shuffle_custom_colors.py` 用了 `io.Schema(search_aliases=...)`，
而本机的 ComfyUI 0.9.2 还不认这个参数，于是：

```
TypeError: Schema.__init__() got an unexpected keyword argument 'search_aliases'
```

好消息是 ComfyUI 在 `/object_info` 里对每个节点单独 try/except，**只有
`ShuffleCustomColors` 一个节点没注册进来**，同包的其余 8 个（`AOApproximator`、
`HeightToNormal`、`ChannelPackerORMA`、`NormalConverter` 等，全被我们用着）照常可用。

值得记一笔是因为下次未必这么走运：节点包会自动跟进上游新 API，而整合包的
ComfyUI 版本是滞后的。环境自检里的「工作流节点」那一项比对的正是节点类是否真的
注册成功，就是为了在这种情况下直接把缺的节点名报出来，而不是等跑到一半才崩。

### 1. `status_str: success` 不代表成功

某条输出分支校验失败（典型是缺模型文件）时，ComfyUI **照常执行其余分支**，`/history` 里
`status_str` 依然是 `success`、`completed` 依然是 `true`，只是 `outputs` 少了几个键。

判定成功必须两步都做：

1. 检查 `POST /prompt` 响应体里的 `node_errors`（校验期错误在这里，不在 history 里）；
2. 比对模板里期望的 `SaveImage` 节点集合与 `outputs` 的实际键集合，缺失即判失败。

### 2. 节点缓存会让 history 报出已不存在的文件

重复提交内容相同的图时，ComfyUI 命中节点缓存直接返回上次的输出记录——**包括文件名**。
如果那些文件已被删除或换了目录，`/history` 仍会报出旧路径。后端必须：

- 取回产物前校验文件真实存在；
- 每个任务用唯一输出前缀（避免不同任务互相覆盖，也让缓存命中不会污染新任务）。

### 3. 上游节点的张量形状不符合 IMAGE 约定

ComfyUI 的 IMAGE 是 `[B, H, W, C]`，但 ComfyUI-Chord 有两处返回 `[B, H, W]`：

- `ChordMaterialEstimation` 只对 `ndim == 4` 的输出做 permute，单通道的 `roughness` / `metalness` 原样漏出；
- `ChordNormalToHeight` 内部 `[None, None].squeeze(1)` 净得 3 维。

`SaveImage` 恰好能容忍（存成灰度 PNG），所以官方示例工作流没暴露；但一接进按 4 维解包的节点
（`AOApproximator` 的 `batch, h, w, channels = height.shape`、`ChannelPackerORMA` 的 `image[:, :, :, 0:3]`）就崩。

不改上游文件（升级会丢），用本仓库的 `DT_EnsureImageShape` 在中间过一道。
接入新的 PBR 估计模型时要重新检查这一点。

### 显存被别的程序占了 → 不报错，改用内存硬算

**症状**：GPU 占用 99%、显存打满、进度条停在某个百分比再也不动，日志最后一行
是 `Requested to load AutoencoderKL / loaded completely`，然后什么都不再输出。
等半小时也不会结束，也不会报错。

**成因**：ComfyUI 只在**装载模型那一刻**看一眼空闲显存，之后不再复查。等它把
显存吃到只剩几百兆，别的程序（这台机器上典型是**虚幻编辑器**）一涨、
或者解码环节要一大块连续显存，Windows 显卡驱动的"系统内存回退"就会接管——
它不抛 OOM，而是改用系统内存走 PCIe 算，慢几十倍。于是既没有报错也没有进展。

实测过的一次：ComfyUI 报"可用 3521 MB"，而 `nvidia-smi` 显示 15488/16375 MiB
已被占用，torch 自己的池握着 9408 MB。同一张图，关掉占显存的程序之后 **12 秒**跑完。

**三层处理**：

1. `--reserve-vram`（配置项 `comfy.reserve_vram_gb`，默认 1）。让 ComfyUI
   主动把模型换出到内存，而不是等驱动去悄悄降级。同时开着虚幻编辑器时调到 2~3。
2. 看门狗（`internal/job/stall.go`）。8 分钟没有任何动静就判定卡死、打断 ComfyUI、
   把任务判负，并现场读一次显存写进报错——不打断的话后面排队的任务全跟着陪葬。
   阈值定得宽是因为没有进度事件的环节确实存在且合法（VAEDecode、CHORD 分解、
   大图落盘都会一声不吭跑上几十秒），但差着一个数量级。
3. 自检里单列一条「显存」（`internal/api/vram.go`），并用 `nvidia-smi` 点名
   正在用显卡的程序。看见 `UnrealEditor` 比看见一个数字有用得多。

> WDDM 驱动模型下 `nvidia-smi` 查不到每个进程占了多少显存（一律 N/A），
> 所以只报名字，不编造数值。

**还想更彻底**：NVIDIA 控制面板 →「管理 3D 设置」→「CUDA - 系统内存回退策略」
改成「优先不使用系统内存回退」。改完之后显存不够会**立刻报 OOM**，
而不是无声地慢几十倍。

自检里会如实报告这一项当前是开是关（`internal/nvidia`），下一节说清了
为什么只能读不能改。

### 不要自己设 `PYTORCH_CUDA_ALLOC_CONF`

`cuda_malloc.py` 会往这个环境变量后面**追加** `backend:cudaMallocAsync`：

```python
env_var = os.environ.get('PYTORCH_CUDA_ALLOC_CONF', None)
if env_var is None: env_var = "backend:cudaMallocAsync"
else:               env_var += ",backend:cudaMallocAsync"
```

而 `expandable_segments` 只对原生缓存分配器有效，拼在一起等于白写——
torch 静默忽略，看不出任何异常。想换分配器就往 `extra_args` 里加
`--disable-cuda-malloc`，那才是 ComfyUI 认的开关。

## 启动参数做成了控件

设置页「性能」那一节把 ComfyUI 的启动参数图形化，见 `internal/comfy/flags.go`
与 `flags_parse.go`。清单对着本机 0.9.2 的 `comfy/cli_args.py` 核过。

**互斥组一一对应成下拉框。** 这是做这件事最主要的理由：`vram_group`、
`attn_group`、`fpunet_group` 这些组里同时写两个，ComfyUI 直接起不来，
而用户看到的只是"启动失败"，没有任何线索指向参数冲突。做成下拉之后
冲突的选项根本选不到一起。

**认不出来的参数原样保留。** `Parse` 返回 `leftover`，它们落进界面上的
「其他参数」。这份目录跟不上上游是早晚的事，跟不上的时候把用户手写的
`--whitelist-custom-nodes` 吃掉是不可接受的。`flags_test.go` 盯着这条。

几个容易写错的地方：

- **`--k=v` 和 `--k v` 都要认。** argparse 两种都吃；不统一的话
  `--preview-method=none` 匹配不上，界面显示"自动"而实际跑的是 none
- **长选项优先匹配。** `--fast fp16_accumulation` 必须先于光秃秃的 `--fast`
  试，否则界面把"仅 fp16 累加"显示成"全开"，那是两种不同的行为
- **反向开关。** ComfyUI 的开关多是 `--disable-xxx`，界面上正着说才好懂，
  所以有 `Invert`：开着=不加参数，关掉才加。别让用户去理解双重否定
- **`--listen` / `--port` / `--reserve-vram` 由后端自己填**，写进用户参数里
  只会和它打架（后写的赢），保存时会被拦下

### 使用共享显存 / 系统内存回退：能读，不能写

不是命令行参数，所以不在启动参数那份目录里。它是驱动里针对某个 exe 的
应用配置，编号 **`0x10ECECC9`**，值 1 = 不使用回退。

**这个编号是实测出来的，不能靠记。** 我一开始记成 `0x10F9DC81`，
`NvAPI_DRS_GetSettingNameFromId` 一查那其实是 "Enable application for Optimus"
——照着写会去改一个完全无关的设置。

拿到它的过程：这一项在驱动里**没有名字**，`NvAPI_DRS_EnumAvailableSettingIds`
列出的 130 个具名设置里没有它，`GetSettingIdFromName` 按名字反查也全部失败。
只能让写过它的程序先写一次（把秋叶启动器的「使用共享显存」关掉），
再用 `NvAPI_DRS_SaveSettingsToFile` 把整个配置库导出来比对。同一份导出里
MXGP 那条配置的 `0x1033CEC2` / `0x1033DCD3` 与具名清单对得上，说明解析没错。

**读要绕一圈。** 无名设置会被公开读接口过滤掉：profile 明明报
`numOfSettings=1`，`EnumSettings` 返回 0 项，`GetSetting` 报
`SETTING_NOT_FOUND`。所以 `internal/nvidia` 的做法是——用
`FindApplicationByName` 问出这个 exe 归哪条配置管（这一步公开接口好使），
再从导出的配置库里把值抠出来。

**写不了。** `NvAPI_DRS_SetSetting` 会拿编号去校验驱动自己的设置库，
无名设置一律 `SETTING_NOT_FOUND`。实测过六种写法（补 `settingName`、
补 `settingType`、补 `settingLocation`…）全部被拒，而同一段代码写已知的
Vertical Sync 立刻 `rc=0`——所以不是调用姿势的问题。

剩下唯一的路是 `NvAPI_DRS_LoadSettingsFromFile`：把整个配置库（本机 7962 条）
导出、按未公开的二进制格式改、再整体导回。那个格式里全是绝对偏移，插入一条
就得全局修正，改错一次影响的是机器上所有配置。**一个材质工具不该干这个**，
所以到此为止：我们只报告状态，改还是去 NVIDIA 控制面板，或者用秋叶启动器
那个开关。

> 导出格式（逆出来的，无文档）：一条配置的记录是
> `53 00 xx 00 | size | count | nameOffset`，其后 `count` 个 16 字节条目
> `A4 00 10 00 | id | type | value`；`nameOffset` 是文件内的绝对偏移，
> 指向 UTF-16 的配置名。解析见 `internal/nvidia/sysmem_windows.go`。

## 工作流模板约定

- 模板是 ComfyUI 导出的 **API format** JSON，与参数声明 sidecar `<id>.params.json` 成对存放。
- 所有需要注入的节点必须有唯一 `_meta.title`，统一 `dt.` 前缀。按 title 定位比按节点 id 抗重排。
- 输出节点统一命名 `dt.out.<通道名>`，后端据此把产物映射到 manifest 的 `maps`。
- PBR 估计段独立命名（`dt.pbr_estimate` / `dt.chord_model`），便于将来整段替换掉 research-only 的 CHORD。
- 参数注入原型见 `scratchpad/run_wf.py`，支持 `--set 标题.输入名=值` 与节点 bypass，Go 侧照此实现。

## 无缝平铺

`SeamlessTile`（改 UNet 卷积为 circular padding）+ `MakeCircularVAE` 两个节点即可，
实测 3×3 平铺零接缝。**仅对 SDXL 这类 UNet 架构有效**，将来换 DiT 底座（Flux / Z-Image）
必须改走「offset + inpaint 修补」后处理。

CHORD 内部会 `apply_circular_padding`，分解不破坏无缝，无需额外处理。
