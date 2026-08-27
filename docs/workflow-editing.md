# 在 ComfyUI 里改工作流

界面在 `#/workflows`。顶栏还有一个直达 ComfyUI 的入口。

## 往返流程

```
DreamTexture 模板  ──「在 ComfyUI 中编辑」──▶  ComfyUI 编辑器（可视化改）
       ▲                                              │
       └──────────「导入」◀── 工作流 → 导出(API) ──────┘
```

1. 在工作流页点**「在 ComfyUI 中编辑」**：后端把模板转成编辑器格式，写进 ComfyUI
   的用户工作流目录，并打开 ComfyUI。
2. 在 ComfyUI 左侧「工作流」面板里打开 `dreamtexture-<id>.json`，随意改。
3. 改完用**「工作流 → 导出(API)」**导出 JSON。
4. 回 DreamTexture 点**「导入」**，填同一个 id 并勾选覆盖。

也可以直接编辑 `workflows/` 下的文件，然后点**「重新加载」**——不用重启后端。

## 为什么需要格式转换

ComfyUI 有两种工作流格式：

| | 内容 | 用途 |
|---|---|---|
| **API 格式** | 节点类型 + 输入取值 | 提交给 `/prompt` 执行。DreamTexture 的模板用这个 |
| **UI 格式** | 再加上槽位、连线表、坐标、控件数组 | 编辑器打开的就是这个 |

**API 格式直接丢给编辑器会得到一张空图**——实测确认：写进用户目录再加载，
节点数为 0。所以「在 ComfyUI 中编辑」必须先转换。

转换靠 `/object_info` 的声明重建缺失的信息（`internal/workflow/editor.go`）：

- `input_order` 给出输入的规范顺序。**这个顺序是必须的**：Go 的 map 无序，而
  `widgets_values` 是位置相关的数组，顺序错了参数就会串到别的控件上。
- 输入是连线还是控件，按 API 图里的实际取值判断——值是 `[节点id, 槽位]` 就是连线，
  否则是控件值。比查类型表可靠。
- **`control_after_generate` 的坑**：种子这类控件在编辑器里额外带一个"生成后如何变化"
  的下拉，它也占 `widgets_values` 的一个位置。漏了它，后面所有值都会错位。
  这个标记在输入规格的第二项里，是权威依据。
- 坐标按依赖深度自动分列（`layerDepths`），至少能一眼看出数据从左往右流。

## 往返一致性

实测把 `realistic-chord-v1` 导出到 ComfyUI、在编辑器里加载、再用 `graphToPrompt`
导回 API 格式，与原模板逐节点逐字段比对：

- 28 节点 / 33 连线全部还原，28 个 `dt.*` 标题全部保留，零错误节点
- KSampler 的每个控件都按名字对上了正确的值
- **唯一差异**：`dt.reference_image.image` 从 `""` 变成了 input 目录里的第一张图。
  `LoadImage` 的图片选择是个下拉框，空字符串不是合法选项，编辑器会自动选第一个。
  这个字段本来就由参考图逻辑控制（有图时注入、无图时整条支路删掉），不影响。

## 导入的约定

只接受 **API 格式**。导入时会校验：

- 每个节点必须有 `class_type`（没有就说明导出的是 UI 格式，会明确提示）
- 至少有一个 `dt.out.*` 命名的 `SaveImage` 节点——DreamTexture 靠标题把产物对上通道
- id 只能用小写字母、数字和 `. _ -`

没传参数声明时会自动生成一份最小可用的：识别 `dt.positive` / `dt.negative` /
`dt.sampler` 生成对应的可调参数，`dt.out.*` 生成输出映射。认不出来的参数不会凭空捏造——
先让工作流能跑，用户再自己去 `params.json` 里补。

**导入后立刻重载校验**：声明和图对不上就把刚写的文件撤掉再重载回去，
不会在目录里留下一个坏模板。

## 接口

| 方法与路径 | 说明 |
|---|---|
| `POST /api/workflows/{id}/open-in-comfy` | 转成编辑器格式并写进 ComfyUI 用户目录 |
| `GET /api/workflows/{id}/template` | 下载原始 API 格式 JSON |
| `POST /api/workflows/import` | 导入，body `{id, name, style, graph, override}` |
| `POST /api/workflows/reload` | 重新扫描工作流目录 |
