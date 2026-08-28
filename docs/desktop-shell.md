# 桌面外壳（Tauri）

`src-tauri/` 是一层 Tauri 外壳，把界面装进一个原生窗口，不用再开浏览器输地址。

**它只做三件事**：把 Go 后端作为 sidecar 拉起来、等它开始监听、把窗口导航过去。
界面仍然由后端在 `127.0.0.1:8777` 上提供——外壳里没有第二份前端，也没有 IPC。

这么分是因为后端已经是这个应用的主体了：ComfyUI 子进程、任务队列、SQLite、
WebSocket 都在它那儿。外壳再管一遍只会多一条要对齐的进程链。

## 构建

```bash
go build -o "src-tauri/binaries/dreamtexture-$(rustc -vV | sed -n 's/^host: //p').exe" ./cmd/dreamtexture
```

```bash
pnpm install && pnpm tauri build
```

产物是 `src-tauri/target/release/bundle/nsis/DreamTexture_<版本>_x64-setup.exe`（约 13 MB）。
开发时用 `pnpm tauri dev`。

sidecar 的文件名必须带目标三元组，这是 Tauri 的约定；`binaries/` 已经 gitignore，
它是 `go build` 的产物，不进仓库。

> 改前端仍然要先 `cd web && pnpm build`——产物进 `internal/web/dist`，
> 由 Go 侧 `go:embed`，和外壳无关。

## 两条启动路径

**端口没人占** → 拉起 sidecar，等端口通了再显示窗口。

**端口已经通了** → 直接用那个后端，不再拉一个。这顺手把单实例做掉了：
再点一次图标只会多开一个窗口，而不是抢 8777 然后双双失败。

## 外链交给系统浏览器

窗口里没有标签页，`target="_blank"` 和 `window.open` 的默认行为是**什么都不发生**
——界面上那个「打开 ComfyUI」点下去毫无反应。

所以主窗口在 Rust 里建（不写在 tauri.conf.json 里），为的就是挂两个钩子：

| 钩子 | 管什么 |
|---|---|
| `on_new_window` | `target="_blank"` / `window.open` |
| `on_navigation` | 同窗口内跳到站外 |

两者都一样：不是我们后端的地址，就交给系统浏览器并拦下。ComfyUI 的节点编辑器
是个完整的应用，本来也不该塞进我们这个窗口里；Civitai、HuggingFace 那些页面同理。

只放行 `http(s)`。别的协议（`file:`、自定义协议…）交给系统等于让页面有能力
拉起本机程序，不值这个风险。

实测方式记一笔，因为不好测：没法在 Tauri 窗口里模拟点击，所以在真实界面上用
`win.eval` 触发 `window.open` 与 `location.href`，同时起一个本地监听端口收请求。
结果是两条路径都命中了监听端口，而 `msedgewebview2` 仍然连着 `127.0.0.1:8777`
——**窗口没有被带走**，正是要的行为。

## 谁起的谁负责关

关窗口退出时，**只杀自己拉起来的那个后端**；附着到已有实例时什么都不做。

后端被杀之后会把 ComfyUI 一起收干净（Windows 上用 Job Object，见
`internal/comfy`）。这条链实测验证过：关窗口 → 两个进程都没了、8777 释放、
没有孤儿；先手动起后端再开外壳 → 关窗口后那个后端还活着。

## 相对路径按配置文件的位置解析，不按 cwd

`config.finalize` 把 `output_dir` 这些相对路径补成绝对路径时，基准是
**配置文件所在目录的上一级**。

这条是做外壳时才暴露的：以前按 `cwd` 解析，从仓库根手动跑时两者恰好一样，
所以一直没事。而双击启动、被外壳拉起来、从服务里启动，cwd 各不相同——
按 cwd 解析的话 `output/` 和 `data/` 会落到谁也想不到的地方，**而且不报错**，
只是"我的素材呢"。`internal/config/config_test.go` 盯着这条。

## 起不来的时候

窗口里有个占位页，后端 90 秒内没开始监听就把原因显示在上面，并指向
`logs/dreamtexture.log`。用页面不用弹窗，是因为错误信息里有路径——弹窗里没法复制。

日志见 [backend.md](backend.md)：默认落在程序旁边的 `logs/`，那儿写不了就退到
用户缓存目录。这正是外壳最需要它的地方——没有控制台，标准输出直接丢掉。
