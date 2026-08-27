# tools

一次性验证脚本。不参与构建，也不进二进制。

用 ComfyUI 整合包自带的解释器跑就行（省得再配一套环境）：

```bash
I:/ComfyUI/ComfyUI-aki-v3/python/python.exe tools/<脚本>
```

| 脚本 | 用途 |
|---|---|
| `fake_openai.py` | OpenAI 图像接口的兼容桩，监听 `127.0.0.1:8899`。出图时故意带 0.28 的径向暗角，用来在**没有真实令牌**的情况下跑通整条云端底图链路——顺带验证自定义 base URL、消暗角、用量记账。见 [../docs/cloud-source.md](../docs/cloud-source.md) |
| `gen_api_workflows.py` | 从本地工作流派生出「API 底图」版本的模板。改了 `realistic-chord-v1` / `stylized-derive-v1` 的图之后重跑一次，两条云端管线就跟着更新，不用手改 JSON |

一键部署产出的运行时在 `runtime/`（已 gitignore，约 4GB）：
`runtime/venv` 是虚拟环境，`runtime/ComfyUI` 是 ComfyUI 本体。
删掉整个目录就等于回到没部署过的状态，重跑一次部署即可。见
[../docs/deploy.md](../docs/deploy.md)。

`gen_api_workflows.py` 会**覆盖** `workflows/api-*.json` 与 `api-*.params.json`，
在那两个文件里手动改的东西会丢。要长期保留的改动请改到派生脚本里。
