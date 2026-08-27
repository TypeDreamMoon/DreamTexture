# 材质套装契约 · manifest v1

**状态：2026-08-27 冻结。** 这是 Web 前端、Go 后端与 UE 插件之间唯一的产物契约。
消费端只读 manifest，不假设贴图是怎么生成的——底层换 workflow、换 PBR 估计模型，消费端都不动。

参考实现见 [example-material/](example-material/)，由 `realistic-chord-v1` 实跑产出。

## 目录结构

```
output/<material_id>/
├─ basecolor.png
├─ normal.png
├─ roughness.png
├─ metallic.png
├─ ao.png
├─ height.png
├─ orm.png          # R=AO, G=Roughness, B=Metallic
├─ preview.png      # 3×3 平铺预览，供列表页与验收
└─ manifest.json
```

`orm.png` 与分通道图并存：UE 材质用 ORM 省采样，其他 DCC 用分通道。

## 字段

| 字段 | 说明 |
|---|---|
| `schema` / `schema_version` | 固定 `dreamtexture/material-manifest` / `1`，消费端先校验再读 |
| `id` | 材质 ID，同时是目录名 |
| `name` | 展示名，默认取提示词主体 |
| `style` | `realistic` \| `stylized` |
| `workflow` | `{id, version}`，可追溯到 `workflows/<id>.json` |
| `prompt` / `negative` / `seed` / `resolution` | 生成参数，支持「参数回填再来一张」 |
| `tileable` | 是否无缝。**按次判定，不是模板上的静态值**——云端底图管线开了无缝重整才为 `true` |
| `reference` | img2img 的参考图信息；纯文生图为 `null` |
| `source` | 底图来自云端模型时的出处；全本地生成时该字段不存在。**见下** |
| `maps` | 见下 |
| `preview` | 平铺预览文件名 |
| `created_at` | RFC3339，带时区 |
| `generator` | ComfyUI 版本、底模、PBR 估计模型、节点包清单——复现和排障用。**只记这一次真的进了图的模型**，见下 |
| `license_flags` | `commercial_use` 及其原因。CHORD 为 research-only，写实管线产物标 `false` |

### generator 只记真的用到的模型

`generator.checkpoint` / `pbr_estimator` 取自**实际提交的节点图**，不是模板声明的
`model_requirements`。两者会不一致：条件接线可以整段删掉支路，云端底图调成直出时
SDXL 那一段根本不在图里，底模一次都没加载过。照抄声明会让 manifest 写上一个从未
参与的底模——排障时能把人带得很远。

（这条是端到端跑通之后翻素材详情页才发现的：一次纯直出的任务，"底模"那栏赫然写着
Juggernaut。守这条的回归测试在 `internal/workflow/loaded_models_test.go`。）

### source 条目

```json
"source": {
  "provider": "openai", "model": "gpt-image-2",
  "prompt": "texture of ... , flat even lighting, no shadows",
  "size": "1024x1024", "quality": "medium",
  "input_tokens": 52, "output_tokens": 1568,
  "cost_usd": 0.0473, "elapsed_ms": 24310,
  "flattened": true, "falloff_before": 0.912, "falloff_after": 0.981
}
```

**这个字段存在本身就是一条信息：它一出现就说明这份材质复现不了。** 云端图像接口
不支持种子，同样的提示词每次结果都不同；manifest 里的 `seed` 只作用于本地那一半
（无缝重整），单凭它拿不回同一张图。

所以没有另设一个"可否复现"的布尔量——老素材没有 `source` 字段，天然读成
"本地、可复现"，不需要迁移。消费端的判断规则就一条：**有 `source` 就别指望
用同样的参数复现它**。

`flattened` 记录有没有做过亮度场压平（消暗角），以及压平前后的边缘/中心亮度比。
详见 [cloud-source.md](cloud-source.md)。

### maps 条目

```json
"normal": {
  "file": "normal.png",
  "colorspace": "linear",
  "y": "directx",
  "width": 1024, "height": 1024,
  "bytes": 1626802,
  "sha256": "..."
}
```

- `colorspace`：`srgb` 仅 basecolor；其余全部 `linear`。UE 导入据此设 sRGB 开关。
- `y`：仅法线有。**v1 两条管线统一输出 `directx`（Y 向下，UE/Maya 惯例）**，UE 导入无需翻 G 通道。
  Three.js/Blender 等 OpenGL 系消费端自行翻转即可。
- `sha256`：取前 16 位十六进制，用于素材库去重与完整性校验。

## 法线方向的实测依据

CHORD 原生输出的是 **OpenGL (Y+)** 法线——实测方法：按 TextureAlchemy `HeightToNormal` 的定义
（OpenGL 时 `G = -grad_y`，DirectX 时 `G = +grad_y`），对 `G - 0.5` 与高度图的行方向梯度求相关：

- CHORD 原始输出：`corr = -0.9085` → OpenGL
- 经 `dt.normal_to_ue`（NormalConverter, OpenGL_to_DirectX）后：`corr = +0.9085` → DirectX

判定脚本见 `scratchpad/normal_convention.py`，换 PBR 估计模型后应重跑一次确认。

## UE 导入映射（Phase B 用）

| 贴图 | 压缩设置 | sRGB |
|---|---|---|
| basecolor | Default (BC1/BC7) | 开 |
| normal | TC_Normalmap | 关 |
| orm | Masks (TC_Masks) | 关 |
| roughness / metallic / ao / height | Grayscale 或 Masks | 关 |

法线已是 DirectX，导入时**不要**再翻 G 通道。

## 兼容性约定

- 新增字段不算破坏性变更，消费端必须忽略未知字段。
- 删除或改变已有字段语义须升 `schema_version`，且后端保留旧版本读取路径。
- `maps` 中缺某个通道是合法的（例如将来某条管线不产 height），消费端要能降级。
