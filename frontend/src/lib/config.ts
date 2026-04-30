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

export { STORAGE_KEY, TOKEN_KEY };
