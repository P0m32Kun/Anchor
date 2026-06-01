const STORAGE_KEY = "anchor_api_base";
const TOKEN_KEY = "anchor_api_token";
const DEFAULT_API_BASE = "/api";

export function getApiBase(): string {
  return localStorage.getItem(STORAGE_KEY) || DEFAULT_API_BASE;
}

/**
 * 是否需要用户配置 API 地址。
 * 生产环境（nginx 反代）默认 /api，不需要配置；
 * 只有 localStorage 完全为空且不是默认值时才需要。
 */
export function needsApiBaseConfig(): boolean {
  return false;
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
 * 尝试从 URL query params 中读取自动连接配置。
 * install.sh 完成后输出的浏览器地址会带上 api_base 和 api_token。
 * 读取成功后写入 localStorage 并清除 URL 参数。
 * 返回 true 表示成功设置了连接信息。
 */
export function tryAutoConnectFromUrl(): boolean {
  if (!needsApiBaseConfig() && !needsApiToken()) return false;

  const params = new URLSearchParams(window.location.search);
  const apiBase = params.get("api_base");
  const apiToken = params.get("api_token");

  if (apiBase) {
    setApiBase(apiBase);
    if (apiToken) setApiToken(apiToken);
    // 清除 URL 参数，避免刷新时重复写入
    const cleanUrl = window.location.pathname;
    window.history.replaceState({}, "", cleanUrl);
    return true;
  }
  return false;
}

export { STORAGE_KEY, TOKEN_KEY };
