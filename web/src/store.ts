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

/** segments 是"半条管线"的声明，自己提交不了，只用来渲染出图/分解那两个下拉。 */
export const segments = ref<WorkflowMeta[]>([])
export const health = ref<ComfyHealth | null>(null)
export const wsConnected = ref(false)

/**
 * wsDown 是"断开且一时半会儿没回来"。
 *
 * 与 wsConnected 分开是因为后者有两个会误报的瞬间：页面刚加载还没握上手，
 * 以及重连成功前的那一两秒。拿它直接驱动界面，每次刷新都会闪一下红。
 */
export const wsDown = ref(false)

/** 断开多久之后才认为值得告诉用户。 */
const DOWN_GRACE = 3000

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
  if (wf.status === 'fulfilled') {
    workflows.value = wf.value.workflows
    segments.value = wf.value.segments
  }
  if (hs.status === 'fulfilled') health.value = hs.value
  if (js.status === 'fulfilled') js.value.forEach(upsertJob)
}

/**
 * STALE_AFTER 是多久没收到任何消息就判定这条连接已经死了。
 *
 * 后端每 15 秒发一个 {"type":"hb"}（internal/api/websocket.go 的
 * HeartbeatInterval），这里给到两拍多一点的余量——短暂的 GC 或调度停顿
 * 不该把一条好连接掐掉，但也不能等太久，超时期间界面显示的是过期状态。
 */
const STALE_AFTER = 40000

/** 连接事件流，断线自动重连。 */
export function connectEvents() {
  let retry = 1000
  let closed = false
  let timer: ReturnType<typeof setTimeout> | undefined
  let grace: ReturnType<typeof setTimeout> | undefined

  const setConnected = (ok: boolean) => {
    wsConnected.value = ok
    clearTimeout(grace)
    if (ok) {
      wsDown.value = false
    } else if (!closed) {
      grace = setTimeout(() => {
        wsDown.value = true
      }, DOWN_GRACE)
    }
  }

  const schedule = () => {
    if (closed) return
    clearTimeout(timer)
    timer = setTimeout(open, retry)
    retry = Math.min(retry * 2, 15000)
  }

  function open() {
    // new WebSocket 是会同步抛的（地址不合法、某些容器里的 SecurityError）。
    // 它跑在 setTimeout 的回调里，抛出去没有任何人接得住，重连链就此断掉，
    // 界面从此永远停在"事件流断开"——比连不上更糟的是再也不重试。
    let ws: WebSocket
    try {
      const proto = location.protocol === 'https:' ? 'wss' : 'ws'
      ws = new WebSocket(`${proto}://${location.host}/api/ws`)
    } catch {
      setConnected(false)
      schedule()
      return
    }

    // settled 保证一条连接只安排一次重连。
    //
    // 没有它的话：看门狗把一条僵死的连接 close 掉、新连接已经建好，此时旧连接
    // 迟到的 close 事件才到，于是又排一次重连——同时挂两条，之后越滚越多。
    let settled = false
    let watchdog: ReturnType<typeof setTimeout> | undefined

    const die = () => {
      clearTimeout(watchdog)
      if (settled) return
      settled = true
      setConnected(false)
      schedule()
    }

    // 看门狗盯的是"有没有消息进来"，而不是 close 事件。
    //
    // close 事件不保证会来：标签页被冻结、机器休眠再唤醒、半开连接，都会让它
    // 丢掉。丢掉之后界面会一直显示"已连接"外加一份过期的状态——实测把后端
    // 进程杀干净，页面照旧显示 ComfyUI 就绪、显存 14.8G。协议层的 ping/pong
    // 帮不上忙：浏览器在网络层就把 pong 回掉了，JS 收不到任何事件。
    const kick = () => {
      clearTimeout(watchdog)
      watchdog = setTimeout(() => {
        die()
        try {
          ws.close()
        } catch {
          /* 已经关掉了 */
        }
      }, STALE_AFTER)
    }

    ws.onopen = () => {
      setConnected(true)
      retry = 1000
      kick()
    }
    ws.onmessage = (e) => {
      kick()
      let ev: BusEvent
      try {
        ev = JSON.parse(e.data as string)
      } catch {
        return
      }
      if (ev.type === 'hb') {
        return
      } else if (ev.type === 'comfy.status') {
        health.value = ev.data as ComfyHealth
      } else if (ev.type === 'model.download') {
        upsertDownload(ev.data as Download)
      } else if (ev.job) {
        upsertJob(ev.job)
      }
    }
    ws.onclose = die
    ws.onerror = () => {
      try {
        ws.close()
      } catch {
        /* 忽略 */
      }
      die()
    }
  }

  // 从休眠/后台回来时立刻试一次，不用干等最长 15 秒的退避。
  // 这两件事恰好是最容易丢掉 close 事件的场合。
  const wake = () => {
    if (closed || wsConnected.value) return
    if (document.visibilityState !== 'visible') return
    retry = 1000
    clearTimeout(timer)
    timer = setTimeout(open, 0)
  }
  document.addEventListener('visibilitychange', wake)
  window.addEventListener('online', wake)

  open()
  return () => {
    closed = true
    clearTimeout(timer)
    clearTimeout(grace)
    document.removeEventListener('visibilitychange', wake)
    window.removeEventListener('online', wake)
  }
}
