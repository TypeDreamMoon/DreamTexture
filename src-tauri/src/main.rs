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

use tauri::webview::{NewWindowResponse, WebviewWindowBuilder};
use tauri::{Manager, RunEvent, WebviewUrl, WebviewWindow};
use tauri_plugin_opener::OpenerExt;
use tauri_plugin_shell::process::CommandChild;
use tauri_plugin_shell::ShellExt;
use tauri::Url;

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
        .plugin(tauri_plugin_opener::init())
        .manage(Backend(Mutex::new(None)))
        .setup(|app| {
            let handle = app.handle().clone();
            let addr = backend_addr(&handle);
            let url = format!("http://{addr}");

            // 已经有一个在跑就直接用它。等于顺手做了单实例：
            // 再点一次图标只会多开一个窗口，而不是抢端口然后双双失败。
            let already_up = port_open(&addr);
            if !already_up {
                match spawn_backend(&handle) {
                    Ok(child) => {
                        *handle.state::<Backend>().0.lock().unwrap() = Some(child);
                    }
                    Err(e) => {
                        let win = build_window(&handle, &url)?;
                        fatal(&win, &format!("后端启动失败：{e}"));
                        return Ok(());
                    }
                }
            }

            let window = build_window(&handle, &url)?;

            // 等它开始监听。窗口这时还是隐藏的，占位页只在出错时才露面。
            let deadline = Instant::now() + STARTUP_TIMEOUT;
            let win = window.clone();
            std::thread::spawn(move || {
                while Instant::now() < deadline {
                    if port_open(&addr) {
                        if let Ok(u) = url.parse::<Url>() {
                            let _ = win.navigate(u);
                        }
                        let _ = win.show();
                        let _ = win.set_focus();
                        return;
                    }
                    std::thread::sleep(Duration::from_millis(200));
                }
                fatal(
                    &win,
                    &format!(
                        "后端在 {} 秒内没有开始监听 {addr}。\n日志在程序旁边的 logs/dreamtexture.log 里。",
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

/// build_window 建主窗口，并把外链的去处定好。
///
/// 窗口在这里建而不是写在 tauri.conf.json 里，就是为了挂下面两个钩子——
/// 配置声明的窗口没地方挂。
fn build_window(app: &tauri::AppHandle, origin: &str) -> tauri::Result<WebviewWindow> {
    let own = origin.to_string();
    let app_for_new = app.clone();
    let app_for_nav = app.clone();
    let own_for_nav = own.clone();

    WebviewWindowBuilder::new(app, "main", WebviewUrl::App("index.html".into()))
        .title("DreamTexture")
        .inner_size(1440.0, 900.0)
        .min_inner_size(1024.0, 640.0)
        .center()
        .visible(false)
        // target="_blank" 和 window.open 走这里。
        //
        // 窗口里没有浏览器的标签页，默认行为是什么都不发生——界面上那个
        // 「打开 ComfyUI」点下去毫无反应。一律交给系统浏览器：ComfyUI 的
        // 节点编辑器是个完整的应用，本来也不该塞进我们这个窗口里。
        .on_new_window(move |url, _features| {
            open_outside(&app_for_new, url.as_str());
            NewWindowResponse::Deny
        })
        // 同窗口内跳到站外也一样处理。这里同时是一道边界：这个 webview 只该
        // 显示我们自己的后端，别的地址一概不在窗口里打开。
        .on_navigation(move |url| {
            if is_ours(url, &own_for_nav) {
                return true;
            }
            open_outside(&app_for_nav, url.as_str());
            false
        })
        .build()
}

/// is_ours 判断这个地址是不是"我们自己的页面"。
///
/// 两类：后端提供的界面，以及 Tauri 自己那份占位页（协议是 tauri://，
/// Windows 上表现为 http://tauri.localhost）。别的都算站外。
fn is_ours(url: &Url, origin: &str) -> bool {
    let s = url.as_str();
    s.starts_with(origin)
        || url.scheme() == "tauri"
        || url.host_str() == Some("tauri.localhost")
        || s == "about:blank"
}

fn open_outside(app: &tauri::AppHandle, url: &str) {
    // 只放行 http(s)。别的协议（file:、自定义协议…）交给系统等于让页面
    // 有能力拉起本机程序，不值这个风险。
    if !(url.starts_with("http://") || url.starts_with("https://")) {
        return;
    }
    let _ = app.opener().open_url(url, None::<&str>);
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
        .resolve(
            "configs/dreamtexture.json",
            tauri::path::BaseDirectory::Resource,
        )
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
