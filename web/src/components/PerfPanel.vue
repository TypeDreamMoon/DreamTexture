<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NAlert, NButton, NInput, NSelect, NSpin, NSwitch, useMessage } from 'naive-ui'
import SettingRow from './SettingRow.vue'
import { api } from '../api/client'
import type { FlagOption } from '../api/types'

const emit = defineEmits<{ saved: [string[]] }>()
const message = useMessage()

const catalog = ref<FlagOption[]>([])
const values = ref<Record<string, string>>({})
const extra = ref('')
const managed = ref<string[]>([])
/** 后端算出来的当前参数。没改动时直接显示它，避免和前端的预览逻辑对不上。 */
const savedArgs = ref('')
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const showAdvanced = ref(false)

// 记住加载时的样子，用来判断有没有改动——没改动时把保存按钮点亮是种噪音。
const baseline = ref('')
function snapshot() {
  return JSON.stringify({ v: values.value, e: extra.value.trim() })
}
const dirty = computed(() => snapshot() !== baseline.value)

const basic = computed(() => catalog.value.filter((o) => !o.advanced))
const advanced = computed(() => catalog.value.filter((o) => o.advanced))

/**
 * 最终会传给 ComfyUI 的那串参数。
 *
 * 没改动时用后端给的原值，只有改动之后才用下面这段本地推算——它和后端的
 * Build 是同一条规则的两份实现，两份实现迟早会漂。让"没动过"这个最常见的
 * 情形始终显示真值，漂了也只会短暂地出现在编辑当中，保存后立刻被纠回来。
 */
const preview = computed(() => (dirty.value ? localPreview.value : savedArgs.value))

const localPreview = computed(() => {
  const out: string[] = []
  for (const o of catalog.value) {
    const v = values.value[o.key]
    if (v === undefined) continue
    if (o.kind === 'bool') {
      // 与后端 Build 同一条规则：普通项开着才加，invert 的项关掉才加。
      if ((v === 'true') !== !!o.invert) out.push(o.flag ?? '')
    } else {
      const c = o.choices?.find((x) => x.value === v)
      if (c?.args?.length) out.push(...c.args)
    }
  }
  const tail = extra.value.trim()
  return [...out.filter(Boolean), ...(tail ? tail.split(/\s+/) : [])].join(' ')
})

function optionsFor(o: FlagOption) {
  return (o.choices ?? []).map((c) => ({
    label: c.note ? `${c.label} · ${c.note}` : c.label,
    value: c.value,
  }))
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const r = await api.comfyFlags()
    catalog.value = r.catalog
    values.value = { ...r.values }
    extra.value = r.extra
    savedArgs.value = r.raw
    // ?? [] 是第二道防线：后端已经保证返回数组了，但这个字段一旦是 null，
    // 抛的错会把整个组件从页面上抹掉，代价远大于少显示一条提醒。
    managed.value = r.managed ?? []
    baseline.value = snapshot()
  } catch (e) {
    error.value = String((e as Error).message)
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function save() {
  saving.value = true
  try {
    const r = await api.setComfyFlags(values.value, extra.value)
    // 以后端合成的结果为准重新读一遍：它才是最终写进配置的东西，
    // 界面自己算的那份只是预览。
    await load()
    emit('saved', r.need_restart ?? [])
    message.warning('已保存。ComfyUI 启动参数需要重启后端才生效')
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    saving.value = false
  }
}

function reset() {
  void load()
}
</script>

<template>
  <div v-if="loading" class="center"><NSpin /></div>
  <NAlert v-else-if="error" type="error" :bordered="false">{{ error }}</NAlert>

  <template v-else>
    <NAlert v-if="managed.length" type="warning" :bordered="false" class="warn">
      「其他参数」里出现了 <span class="dt-mono">{{ managed.join(' ') }}</span
      >，这些由 DreamTexture 自己填。保存时会被拦下——监听地址改「ComfyUI 地址」，
      显存余量有单独一项。
    </NAlert>

    <div class="rows">
      <SettingRow
        v-for="o in basic"
        :key="o.key"
        :icon="o.icon || 'sliders'"
        :title="o.label"
        :desc="o.help"
        restart
      >
        <NSwitch
          v-if="o.kind === 'bool'"
          :value="values[o.key] === 'true'"
          @update:value="(v: boolean) => (values[o.key] = v ? 'true' : 'false')"
        />
        <NSelect
          v-else
          v-model:value="values[o.key]"
          :options="optionsFor(o)"
          size="small"
          class="pick"
        />
      </SettingRow>
    </div>

    <button class="more" @click="showAdvanced = !showAdvanced">
      {{ showAdvanced ? '收起' : '展开' }}计算精度与分配器等 {{ advanced.length }} 项
    </button>

    <div v-if="showAdvanced" class="rows">
      <SettingRow
        v-for="o in advanced"
        :key="o.key"
        :icon="o.icon || 'sliders'"
        :title="o.label"
        :desc="o.help"
        restart
      >
        <NSelect
          v-model:value="values[o.key]"
          :options="optionsFor(o)"
          size="small"
          class="pick"
        />
      </SettingRow>
    </div>

    <div class="rows">
      <SettingRow
        icon="terminal"
        title="其他参数"
        desc="上面没覆盖到的，按命令行原样写在这里。这一栏也是「认不出来的参数」的去处——目录跟不上 ComfyUI 更新时，你手写的东西会留在这儿而不是被吃掉"
        stack
      >
        <NInput
          v-model:value="extra"
          size="small"
          placeholder="例如 --whitelist-custom-nodes ComfyUI-Chord"
        />
      </SettingRow>
    </div>

    <div class="foot">
      <p class="cmd dt-mono dt-faint">
        <span class="lead">最终参数</span>{{ preview || '（无）' }}
      </p>
      <div class="acts">
        <NButton v-if="dirty" size="small" tertiary @click="reset">放弃改动</NButton>
        <NButton size="small" type="primary" :disabled="!dirty" :loading="saving" @click="save">
          保存
        </NButton>
      </div>
    </div>
  </template>
</template>

<style scoped>
.center {
  display: flex;
  justify-content: center;
  padding: 28px 0;
}
.warn {
  margin-bottom: 12px;
}
.rows {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.pick {
  min-width: 240px;
}
.more {
  margin: 10px 0;
  padding: 0;
  background: none;
  border: 0;
  cursor: pointer;
  font: inherit;
  font-size: var(--dt-fs-sm);
  color: var(--dt-accent);
}
.more:hover {
  text-decoration: underline;
}
.foot {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--dt-border);
}
.cmd {
  flex: 1;
  margin: 0;
  font-size: var(--dt-fs-xs);
  line-height: 1.7;
  word-break: break-all;
}
.lead {
  display: inline-block;
  margin-right: 8px;
  color: var(--dt-ink-faint);
}
.acts {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}
</style>
