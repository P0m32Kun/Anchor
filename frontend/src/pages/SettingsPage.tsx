import { useState, useEffect } from "react";
import { getApiBase, setApiBase, resetApiBase, getApiToken, setApiToken, resetApiToken } from "../lib/config";
import { 
  Card, 
  CardHeader, 
  CardTitle, 
  CardDescription, 
  CardContent, 
  Button, 
  Input 
} from "../components";
import { Eye, EyeOff, Save, RotateCcw, Info, Server, Key } from "lucide-react";

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

  useEffect(() => {
    setApiBaseState(getApiBase());
    setApiTokenState(getApiToken());
  }, []);

  const handleSave = () => {
    setApiBase(apiBase);
    setApiToken(apiToken);
    setSaved(true);
    setTimeout(() => {
        setSaved(false);
        window.location.reload();
    }, 500);
  };

  const handleReset = () => {
    resetApiBase();
    resetApiToken();
    setApiBaseState("http://localhost:17421");
    setApiTokenState("");
    setSaved(true);
    setTimeout(() => {
        setSaved(false);
        window.location.reload();
    }, 500);
  };

  return (
    <div className="space-y-8 animate-in fade-in duration-500">
      <div>
        <h1 className="text-3xl font-bold tracking-tight text-foreground">设置</h1>
        <p className="text-muted-foreground mt-1">
          配置应用连接和系统偏好。
        </p>
      </div>

      <div className="grid gap-6">
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
                <Server className="h-5 w-5 text-primary" />
                <CardTitle>后端连接</CardTitle>
            </div>
            <CardDescription>
                Web 模式：设置后端 API 接口地址。
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="space-y-2">
              <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                Server 地址
              </label>
              <div className="flex gap-3">
                <Input
                  type="text"
                  value={apiBase}
                  onChange={(e) => setApiBaseState(e.target.value)}
                  placeholder={placeholderText}
                  className="max-w-md"
                />
                <Button
                  onClick={handleSave}
                  variant="primary"
                  loading={saved}
                >
                  <Save className="mr-2 h-4 w-4" />
                  {saved ? "已保存" : "保存并刷新"}
                </Button>
                <Button
                  onClick={handleReset}
                  variant="secondary"
                >
                  <RotateCcw className="mr-2 h-4 w-4" />
                  重置
                </Button>
              </div>
              {isDefaultRelative && (
                <p className="text-xs text-muted-foreground">
                  当前处于自动代理模式：{rawBase || "/api"}
                </p>
              )}
            </div>

            <div className="space-y-2 pt-4 border-t">
              <label className="text-sm font-medium leading-none flex items-center gap-2">
                <Key className="h-4 w-4 text-muted-foreground" />
                API Token
              </label>
              <p className="text-xs text-muted-foreground">
                连接 Server 所需的认证令牌（由管理员提供）。
              </p>
              <div className="relative max-w-md">
                <Input
                  type={showToken ? "text" : "password"}
                  value={apiToken}
                  onChange={(e) => setApiTokenState(e.target.value)}
                  placeholder="输入 API Token"
                  className="pr-10 font-mono"
                />
                <Button
                  variant="ghost"
                  size="sm"
                  className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                  onClick={() => setShowToken(!showToken)}
                >
                  {showToken ? (
                    <EyeOff className="h-4 w-4 text-muted-foreground" />
                  ) : (
                    <Eye className="h-4 w-4 text-muted-foreground" />
                  )}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>


        <Card className="bg-muted/30">
          <CardHeader>
            <div className="flex items-center gap-2">
                <Info className="h-5 w-5 text-muted-foreground" />
                <CardTitle className="text-base text-muted-foreground">系统信息</CardTitle>
            </div>
          </CardHeader>
          <CardContent className="space-y-2">
            <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">版本</span>
                <span className="font-medium">v0.2.0 (Standard Edition)</span>
            </div>
            <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">构建环境</span>
                <span className="font-mono text-xs">React Web</span>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
