# -*- coding: utf-8 -*-
"""从现有的本地工作流派生出「API 底图」版本。

改动就三件事：
  1. 把 reference_* 三个节点改名成 source_*，它们从"可选参考图"升级为"必经的底图入口"
  2. 采样器的 latent 恒定接 source_encode（底图总是存在），EmptyLatentImage 删掉
  3. 声明 rewire_when_zero / drop_when_zero：无缝重整强度为 0 时，
     把分解节点直接接到底图上，整条 SDXL 支路删掉
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

def by_title(g):
    return {n.get("_meta", {}).get("title", ""): nid for nid, n in g.items()}

def retitle(g, old, new):
    t = by_title(g)
    assert old in t, "找不到 " + old
    g[t[old]]["_meta"]["title"] = new

def link(g, title, input_name, src_title, slot=0):
    t = by_title(g)
    g[t[title]]["inputs"][input_name] = [t[src_title], slot]

def drop(g, title):
    t = by_title(g)
    if title in t:
        del g[t[title]]

def build(src_json, out_json):
    g = load(src_json)
    retitle(g, "dt.reference_image", "dt.source_image")
    retitle(g, "dt.reference_scale", "dt.source_scale")
    retitle(g, "dt.reference_encode", "dt.source_encode")
    # 底图恒定存在，采样器不再从空 latent 起步
    link(g, "dt.sampler", "latent_image", "dt.source_encode", 0)
    drop(g, "dt.latent")
    save(out_json, g)
    return g

# ---------- 写实 / CHORD ----------
g = build("realistic-chord-v1.json", "api-chord-v1.json")

SDXL_BRANCH = ["dt.checkpoint", "dt.seamless_model", "dt.seamless_vae",
               "dt.positive", "dt.negative", "dt.sampler", "dt.decode",
               "dt.source_encode"]

chord = collections.OrderedDict([
  ("id", "api-chord-v1"),
  ("version", 1),
  ("name", "API 底图 · CHORD 分解"),
  ("style", "realistic"),
  ("description",
   "底图交给云端模型出（对提示词的理解明显强于本地 SDXL），再由本地 CHORD 分解成完整 PBR 套装。"
   "云端出的图天然不无缝，可以让本地 SDXL 用循环卷积重整一道换取可平铺，也可以调到 0 直接分解。"),
  ("template", "api-chord-v1.json"),
  ("resolution", 1024),
  ("tileable", False),
  ("tileable_when_positive", "tile_fix"),
  ("source", collections.OrderedDict([
      ("kind", "api"),
      ("provider", "openai"),
      ("image_param", "source_image"),
      ("roles", collections.OrderedDict([
          ("model", "api_model"),
          ("prompt", "prompt"),
          ("size", "api_size"),
          ("quality", "api_quality"),
          ("background", "api_background"),
          ("reference", "reference"),
      ])),
  ])),
  ("license_notice", collections.OrderedDict([
      ("component", "CHORD (dt.pbr_estimate / dt.chord_model)"),
      ("license", "Ubisoft Machine Learning License (Research-Only, Copyleft)"),
      ("commercial", False),
      ("replaceable_segment", "dt.pbr_estimate"),
      ("alternatives", ["Marigold IID appearance (OpenRAIL++)", "PBRFusion4 (Apache 2.0)"]),
  ])),
  ("model_requirements", [
      collections.OrderedDict([
          ("kind", "checkpoint"), ("file", "chord_v1.safetensors"), ("dir", "checkpoints"),
          ("size_bytes", 2761211350),
          ("source", "https://huggingface.co/Ubisoft/ubisoft-laforge-chord"),
          ("download_url", "https://huggingface.co/Ubisoft/ubisoft-laforge-chord/resolve/main/chord_v1.safetensors"),
          ("auth", "hf-gated"),
          ("note", "分解必需。gated=auto：先在模型页同意协议，再在设置里填 HF 令牌即可自动下载"),
      ]),
      collections.OrderedDict([
          ("kind", "checkpoint"), ("file", "Juggernaut-XL-v9.safetensors"), ("dir", "checkpoints"),
          ("size_bytes", 7105348188),
          ("source", "https://huggingface.co/RunDiffusion/Juggernaut-XL-v9"),
          ("download_url", "https://huggingface.co/RunDiffusion/Juggernaut-XL-v9/resolve/main/Juggernaut-XL_v9_RunDiffusionPhoto_v2.safetensors"),
          ("auth", "none"),
          ("note", "只有「无缝重整」大于 0 时才用得到；调成 0 走直出，不下这个模型也能跑"),
      ]),
  ]),
  ("node_packs", ["ComfyUI-Chord", "ComfyUI-seamless-tiling", "ComfyUI-TextureAlchemy"]),
  ("params", [
    collections.OrderedDict([
      ("key", "prompt"), ("label", "提示词"), ("type", "string"), ("multiline", True),
      ("target", "dt.positive.text"),
      ("prefix", "texture of "),
      ("suffix", ", top down view, seamless tileable, orthographic, flat even lighting, no shadows"),
      ("default", "large old broken rocks with moss"),
      ("note", "前后缀会一并送给云端模型。CHORD 对平光输入敏感，平光约束不要去掉"),
    ]),
    collections.OrderedDict([
      ("key", "api_model"), ("label", "云端模型"), ("type", "string"),
      ("widget", "imagen-model"), ("default", ""),
      ("note", "可用模型按你的 API 令牌实际权限列出"),
    ]),
    collections.OrderedDict([
      ("key", "api_quality"), ("label", "画质档"), ("type", "enum"),
      ("options", ["low", "medium", "high"]), ("default", "medium"),
      ("note", "直接决定单张花费与耗时"),
    ]),
    collections.OrderedDict([
      ("key", "tile_fix"), ("label", "无缝重整"), ("type", "float"),
      ("min", 0), ("max", 0.85), ("target", "dt.sampler.denoise"), ("default", 0.45),
      ("rewire_when_zero", [
        collections.OrderedDict([("node","dt.pbr_estimate"),("input","image"),("source","dt.source_scale"),("slot",0)]),
        collections.OrderedDict([("node","dt.out.source"),("input","images"),("source","dt.source_scale"),("slot",0)]),
      ]),
      ("drop_when_zero", SDXL_BRANCH),
      ("note", "云端出的图不保证无缝。本地 SDXL 用循环卷积重画一道可以让它可平铺，"
               "数值越大越无缝、也越偏离原图；0.35~0.55 通常够用。调成 0 则直接分解，"
               "不经过本地 SDXL，也就不需要下底模"),
    ]),
    collections.OrderedDict([
      ("key", "reference"), ("label", "参考图"), ("type", "image"),
      ("note", "留空为纯文生图。给了参考图就走云端的图像编辑接口，在它基础上改"),
    ]),
    collections.OrderedDict([
      ("key", "source_image"), ("label", "底图"), ("type", "image"),
      ("target", "dt.source_image.image"), ("hidden", True),
      ("note", "由后端填：云端出的图上传到 ComfyUI 之后的文件名"),
    ]),
  ]),
  ("advanced", [
    collections.OrderedDict([
      ("key", "api_size"), ("label", "云端出图尺寸"), ("type", "enum"),
      ("options", ["1024x1024", "1536x1024", "1024x1536"]), ("default", "1024x1024"),
      ("note", "纹理基本只用正方形；非正方形会在缩放到工作分辨率时被拉伸"),
    ]),
    collections.OrderedDict([
      ("key", "api_background"), ("label", "背景"), ("type", "enum"),
      ("options", ["opaque", "auto", "transparent"]), ("default", "opaque"),
      ("note", "transparent 用于贴花类素材；普通纹理保持 opaque"),
    ]),
    collections.OrderedDict([
      ("key", "resolution"), ("label", "工作分辨率"), ("type", "enum"), ("options", [768, 1024]),
      ("target", ["dt.source_scale.width", "dt.source_scale.height"]), ("default", 1024),
      ("note", "云端出的图会先缩放到这个尺寸。CHORD 最佳工作分辨率为 1024"),
    ]),
    collections.OrderedDict([
      ("key", "negative"), ("label", "负面词"), ("type", "string"), ("multiline", True),
      ("target", "dt.negative.text"),
      ("default", "perspective, vignette, cast shadow, specular highlight, blurry, watermark, text, border"),
      ("note", "只作用于本地的无缝重整那一道；云端接口不收负面词"),
    ]),
    collections.OrderedDict([
      ("key", "seed"), ("label", "种子"), ("type", "int"), ("target", "dt.sampler.seed"), ("default", -1),
      ("note", "只影响本地重整。云端接口不支持种子，同一提示词每次结果都不同"),
    ]),
    collections.OrderedDict([("key","steps"),("label","步数"),("type","int"),("min",10),("max",60),("target","dt.sampler.steps"),("default",30)]),
    collections.OrderedDict([("key","cfg"),("label","CFG"),("type","float"),("min",1),("max",15),("target","dt.sampler.cfg"),("default",6)]),
    collections.OrderedDict([("key","sampler"),("label","采样器"),("type","string"),("target","dt.sampler.sampler_name"),("default","dpmpp_2m")]),
    collections.OrderedDict([("key","scheduler"),("label","调度器"),("type","string"),("target","dt.sampler.scheduler"),("default","karras")]),
    collections.OrderedDict([("key","ao_radius"),("label","AO 半径"),("type","int"),("min",4),("max",64),("target","dt.derive_ao.radius"),("default",36)]),
    collections.OrderedDict([("key","ao_strength"),("label","AO 强度"),("type","float"),("min",0),("max",3),("target","dt.derive_ao.strength"),("default",2.2)]),
  ]),
  ("outputs", collections.OrderedDict([
    ("source",    {"node":"dt.out.source","colorspace":"srgb","role":"reference",
                   "note":"送进分解之前的底图。无缝重整为 0 时即云端原图，否则是重整后的版本"}),
    ("basecolor", {"node":"dt.out.basecolor","colorspace":"srgb"}),
    ("normal",    {"node":"dt.out.normal","colorspace":"linear","y":"directx",
                   "note":"CHORD 原生输出 OpenGL(Y+)，工作流内经 dt.normal_to_ue 转成 DirectX"}),
    ("roughness", {"node":"dt.out.roughness","colorspace":"linear"}),
    ("metallic",  {"node":"dt.out.metallic","colorspace":"linear"}),
    ("ao",        {"node":"dt.out.ao","colorspace":"linear"}),
    ("height",    {"node":"dt.out.height","colorspace":"linear"}),
    ("orm",       {"node":"dt.out.orm","colorspace":"linear","packing":"R=ao,G=roughness,B=metallic"}),
  ])),
])
save("api-chord-v1.params.json", chord)
print("写出 api-chord-v1（%d 节点）" % len(g))

# ---------- 手绘 / 派生 ----------
g2 = build("stylized-derive-v1.json", "api-derive-v1.json")

base = load("stylized-derive-v1.params.json")
derive = collections.OrderedDict(chord)  # 复用大部分声明，再逐项替换
derive["id"] = "api-derive-v1"
derive["name"] = "API 底图 · 手绘派生"
derive["style"] = "stylized"
derive["description"] = (
    "底图交给云端模型出，再用传统亮度推导得到法线/高度/AO/粗糙度。"
    "适合手绘、卡通、风格化材质——这类材质本来就没有物理正确的分解可言，推导反而更可控。")
derive["template"] = "api-derive-v1.json"
derive.pop("license_notice", None)
derive["model_requirements"] = [
    collections.OrderedDict([
        ("kind", "checkpoint"), ("file", "Juggernaut-XL-v9.safetensors"), ("dir", "checkpoints"),
        ("size_bytes", 7105348188),
        ("source", "https://huggingface.co/RunDiffusion/Juggernaut-XL-v9"),
        ("download_url", "https://huggingface.co/RunDiffusion/Juggernaut-XL-v9/resolve/main/Juggernaut-XL_v9_RunDiffusionPhoto_v2.safetensors"),
        ("auth", "none"),
        ("note", "只有「无缝重整」大于 0 时才用得到；调成 0 走直出，不下这个模型也能跑"),
    ]),
]
for r in base.get("model_requirements", []):
    if "lora" in r.get("kind", "").lower():
        r = collections.OrderedDict(r)
        r["note"] = "只在无缝重整大于 0 且风格强度大于 0 时用到"
        derive["model_requirements"].append(r)
derive["node_packs"] = base["node_packs"]

def find(lst, key):
    for p in lst:
        if p["key"] == key:
            return p
    return None

# 提示词换成手绘管线的前后缀与默认值
bp = find(base["params"], "prompt")
p = collections.OrderedDict(find(derive["params"], "prompt"))
p["prefix"], p["suffix"], p["default"] = bp.get("prefix",""), bp.get("suffix",""), bp["default"]
p["note"] = "前后缀会一并送给云端模型"
derive["params"] = [p] + [x for x in derive["params"] if x["key"] != "prompt"]

# 派生管线里 SDXL 支路多一个 LoRA 节点，分解入口也不同
tf = find(derive["params"], "tile_fix")
tf["rewire_when_zero"] = [
    collections.OrderedDict([("node","dt.out.basecolor"),("input","images"),("source","dt.source_scale"),("slot",0)]),
    collections.OrderedDict([("node","dt.luminance"),("input","color"),("source","dt.source_scale"),("slot",0)]),
]
tf["drop_when_zero"] = ["dt.checkpoint", "dt.style_lora", "dt.seamless_model", "dt.seamless_vae",
                        "dt.positive", "dt.negative", "dt.sampler", "dt.decode", "dt.source_encode"]

adv = [x for x in derive["advanced"] if x["key"] not in ("ao_radius", "ao_strength")]
r = find(adv, "resolution")
r["note"] = "云端出的图会先缩放到这个尺寸"
r["options"] = [768, 1024]
derive["advanced"] = adv + [x for x in base["advanced"]
                            if x["key"] in ("style_strength","height_amplify","normal_strength",
                                            "ao_radius","ao_strength","roughness_floor","roughness_ceiling")]
derive["outputs"] = base["outputs"]
save("api-derive-v1.params.json", derive)
print("写出 api-derive-v1（%d 节点）" % len(g2))
