"""DreamTexture 胶水节点包。

只放"把上游节点接起来"所必需的最小逻辑，不做效果性图像处理——
效果性处理一律用生态里现成的节点（TextureAlchemy 等）。
"""

from .shape_utils import NODE_CLASS_MAPPINGS, NODE_DISPLAY_NAME_MAPPINGS

__all__ = ["NODE_CLASS_MAPPINGS", "NODE_DISPLAY_NAME_MAPPINGS"]
