# 参考图 / img2img（M4）

给一张图，在它的基础上重绘出材质。留空则是纯文生图。

## 一套模板同时覆盖两种模式

模板里同时留着两条路：

```
EmptyLatentImage ─────────────────┐
                                  ├─→ KSampler.latent_image
LoadImage → ImageScale → VAEEncode ┘
```

没传参考图时走上面那条，传了就把 `KSampler.latent_image` 改接到下面那条。
这样不必维护两份几乎一样的工作流。

机制由参数声明驱动，见 `<id>.params.json`：

```json
{
  "key": "reference", "type": "image",
  "target": "dt.reference_image.image",
  "rewire_when_set": {
    "node": "dt.sampler", "input": "latent_image",
    "source": "dt.reference_encode", "slot": 0
  },
  "drop_when_empty": ["dt.reference_image", "dt.reference_scale", "dt.reference_encode"]
}
```

- `rewire_when_set`：有值时改接一条连线。
- `drop_when_empty`：没值时删掉那条支路。留着的话 ComfyUI 会去校验一个填不出值的
  `LoadImage.image`。

还有一对判据是数值零而不是空串的版本，供数值参数使用（数值参数永远"有值"，
走不到 `drop_when_empty`）：

- `rewire_when_zero`：参数为 0 时改接**若干**条连线（数组，因为一次形态切换往往要
  同时改好几个下游）。
- `drop_when_zero`：参数为 0 时删掉这些节点。

改接一律排在删除之前，否则要找的源节点可能已经被同一个参数删掉了。云端底图的
「无缝重整调到 0 就绕开整条 SDXL 支路」就是靠这一对，见 [cloud-source.md](cloud-source.md)。

条件删除很容易漏掉一条下游引用，而 ComfyUI 的报错是 `Node ID '#9' not found`
——不说谁引用的，也不说那节点原本是什么。所以 `Render` 结束前会扫一遍悬空连线，
用模板里的标题报出来。

参考图先经 `ImageScale`（lanczos + 居中裁切）缩到出图分辨率，免得比例不一致时出怪东西。

## 上传

`POST /api/uploads`，multipart 字段名 `image`。后端转存到 ComfyUI 的 input 目录，
返回可直接填进 `LoadImage` 的文件名。

- 只收 PNG / JPG / WebP / BMP，上限 32 MB——参考图最终会被缩到 1K 附近，
  几十 MB 的原图没有意义，早点拒绝比让它穿过整条管线再失败要好。
- **文件名由后端定**（`dt_ref_<随机>.<扩展名>`），不用客户端传来的：避免路径穿越，
  也避免同名互相覆盖。
- 不落在 DreamTexture 自己的目录：ComfyUI 只认它自己 input 目录下的文件，转一手比
  让两边共享目录更少踩坑（attach 模式下 ComfyUI 甚至可能在另一台机器上）。

## 重绘幅度

`denoise` 决定离原图多远。界面上跟参考图放在一起，并把档位翻译成人话：

| 幅度 | 含义 |
|---|---|
| ≤ 0.35 | 几乎照抄原图，只做细节翻新 |
| ≤ 0.55 | 保留原图结构，换材质表现 |
| ≤ 0.75 | 借用大色调与构成，重新生成 |
| > 0.75 | 基本无视原图 |

传了参考图而 `denoise` 还停在 1.0，等于把参考图完全盖掉——所以前端在上传成功后
自动把它调到 0.6，用户不至于一脸困惑地发现参考图"没起作用"。

实测：拿一张写实红砖墙做参考、手绘管线、`denoise 0.65`，产出保留了原图的配色与
砖块布局，但形体被简化成手绘质感。

## 记录

manifest 的 `reference` 字段记下用了哪张图和当时的重绘幅度：

```json
"reference": { "file": "dt_ref_cda0a61398.png", "denoise": 0.65 }
```

`origin` 字段留给 Phase B：从 UE 里选中的 Texture 上传时标记来源。
