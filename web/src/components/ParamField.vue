<script setup lang="ts">
import { computed } from 'vue'
import { NInput, NInputNumber, NSelect, NSwitch, NSlider, NTooltip } from 'naive-ui'
import ImagenModelPicker from './ImagenModelPicker.vue'
import type { Param } from '../api/types'

// 参数控件完全由后端的 params.json 声明驱动。前端不认识任何具体参数名——
// 工作流加一个可调项，这里自动就多一个控件，不用改前端。
//
// widget 是这条规则唯一的例外，留给"选项要运行时才知道"的参数：云端模型
// 清单得现问服务端，没法写进静态的 params.json。provider 由调用方传进来，
// 同样来自后端声明，前端仍然不认识任何具体来源名。
const props = defineProps<{ param: Param; modelValue: unknown; provider?: string }>()
const emit = defineEmits<{ 'update:modelValue': [unknown] }>()

const value = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const options = computed(() =>
  (props.param.options ?? []).map((o) => ({ label: String(o), value: o as string | number })),
)

// 有上下界的数值给滑条，更好调；无界的给数字框。
const isRanged = computed(
  () =>
    (props.param.type === 'float' || props.param.type === 'int') &&
    props.param.min !== undefined &&
    props.param.max !== undefined,
)

const step = computed(() => (props.param.type === 'int' ? 1 : 0.05))
</script>

<template>
  <div class="field">
    <div class="head">
      <span class="name">{{ param.label }}</span>
      <NTooltip v-if="param.note" trigger="hover" :style="{ maxWidth: '320px' }">
        <template #trigger><span class="hint">?</span></template>
        {{ param.note }}
      </NTooltip>
      <span class="key dt-mono">{{ param.key }}</span>
    </div>

    <ImagenModelPicker
      v-if="param.widget === 'imagen-model'"
      :provider="provider ?? 'openai'"
      v-model="value"
    />

    <NInput
      v-else-if="param.type === 'string'"
      v-model:value="value as string"
      :type="param.multiline ? 'textarea' : 'text'"
      :autosize="param.multiline ? { minRows: 2, maxRows: 6 } : undefined"
      :placeholder="String(param.default ?? '')"
    />

    <NSelect v-else-if="param.type === 'enum'" v-model:value="value as string" :options="options" />

    <NSwitch v-else-if="param.type === 'bool'" v-model:value="value as boolean" />

    <div v-else-if="isRanged" class="ranged">
      <NSlider
        v-model:value="value as number"
        :min="param.min"
        :max="param.max"
        :step="step"
        :tooltip="false"
      />
      <NInputNumber
        v-model:value="value as number"
        :min="param.min"
        :max="param.max"
        :step="step"
        size="small"
        class="num"
      />
    </div>

    <NInputNumber
      v-else
      v-model:value="value as number"
      :min="param.min"
      :max="param.max"
      :step="step"
      class="full"
    />
  </div>
</template>

<style scoped>
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.head {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.name {
  font-size: var(--dt-fs-base);
  font-weight: 500;
}
.key {
  font-size: var(--dt-fs-xs);
  color: var(--dt-ink-faint);
  margin-left: auto;
}
.hint {
  display: grid;
  place-items: center;
  width: 14px;
  height: 14px;
  font-size: var(--dt-fs-2xs);
  color: var(--dt-ink-faint);
  border: 1px solid var(--dt-border-strong);
  cursor: help;
}
.ranged {
  display: grid;
  grid-template-columns: 1fr 96px;
  gap: 12px;
  align-items: center;
}
.full {
  width: 100%;
}
</style>
