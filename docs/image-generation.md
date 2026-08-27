# 图片生成（M6）

在材质之外，再产出**单张图片**。同一个生成台，顶上分「材质 / 图片」两档。

两档分开而不是混成一个预设列表，是因为两者的参数、产物、去处都不一样：
材质出的是一整套 PBR 通道、进 `output/<id>/`、有 manifest；图片出的是一张 PNG、
进 `output/images/`、配一份同名 json。混在一起用户得先认出哪个预设产出什么。

## 两条管线

| 预设 | 底图来自 | 走不走 ComfyUI | 可复现 |
|---|---|---|---|
| `local-image-v1` 本地出图 | 本机 SDXL | 走 | 是（种子） |
| `cloud-image-v1` 云端出图 | OpenAI 图像接口 | **不走** | 否 |

### 云端出图不经过 ComfyUI

模板里 `source.direct_output: true` 的，拿回来就是成品，整条链路上没有
ComfyUI 的事：

```go
if tpl.Meta.Direct() {
    return r.executeDirect(ctx, j, tpl)   // 在 waitComfyAvailable 之前
}
```

这一点很要紧——否则"想用云端出张图"就得先有一个装好的 ComfyUI，
而那正是新用户还没有的东西。

直出也**不做亮度场压平**（见 [cloud-source.md](cloud-source.md)）：压平是给
"要拿去平铺"的底图准备的，单张图片按原样保留模型给的明暗才对。

### 直出模板没有节点图

于是任何"先渲染一遍"的代码都得先分流。踩过一次：`Submit` 一律拿 `Render`
校验参数，而 `Render` 见到直出模板就报"没有节点图可渲染"——云端出图在提交
那一步就死了，一次 API 都没发出去。现在是：

```go
func validateParams(tpl *workflow.Template, params map[string]any) error {
    if tpl.Meta.Direct() {
        _, err := tpl.Resolve(params)   // 只验参数
        return err
    }
    _, err := tpl.Render(params, "probe")
    return err
}
```

`internal/job/validate_test.go` 盯着这条路。

## 落盘格式

`output/images/<id>.png` 与 `output/images/<id>.json`，同名并排。

json 是 `dreamtexture/picture` schema，字段见 `internal/picture/picture.go`。
和材质的 manifest 一样，`source` 字段一旦存在就说明**这张图复现不了**：
云端图像接口不支持种子，`seed` 只对本地那一半有意义，所以云端出的图
`seed` 直接写 0，而不是写一个看着像能复现的数字。

## 产物类型按工作流判定，不按界面档位

任务列表是不分档的。切到"材质"档时，之前那些图片任务还在列表里——
卡片如果跟着当前档位渲染，就会把它们全渲染成材质，点进去是个不存在的详情页。
所以 `JobCard` 自己按 `job.workflow_id` 去工作流表里查 `kind`。

---

# 提示词扩写

生成台的提示词框旁边有「让模型扩写」。走对话补全接口，把一句话铺开成
适合图像模型的提示词。

## 两套系统提示，不能共用

材质的提示词必须服从「正交俯视、平光无影、可平铺」这些硬约束——那是 PBR 分解
能不能 work 的前提，扩写时把这些丢了，出来的图再好看也没法用。
而普通出图恰恰相反，硬塞平光约束等于把画面限死。

所以 `Purpose` 分 `texture` / `image` 两套（`internal/imagen/refine.go`）。

## 扩写可以走另一个网关

**出图网关不一定提供对话接口。** 实测遇到的情况：某个只做图像的中转，
`/models` 和 `/images/generations` 都正常，`/chat/completions` 返回的却是
它自己的前端页面——还带着 HTTP 200。

所以设置页里「文本模型（提示词扩写）」是独立一项，有自己的令牌与地址；
两项都留空就沿用上面那套 OpenAI 的。保存地址时会真扩写一次去探通不通，
只花几十个 token，换来"填完立刻知道对不对"。

对应地，响应体解析不了时要把实际收到的东西摆出来。光报
`invalid character '<'` 等于什么都没说：

```
对话接口 返回的不是 JSON（HTTP 200, text/html）：<!doctype html> <html lang="zh-CN"> …
收到的是一个网页而不是接口响应。多半是接口地址填得不对，或者这个网关不提供该接口
```

## 结果不自动覆盖输入框

扩写完让用户自己看一眼再决定用不用。模型有时会自作主张加东西，
直接盖掉输入框会让人莫名其妙地丢掉自己写的要求。

---

# 参考图库

`output/refs/`，一张图一条记录。解决的是"上次那张参考图我放哪儿了"。

## 三个入口

- 生成台的参考图区域点「图库」，从已存的里挑
- 图片详情页点「存为参考图」，把刚生成的图收进去
- 直接拖文件进参考图区域（这一条不入库，是一次性的）

## `use` 才上传到 ComfyUI

`POST /api/refs/{id}/use` 返回 ComfyUI 侧的文件名，并把它记进
`refs.comfy_name`。分成两步是因为 ComfyUI 可能重启、input 目录可能被清，
入库那一刻传上去的东西不保证到用时还在。

云端出图用参考图时走的是图生图接口，本地不参与——两条路的
"参考图"是同一个概念，但落地方式完全不同。
