use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;

#[derive(Serialize, Deserialize)]
struct AutoConnectInfo {
    api_base: String,
    api_token: String,
}

#[tauri::command]
fn save_file(path: String, contents: Vec<u8>) -> Result<(), String> {
    std::fs::write(&path, &contents).map_err(|e| e.to_string())
}

#[tauri::command]
fn read_auto_connect() -> Result<Option<AutoConnectInfo>, String> {
    let path = PathBuf::from("/tmp/anchor-auto-connect.json");
    if !path.exists() {
        return Ok(None);
    }
    let data = fs::read_to_string(&path).map_err(|e| e.to_string())?;
    let info: AutoConnectInfo = serde_json::from_str(&data).map_err(|e| e.to_string())?;
    // 读取后删除文件，只使用一次
    let _ = fs::remove_file(&path);
    Ok(Some(info))
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init())
        .invoke_handler(tauri::generate_handler![save_file, read_auto_connect])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
