<script setup lang="ts">
import { computed, ref } from 'vue'

// 平铺预览不是装饰，是验收工具：无缝是硬指标，而接缝只有拼起来才看得见。
const props = defineProps<{ src: string; alt?: string }>()

const tiles = ref(2)
const zoom = ref(false)
const TILE_OPTIONS = [1, 2, 3, 4]

const style = computed(() => ({
  backgroundImage: `url(${props.src})`,
  backgroundSize: `${100 / tiles.value}% ${100 / tiles.value}%`,
  backgroundRepeat: 'repeat',
  // 放大到 1:1 看像素时关掉插值，才能看清实际的噪点与压缩痕迹。
  imageRendering: zoom.value ? ('pixelated' as const) : ('auto' as const),
}))
</script>

<template>
  <div class="viewer">
    <!-- 容器保持正方形，但按视口高度收敛：宽列下 aspect-ratio:1 会撑出
         上千像素高，必须滚动才看得全，反而没法整体判断接缝。 -->
    <div class="frame">
      <div class="canvas dt-swatch" :style="style" role="img" :aria-label="alt ?? ''" />
    </div>

    <div class="controls">
      <span class="dt-label">平铺</span>
      <button
        v-for="n in TILE_OPTIONS"
        :key="n"
        class="chip dt-mono"
        :class="{ on: tiles === n }"
        @click="tiles = n"
      >
        {{ n }}×{{ n }}
      </button>
      <button class="chip" :class="{ on: zoom }" @click="zoom = !zoom">
        {{ zoom ? '平滑' : '像素' }}
      </button>
      <span class="dt-faint tip">拼接后看不到接缝即为无缝</span>
    </div>
  </div>
</template>

<style scoped>
.viewer {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.frame {
  display: flex;
  justify-content: center;
}
.canvas {
  width: min(100%, clamp(320px, 52vh, 560px));
  aspect-ratio: 1;
  border: 1px solid var(--dt-border);
  border-radius: var(--dt-radius-card);
}
.controls {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.controls .dt-label {
  margin-right: 2px;
}
.chip {
  font: inherit;
  font-size: var(--dt-fs-sm);
  padding: 4px 10px;
  border-radius: var(--dt-radius);
  color: var(--dt-ink-muted);
  background: transparent;
  border: 1px solid var(--dt-border);
  cursor: pointer;
  transition:
    color 0.18s ease,
    border-color 0.18s ease,
    background 0.18s ease;
}
.chip:hover {
  color: var(--dt-ink);
  border-color: var(--dt-border-strong);
}
.chip.on {
  color: var(--dt-accent);
  border-color: var(--dt-accent);
  background: var(--dt-accent-soft);
}
.tip {
  font-size: var(--dt-fs-xs);
  margin-left: auto;
}
</style>
