<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, useMessage } from 'naive-ui'
import { api } from '../api/client'
import type { CheckResult } from '../api/types'

// 环境自检横幅：只在有问题时出现，一切正常就完全不打扰。
//
// 同类工具最大的劝退点是装不上、跑不起来，而报错往往等到点了生成才出现。
// 与其让用户对着一句 "node not found" 发愣，不如进门就说清楚缺什么、怎么补。
const router = useRouter()
const message = useMessage()
const result = ref<CheckResult | null>(null)
const busy = ref(false)
const dismissed = ref(false)

async function run() {
  try {
    result.value = await api.checks()
  } catch {
    result.value = null
  }
}

onMounted(run)

const problems = computed(() => result.value?.checks.filter((c) => c.status !== 'ok') ?? [])
const show = computed(() => !dismissed.value && problems.value.length > 0)
const fatal = computed(() => problems.value.some((c) => c.status === 'fail'))

async function fix(kind: string) {
  if (kind === 'open-models') {
    router.push('/models')
    return
  }
  if (kind === 'restart-comfy') {
    busy.value = true
    try {
      await api.comfyRestart()
      message.success('ComfyUI 已重启')
      await run()
    } catch (e) {
      message.error(String((e as Error).message))
    } finally {
      busy.value = false
    }
  }
}
</script>

<template>
  <Transition name="page">
    <section v-if="show" class="check dt-panel dt-enter" :class="{ fatal }">
      <div class="rows">
        <div v-for="c in problems" :key="c.key" class="row">
          <span class="dot" :class="c.status" />
          <div class="text">
            <p class="t">
              <span class="lab">{{ c.label }}</span>
              <span class="dt-muted">{{ c.detail }}</span>
            </p>
            <p v-if="c.items?.length" class="items dt-mono dt-faint">
              {{ c.items.join('　') }}
            </p>
          </div>
          <NButton v-if="c.fix" size="tiny" :loading="busy" @click="fix(c.fix)">
            {{ c.fix === 'open-models' ? '去处理' : '重启 ComfyUI' }}
          </NButton>
        </div>
      </div>
      <div class="acts">
        <button class="link" @click="run">重新检查</button>
        <button class="link" @click="dismissed = true">本次忽略</button>
      </div>
    </section>
  </Transition>
</template>

<style scoped>
.check {
  padding: 14px 16px;
  margin-bottom: 16px;
  border-left: 3px solid var(--dt-warn);
  display: flex;
  align-items: flex-start;
  gap: 16px;
}
.check.fatal {
  border-left-color: var(--dt-danger);
}
.rows {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 9px;
}
.row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}
.dot {
  flex: none;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  margin-top: 6px;
}
.dot.warn {
  background: var(--dt-warn);
}
.dot.fail {
  background: var(--dt-danger);
}
.text {
  flex: 1;
  min-width: 0;
}
.t {
  margin: 0;
  font-size: var(--dt-fs-base);
  line-height: 1.55;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.lab {
  font-weight: 500;
}
.items {
  margin: 4px 0 0;
  font-size: var(--dt-fs-xs);
  line-height: 1.6;
  word-break: break-all;
}
.acts {
  display: flex;
  flex-direction: column;
  gap: 5px;
  flex: none;
}
.link {
  font: inherit;
  font-size: var(--dt-fs-sm);
  color: var(--dt-ink-faint);
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  text-align: right;
}
.link:hover {
  color: var(--dt-accent);
}
</style>
