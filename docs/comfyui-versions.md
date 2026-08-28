# ComfyUI 版本管理

设置页「ComfyUI 版本」那一节：看当前装的是哪个、能切到哪些、切过去。
稳定版（tag）与开发版（主分支提交）分两个页签，和秋叶启动器那一页是一个意思。

## 分工

| | |
|---|---|
| `internal/comfyver` | 只做 git 那层：读状态、列版本、补历史、checkout |
| `internal/deploy` 的 `Switch` | 切换 → **重装依赖** → 验证，复用部署那套步骤机制 |

**重装依赖不是可选项。** ComfyUI 的 `requirements.txt` 在版本之间是会变的，
只 checkout 不装依赖，轻则某些节点报缺包，重则一起来就崩——而症状看着完全不像
"版本切歪了"。装完还要过一遍 `ensureCUDATorch`：ComfyUI 的 requirements 里列着
torch，而 PyPI 上的是 CPU 版，会把 CUDA 版顶掉（[部署那篇](deploy.md)里同一个坑）。

## 浅克隆是这件事最大的障碍

一键部署原本用的是 `git clone --depth 1`，只有一个提交、没有 tag。
实测那份运行时：

```
git describe --tags   → fatal: No names found
rev-parse --is-shallow-repository → true
```

**一个版本都列不出来，而且不是 bug，是当时的部署方式把这条路堵上了。**

现在的做法是 `Fetch()`：浅克隆时 `git fetch --filter=blob:none --unshallow --tags`,
只补提交图和 tag，不拉旧文件。实测那份运行时 **7 秒**、`.git` 从 12 MB 到 26 MB，
补完列出 179 个稳定版。服务端不支持部分克隆就退回老实全拉。

界面上浅克隆会单独提示并给一个「补齐历史」按钮——否则用户只看到一张空表，
不知道该干嘛。

## 几个刻意的选择

**用游离头指针（detached HEAD），不建分支。** 我们要的是"钉在这个版本上"，
建一堆本地分支只会让用户之后自己用 git 时一头雾水。

**工作区脏就拦住。** ComfyUI 目录里有未提交的改动时，checkout 会把它们冲掉。
界面上把改动的文件名列出来，让用户知道是什么挡住了，而不是只说"失败"。

**tag 要解引用一次。** 附注标签指向的是 tag 对象不是 commit，不解引用的话
"当前"那一列永远对不上。

**开发版按日期重排。** `git log` 默认是拓扑序，遇到合并会把时间穿插开，
而界面上"最新的在上面"是硬预期。

**只列最近 60 条开发版**，要更早的提交就用下面那个手动输入框填提交号。

**仓库位置由后端从配置里的 `main_py` 推出来，不接受客户端指定**——
那个值决定了我们去哪个目录跑 `git checkout`，让客户端说了算等于开了个后门。

## 和素材的关系

`manifest.json` 的 `generator.comfyui` 记了出图时用的版本，所以换版本之后
老素材仍然知道自己是哪个版本产的。
