<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NAlert, NButton, NInput, NSpin, NTag, useDialog, useMessage } from 'naive-ui'
import { api } from '../api/client'
import type { ComfyVersion, ComfyVersionStatus } from '../api/types'

const emit = defineEmits<{ switching: [] }>()
const message = useMessage()
const dialog = useDialog()

const kind = ref<'stable' | 'dev'>('stable')
const status = ref<ComfyVersionStatus | null>(null)
const versions = ref<ComfyVersion[]>([])
const loading = ref(true)
const busy = ref('')
const error = ref('')
const manual = ref('')

const KINDS = [
  { label: '稳定版', value: 'stable' as const },
  { label: '开发版', value: 'dev' as const },
]

async function load(k = kind.value) {
  kind.value = k
  loading.value = true
  error.value = ''
  try {
    const r = await api.comfyVersions(k)
    status.value = r.status
    versions.value = r.versions
    error.value = r.error ?? ''
  } catch (e) {
    error.value = String((e as Error).message)
  } finally {
    loading.value = false
  }
}
onMounted(() => load())

/** 补历史。浅克隆下一个版本都列不出来，得先把提交图和 tag 拉回来。 */
async function fetchAll() {
  busy.value = 'fetch'
  try {
    await api.fetchComfyVersions()
    await load()
    message.success('历史已补齐')
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    busy.value = ''
  }
}

function confirmSwitch(ref_: string, name: string) {
  dialog.warning({
    title: '切换 ComfyUI 版本',
    content:
      `要切到 ${name} 吗？切换会重装 ComfyUI 的依赖（requirements.txt 在版本之间会变），` +
      `视网络情况可能要几分钟。切完需要重启 ComfyUI 才生效。`,
    positiveText: '切换',
    negativeText: '取消',
    onPositiveClick: () => doSwitch(ref_),
  })
}

async function doSwitch(ref_: string) {
  busy.value = ref_
  try {
    await api.switchComfyVersion(ref_)
    // 进度走的是部署那套步骤，让设置页把部署面板亮出来。
    emit('switching')
    message.info('已开始切换，进度看下面的部署面板')
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    busy.value = ''
  }
}

const shown = computed(() => versions.value)

function fmtDate(s: string) {
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return ''
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}
</script>

<template>
  <div v-if="loading" class="center"><NSpin /></div>

  <template v-else>
    <NAlert v-if="status && !status.available" type="warning" :bordered="false" class="note">
      {{ status.reason || '这份 ComfyUI 管不了版本' }}
    </NAlert>

    <template v-else-if="status">
      <dl class="head">
        <dt>远端地址</dt>
        <dd class="dt-mono">{{ status.remote || '—' }}</dd>
        <dt>当前分支</dt>
        <dd class="dt-mono">{{ status.branch || '—' }}</dd>
        <dt>当前版本</dt>
        <dd class="dt-mono cur">
          {{ status.short }}
          <span v-if="status.name" class="cname">{{ status.name }}</span>
          <span v-if="status.date" class="dt-faint">（{{ fmtDate(status.date) }}）</span>
        </dd>
      </dl>

      <NAlert v-if="status.dirty" type="warning" :bordered="false" class="note">
        ComfyUI 目录里有未提交的改动，切版本会把它们冲掉，所以会被拦下：
        <span class="dt-mono">{{ (status.dirty_files ?? []).join('、') }}</span>
      </NAlert>

      <!-- 浅克隆是"一个版本都列不出来"的根因，单独说清楚并给出口，
           否则用户只会看到一张空表，不知道该干嘛。 -->
      <NAlert v-if="status.shallow" type="info" :bordered="false" class="note">
        这份 ComfyUI 是浅克隆（只有一个提交），列不出可切换的版本。
        补一次历史就好——只拉提交记录不拉旧文件，通常几秒到几十秒。
        <div class="fetchrow">
          <NButton size="small" :loading="busy === 'fetch'" @click="fetchAll">补齐历史</NButton>
        </div>
      </NAlert>

      <div class="bar">
        <div class="tabs">
          <button
            v-for="k in KINDS"
            :key="k.value"
            class="tab"
            :class="{ on: kind === k.value }"
            @click="load(k.value)"
          >
            {{ k.label }}
          </button>
        </div>
        <NButton size="tiny" tertiary :loading="busy === 'fetch'" @click="fetchAll">
          刷新列表
        </NButton>
      </div>

      <NAlert v-if="error" type="error" :bordered="false" class="note">{{ error }}</NAlert>

      <div v-if="shown.length" class="tablewrap">
        <table class="tbl">
          <thead>
            <tr>
              <th class="c-id">版本 ID</th>
              <th>{{ kind === 'stable' ? '版本' : '更新内容' }}</th>
              <th class="c-date">日期</th>
              <th class="c-act"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="v in shown" :key="v.ref" :class="{ cur: v.current }">
              <td class="c-id dt-mono">{{ v.short }}</td>
              <td class="c-name" :title="v.name">{{ v.name }}</td>
              <td class="c-date dt-mono dt-faint">{{ fmtDate(v.date) }}</td>
              <td class="c-act">
                <NTag v-if="v.current" size="tiny" type="success" :bordered="false">当前</NTag>
                <NButton
                  v-else
                  size="tiny"
                  :disabled="status.dirty"
                  :loading="busy === v.ref"
                  @click="confirmSwitch(v.ref, v.name || v.short)"
                >
                  切换
                </NButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else-if="!status.shallow" class="empty dt-faint">这一档里没有版本。</p>

      <!-- 开发版只列最近 60 条。要更早的提交时，照着别处给的号直接填。 -->
      <div class="manual">
        <NInput
          v-model:value="manual"
          size="small"
          placeholder="也可以直接填提交号或 tag，例如 v0.31.0"
        />
        <NButton
          size="small"
          :disabled="!manual.trim() || status.dirty"
          @click="confirmSwitch(manual.trim(), manual.trim())"
        >
          切到这个
        </NButton>
      </div>
    </template>
  </template>
</template>

<style scoped>
.center {
  display: flex;
  justify-content: center;
  padding: 28px 0;
}
.note {
  margin-bottom: 12px;
}
.fetchrow {
  margin-top: 8px;
}
.head {
  display: grid;
  grid-template-columns: 76px 1fr;
  gap: 5px 12px;
  margin: 0 0 14px;
  font-size: var(--dt-fs-sm);
  line-height: 1.6;
}
.head dt {
  color: var(--dt-ink-faint);
}
.head dd {
  margin: 0;
  word-break: break-all;
}
.cur .cname,
.head .cname {
  color: var(--dt-accent);
  margin-left: 8px;
}
.bar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}
.tabs {
  display: flex;
  gap: 2px;
  margin-right: auto;
}
.tab {
  padding: 5px 14px;
  border: 0;
  border-bottom: 2px solid transparent;
  background: none;
  color: var(--dt-ink-muted);
  font: inherit;
  font-size: var(--dt-fs-sm);
  cursor: pointer;
}
.tab.on {
  color: var(--dt-ink);
  border-bottom-color: var(--dt-accent);
}
.tablewrap {
  max-height: 380px;
  overflow: auto;
  border: 1px solid var(--dt-border);
  border-radius: var(--dt-radius);
}
.tbl {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--dt-fs-sm);
}
.tbl th {
  position: sticky;
  top: 0;
  z-index: 1;
  text-align: left;
  font-weight: 500;
  color: var(--dt-ink-faint);
  background: var(--dt-surface2);
  padding: 7px 10px;
  border-bottom: 1px solid var(--dt-border);
}
.tbl td {
  padding: 6px 10px;
  border-bottom: 1px solid var(--dt-border);
}
.tbl tr:last-child td {
  border-bottom: 0;
}
.tbl tr.cur {
  background: color-mix(in srgb, var(--dt-accent) 10%, transparent);
}
.c-id {
  width: 92px;
  color: var(--dt-accent);
}
.c-date {
  width: 132px;
  white-space: nowrap;
  font-size: var(--dt-fs-xs);
}
.c-act {
  width: 64px;
  text-align: right;
}
/* 提交标题可以很长，压成一行，完整内容给 title */
.c-name {
  max-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.empty {
  padding: 16px 0;
  font-size: var(--dt-fs-sm);
}
.manual {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
}
</style>
