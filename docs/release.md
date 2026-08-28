# 打包与发布

`.github/workflows/release.yml`：打 `v*` 的 tag 就出一个 Windows 安装包并落成
Release 草稿；也可以在 Actions 页手动跑，那时只上传产物不发布——想先验一下
包能不能装能不能起来，不必先把 tag 推出去。

## 产物

| | |
|---|---|
| `DreamTexture_<版本>_x64-setup.exe` | NSIS 安装包，约 13 MB |
| `dreamtexture-shell.exe` | 外壳本体，4.3 MB（调试用） |

**不含 ComfyUI，也不含模型。** 装完让用户自己走「设置 → ComfyUI 环境 →
一键部署」——那一步会用 uv 装一份独立的 ComfyUI，不动他已有的整合包。

## 发布之前修掉的两个坑

**安装包原本带的是开发机的配置。** `tauri.conf.json` 的 resources 指向
`configs/dreamtexture.json`，而那份里写着 `I:\ComfyUI\ComfyUI-aki-v3\...`，
发出去别人一装就是坏的。现在带的是 `configs/dreamtexture.default.json`——
干净模板，`mode=attach`、路径留空。

**全新安装原本过不了配置校验。** `Validate` 要求 managed 模式下 python 与
main_py 非空，而新用户正是靠一键部署来填这两个值：后端起不来 → 那个页面打不开
→ 永远填不上。现在放宽了，真到要用时由 `comfy.pathsReady` 拦，那儿给的话
也具体得多。

## 配置只在第一次铺，升级不覆盖

外壳的 `config_path`：程序旁边有 `configs/dreamtexture.json` 就用它，没有才从
资源里的模板复制一份过去。

**顺序不能反，也不能每次都铺**。指向模板的话用户改完发现没生效；每次都铺的话
重装一次设置全没了。实测验证过：改一项设置 → 重启 → 那一项还在。

## 版本号

从 tag 推出来，写进三处：Tauri 的包信息、安装包文件名、以及界面上报的后端版本
（`-ldflags -X main.version=`）。手动跑时保持仓库里写的那个。

## CI 里重建前端

`internal/web/dist` 是提交进仓库的（`go:embed` 的目标，让 clone 下来直接
`go build` 就能用），但 CI 里仍然重建一次并比对——提交的那份和源码漂了的话，
不重建是不会有人发现的。不一致会打一条 warning，用重建的那份。
