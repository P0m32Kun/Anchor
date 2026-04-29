import { useState, useEffect } from "react";
import { getApiBase, setApiBase, resetApiBase } from "../lib/config";

export default function SettingsPage() {
  const rawBase = getApiBase();
  const [apiBase, setApiBaseState] = useState(rawBase);
  const [saved, setSaved] = useState(false);

  const isDefaultRelative = rawBase === "/api" || rawBase === "";
  const placeholderText = isDefaultRelative
    ? "http://localhost:17421 (auto)"
    : rawBase || "http://localhost:17421";
  const [portRangePreset, setPortRangePreset] = useState<"top100" | "top1000" | "common" | "custom">("top100");
  const [customPortRange, setCustomPortRange] = useState("");
  const isTauri = !!(window as any).__TAURI__;

  useEffect(() => {
    setApiBaseState(getApiBase());
  }, []);

  const handleSave = () => {
    setApiBase(apiBase);
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
    // Force reload to pick up new API_BASE
    window.location.reload();
  };

  const handleReset = () => {
    resetApiBase();
    setApiBaseState("http://localhost:17421");
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

      <div className="liquid-glass rounded-xl p-5 space-y-4">
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
          <div className="text-sm font-medium mb-2">扫描配置</div>
          <div className="text-xs text-text-tertiary mb-2">
            端口范围设置（Project 级别）
          </div>
          <div className="space-y-3">
            <div>
              <label className="text-xs text-text-tertiary block mb-1">端口范围</label>
              <select
                className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-brand-primary/50"
                value={portRangePreset}
                onChange={(e) => setPortRangePreset(e.target.value as any)}
              >
                <option value="top100">Top 100 常用端口</option>
                <option value="top1000">Top 1000 常用端口</option>
                <option value="common">全部常见端口</option>
                <option value="custom">自定义</option>
              </select>
            </div>
            {portRangePreset === "custom" && (
              <div>
                <label className="text-xs text-text-tertiary block mb-1">自定义端口</label>
                <input
                  type="text"
                  value={customPortRange}
                  onChange={(e) => setCustomPortRange(e.target.value)}
                  placeholder="80,443,8080 或 1-65535"
                  className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-brand-primary/50"
                />
                <div className="text-xs text-text-tertiary mt-1">
                  支持格式：单端口 80，多端口 80,443,8080，范围 1-1000
                </div>
              </div>
            )}
            {/* TODO: Enable when project context is available.
              Requires: current project ID + api.updateProject(id, { port_range: ... })
              Blocked by: No global current-project selector in Zustand store.
              See: e2e/tests/SettingsPage.e2e.md Test 4
            */}
            <button
              disabled
              className="bg-brand-primary/40 text-white/60 text-sm px-4 py-2 rounded-lg cursor-not-allowed"
              title="需在项目内才能保存扫描配置"
            >
              保存扫描配置
            </button>
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
