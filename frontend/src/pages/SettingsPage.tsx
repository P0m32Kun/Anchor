import { useState, useEffect } from "react";
import { getApiBase, setApiBase, resetApiBase, getApiToken, setApiToken, resetApiToken } from "../lib/config";

export default function SettingsPage() {
  const rawBase = getApiBase();
  const rawToken = getApiToken();
  const [apiBase, setApiBaseState] = useState(rawBase);
  const [apiToken, setApiTokenState] = useState(rawToken);
  const [showToken, setShowToken] = useState(false);
  const [saved, setSaved] = useState(false);

  const isDefaultRelative = rawBase === "/api" || rawBase === "";
  const placeholderText = isDefaultRelative
    ? "http://localhost:17421 (auto)"
    : rawBase || "http://localhost:17421";
  const isTauri = !!(window as any).__TAURI__;

  useEffect(() => {
    setApiBaseState(getApiBase());
    setApiTokenState(getApiToken());
  }, []);

  const handleSave = () => {
    setApiBase(apiBase);
    setApiToken(apiToken);
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
    // Force reload to pick up new API_BASE and API token
    window.location.reload();
  };

  const handleReset = () => {
    resetApiBase();
    resetApiToken();
    setApiBaseState("http://localhost:17421");
    setApiTokenState("");
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
    window.location.reload();
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold">Settings</h1>
        <p className="text-sm text-text-tertiary mt-1">
          应用配置和偏好设置
        </p>
      </div>

      <div className="cyber-glass p-5 space-y-4">
        {/* Server URL */}
        <div>
          <div className="text-sm font-medium mb-2">Server 地址</div>
          <div className="text-xs text-text-tertiary mb-2">
            {isTauri
              ? "桌面模式：可连接远程 Server，或使用本地内置 Server"
              : "Web 模式：输入 Anchor Server 的地址"}
          </div>
          <div className="flex gap-2">
            <input
              type="text"
              value={apiBase}
              onChange={(e) => setApiBaseState(e.target.value)}
              placeholder={placeholderText}
              className="flex-1 bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-brand-primary/50"
            />
            <button
              onClick={handleSave}
              className="bg-brand-primary text-white text-sm px-4 py-2 rounded-lg hover:opacity-90 transition-opacity"
            >
              {saved ? "已保存 ✓" : "保存并刷新"}
            </button>
            <button
              onClick={handleReset}
              className="bg-white/5 text-text-secondary text-sm px-4 py-2 rounded-lg hover:bg-white/10 transition-colors"
            >
              重置
            </button>
          </div>
          {isDefaultRelative && (
            <div className="text-xs text-text-tertiary mt-1.5">
              当前实际 API Base：{rawBase}（Vite proxy 自动转发到 http://localhost:17421）
            </div>
          )}
        </div>

        <div className="border-t border-white/[0.06] pt-4">
          <div className="text-sm font-medium mb-2">API Token</div>
          <div className="text-xs text-text-tertiary mb-2">
            连接 Server 所需的认证 Token（由 Server 管理员提供）
          </div>

          <div className="flex gap-2">
            <div className="flex-1 relative">
              <input
                type={showToken ? "text" : "password"}
                value={apiToken}
                onChange={(e) => setApiTokenState(e.target.value)}
                placeholder="输入新的 API Token"
                className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-brand-primary/50 pr-10"
              />
              <button
                type="button"
                onClick={() => setShowToken((s) => !s)}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-text-tertiary hover:text-text-secondary"
                title={showToken ? "隐藏" : "显示"}
              >
                {showToken ? "🙈" : "👁️"}
              </button>
            </div>
          </div>
        </div>

        {isTauri && (
          <>
            <div className="border-t border-white/[0.06] pt-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm font-medium">本地 Worker 自动启动</div>
                  <div className="text-xs text-text-tertiary mt-0.5">
                    应用启动时自动启动本地 Worker（仅本地模式）
                  </div>
                </div>
                {/* TODO: Static UI only — no state binding or click handler.
                  Requires: Tauri config store or backend preference API.
                  Currently always shows "ON" with no way to toggle.
                  See: e2e/tests/SettingsPage.e2e.md Test 4
                */}
                <div className="w-10 h-5 bg-brand-primary rounded-full relative cursor-pointer">
                  <div className="w-4 h-4 bg-white rounded-full absolute right-0.5 top-0.5" />
                </div>
              </div>
            </div>

            <div className="border-t border-white/[0.06] pt-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm font-medium">数据目录</div>
                  <div className="text-xs text-text-tertiary mt-0.5 font-mono">
                    ~/.anchor
                  </div>
                </div>
              </div>
            </div>
          </>
        )}

        <div className="border-t border-white/[0.06] pt-4">
          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm font-medium">版本</div>
              <div className="text-xs text-text-tertiary mt-0.5">
                v0.2.0
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
