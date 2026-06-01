const STORAGE_KEY = "anchor_api_base";
const TOKEN_KEY = "anchor_api_token";

export function isTauriEnv(): boolean {
  // Tauri v1: window.__TAURI__, Tauri v2: globalThis.isTauri
  return !!(window as any).__TAURI__ || !!(globalThis as any).isTauri;
}

export function getApiBase(): string {
  return localStorage.getItem(STORAGE_KEY) || "";
}

export function needsApiBaseConfig(): boolean {
  return !localStorage.getItem(STORAGE_KEY);
}

export function setApiBase(url: string) {
  localStorage.setItem(STORAGE_KEY, url.replace(/\/$/, ""));
}

export function resetApiBase() {
  localStorage.removeItem(STORAGE_KEY);
}

export function getApiToken(): string {
  return localStorage.getItem(TOKEN_KEY) || "";
}

export function setApiToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token);
}

export function resetApiToken() {
  localStorage.removeItem(TOKEN_KEY);
}

export function needsApiToken(): boolean {
  return !localStorage.getItem(TOKEN_KEY);
}

/**
 * 尝试从 Tauri 自动连接文件中读取配置。
 * install.sh 在打开桌面 App 前会写入 /tmp/anchor-auto-connect.json。
 * 读取成功后写入 localStorage，后续启动不再需要。
 * 返回 true 表示成功设置了连接信息。
 */
export async function tryAutoConnect(): Promise<boolean> {
  if (!isTauriEnv()) return false;
  if (!needsApiBaseConfig() && !needsApiToken()) return false;

  try {
    const { invoke } = await import("@tauri-apps/api/core");
    const result = await invoke<{ api_base: string; api_token: string } | null>(
      "read_auto_connect"
    );
    if (result) {
      setApiBase(result.api_base);
      setApiToken(result.api_token);
      return true;
    }
  } catch (e) {
    console.warn("[auto-connect] 读取自动连接配置失败:", e);
  }
  return false;
}

export { STORAGE_KEY, TOKEN_KEY };
