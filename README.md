# DreamTexture

把一句提示词变成一整套能直接进虚幻引擎的 PBR 材质。底层跑本机 ComfyUI，
上层是一个自带界面的 Go 后端——不用在 ComfyUI 的节点图里连线，也不用记
那些启动参数。

> 开发中。目前后端与 Web 界面可用，UE 编辑器插件还没开始。

## 它做什么

| | 产物 |
|---|---|
| **材质** | BaseColor / Normal / Roughness / Metallic / AO / Height / ORM 一整套，配一份 `manifest.json` |
| **图片** | 单张 PNG，配一份同名 json |

两条材质管线：写实走 SDXL 无缝纹理 + [CHORD](https://github.com/ubisoft/ComfyUI-Chord)
分解，手绘走 LoRA + 传统亮度推导。底图也可以交给云端模型出（OpenAI 图像接口，
Go 直连，花的是你自己的额度），再由本地分解。

围绕这条主线还有：模型管理与下载、节点包管理、参考图库、提示词扩写、
运行日志控制台、ComfyUI 一键部署。

## 跑起来

需要 Go 1.26+。前端产物已经提交进仓库（`internal/web/dist`，`go:embed` 的目标），
所以**不装 node 也能构建**：

```bash
go build -o dreamtexture.exe ./cmd/dreamtexture
```

```bash
./dreamtexture.exe -config configs/dreamtexture.json
```

然后开 http://127.0.0.1:8777 。ComfyUI 没装好也能启动——设置页里有一键部署
（用 uv 装一份独立的 ComfyUI，不动你已有的整合包）。

改前端要装依赖，产物直接进 embed 目录：

```bash
cd web && pnpm install && pnpm build
```

## 配置

`configs/dreamtexture.json` 是运行配置，设置页的改动写回它。

访问令牌与自定义接口地址存在 `configs/secrets.json`（权限 0600，已 gitignore）。
这个文件**只写不读**：接口只报告某个令牌"有没有设置"，永远不把值回传，
接口地址也只回传协议+主机——不少第三方网关把密钥直接嵌在路径里。

## 文档

踩过的坑都写在 `docs/` 里，尤其是这几篇：

| | |
|---|---|
| [comfyui-notes.md](docs/comfyui-notes.md) | ComfyUI 的行为坑。启动期派活会慢 25 倍且探针测不出；显存被别的程序占了驱动会静默回退到内存 |
| [backend.md](docs/backend.md) | Go 后端的结构与约定 |
| [frontend.md](docs/frontend.md) | Web 前端，含视图偏好留存 |
| [cloud-source.md](docs/cloud-source.md) | 云端底图与消暗角算法（两个反直觉的教训） |
| [image-generation.md](docs/image-generation.md) | 图片生成、提示词扩写、参考图库 |
| [manifest-v1.md](docs/manifest-v1.md) | 材质套装的落盘格式 |
| [deploy.md](docs/deploy.md) | 一键部署 ComfyUI |
| [models.md](docs/models.md) · [nodes-and-catalog.md](docs/nodes-and-catalog.md) | 模型与节点包管理 |
| [workflow-editing.md](docs/workflow-editing.md) | 工作流模板的往返编辑 |
| [reference-image.md](docs/reference-image.md) | 参考图 / img2img |

## 许可

代码本身还没定许可证。

**注意产物的可商用性**：写实管线用的 CHORD 是 Ubisoft 的 **research-only**
许可，用它分解出来的材质不可商用。界面上会标出来，`manifest.json` 里也有
`license_flags`。PBR 估计那一段是按"可整段替换"设计的，换掉就没有这个限制。
