# 工作流分段与组合

材质管线天然分两段：**出图**（产出一张底图）和**分解**（把一张底图拆成 PBR 通道）。
以前每种搭配都是一份完整模板，四个预设其实是 2×2 的笛卡尔积——加一个分解模型就要
为每个出图源各补一份文件，而那些文件之间只有中间一根连线不同。

现在两段各自成文件，**在加载期两两拼成一份普通 `Template`**。下游（`Render`、
提交、manifest、环境自检）完全不知道有"段"这回事，一行都没改。

```
workflows/
  src-local-realistic-v1.{json,params.json}   出图段 · 本地 SDXL 写实
  src-local-stylized-v1.{json,params.json}    出图段 · 本地 SDXL 手绘
  src-api-realistic-v1.{json,params.json}     出图段 · 云端底图 写实
  src-api-stylized-v1.{json,params.json}      出图段 · 云端底图 手绘
  dec-chord-v1.{json,params.json}             分解段 · CHORD
  dec-derive-v1.{json,params.json}            分解段 · 传统派生
```

4 × 2 = 8 条管线，其中 4 条与改造前的预设完全等价。再加一个分解模型
（Marigold IID、PBRFusion4）只需要**一个文件**，8 条变 12 条。

## 为什么拼接这么便宜

这套模板全程按 `_meta.title` 寻址：参数的 `target`、`rewire_*`、`bypass_when_zero`、
`drop_when_*`、产物声明，引用的全是标题。节点 id 只是 JSON 的 map 键和连线里的第一个
元素。所以拼接要做的只有两件事：给一段的节点重编号、把连线跟着改。那些声明一个字
都不用动。

## 声明

出图段：

```json
{
  "segment": "source",
  "domain": "realistic",
  "export": { "node": "dt.decode", "slot": 0 }
}
```

分解段：

```json
{
  "segment": "decompose",
  "expects_domain": ["realistic"],
  "imports": [
    { "node": "dt.pbr_estimate", "input": "image" },
    { "node": "dt.out.source", "input": "images" }
  ]
}
```

**`imports` 是复数**，因为底图在分解段里往往不止一个去处：CHORD 段既要喂给估计
节点、也要原样存一份底图；传统派生段则是底图**本身就是** BaseColor，同时还要抽亮度。
只接一个的话另一处会悬空，而 ComfyUI 报的错只会说某节点缺输入，看不出是拼接漏了。

### `@import`

出图段不知道对面配的是哪个分解段，所以它引用不到 `dt.pbr_estimate` 或 `dt.luminance`
这些名字。云端那条管线偏偏需要：无缝重整强度调到 0 时要把底图**直接**喂给分解链、
把整条 SDXL 支路删掉。写死任何一个名字都会让这个出图段只能配那一种分解。

于是有了占位符：

```json
"rewire_when_zero": [{ "node": "@import", "source": "dt.source_scale", "slot": 0 }]
```

拼接时它展开成分解段的每一个 import 端口。只在 `segment: "source"` 里放行——完整
模板里出现它仍然是错的。

## 拼接时会拦下什么

两类都是**不拦就会静默出错**的：

- **节点标题撞了。** 整套寻址都靠标题，撞了会让参数注入到另一段的同名节点上。
  图仍然合法，ComfyUI 照跑，只是行为完全不对。约定：出图段用 `dt.*`，分解段用
  各自独立的名字，两段之间不许重名。
- **参数键撞了且类型不同。** 前端只会渲染一个控件，另一段那个参数永远停在默认值。

同名**同类型**的参数不是错误，而是合并、`target` 取并集。分辨率就是这种：出图段拿
它定 latent 尺寸，传统派生段拿它定常量金属度图的尺寸，说的是同一个值。

## 许可要取并集

组合管线的 `licenses` 是两段的并集，判断能否商用一律走 `Meta.Commercial()`。

这里踩过一次：manifest 那边读的是单数的 `LicenseNotice`，而组合模板只填复数的
`Licenses`，于是所有走 CHORD 的组合落盘时**没有** `license_flags`——产物看上去可以
随便用。这类错误不会有任何报错，只会在某天变成法务问题。
`TestChordCombosStayNonCommercial` 现在守着它。

## 搭不上只提醒，不禁止

`domain` 与 `expects_domain` 对不上时，组合会带一条 `mismatch`，界面上以说明体
显示。手绘图对 CHORD 是分布外输入，法线会糊——但要不要试是用户的事，拦掉等于把
一条合法的探索路堵死。

## 改动这些文件之后

`internal/workflow/testdata/legacy/` 冻着改造前那四份完整模板，
`TestComposedMatchesLegacy` 会把拼出来的图和它们**逐节点逐连线**比一遍
（比的是渲染后的图，所以"多一个默认关闭的 LoRA 节点"这类差异会被 bypass 消掉）。

这是整套改造唯一的安全网。拼错一根线不会有任何报错——图仍然合法，ComfyUI 照跑，
只是出来的材质悄悄变了，而材质好不好看，人是分辨不出"是不是接错了"的。改段文件时
如果这个测试红了，先怀疑自己而不是改基准。
