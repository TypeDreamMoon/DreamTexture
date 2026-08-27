// 应用级共享状态。
//
// 没上 Pinia：这个工具只有一份全局状态（任务队列 + ComfyUI 健康），
// 一个响应式模块就够了，也少一层依赖的 API 变动风险。
import { reactive, ref, computed } from 'vue'
import { api } from './api/client'
import type {
  BusEvent,
  ComfyHealth,
  Download,
  ImagenModel,
  ImagenProvider,
  Job,
  WorkflowMeta,
} from './api/types'

export const workflows = ref<WorkflowMeta[]>([])
export const health = ref<ComfyHealth | null>(null)
export const wsConnected = ref(false)

/** 任务表，以 id 为键，供 WebSocket 事件就地更新。 */
const jobMap = reactive(new Map<string, Job>())

export const jobs = computed(() =>
  [...jobMap.values()].sort((a, b) => b.created_at.localeCompare(a.created_at)),
)

export const activeJobs = computed(() =>
  jobs.value.filter((j) => j.status === 'queued' || j.status === 'running'),
)

export function upsertJob(job: Job) {
  jobMap.set(job.id, job)
}

/** 模型下载表，同样由 WebSocket 事件就地更新。 */
const downloadMap = reactive(new Map<string, Download>())

export const downloads = computed(() =>
  [...downloadMap.values()].sort((a, b) => b.started_at.localeCompare(a.started_at)),
)

export const activeDownloads = computed(() =>
  downloads.value.filter((d) => d.state === 'queued' || d.state === 'running'),
)

export function upsertDownload(d: Download) {
  downloadMap.set(d.id, d)
}

export function jobById(id: string): Job | undefined {
  return jobMap.get(id)
}

/**
 * 云端底图来源。
 *
 * 放在 store 而不是组件里，是因为拉一次要跨境访问服务端问模型清单，
 * 秒级起步——多个组件各拉一次的话，切一次预设就要等好几秒。
 */
export const imagenProviders = ref<ImagenProvider[] | null>(null)
export const imagenError = ref('')
let imagenPending: Promise<void> | null = null

export function loadImagen(force = false): Promise<void> {
  if (imagenPending) return imagenPending
  if (imagenProviders.value && !force) return Promise.resolve()
  imagenPending = api
    .imagenProviders()
    .then((r) => {
      imagenProviders.value = r.providers
      imagenError.value = ''
    })
    .catch((e: Error) => {
      imagenError.value = String(e.message)
      imagenProviders.value = []
    })
    .finally(() => {
      imagenPending = null
    })
  return imagenPending
}

/** 某个来源的模型清单；来源不存在或未配置时为空数组。 */
export function modelsOf(provider: string): ImagenModel[] {
  return imagenProviders.value?.find((p) => p.id === provider)?.models ?? []
}

export function providerOf(provider: string): ImagenProvider | undefined {
  return imagenProviders.value?.find((p) => p.id === provider)
}

/** 同一次提交产生的全部变体，按创建时间正序。 */
export function batchJobs(batchID: string): Job[] {
  return [...jobMap.values()]
    .filter((j) => j.batch_id === batchID)
    .sort((a, b) => a.created_at.localeCompare(b.created_at))
}

export async function bootstrap() {
  const [wf, hs, js] = await Promise.allSettled([
    api.workflows(),
    api.comfyStatus(),
    api.jobs(60),
  ])
  if (wf.status === 'fulfilled') workflows.value = wf.value
  if (hs.status === 'fulfilled') health.value = hs.value
  if (js.status === 'fulfilled') js.value.forEach(upsertJob)
}

/** 连接事件流，断线自动重连。 */
export function connectEvents() {
  let retry = 1000
  let closed = false

  const open = () => {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const ws = new WebSocket(`${proto}://${location.host}/api/ws`)

    ws.onopen = () => {
      wsConnected.value = true
      retry = 1000
    }
    ws.onmessage = (e) => {
      let ev: BusEvent
      try {
        ev = JSON.parse(e.data as string)
      } catch {
        return
      }
      if (ev.type === 'comfy.status') {
        health.value = ev.data as ComfyHealth
      } else if (ev.type === 'model.download') {
        upsertDownload(ev.data as Download)
      } else if (ev.job) {
        upsertJob(ev.job)
      }
    }
    ws.onclose = () => {
      wsConnected.value = false
      if (closed) return
      setTimeout(open, retry)
      retry = Math.min(retry * 2, 15000)
    }
    ws.onerror = () => ws.close()
  }

  open()
  return () => {
    closed = true
  }
}
