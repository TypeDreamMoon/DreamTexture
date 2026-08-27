<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import { RoomEnvironment } from 'three/examples/jsm/environments/RoomEnvironment.js'
import { fileURL } from '../api/client'
import type { Manifest } from '../api/types'

// 真正把各通道合起来打光看一眼。逐张看通道图判断不了材质好坏——
// 法线对不对、粗糙度分布合不合理，只有在光扫过表面时才看得出来。
const props = defineProps<{ manifest: Manifest }>()

type Shape = 'plane' | 'sphere' | 'cube'

const host = ref<HTMLDivElement | null>(null)
const shape = ref<Shape>('plane')
const tiling = ref(1)
const displacement = ref(0.06)
const loading = ref(true)

// 光照参数。默认给一束偏低的强光配一点点环境光——
// 掠射光是看出法线与高度细节最有效的方式；环境光给多了会把方向光冲淡，
// 表面就变成一片没有信息的均匀亮面。
const lightIntensity = ref(3.4)
const lightElevation = ref(22) // 度，越低越掠射
const lightAzimuth = ref(50) // 度
const envIntensity = ref(0.3)
const lightWarmth = ref(0.35) // 0 冷白 → 1 暖黄
const spinLight = ref(true)

const SHAPES: { key: Shape; label: string }[] = [
  { key: 'plane', label: '平面' },
  { key: 'sphere', label: '球体' },
  { key: 'cube', label: '立方体' },
]

// three 的对象不需要响应式代理——深度代理一个场景图既慢又容易出怪问题。
const gl = shallowRef<THREE.WebGLRenderer>()
const scene = shallowRef<THREE.Scene>()
const camera = shallowRef<THREE.PerspectiveCamera>()
const controls = shallowRef<OrbitControls>()
const material = shallowRef<THREE.MeshStandardMaterial>()
const mesh = shallowRef<THREE.Mesh>()
const keyLight = shallowRef<THREE.DirectionalLight>()

const fillLight = shallowRef<THREE.DirectionalLight>()
const textures: THREE.Texture[] = []
let raf = 0
let ro: ResizeObserver | null = null

/** 按当前的强度、方位、高度、色温摆好光。 */
function applyLight() {
  const key = keyLight.value
  const fill = fillLight.value
  const sc = scene.value
  if (!key || !fill || !sc) return

  const az = (lightAzimuth.value * Math.PI) / 180
  const el = (lightElevation.value * Math.PI) / 180
  const r = 5
  key.position.set(
    Math.cos(az) * Math.cos(el) * r,
    Math.sin(el) * r,
    Math.sin(az) * Math.cos(el) * r,
  )
  key.intensity = lightIntensity.value
  // 冷白 → 暖黄之间插值，模拟从阴天到白炽的光色。
  key.color.setRGB(1, 1 - lightWarmth.value * 0.14, 1 - lightWarmth.value * 0.34)

  // 补光始终在主光背面，强度按主光的一小部分走，免得把暗部彻底填平。
  fill.position.set(-key.position.x, Math.max(key.position.y * 0.6, 1.5), -key.position.z)
  fill.intensity = lightIntensity.value * 0.18

  sc.environmentIntensity = envIntensity.value
}

function loadTexture(loader: THREE.TextureLoader, file: string, srgb: boolean) {
  const t = loader.load(fileURL(props.manifest.id, file))
  t.colorSpace = srgb ? THREE.SRGBColorSpace : THREE.NoColorSpace
  t.wrapS = t.wrapT = THREE.RepeatWrapping
  t.anisotropy = gl.value?.capabilities.getMaxAnisotropy() ?? 1
  textures.push(t)
  return t
}

function buildMaterial() {
  const m = props.manifest.maps
  const loader = new THREE.TextureLoader()
  const mat = new THREE.MeshStandardMaterial({ side: THREE.DoubleSide })

  if (m.basecolor) mat.map = loadTexture(loader, m.basecolor.file, true)
  if (m.roughness) mat.roughnessMap = loadTexture(loader, m.roughness.file, false)
  if (m.metallic) mat.metalnessMap = loadTexture(loader, m.metallic.file, false)
  if (m.ao) mat.aoMap = loadTexture(loader, m.ao.file, false)
  if (m.height) mat.displacementMap = loadTexture(loader, m.height.file, false)

  if (m.normal) {
    mat.normalMap = loadTexture(loader, m.normal.file, false)
    // 这里就是 manifest 里 normal.y 字段的用处：我们的管线统一输出 DirectX(Y-)，
    // 而 three 按 OpenGL(Y+) 解释法线。翻 Y 分量即可，不必在落盘时为了迁就
    // 预览而改变产物本身——UE 才是主消费方。
    const flip = (m.normal.y ?? 'opengl').toLowerCase() === 'directx' ? -1 : 1
    mat.normalScale = new THREE.Vector2(1, flip)
  }

  // 有贴图时基础色/粗糙度/金属度的标量必须放开，否则会把贴图整体压暗或压平。
  mat.roughness = 1
  mat.metalness = m.metallic ? 1 : 0
  mat.aoMapIntensity = 1
  material.value = mat
  return mat
}

function makeGeometry(kind: Shape): THREE.BufferGeometry {
  let g: THREE.BufferGeometry
  switch (kind) {
    case 'sphere':
      g = new THREE.SphereGeometry(1.15, 256, 192)
      break
    case 'cube':
      g = new THREE.BoxGeometry(1.9, 1.9, 1.9, 128, 128, 128)
      break
    default:
      // 平面细分给足，置换才不会呈阶梯状。
      g = new THREE.PlaneGeometry(3, 3, 320, 320)
  }
  // three 的 aoMap 走第二套 UV，而这些内置几何体只带一套，复制一份即可。
  const uv = g.getAttribute('uv')
  if (uv && !g.getAttribute('uv1')) g.setAttribute('uv1', uv.clone())
  return g
}

function applyShape() {
  const s = scene.value
  if (!s || !material.value) return
  if (mesh.value) {
    s.remove(mesh.value)
    mesh.value.geometry.dispose()
  }
  const m = new THREE.Mesh(makeGeometry(shape.value), material.value)
  if (shape.value === 'plane') m.rotation.x = -Math.PI / 2.6
  s.add(m)
  mesh.value = m
}

function applyTiling() {
  const n = tiling.value
  for (const t of textures) {
    // 只改 repeat 就够了：纹理矩阵会自动重算。
    // 这里不能顺手设 needsUpdate —— 贴图是异步加载的，在图还没到位时把它标脏，
    // three 会刷一屏 "Texture marked for update but no image data found"。
    t.repeat.set(n, n)
  }
}

function applyDisplacement() {
  if (!material.value) return
  // 球和立方体上置换更容易露馅（边缘会裂开），给它减半。
  const scale = shape.value === 'plane' ? 1 : 0.5
  material.value.displacementScale = displacement.value * scale
  material.value.displacementBias = -displacement.value * scale * 0.5
  material.value.needsUpdate = true
}

function init() {
  const el = host.value
  if (!el) return

  const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true })
  renderer.setPixelRatio(Math.min(devicePixelRatio, 2))
  renderer.toneMapping = THREE.ACESFilmicToneMapping
  renderer.toneMappingExposure = 1.05
  el.appendChild(renderer.domElement)
  gl.value = renderer

  const sc = new THREE.Scene()
  scene.value = sc

  // 用 three 自带的房间环境生成 IBL：PBR 没有环境光就是一坨死板的塑料，
  // 而这个环境是程序生成的，不需要外部 HDRI 文件。
  const pmrem = new THREE.PMREMGenerator(renderer)
  sc.environment = pmrem.fromScene(new RoomEnvironment(), 0.04).texture
  pmrem.dispose()

  const cam = new THREE.PerspectiveCamera(38, 1, 0.1, 100)
  cam.position.set(0, 1.9, 3.5)
  camera.value = cam

  const key = new THREE.DirectionalLight(0xffffff, 1)
  sc.add(key)
  keyLight.value = key

  const fill = new THREE.DirectionalLight(0xdce6ff, 0.5)
  sc.add(fill)
  fillLight.value = fill

  const ctl = new OrbitControls(cam, renderer.domElement)
  ctl.enableDamping = true
  ctl.dampingFactor = 0.08
  ctl.minDistance = 1.6
  ctl.maxDistance = 8
  ctl.target.set(0, 0, 0)
  controls.value = ctl

  buildMaterial()
  applyShape()
  applyTiling()
  applyDisplacement()
  applyLight()
  loading.value = false

  ro = new ResizeObserver(resize)
  ro.observe(el)
  resize()

  const timer = new THREE.Timer()
  const tick = (t: number) => {
    raf = requestAnimationFrame(tick)
    timer.update(t)
    if (spinLight.value) {
      // 让光慢慢扫过表面——判断法线方向和粗糙度分布最直接的方式。
      lightAzimuth.value = (lightAzimuth.value + timer.getDelta() * 26) % 360
    }
    ctl.update()
    renderer.render(sc, cam)
  }
  raf = requestAnimationFrame(tick)
}

function resize() {
  const el = host.value
  const renderer = gl.value
  const cam = camera.value
  if (!el || !renderer || !cam) return
  const w = el.clientWidth
  const h = el.clientHeight
  if (w === 0 || h === 0) return
  renderer.setSize(w, h, false)
  cam.aspect = w / h
  cam.updateProjectionMatrix()
}

function resetView() {
  camera.value?.position.set(0, 1.9, 3.5)
  controls.value?.target.set(0, 0, 0)
  controls.value?.update()
}

// 三档常用打光。掠射是检查法线/高度的主力，柔和用来看整体色调，
// 强反差用来挑粗糙度分布不匀的地方。
const LIGHT_PRESETS = [
  { key: 'grazing', label: '掠射', intensity: 4.2, elevation: 12, env: 0.18, warmth: 0.35 },
  { key: 'soft', label: '柔和', intensity: 2.0, elevation: 45, env: 0.7, warmth: 0.25 },
  { key: 'harsh', label: '强反差', intensity: 6.0, elevation: 28, env: 0.05, warmth: 0.6 },
] as const

const activePreset = ref<string>('')

function usePreset(p: (typeof LIGHT_PRESETS)[number]) {
  lightIntensity.value = p.intensity
  lightElevation.value = p.elevation
  envIntensity.value = p.env
  lightWarmth.value = p.warmth
  activePreset.value = p.key
}

// 手动调过任何一项就取消预设的高亮，免得显示与实际不符。
watch([lightIntensity, lightElevation, envIntensity, lightWarmth], () => {
  const p = LIGHT_PRESETS.find((x) => x.key === activePreset.value)
  if (
    !p ||
    p.intensity !== lightIntensity.value ||
    p.elevation !== lightElevation.value ||
    p.env !== envIntensity.value ||
    p.warmth !== lightWarmth.value
  ) {
    activePreset.value = ''
  }
})

watch(shape, () => {
  applyShape()
  applyDisplacement()
})
watch(tiling, applyTiling)
watch(displacement, applyDisplacement)
watch([lightIntensity, lightElevation, lightAzimuth, envIntensity, lightWarmth], applyLight)
watch(() => props.manifest.id, () => {
  // 换材质时把旧贴图释放掉，否则来回翻几个材质就吃掉几百 MB 显存。
  disposeTextures()
  const mat = buildMaterial()
  if (mesh.value) mesh.value.material = mat
  applyTiling()
  applyDisplacement()
})

function disposeTextures() {
  for (const t of textures) t.dispose()
  textures.length = 0
  material.value?.dispose()
}

onMounted(init)
onBeforeUnmount(() => {
  cancelAnimationFrame(raf)
  ro?.disconnect()
  controls.value?.dispose()
  mesh.value?.geometry.dispose()
  disposeTextures()
  scene.value?.environment?.dispose()
  gl.value?.dispose()
})
</script>

<template>
  <div class="wrap">
    <div ref="host" class="stage dt-swatch">
      <p v-if="loading" class="loading dt-faint">正在装配材质…</p>
    </div>

    <div class="bar">
      <div class="grp">
        <span class="dt-label">几何体</span>
        <button
          v-for="s in SHAPES"
          :key="s.key"
          class="chip"
          :class="{ on: shape === s.key }"
          @click="shape = s.key"
        >
          {{ s.label }}
        </button>
      </div>

      <div class="grp">
        <span class="dt-label">平铺</span>
        <button
          v-for="n in [1, 2, 3, 4]"
          :key="n"
          class="chip dt-mono"
          :class="{ on: tiling === n }"
          @click="tiling = n"
        >
          {{ n }}×
        </button>
      </div>

      <label class="grp slider">
        <span class="dt-label">置换</span>
        <input v-model.number="displacement" type="range" min="0" max="0.2" step="0.005" />
        <span class="dt-mono val">{{ displacement.toFixed(3) }}</span>
      </label>

      <div class="grp right">
        <button class="chip" @click="resetView">复位视角</button>
      </div>
    </div>

    <div class="bar light">
      <div class="grp">
        <span class="dt-label">打光</span>
        <button
          v-for="p in LIGHT_PRESETS"
          :key="p.key"
          class="chip"
          :class="{ on: activePreset === p.key }"
          @click="usePreset(p)"
        >
          {{ p.label }}
        </button>
      </div>

      <label class="grp slider">
        <span class="dt-label">光强</span>
        <input v-model.number="lightIntensity" type="range" min="0" max="8" step="0.1" />
        <span class="dt-mono val">{{ lightIntensity.toFixed(1) }}</span>
      </label>

      <label class="grp slider">
        <span class="dt-label">高度角</span>
        <input v-model.number="lightElevation" type="range" min="3" max="85" step="1" />
        <span class="dt-mono val">{{ lightElevation }}°</span>
      </label>

      <label class="grp slider">
        <span class="dt-label">环境光</span>
        <input v-model.number="envIntensity" type="range" min="0" max="1.5" step="0.05" />
        <span class="dt-mono val">{{ envIntensity.toFixed(2) }}</span>
      </label>

      <label class="grp slider">
        <span class="dt-label">色温</span>
        <input v-model.number="lightWarmth" type="range" min="0" max="1" step="0.05" />
        <span class="dt-mono val">{{ lightWarmth < 0.34 ? '冷' : lightWarmth < 0.67 ? '中' : '暖' }}</span>
      </label>

      <div class="grp right">
        <button class="chip" :class="{ on: spinLight }" @click="spinLight = !spinLight">
          {{ spinLight ? '光在转' : '光已停' }}
        </button>
        <label v-if="!spinLight" class="grp slider">
          <span class="dt-label">方位</span>
          <input v-model.number="lightAzimuth" type="range" min="0" max="360" step="1" />
        </label>
      </div>
    </div>

    <p class="tip dt-faint">
      拖拽旋转视角 · 滚轮缩放 · <b>高度角调低到掠射</b>最容易看出法线和高度的细节，
      环境光调低则暗部更分明
    </p>
  </div>
</template>

<style scoped>
.wrap {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.stage {
  position: relative;
  width: 100%;
  /* 不用 aspect-ratio:1 —— 宽列下会撑出上千像素高、必须滚动才看得全。
     按视口高度收敛，保证一屏之内看完。 */
  height: clamp(320px, 52vh, 560px);
  border: 1px solid var(--dt-border);
  border-radius: var(--dt-radius-card);
  overflow: hidden;
}
.stage :deep(canvas) {
  display: block;
  width: 100%;
  height: 100%;
}
.loading {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  margin: 0;
  font-size: var(--dt-fs-base);
}

.bar {
  display: flex;
  align-items: center;
  gap: 18px;
  flex-wrap: wrap;
}
/* 光照这一排单独分组：调参时视线不用在几何体和光之间来回跳 */
.bar.light {
  padding-top: 10px;
  border-top: 1px solid var(--dt-border);
  gap: 14px;
}
.grp {
  display: flex;
  align-items: center;
  gap: 6px;
}
.grp.right {
  margin-left: auto;
}
.grp .dt-label {
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
.slider {
  cursor: pointer;
}
.slider input {
  width: 92px;
  accent-color: var(--dt-accent);
}
.val {
  font-size: var(--dt-fs-xs);
  color: var(--dt-ink-faint);
  min-width: 30px;
}
.tip b {
  font-weight: 500;
  color: var(--dt-ink-muted);
}
.tip {
  margin: 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.5;
}
</style>
