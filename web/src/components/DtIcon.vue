<script setup lang="ts">
// 一小组描边图标。内联 SVG 而不是引图标库：整个应用只用到十来个，
// 为此拖进一个几百 KB 的包不值当，而且内联的能直接跟着 currentColor 变色。
const props = withDefaults(defineProps<{ name: string; size?: number }>(), { size: 18 })

const PATHS: Record<string, string> = {
  key: 'M14 7a4 4 0 1 1-3.9 5H8v2H6v2H3v-3l5.1-5.1A4 4 0 0 1 14 7zm1.5 2.5h.01',
  cloud: 'M6.5 18a4 4 0 0 1-.4-8A5.5 5.5 0 0 1 17 10.2 3.9 3.9 0 0 1 16.5 18h-10z',
  globe: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zm0 0c-2.5 2.4-3.8 5.4-3.8 9s1.3 6.6 3.8 9m0-18c2.5 2.4 3.8 5.4 3.8 9s-1.3 6.6-3.8 9M3.4 9h17.2M3.4 15h17.2',
  chip: 'M7 7h10v10H7zM9.5 3v4M14.5 3v4M9.5 17v4M14.5 17v4M3 9.5h4M3 14.5h4M17 9.5h4M17 14.5h4',
  folder: 'M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z',
  download: 'M12 3v12m0 0 4.5-4.5M12 15l-4.5-4.5M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2',
  terminal: 'M4 5h16v14H4zM7.5 9.5l3 2.5-3 2.5M13 15h4',
  sliders: 'M4 7h9M17 7h3M4 17h4M12 17h8M15 4v6M8 14v6',
  image: 'M4 5h16v14H4zM4 15l4.5-4.5 3.5 3.5 3-3L20 15M9 9.5h.01',
  gauge: 'M12 20a8 8 0 1 1 8-8M12 12l4.5-3.5',
  shield: 'M12 3l7 3v5c0 4.5-3 8-7 10-4-2-7-5.5-7-10V6l7-3z',
  wrench: 'M14.5 4a4.5 4.5 0 0 0-4 6.6L4 17.1V20h2.9l6.5-6.5A4.5 4.5 0 1 0 14.5 4z',

  // 导航用
  wand: 'M4 20l9.5-9.5M15 4l.9 2.1L18 7l-2.1.9L15 10l-.9-2.1L12 7l2.1-.9zM19 13l.6 1.4 1.4.6-1.4.6-.6 1.4-.6-1.4-1.4-.6 1.4-.6z',
  grid: 'M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h6v6h-6z',
  flow: 'M4 5h5v4H4zM15 4h5v4h-5zM15 16h5v4h-5zM9 7h3a2 2 0 0 1 2 2v9M9 7h6',
  layers: 'M12 3l8 4.5-8 4.5-8-4.5zM4 12l8 4.5 8-4.5M4 16.5L12 21l8-4.5',
  panelLeft: 'M4 5h16v14H4zM9.5 5v14',
}
</script>

<template>
  <svg
    class="dt-icon"
    :width="props.size"
    :height="props.size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.5"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path :d="PATHS[props.name] ?? PATHS.sliders" />
  </svg>
</template>

<style scoped>
.dt-icon {
  flex: none;
  display: block;
}
</style>
