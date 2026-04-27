const STORAGE_KEY = "anchor_api_base";

export function getApiBase(): string {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored) return stored;

  // Tauri desktop: check if we are in desktop mode
  const isTauri = !!(window as any).__TAURI__;
  if (isTauri) {
    // Desktop defaults to embedded local server
    return "http://localhost:17421";
  }

  // Web: try current host (if served from same origin as API)
  return "http://localhost:17421";
}

export function setApiBase(url: string) {
  localStorage.setItem(STORAGE_KEY, url.replace(/\/$/, ""));
}

export function resetApiBase() {
  localStorage.removeItem(STORAGE_KEY);
}
