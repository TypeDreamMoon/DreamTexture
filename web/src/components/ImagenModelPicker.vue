<script setup lang="ts">
import { computed, h, onMounted, watch } from 'vue'
import { NSelect, NSpin } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { RouterLink } from 'vue-router'
import { imagenProviders, imagenError, loadImagen, modelsOf, providerOf } from '../store'
import type { ImagenModel } from '../api/types'

const props = defineProps<{ provider: string; modelValue: unknown }>()
const emit = defineEmits<{ 'update:modelValue': [unknown] }>()

onMounted(() => loadImagen())

const prov = computed(() => providerOf(props.provider))
const models = computed(() => modelsOf(props.provider))
const loading = computed(() => imagenProviders.value === null)

const options = computed(() =>
  models.value.map((m) => ({ label: m.label, value: m.id, note: m.note, known: m.known })),
)

// 下拉项自己渲染，好在名字后面挂一行说明——各模型的速度与花费差着一个量级，
// 光看 "GPT Image 2 / 1.5 / 1 mini" 这几个名字选不出来。
//
// 类名用全局的 dt-opt-*：下拉菜单被 teleport 到 body 之外，组件的 scoped
// 样式（哪怕 :deep）都够不着它。
function renderLabel(opt: SelectOption) {
  const note = opt.known === false ? '未收录，参数按同族推定' : (opt.note as string) || ''
  return h('div', { class: 'dt-opt' }, [
    h('span', { class: 'dt-opt-name' }, String(opt.label)),
    note
      ? h('span', { class: opt.known === false ? 'dt-opt-tag dt-opt-tag-warn' : 'dt-opt-note' }, note)
      : null,
  ])
}

const value = computed({
  get: () => String(props.modelValue ?? ''),
  set: (v: string) => emit('update:modelValue', v),
})

// 模型清单是异步到位的，而参数默认值是空串——清单一来就替用户选上第一个
// （已收录的排在前面，所以第一个是当前推荐的那个）。用户自己选过就不再动。
watch(
  models,
  (list) => {
    if (!list.length) return
    if (list.some((m) => m.id === value.value)) return
    value.value = list[0]!.id
  },
  { immediate: true },
)

const current = computed<ImagenModel | undefined>(() =>
  models.value.find((m) => m.id === value.value),
)
</script>

<template>
  <div class="picker">
    <div v-if="loading" class="loading dt-faint">
      <NSpin :size="12" /> 正在查可用模型…
    </div>

    <template v-else-if="!prov?.configured">
      <p class="notice">
        还没填 {{ prov?.label ?? provider }} 的 API Key。
        <RouterLink to="/models" class="link">去设置</RouterLink>
        填上之后才能用云端底图。
      </p>
    </template>

    <template v-else-if="prov?.error || imagenError">
      <p class="notice">
        连不上 {{ prov?.label ?? provider }}：{{ prov?.error || imagenError }}
      </p>
    </template>

    <template v-else>
      <NSelect
        v-model:value="value"
        :options="options"
        :render-label="renderLabel"
        :consistent-menu-width="false"
      />
      <p v-if="current && !current.known" class="tiny warn">
        这个模型不在收录表里，尺寸与画质选项按 gpt-image 系列推定，可能不完全适用。
      </p>
    </template>
  </div>
</template>

<style scoped>
.picker {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.loading {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--dt-fs-sm);
  height: 34px;
}
.notice {
  margin: 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.65;
  color: var(--dt-warn);
  border-left: 2px solid var(--dt-warn);
  padding-left: 9px;
}
.link {
  color: var(--dt-accent);
}
/* 下拉项的样式在全局表里（style.css 的 .dt-opt-*）——菜单被 teleport
   到 body 之外，scoped 样式够不着 */
.tiny {
  margin: 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.5;
}
.warn {
  color: var(--dt-warn);
}
</style>
