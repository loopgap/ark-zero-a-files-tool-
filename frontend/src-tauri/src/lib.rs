use rfd::FileDialog;
use serde::Serialize;
use serde_json::{json, Value};
use std::env;
use std::fs::{self, OpenOptions};
use std::io::{BufRead, BufReader, Write};
use std::path::PathBuf;
use std::process::{Child, ChildStdin, ChildStdout, Command, Stdio};
use std::sync::Mutex;
use std::time::{SystemTime, UNIX_EPOCH};
use tauri::State;
#[cfg(windows)]
use std::os::windows::process::CommandExt;

#[cfg(windows)]
const CREATE_NO_WINDOW: u32 = 0x08000000;

#[derive(Serialize)]
struct BootstrapPayload {
    #[serde(rename = "baseUrl")]
    base_url: String,
}

struct SidecarClient {
    child: Child,
    stdin: ChildStdin,
    stdout: BufReader<ChildStdout>,
    next_id: u64,
}

impl SidecarClient {
    fn call(&mut self, method: &str, params: Value) -> Result<Value, String> {
        let request = json!({
            "jsonrpc": "2.0",
            "id": self.next_id,
            "method": method,
            "params": params
        });
        self.next_id += 1;

        writeln!(self.stdin, "{request}").map_err(|err| err.to_string())?;
        self.stdin.flush().map_err(|err| err.to_string())?;

        let mut line = String::new();
        self.stdout
            .read_line(&mut line)
            .map_err(|err| err.to_string())?;
        let response: Value = serde_json::from_str(line.trim()).map_err(|err| err.to_string())?;

        if let Some(error) = response.get("error") {
            let message = error
                .get("message")
                .and_then(Value::as_str)
                .unwrap_or("unknown sidecar error");
            return Err(message.to_string());
        }

        Ok(response.get("result").cloned().unwrap_or(Value::Null))
    }
}

impl Drop for SidecarClient {
    fn drop(&mut self) {
        let _ = self.child.kill();
        let _ = self.child.wait();
    }
}

struct AppState {
    base_url: String,
    sidecar: Mutex<SidecarClient>,
}

#[tauri::command]
fn bootstrap(state: State<'_, AppState>) -> BootstrapPayload {
    BootstrapPayload {
        base_url: state.base_url.clone(),
    }
}

#[tauri::command]
fn rpc(method: String, params: Value, state: State<'_, AppState>) -> Result<Value, String> {
    let mut client = state
        .sidecar
        .lock()
        .map_err(|_| "sidecar lock poisoned".to_string())?;
    client.call(&method, params)
}

#[tauri::command]
fn pick_directory(initial: Option<String>) -> Result<Option<String>, String> {
    let mut dialog = FileDialog::new();
    if let Some(path) = initial {
        dialog = dialog.set_directory(path);
    }
    Ok(dialog
        .pick_folder()
        .map(|path| path.to_string_lossy().to_string()))
}

fn sidecar_path_candidates() -> Vec<PathBuf> {
    let mut candidates = Vec::new();
    if let Ok(current_exe) = env::current_exe() {
        if let Some(parent) = current_exe.parent() {
            candidates.push(parent.join("arkkb-sidecar.exe"));
            candidates.push(parent.join("arkkb-sidecar"));
        }
    }

    let manifest_dir = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    candidates.push(manifest_dir.join("bin").join("arkkb-sidecar.exe"));
    candidates.push(manifest_dir.join("bin").join("arkkb-sidecar"));
    candidates.push(
        manifest_dir
            .join("..")
            .join("..")
            .join("bin")
            .join("arkkb-sidecar.exe"),
    );
    candidates.push(
        manifest_dir
            .join("..")
            .join("..")
            .join("bin")
            .join("arkkb-sidecar"),
    );
    candidates
}

fn append_desktop_log(message: &str) {
    let home = env::var_os("USERPROFILE").or_else(|| env::var_os("HOME"));
    let Some(home) = home else {
        return;
    };
    let log_path = PathBuf::from(home)
        .join(".arkkb")
        .join("logs")
        .join("desktop.log");
    if let Some(parent) = log_path.parent() {
        let _ = fs::create_dir_all(parent);
    }
    let Ok(mut file) = OpenOptions::new().create(true).append(true).open(log_path) else {
        return;
    };
    let ts = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_secs())
        .unwrap_or(0);
    let _ = writeln!(file, "[{}] {}", ts, message);
}

fn resolve_sidecar_path() -> Result<PathBuf, String> {
    let candidates = sidecar_path_candidates();
    let resolved = candidates
        .iter()
        .find(|candidate| candidate.exists() && candidate.is_file())
        .cloned();
    match resolved {
        Some(path) => {
            append_desktop_log(&format!("resolved sidecar path: {}", path.display()));
            Ok(path)
        }
        None => {
            append_desktop_log(&format!(
                "sidecar binary not found; checked: {}",
                candidates
                    .iter()
                    .map(|path| path.display().to_string())
                    .collect::<Vec<_>>()
                    .join(", ")
            ));
            Err("sidecar binary not found".to_string())
        }
    }
}

fn spawn_sidecar() -> Result<(SidecarClient, String), String> {
    let sidecar_path = resolve_sidecar_path()?;
    append_desktop_log(&format!("spawning sidecar: {}", sidecar_path.display()));
    let mut command = Command::new(&sidecar_path);
    command
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::null());
    #[cfg(windows)]
    command.creation_flags(CREATE_NO_WINDOW);
    let mut child = command
        .spawn()
        .map_err(|err| format!("spawn sidecar {}: {}", sidecar_path.display(), err))?;

    let stdin = child
        .stdin
        .take()
        .ok_or_else(|| "sidecar stdin unavailable".to_string())?;
    let stdout = child
        .stdout
        .take()
        .ok_or_else(|| "sidecar stdout unavailable".to_string())?;

    let mut client = SidecarClient {
        child,
        stdin,
        stdout: BufReader::new(stdout),
        next_id: 1,
    };

    let bootstrap = client.call("app.bootstrap", json!({}))?;
    let base_url = bootstrap
        .get("baseUrl")
        .and_then(Value::as_str)
        .ok_or_else(|| "sidecar bootstrap missing baseUrl".to_string())?
        .to_string();
    append_desktop_log(&format!("sidecar bootstrap complete; baseUrl={}", base_url));

    Ok((client, base_url))
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    append_desktop_log(&format!(
        "starting desktop shell; frontend_source={}",
        if cfg!(debug_assertions) { "devUrl" } else { "frontendDist" }
    ));
    let (client, base_url) = match spawn_sidecar() {
        Ok(payload) => payload,
        Err(err) => {
            append_desktop_log(&format!("failed to start Go sidecar: {}", err));
            panic!("failed to start Go sidecar: {}", err);
        }
    };

    tauri::Builder::default()
        .manage(AppState {
            base_url,
            sidecar: Mutex::new(client),
        })
        .plugin(
            tauri_plugin_log::Builder::default()
                .level(log::LevelFilter::Info)
                .build(),
        )
        .invoke_handler(tauri::generate_handler![bootstrap, rpc, pick_directory])
        .run(tauri::generate_context!())
        .unwrap_or_else(|err| {
            append_desktop_log(&format!("error while running tauri application: {}", err));
            panic!("error while running tauri application: {}", err);
        });
}
