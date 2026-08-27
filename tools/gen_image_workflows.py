# -*- coding: utf-8 -*-
"""生成两条「图片」管线的模板。

  cloud-image-v1  纯云端直出，没有节点图
  local-image-v1  本地 SDXL 出图，从写实材质管线里裁出采样那一段

本地那条是**裁**出来的而不是新写的：节点各自的输入名从既有模板里原样带走，
比照着文档手写可靠——写错一个输入名要等到点生成才报错。
"""
import json, io, os, collections

WF = r"G:\Workflow\DreamTexture\workflows"

def load(p):
    with io.open(os.path.join(WF, p), encoding="utf-8") as f:
        return json.load(f, object_pairs_hook=collections.OrderedDict)

def save(p, obj):
    with io.open(os.path.join(WF, p), "w", encoding="utf-8", newline="\n") as f:
        json.dump(obj, f, ensure_ascii=False, indent=2)
        f.write("\n")

# ───────────────────────── 本地出图的节点图 ─────────────────────────
src = load("realistic-chord-v1.json")
by_title = {n.get("_meta", {}).get("title", ""): nid for nid, n in src.items()}

# 只留文生图/图生图这一段，PBR 分解整条丢掉
keep = ["dt.checkpoint", "dt.positive", "dt.negative", "dt.latent",
        "dt.sampler", "dt.decode",
        "dt.reference_image", "dt.reference_scale", "dt.reference_encode"]
g = collections.OrderedDict()
for t in keep:
    nid = by_title[t]
    g[nid] = src[nid]

# 采样器原来接的是 seamless_model（循环卷积），普通出图不要无缝，接回底模
g[by_title["dt.sampler"]]["inputs"]["model"] = [by_title["dt.checkpoint"], 0]
# VAE 也从 seamless_vae 换回底模自带的
g[by_title["dt.decode"]]["inputs"]["vae"] = [by_title["dt.checkpoint"], 2]
g[by_title["dt.reference_encode"]]["inputs"]["vae"] = [by_title["dt.checkpoint"], 2]

# 输出节点：从原来的 dt.out.source 改名，只留这一路
out_id = by_title["dt.out.source"]
g[out_id] = src[out_id]
g[out_id]["_meta"]["title"] = "dt.out.image"
g[out_id]["inputs"]["images"] = [by_title["dt.decode"], 0]

save("local-image-v1.json", g)
print("写出 local-image-v1.json（%d 节点）" % len(g))

# ───────────────────────── 参数声明 ─────────────────────────
# 图片的尺寸档比材质多：材质基本只用正方形，图片要竖构图和横构图
IMG_SIZES = ["1024x1024", "1024x1536", "1536x1024",
             "2048x2048", "1152x2048", "2048x1152", "2880x2880", "2160x3840", "3840x2160"]

cloud = collections.OrderedDict([
  ("id", "cloud-image-v1"),
  ("kind", "image"),
  ("version", 1),
  ("name", "云端出图"),
  ("style", "image"),
  ("description",
   "直接用云端模型出一张图，不做 PBR 分解、不走本地 ComfyUI。"
   "适合出概念图、参考图，或者需要图内文字与符号的场合。"),
  ("resolution", 1024),
  ("tileable", False),
  ("source", collections.OrderedDict([
      ("kind", "api"),
      ("provider", "openai"),
      ("direct_output", True),
      ("roles", collections.OrderedDict([
          ("model", "api_model"),
          ("prompt", "prompt"),
          ("size", "api_size"),
          ("quality", "api_quality"),
          ("background", "api_background"),
          ("reference", "reference"),
      ])),
  ])),
  ("model_requirements", []),
  ("node_packs", []),
  ("params", [
    collections.OrderedDict([
      ("key", "prompt"), ("label", "提示词"), ("type", "string"), ("multiline", True),
      ("default", ""),
      ("note", "普通出图不加任何固定前后缀——材质管线那套平光约束是给 PBR 分解准备的，"
               "用在这儿只会限制画面"),
    ]),
    collections.OrderedDict([
      ("key", "api_model"), ("label", "模型"), ("type", "string"),
      ("widget", "imagen-model"), ("default", ""),
    ]),
    collections.OrderedDict([
      ("key", "api_size"), ("label", "尺寸"), ("type", "enum"),
      ("options", IMG_SIZES), ("default", "1024x1024"),
      ("note", "正方形最大 2880²；更大只能走非正方形（3840x2160）"),
    ]),
    collections.OrderedDict([
      ("key", "api_quality"), ("label", "画质档"), ("type", "enum"),
      ("options", ["low", "medium", "high"]), ("default", "medium"),
      ("note", "直接决定花费与耗时。high 档单张要两三分钟"),
    ]),
    collections.OrderedDict([
      ("key", "reference"), ("label", "参考图"), ("type", "image"),
      ("note", "给了就走图像编辑接口，在它基础上改"),
    ]),
  ]),
  ("advanced", [
    collections.OrderedDict([
      ("key", "api_background"), ("label", "背景"), ("type", "enum"),
      ("options", ["auto", "opaque", "transparent"]), ("default", "auto"),
      ("note", "transparent 需要模型支持，且只在 PNG/WebP 下有效"),
    ]),
  ]),
  ("outputs", collections.OrderedDict()),
])
save("cloud-image-v1.params.json", cloud)
print("写出 cloud-image-v1.params.json")

local = collections.OrderedDict([
  ("id", "local-image-v1"),
  ("kind", "image"),
  ("version", 1),
  ("name", "本地出图"),
  ("style", "image"),
  ("description",
   "用本机 SDXL 出一张图，不做 PBR 分解也不做无缝。不花钱、没有内容审核、"
   "种子可复现，代价是提示词遵循度不如云端模型。"),
  ("template", "local-image-v1.json"),
  ("resolution", 1024),
  ("tileable", False),
  ("model_requirements", [
      collections.OrderedDict([
          ("kind", "checkpoint"), ("file", "Juggernaut-XL-v9.safetensors"), ("dir", "checkpoints"),
          ("size_bytes", 7105348188),
          ("source", "https://huggingface.co/RunDiffusion/Juggernaut-XL-v9"),
          ("download_url", "https://huggingface.co/RunDiffusion/Juggernaut-XL-v9/resolve/main/Juggernaut-XL_v9_RunDiffusionPhoto_v2.safetensors"),
          ("auth", "none"),
      ]),
  ]),
  ("node_packs", []),
  ("params", [
    collections.OrderedDict([
      ("key", "prompt"), ("label", "提示词"), ("type", "string"), ("multiline", True),
      ("target", "dt.positive.text"), ("default", ""),
    ]),
    collections.OrderedDict([
      ("key", "negative"), ("label", "负面词"), ("type", "string"), ("multiline", True),
      ("target", "dt.negative.text"),
      ("default", "lowres, bad anatomy, worst quality, watermark, text, signature"),
    ]),
    collections.OrderedDict([
      ("key", "width"), ("label", "宽"), ("type", "enum"),
      ("options", [768, 896, 1024, 1152, 1344]),
      ("target", "dt.latent.width"), ("default", 1024),
    ]),
    collections.OrderedDict([
      ("key", "height"), ("label", "高"), ("type", "enum"),
      ("options", [768, 896, 1024, 1152, 1344]),
      ("target", "dt.latent.height"), ("default", 1024),
      ("note", "SDXL 在总像素接近 1024² 时最稳；偏离太远容易出双头、重复构图"),
    ]),
    collections.OrderedDict([
      ("key", "seed"), ("label", "种子"), ("type", "int"),
      ("target", "dt.sampler.seed"), ("default", -1),
      ("note", "-1 为随机。本地出图的种子是真能复现的，和云端不一样"),
    ]),
    collections.OrderedDict([
      ("key", "reference"), ("label", "参考图"), ("type", "image"),
      ("target", "dt.reference_image.image"),
      ("rewire_when_set", collections.OrderedDict([
          ("node", "dt.sampler"), ("input", "latent_image"),
          ("source", "dt.reference_encode"), ("slot", 0)])),
      ("drop_when_empty", ["dt.reference_image", "dt.reference_scale", "dt.reference_encode"]),
      ("note", "留空为纯文生图。给了就在它基础上重绘，幅度在高级参数里调"),
    ]),
  ]),
  ("advanced", [
    collections.OrderedDict([("key","steps"),("label","步数"),("type","int"),("min",10),("max",60),
                             ("target","dt.sampler.steps"),("default",28)]),
    collections.OrderedDict([("key","cfg"),("label","CFG"),("type","float"),("min",1),("max",15),
                             ("target","dt.sampler.cfg"),("default",6.5)]),
    collections.OrderedDict([("key","denoise"),("label","重绘幅度"),("type","float"),("min",0.1),("max",1.0),
                             ("target","dt.sampler.denoise"),("default",1.0),
                             ("note","仅在给了参考图时有意义")]),
    collections.OrderedDict([("key","reference_size"),("label","参考图缩放"),("type","enum"),
                             ("options",[768,1024,1152]),
                             ("target",["dt.reference_scale.width","dt.reference_scale.height"]),
                             ("default",1024)]),
    collections.OrderedDict([("key","sampler"),("label","采样器"),("type","string"),
                             ("target","dt.sampler.sampler_name"),("default","dpmpp_2m")]),
    collections.OrderedDict([("key","scheduler"),("label","调度器"),("type","string"),
                             ("target","dt.sampler.scheduler"),("default","karras")]),
  ]),
  ("outputs", collections.OrderedDict([
    ("image", {"node": "dt.out.image", "colorspace": "srgb"}),
  ])),
])
save("local-image-v1.params.json", local)
print("写出 local-image-v1.params.json")
