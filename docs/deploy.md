# 一键部署与运行时管理

设置页（`#/settings`）里的「一键部署独立环境」会在 DreamTexture 目录下装一套
自带 Python 与 PyTorch 的 ComfyUI，从此不依赖任何外部整合包。

**已有的 ComfyUI 一根手指都不会碰。** 装完之后由用户点「应用到配置」才切过去，
随时能在路径设置里切回来。理由很简单：整合包（绘世之类）自带的环境修复、内核
版本切换是有价值的工具，没道理为了"统一管理"就把它废掉。

## 为什么用 uv

部署要解决三件事：弄一个 Python、装 PyTorch、装一堆依赖。用 uv 是因为：

- **它自己会装 Python**，装的是完整 CPython，不是那种缺 stdlib、要手改 `._pth`
  的嵌入式版。省掉了整个"下 embeddable 包 → 改 _pth → bootstrap pip"的脆弱链条。
- **断网重试和缓存比 pip 强**。torch 那一步要下 2~3GB，在国内网络下必然会断，
  能续上比什么都重要。
- 单个静态二进制，没有自己的运行时依赖。

没有 uv 时部署会直接失败并给出安装命令，而不是硬着头皮走 pip——那条路的失败
方式太多，报错也难懂。

## 步骤

| 步骤 | 做什么 | 幂等策略 |
|---|---|---|
| `tools` | 找 uv 与 git | — |
| `python` | `uv python install 3.13` | uv 自己判重 |
| `venv` | 建虚拟环境 | 已有 `venv/Scripts/python.exe` 就跳过 |
| `torch` | 从 PyTorch 官方源装 CUDA 版 | 已装**且档位相符**才跳过 |
| `comfyui` | git clone | 已有 `main.py` 就跳过 |
| `deps` | ComfyUI 的 requirements | 每次都跑（uv 自己判重） |
| `nodes` | 拉 4 个必需节点包 | 目录已存在就跳过 |
| `dtnodes` | 接入自有节点 | 已存在就跳过 |
| `modelpaths` | 写 `extra_model_paths.yaml` | 未指定模型库就跳过 |
| `verify` | 验 Python / torch / CUDA | — |

每一步都是幂等的。部署动辄十几分钟、要下几个 GB，中途断网、关机、手滑取消都很
正常，重跑必须能接着走而不是从头来。

只装这四个节点包：

| 包 | 为什么必需 |
|---|---|
| ComfyUI-Manager | 节点页要靠它装/卸节点 |
| ComfyUI-Chord | 写实管线的 PBR 分解 |
| ComfyUI-seamless-tiling | 循环卷积无缝平铺 |
| ComfyUI-TextureAlchemy | 法线/AO/通道打包等派生节点 |

不把整合包里那十几个都搬过来——那些是用户按别的需求装的，一个"能跑
DreamTexture"的环境不需要它们，装了反而拖慢启动、扩大依赖冲突的面积。
想要什么另外在节点页里装。

## 两个会静默毁掉整个部署的坑

**ComfyUI 的 requirements.txt 会把 CUDA 版 torch 换成 CPU 版。**

实测抓到的：跳过 torch 步骤后，`deps` 这一步自己从 PyPI 拉来了 `2.13.0+cpu`——
因为 ComfyUI 的 requirements 里列着 `torch`，而 PyPI 上的 torch 就是 CPU 版。
正常顺序下 torch 装在 deps 之前，但只要版本约束对不上，deps 照样会把它顶掉。

**这个故障是静默的**：装完一切正常，ComfyUI 也能起，只是生成慢十几倍，
用户根本想不到该往这儿查。所以 `deps` 跑完会立刻回头验一次
`torch.version.cuda`，被顶掉就用 `--reinstall-package torch` 从 CUDA 源补回来。

**"已装则跳过"必须连档位一起看。** 同一个陷阱的另一面：如果 torch 步骤只验
"能不能 import"，那么一个 CPU 版的 torch 会被当成装好了，于是**永远装不上
CUDA 版**。判据用 `torch.version.cuda`（运行时真实值）而不是版本号里有没有
`+cu` 后缀——本地编译的轮子未必带那个后缀。

## 代理

部署时的代理由后端按设置页的配置填进子进程的环境变量，**不接受请求方指定**——
那个值决定了子进程把流量导去哪儿，让客户端指定等于开了个后门。

必须显式往下传而不是靠继承：git 和 uv 读的是 `HTTPS_PROXY`，而后端进程可能是
从一个没有这些变量的环境里起来的（服务、计划任务、双击 exe）。不传的话
git clone GitHub 会卡到超时，"一键"就成了空话。

镜像开关只影响 PyPI（清华）与 ComfyUI 源（jihulab）。**PyTorch 的 CUDA 轮子
只在它自己的源上**，任何镜像都没有，那一步必须能出网。

## ComfyUI 起不来时后端照常启动

这条是被一键部署逼出来的：全新安装的人根本还没有 ComfyUI，若后端因此拒绝启动，
那个"帮你把 ComfyUI 装起来"的页面就永远打不开——**最需要它的人反而用不上**。

现在的行为：

- managed 模式下路径不存在，**不等超时**直接降级。让新用户对着三分钟的进度条
  干等只会以为程序挂了。
- attach 模式下到点就放弃（不再无限等）：那儿没有我们启的进程，"它在忙"这个
  理由不成立，多半就是根本没开。
- 无论成败都会把巡检协程带起来——失败之后正是最需要它自愈的时候。
- 环境根本没装好时，巡检不会一遍遍试着重启，否则每十秒往日志里刷一条同样的
  失败，把真正有用的信息淹掉。

任务侧本来就会排队等 ComfyUI 回来（`waitComfyAvailable`），健康探针和环境自检
也都如实报告，所以降级运行是安全的。

## 换过去会失去什么

新环境只装 4 个节点包，原整合包里有 17 个。切过去之后这些就不在了
（想要的话在节点页里一个个装回来，DreamTexture 自己一个都不需要）：

`comfyui_controlnet_aux`、`ComfyUI_IPAdapter_plus`、`ComfyUI_UltimateSDUpscale`、
`ComfyUI-GGUF`、`ComfyUI-Impact-Pack`、`ComfyUI-LTXVideo`、`ComfyUI-RMBG`、
`ComfyUI-VideoHelperSuite`、`ComfyUI-WanVideoWrapper`、`rgthree-comfy`、
`ComfyUI_Custom_Nodes_AlekPet`、`ComfyUI-Universal-Seamless-Tiles`

模型不受影响——两边共用同一个模型库目录。

## 启停

设置页和控制台都有启动 / 停止 / 重启。手动停止走的是 `StopByUser` 而不是
后端退出时用的 `Stop`：后者会把巡检一并结束，而这里只是把进程停住，并记下
"用户不想让它跑"——否则自动重启会立刻把它拉起来，那个停止按钮看起来就是坏的。

被手动停止期间健康探针也会跳过，否则界面上分不清"我停的"和"它挂了"。

## 实测记录（2026-08-27，本机）

一次完整部署，**6 分 11 秒**，产出 3.9GB：

```
tools       done     uv 0.11.2 / git 2.55.0.windows.4
python      done     CPython 3.13
venv        done     venv\Scripts\python.exe
torch       done     2.13.0+cu130            ← 检出已装的 2.13.0+cpu 档位不符，重装
comfyui     done     ComfyUI
deps        done
nodes       done     新增 4 个
dtnodes     done     已联接到 G:\Workflow\DreamTexture\comfy_nodes
modelpaths  done     复用 G:\Workflow\ComfyUIModels
verify      done     Python 3.13.12 / torch 2.13.0+cu130 / CUDA True
```

装完之后把它拉起来验收（另开 8189 端口，不影响原有实例）：

- 994 个节点类注册；两条本地管线加两条云端管线用到的 **22 个节点一个不缺**
- 模型库正确接上：`CheckpointLoaderSimple` 能看到 `Juggernaut-XL-v9` 与
  `chord_v1`，都来自共享的 `G:\Workflow\ComfyUIModels`

994 对比原整合包的 1357，差在只装了 4 个节点包而不是 17 个——这是刻意的。

写进配置的路径**能相对就相对**（`runtime\venv\Scripts\python.exe`），
整个 DreamTexture 文件夹换个盘符照样能跑。

## 接口

| 方法与路径 | 说明 |
|---|---|
| `GET /api/deploy` | 部署状态、默认参数、当前配置的路径 |
| `POST /api/deploy` | 开始部署，立刻返回；进度靠轮询，详细输出去控制台看 |
| `POST /api/deploy/cancel` | 中断；已下好的留着，重跑时跳过 |
| `POST /api/deploy/apply` | 把装好的环境写进配置（需重启后端生效） |
| `GET /api/config` · `POST /api/config` | 读写运行时配置，返回哪些项需要重启 |
| `POST /api/comfy/start` · `/stop` · `/restart` | ComfyUI 生命周期 |
| `GET /api/logs?since=&limit=` | 增量拉日志 |

`POST /api/deploy` 用的是 `context.WithoutCancel`：部署要跑十几分钟，挂在 HTTP
请求的生命周期上的话，浏览器一刷新连接就断、ctx 就取消了，装到一半的环境比
没装还麻烦。
