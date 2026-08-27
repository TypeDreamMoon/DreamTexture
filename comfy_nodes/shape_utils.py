"""张量形状归一化。

存在的理由：ComfyUI 的 IMAGE 约定是 [B, H, W, C]，但个别上游节点会返回
[B, H, W]。ComfyUI-Chord 的 ChordNormalToHeight 就是一例——它内部做
`normal_to_height(...)[None, None].squeeze(1)`，产出 3 维张量却声明为
IMAGE 输出。SaveImage 恰好能容忍（当灰度图存），所以官方示例工作流没暴露，
但只要把它接进任何按 4 维解包的节点（如 AOApproximator 的
`batch, h, w, channels = height.shape`）就会 ValueError。

上游修好之前，用这个节点在两者之间过一道。
"""

class DTEnsureImageShape:
    @classmethod
    def INPUT_TYPES(cls):
        return {
            "required": {
                "image": ("IMAGE",),
                "channels": (
                    ["keep", "1", "3"],
                    {
                        "default": "3",
                        "tooltip": "keep=只补齐维度不改通道数；1=单通道；3=复制成 RGB",
                    },
                ),
            }
        }

    RETURN_TYPES = ("IMAGE",)
    RETURN_NAMES = ("image",)
    FUNCTION = "fix"
    CATEGORY = "DreamTexture/Utils"
    DESCRIPTION = "把 [B,H,W] 或 [H,W] 的张量补成 ComfyUI 约定的 [B,H,W,C]，并按需扩展通道数。"

    def fix(self, image, channels):
        x = image
        if x.dim() == 2:                      # [H, W]
            x = x[None, ..., None]
        elif x.dim() == 3:                    # [B, H, W] —— 缺通道维
            x = x[..., None]
        elif x.dim() != 4:
            raise ValueError(
                "DT_EnsureImageShape: 不支持 %d 维张量 %s" % (x.dim(), tuple(x.shape))
            )

        if channels == "3" and x.shape[-1] == 1:
            x = x.repeat(1, 1, 1, 3)
        elif channels == "1" and x.shape[-1] > 1:
            x = x[..., :1]

        return (x.contiguous(),)


NODE_CLASS_MAPPINGS = {"DT_EnsureImageShape": DTEnsureImageShape}
NODE_DISPLAY_NAME_MAPPINGS = {"DT_EnsureImageShape": "DT - 归一化图像形状"}
