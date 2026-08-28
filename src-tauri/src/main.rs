// DreamTexture 的桌面外壳。
//
// 它只做三件事：把 Go 后端作为 sidecar 拉起来、等它开始监听、然后把窗口
// 导航过去。界面本身仍然由后端在 127.0.0.1 上提供——这里没有第二份前端，
// 也没有 IPC，纯粹是个窗口。
//
// 这么分是因为后端已经是这个应用的主体了：它管着 ComfyUI 子进程、任务队列、
// SQLite 和 WebSocket。外壳再去管一遍只会多一条要对齐的进程链。

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::net::{SocketAddr, TcpStream};
use std::path::PathBuf;
use std::sync::Mutex;
use std::time::{Duration, Instant};

use tauri::{Manager, RunEvent, WebviewWindow};
use tauri_plugin_shell::process::CommandChild;
use tauri_plugin_shell::ShellExt;

/// 后端起不来时等多久放弃。
///
/// 给到 90 秒是因为首次启动要建库、扫模型；ComfyUI 本身起得慢不影响这里，
/// 后端不再等它（见 cmd/dreamtexture/main.go 里 sup.Start 那段注释）。
const STARTUP_TIMEOUT: Duration = Duration::from_secs(90);

/// 拉起来的后端。附着到已有实例时是 None——不是我们起的，就不该由我们杀。
struct Backend(Mutex<Option<CommandChild>>);

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(Backend(Mutex::new(None)))
        .setup(|app| {
            let handle = app.handle().clone();
            let window: WebviewWindow = app
                .get_webview_window("main")
                .expect("主窗口不见了，检查 tauri.conf.json");

            let addr = backend_addr(app.handle());
            let url = format!("http://{addr}");

            // 已经有一个在跑就直接用它。等于顺手做了单实例：
            // 再点一次图标只会多开一个窗口，而不是抢 8777 端口然后双双失败。
            let already_up = port_open(&addr);
            if !already_up {
                match spawn_backend(app.handle()) {
                    Ok(child) => {
                        *handle.state::<Backend>().0.lock().unwrap() = Some(child);
                    }
                    Err(e) => {
                        fatal(&window, &format!("后端启动失败：{e}"));
                        return Ok(());
                    }
                }
            }

            // 等它开始监听。窗口这时还是隐藏的，占位页只在出错时才露面。
            let deadline = Instant::now() + STARTUP_TIMEOUT;
            let win = window.clone();
            std::thread::spawn(move || {
                while Instant::now() < deadline {
                    if port_open(&addr) {
                        let _ = win.navigate(url.parse().expect("拼出来的地址不合法"));
                        let _ = win.show();
                        let _ = win.set_focus();
                        return;
                    }
                    std::thread::sleep(Duration::from_millis(200));
                }
                fatal(
                    &win,
                    &format!(
                        "后端在 {} 秒内没有开始监听 {addr}。\\n\
                         日志在程序旁边的 logs/dreamtexture.log 里。",
                        STARTUP_TIMEOUT.as_secs()
                    ),
                );
            });
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("Tauri 初始化失败")
        .run(|app, event| {
            // 关窗口就退出，顺手把后端带走——它再往下会把 ComfyUI 也收干净
            // （Windows 上用的是 Job Object，见 internal/comfy）。
            //
            // 附着到已有实例时这里是 None，什么都不做：别人起的进程不归我们杀。
            if let RunEvent::Exit = event {
                if let Some(child) = app.state::<Backend>().0.lock().unwrap().take() {
                    let _ = child.kill();
                }
            }
        });
}

/// spawn_backend 拉起 sidecar。
fn spawn_backend(app: &tauri::AppHandle) -> Result<CommandChild, String> {
    let mut cmd = app
        .shell()
        .sidecar("dreamtexture")
        .map_err(|e| format!("找不到 sidecar：{e}"))?;

    if let Some(cfg) = config_path(app) {
        cmd = cmd.args(["-config", &cfg.to_string_lossy()]);
    }
    let (_rx, child) = cmd.spawn().map_err(|e| e.to_string())?;
    Ok(child)
}

/// backend_addr 读配置里的监听地址，读不到就用默认值。
///
/// 必须读：用户可能把 addr 改掉了，外壳还傻等 8777 的话就永远起不来，
/// 而且报的错会完全指错方向。
fn backend_addr(app: &tauri::AppHandle) -> String {
    const DEFAULT: &str = "127.0.0.1:8777";
    let Some(path) = config_path(app) else {
        return DEFAULT.into();
    };
    let Ok(text) = std::fs::read_to_string(path) else {
        return DEFAULT.into();
    };
    let Ok(v) = serde_json::from_str::<serde_json::Value>(&text) else {
        return DEFAULT.into();
    };
    v.get("addr")
        .and_then(|a| a.as_str())
        .filter(|a| !a.is_empty())
        .unwrap_or(DEFAULT)
        .to_string()
}

/// config_path 找配置文件：先看程序旁边，再看打包进来的资源。
///
/// 顺序不能反。装好之后用户改的是程序旁边那一份，资源目录里那份是安装包
/// 带的初始值——反过来的话，用户改完发现没生效。
fn config_path(app: &tauri::AppHandle) -> Option<PathBuf> {
    if let Ok(exe) = std::env::current_exe() {
        if let Some(dir) = exe.parent() {
            let p = dir.join("configs").join("dreamtexture.json");
            if p.exists() {
                return Some(p);
            }
        }
    }
    let p = app
        .path()
        .resolve("configs/dreamtexture.json", tauri::path::BaseDirectory::Resource)
        .ok()?;
    p.exists().then_some(p)
}

/// port_open 只探端口通不通，不发 HTTP 请求——为此拉一个 HTTP 客户端进来不值。
/// 后端一旦开始监听，界面就能加载了。
fn port_open(addr: &str) -> bool {
    let Ok(sa) = addr.parse::<SocketAddr>() else {
        return false;
    };
    TcpStream::connect_timeout(&sa, Duration::from_millis(300)).is_ok()
}

/// fatal 把错误显示在占位页上并把窗口露出来。
///
/// 不用弹窗：错误信息里有路径，弹窗里没法复制。
fn fatal(win: &WebviewWindow, msg: &str) {
    let js = format!(
        r#"document.querySelector('.box').innerHTML =
             '<div class="t">DreamTexture 起不来</div><div style="white-space:pre-wrap">'
             + {} + '</div>';"#,
        serde_json::to_string(msg).unwrap_or_else(|_| "\"未知错误\"".into())
    );
    let _ = win.eval(&js);
    let _ = win.show();
    let _ = win.set_focus();
}
