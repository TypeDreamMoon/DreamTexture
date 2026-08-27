// 视图偏好的本地留存。
//
// 只留"用户对界面的选择"——档位、筛选、排版偏好这类**看得见**的东西。
// 不留 loading / busy / error 这些瞬时状态，也不留会让人误判的状态：
// 比如日志页的"暂停"，恢复成暂停态之后界面看着像卡住了，而其实是上次留下的。
import { ref, watch, type Ref } from 'vue'

/**
 * localStorage 在隐私窗口、以及某些把站点数据关掉的浏览器里会直接抛异常。
 * 读写都得包住——存不下无非是不记住，不该让整个页面白屏。
 */
function read(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function write(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    /* 存不了也不影响用 */
  }
}

/**
 * 造一个会自己存回 localStorage 的 ref。
 *
 * `valid` 用来挡住"存下来的值现在已经不合法了"：枚举改名、工作流被删，
 * 都会让旧值把界面卡在一个不存在的选项上——而且没有任何报错，
 * 表现是"这个下拉框显示空白且怎么点都不对"。校验不过就退回默认值。
 */
export function persisted<T>(
  key: string,
  fallback: T,
  valid?: (v: unknown) => boolean,
): Ref<T> {
  const raw = read('dt.' + key)
  let init = fallback
  if (raw !== null) {
    try {
      const parsed = JSON.parse(raw) as T
      if (!valid || valid(parsed)) init = parsed
    } catch {
      /* 存进去的不是合法 JSON，当没存过 */
    }
  }
  const r = ref(init) as Ref<T>
  watch(r, (v) => write('dt.' + key, JSON.stringify(v)))
  return r
}

/** 造一个只认这几个取值的 ref，专给档位、模式这类枚举用。 */
export function persistedEnum<T extends string>(
  key: string,
  fallback: T,
  options: readonly T[],
): Ref<T> {
  return persisted<T>(key, fallback, (v) => options.includes(v as T))
}

/**
 * 原样读一条，只用来判断"存过没有"。
 *
 * 和 persisted 的默认值区分不开这件事有关：偏好为 false 和从没存过，
 * 值一样但含义不同——前者是用户明确选的，后者该让自动逻辑接管。
 */
export function readRaw(key: string): string | null {
  return read('dt.' + key)
}

/**
 * 一次性迁移旧的侧边栏取值。
 *
 * 0.1 版存的是 'collapsed' / 'open' 两个裸字符串。留在这儿是为了让老用户的
 * 偏好平滑过来；等确定没人还带着旧值了可以删。
 */
export function migrateSidebar() {
  const raw = read('dt.sidebar')
  if (raw === 'collapsed' || raw === 'open') {
    write('dt.sidebar', JSON.stringify(raw === 'collapsed'))
  }
}
