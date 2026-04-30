import { useState } from "react";

function validateSetupInput(url: string, token: string): string | null {
  const trimmedUrl = url.trim().replace(/\/$/, "");
  if (!trimmedUrl) return "请输入 Server 地址";
  if (!token.trim()) return "请输入 API Token";
  if (!trimmedUrl.startsWith("http://") && !trimmedUrl.startsWith("https://")) {
    return "地址必须以 http:// 或 https:// 开头";
  }
  return null;
}

export default function ApiBaseSetup() {
  const [url, setUrl] = useState("");
  const [token, setToken] = useState("");
  const [showToken, setShowToken] = useState(false);
  const [error, setError] = useState("");
  const [testing, setTesting] = useState(false);

  const handleSave = async () => {
    const validationError = validateSetupInput(url, token);
    if (validationError) {
      setError(validationError);
      return;
    }

    const trimmedUrl = url.trim().replace(/\/$/, "");
    const trimmedToken = token.trim();

    setTesting(true);
    setError("");
    try {
      const headers = { Authorization: `Bearer ${trimmedToken}` };
      const res = await fetch(`${trimmedUrl}/health`, { method: "GET", mode: "cors", headers });
      if (res.status === 401) {
        setError("Token 无效，请检查输入的 API Token");
        return;
      }
      if (!res.ok) {
        setError(`Server 返回 HTTP ${res.status}，请确认地址正确`);
        return;
      }
      localStorage.setItem("anchor_api_base", trimmedUrl);
      localStorage.setItem("anchor_api_token", trimmedToken);
      window.location.reload();
    } catch (e: any) {
      setError(`无法连接到 Server：${e?.message || "网络错误"}`);
    } finally {
      setTesting(false);
    }
  };

  return (
    <div className="flex flex-col items-center justify-center h-screen bg-surface text-text-primary px-4">
      <div className="text-4xl mb-4">⚓</div>
      <h1 className="text-xl font-semibold mb-2">欢迎使用 Anchor</h1>
      <p className="text-text-secondary mb-6 text-sm text-center max-w-sm">
        请配置 Anchor Server 地址以继续。
      </p>

      <div className="w-full max-w-sm space-y-3">
        <input
          type="text"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="http://localhost:17421"
          className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:border-brand-primary/50"
          onKeyDown={(e) => e.key === "Enter" && handleSave()}
        />
        <div className="relative">
          <input
            type={showToken ? "text" : "password"}
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="请输入 Server 管理员提供的 API Token"
            className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:border-brand-primary/50 pr-10"
            onKeyDown={(e) => e.key === "Enter" && handleSave()}
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
        {error && (
          <div className="text-xs text-red-400">{error}</div>
        )}
        <button
          onClick={handleSave}
          disabled={testing}
          className="w-full bg-brand-primary text-white text-sm px-4 py-2.5 rounded-lg hover:opacity-90 transition-opacity disabled:opacity-50"
        >
          {testing ? "连接测试中..." : "保存并进入"}
        </button>
      </div>
    </div>
  );
}
