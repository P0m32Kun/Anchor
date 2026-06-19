import { useEffect, useState, useCallback } from "react";
import { useProjectId } from "../components/ProjectLayout";
import { useNavigate } from "react-router-dom";
import { api, WatchConfig, AlertWebhook, NotificationChannel } from "../lib/api";
import {
  useToast,
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  Button,
  Input,
  Switch,
  Select,
  EmptyState,
} from "../components";
import {
  Settings,
  Bell,
  Webhook,
  Clock,
  Eye,
  EyeOff,
  Wifi,
  Server,
  Shield,
  Plus,
  Trash2,
  Save,
  RefreshCw,
} from "lucide-react";

const WATCH_INTERVALS = [
  { value: 6, label: "每 6 小时" },
  { value: 12, label: "每 12 小时" },
  { value: 24, label: "每天" },
  { value: 48, label: "每 2 天" },
  { value: 72, label: "每 3 天" },
  { value: 168, label: "每周" },
];

const SEVERITY_OPTIONS = [
  { value: "info", label: "信息" },
  { value: "low", label: "低" },
  { value: "medium", label: "中" },
  { value: "high", label: "高" },
  { value: "critical", label: "严重" },
];

export default function ProjectSettingsPage() {
  const projectId = useProjectId();
  const navigate = useNavigate();
  const toast = useToast();

  const [loading, setLoading] = useState(true);

  // Watch config
  const [watchConfig, setWatchConfig] = useState<WatchConfig | null>(null);
  const [watchSaving, setWatchSaving] = useState(false);

  // Alert webhook
  const [alertWebhook, setAlertWebhook] = useState<AlertWebhook | null>(null);
  const [webhookSaving, setWebhookSaving] = useState(false);
  const [showSecret, setShowSecret] = useState(false);

  // Notification channels
  const [channels, setChannels] = useState<NotificationChannel[]>([]);
  const [newChannel, setNewChannel] = useState({ name: "", url: "" });
  const [addingChannel, setAddingChannel] = useState(false);

  const fetchData = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    try {
      const [wc, wh, chs] = await Promise.all([
        api.getWatchConfig(projectId).catch(() => null),
        api.getAlertWebhook(projectId).catch(() => null),
        api.listNotificationChannels(projectId).catch(() => []),
      ]);
      setWatchConfig(wc);
      setAlertWebhook(wh);
      setChannels(chs);
    } catch {
      toast("加载设置失败", "error");
    } finally {
      setLoading(false);
    }
  }, [projectId, toast]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const saveWatchConfig = async () => {
    if (!projectId || !watchConfig) return;
    setWatchSaving(true);
    try {
      const updated = await api.updateWatchConfig(projectId, {
        watch_enabled: watchConfig.watch_enabled,
        watch_interval_hours: watchConfig.watch_interval_hours,
        watch_passive_only: watchConfig.watch_passive_only,
      });
      setWatchConfig(updated);
      toast("监控配置已保存", "success");
    } catch {
      toast("保存监控配置失败", "error");
    } finally {
      setWatchSaving(false);
    }
  };

  const saveAlertWebhook = async () => {
    if (!projectId || !alertWebhook) return;
    setWebhookSaving(true);
    try {
      const updated = await api.upsertAlertWebhook(projectId, {
        enabled: alertWebhook.enabled,
        url: alertWebhook.url,
        secret: alertWebhook.secret,
        min_severity: alertWebhook.min_severity,
        on_new_asset: alertWebhook.on_new_asset,
        on_asset_gone: alertWebhook.on_asset_gone,
        on_port_change: alertWebhook.on_port_change,
        on_service_change: alertWebhook.on_service_change,
        on_cert_expiry: alertWebhook.on_cert_expiry,
      });
      setAlertWebhook(updated);
      toast("告警 Webhook 已保存", "success");
    } catch {
      toast("保存告警 Webhook 失败", "error");
    } finally {
      setWebhookSaving(false);
    }
  };

  const deleteAlertWebhook = async () => {
    if (!projectId) return;
    try {
      await api.deleteAlertWebhook(projectId);
      setAlertWebhook(null);
      toast("告警 Webhook 已删除", "success");
    } catch {
      toast("删除告警 Webhook 失败", "error");
    }
  };

  const addChannel = async () => {
    if (!projectId || !newChannel.name.trim() || !newChannel.url.trim()) {
      toast("请填写通知渠道名称和 URL", "warning");
      return;
    }
    setAddingChannel(true);
    try {
      const ch = await api.createNotificationChannel(projectId, {
        name: newChannel.name.trim(),
        url: newChannel.url.trim(),
      });
      setChannels((prev) => [...prev, ch]);
      setNewChannel({ name: "", url: "" });
      toast("通知渠道已添加", "success");
    } catch {
      toast("添加通知渠道失败", "error");
    } finally {
      setAddingChannel(false);
    }
  };

  const toggleChannelEnabled = async (ch: NotificationChannel) => {
    if (!projectId) return;
    try {
      const updated = await api.updateNotificationChannel(projectId, ch.id, {
        enabled: !ch.enabled,
      });
      setChannels((prev) => prev.map((c) => (c.id === ch.id ? updated : c)));
    } catch {
      toast("更新通知渠道失败", "error");
    }
  };

  const deleteChannel = async (channelId: string) => {
    if (!projectId) return;
    try {
      await api.deleteNotificationChannel(projectId, channelId);
      setChannels((prev) => prev.filter((c) => c.id !== channelId));
      toast("通知渠道已删除", "success");
    } catch {
      toast("删除通知渠道失败", "error");
    }
  };

  if (!projectId) {
    return (
      <EmptyState
        title="未选择项目"
        description="请先在左侧导航栏选择一个项目"
        actionLabel="返回项目列表"
        onAction={() => navigate("/projects")}
      />
    );
  }

  if (loading) {
    return (
      <div className="space-y-6 animate-in fade-in duration-500">
        <div className="h-8 w-48 bg-muted rounded animate-pulse" />
        <div className="grid gap-6">
          {[1, 2, 3].map((i) => (
            <Card key={i}>
              <CardHeader>
                <div className="h-6 w-40 bg-muted rounded animate-pulse" />
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="h-10 bg-muted rounded animate-pulse" />
                  <div className="h-10 bg-muted rounded animate-pulse" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <Settings className="h-6 w-6 text-primary" />
            项目设置
          </h1>
          <p className="text-muted-foreground mt-1">
            持续资产监控 — 配置定时扫描策略和告警通知渠道
          </p>
        </div>
        <Button variant="secondary" size="sm" onClick={fetchData}>
          <RefreshCw className="h-4 w-4 mr-2" />
          刷新
        </Button>
      </div>

      {/* Watch Configuration */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Clock className="h-5 w-5 text-cyan-400" />
            <CardTitle>持续监控策略</CardTitle>
          </div>
          <CardDescription>
            启用后系统将定期自动扫描，检测资产变化并生成告警信号
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="flex items-center justify-between p-4 rounded-lg bg-muted/30">
            <div>
              <div className="font-medium text-sm">启用持续监控</div>
              <div className="text-xs text-muted-foreground mt-0.5">
                {watchConfig?.watch_enabled
                  ? "系统将按设定的间隔自动扫描"
                  : "关闭后不会自动触发扫描"}
              </div>
            </div>
            <Switch
              checked={watchConfig?.watch_enabled ?? false}
              onCheckedChange={(v) =>
                setWatchConfig((prev) => (prev ? { ...prev, watch_enabled: v } : prev))
              }
            />
          </div>

          {watchConfig?.watch_enabled && (
            <>
              <div className="space-y-2">
                <label className="text-sm font-medium">扫描间隔</label>
                <Select
                  value={String(watchConfig.watch_interval_hours ?? 24)}
                  onChange={(e) =>
                    setWatchConfig((prev) =>
                      prev
                        ? { ...prev, watch_interval_hours: Number(e.target.value) }
                        : prev
                    )
                  }
                >
                  {WATCH_INTERVALS.map((w) => (
                    <option key={w.value} value={String(w.value)}>
                      {w.label}
                    </option>
                  ))}
                </Select>
              </div>

              <div className="flex items-center justify-between p-4 rounded-lg bg-muted/30">
                <div>
                  <div className="font-medium text-sm">仅被动搜索</div>
                  <div className="text-xs text-muted-foreground mt-0.5">
                    关闭后扫描将包含主动探测（端口扫描、目录爆破等）
                  </div>
                </div>
                <Switch
                  checked={watchConfig.watch_passive_only}
                  onCheckedChange={(v) =>
                    setWatchConfig((prev) =>
                      prev ? { ...prev, watch_passive_only: v } : prev
                    )
                  }
                />
              </div>
            </>
          )}

          <div className="flex items-center gap-3 pt-2">
            <Button
              variant="primary"
              onClick={saveWatchConfig}
              loading={watchSaving}
            >
              <Save className="h-4 w-4 mr-2" />
              保存监控配置
            </Button>
            {watchConfig?.watch_last_tick_at && (
              <span className="text-xs text-muted-foreground">
                上次扫描: {new Date(watchConfig.watch_last_tick_at).toLocaleString()}
              </span>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Alert Webhook */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Webhook className="h-5 w-5 text-emerald-400" />
            <CardTitle>告警 Webhook</CardTitle>
          </div>
          <CardDescription>
            资产变化时向指定 URL 发送 JSON 通知（资产新增/消失、端口变更、证书过期等）
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          {alertWebhook ? (
            <>
              <div className="flex items-center justify-between p-4 rounded-lg bg-muted/30">
                <div>
                  <div className="font-medium text-sm">启用告警</div>
                  <div className="text-xs text-muted-foreground mt-0.5">
                    关闭后不会发送告警通知
                  </div>
                </div>
                <Switch
                  checked={alertWebhook.enabled}
                  onCheckedChange={(v) =>
                    setAlertWebhook((prev) =>
                      prev ? { ...prev, enabled: v } : prev
                    )
                  }
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">Webhook URL</label>
                <Input
                  type="url"
                  placeholder="https://hooks.example.com/anchor"
                  value={alertWebhook.url}
                  onChange={(e) =>
                    setAlertWebhook((prev) =>
                      prev ? { ...prev, url: e.target.value } : prev
                    )
                  }
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">签名密钥 (可选)</label>
                <div className="relative">
                  <Input
                    type={showSecret ? "text" : "password"}
                    placeholder="webhook secret for HMAC signature"
                    value={alertWebhook.secret ?? ""}
                    onChange={(e) =>
                      setAlertWebhook((prev) =>
                        prev ? { ...prev, secret: e.target.value } : prev
                      )
                    }
                  />
                  <button
                    type="button"
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                    onClick={() => setShowSecret(!showSecret)}
                  >
                    {showSecret ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">最低严重级别</label>
                <Select
                  value={alertWebhook.min_severity}
                  onChange={(e) =>
                    setAlertWebhook((prev) =>
                      prev ? { ...prev, min_severity: e.target.value } : prev
                    )
                  }
                >
                  {SEVERITY_OPTIONS.map((s) => (
                    <option key={s.value} value={s.value}>
                      {s.label}
                    </option>
                  ))}
                </Select>
              </div>

              <div className="space-y-3">
                <label className="text-sm font-medium">触发事件</label>
                <div className="grid gap-2">
                  {[
                    { key: "on_new_asset", label: "新增资产", icon: Eye },
                    { key: "on_asset_gone", label: "资产消失", icon: EyeOff },
                    { key: "on_port_change", label: "端口变更", icon: Wifi },
                    { key: "on_service_change", label: "服务变更", icon: Server },
                    { key: "on_cert_expiry", label: "证书过期", icon: Shield },
                  ].map(({ key, label, icon: Icon }) => (
                    <div
                      key={key}
                      className="flex items-center justify-between p-3 rounded-lg bg-muted/30"
                    >
                      <div className="flex items-center gap-2">
                        <Icon className="h-4 w-4 text-muted-foreground" />
                        <span className="text-sm">{label}</span>
                      </div>
                      <Switch
                        checked={!!(alertWebhook as any)[key]}
                        onCheckedChange={(v) =>
                          setAlertWebhook((prev) =>
                            prev ? { ...prev, [key]: v } : prev
                          )
                        }
                      />
                    </div>
                  ))}
                </div>
              </div>

              <div className="flex items-center gap-3 pt-2">
                <Button variant="primary" onClick={saveAlertWebhook} loading={webhookSaving}>
                  <Save className="h-4 w-4 mr-2" />
                  保存 Webhook
                </Button>
                <Button variant="secondary" onClick={deleteAlertWebhook}>
                  <Trash2 className="h-4 w-4 mr-2" />
                  删除
                </Button>
              </div>
            </>
          ) : (
            <div className="text-center py-4">
              <Webhook className="h-8 w-8 text-muted-foreground mx-auto mb-2 opacity-30" />
              <p className="text-xs text-muted-foreground mb-4">
                尚未配置告警 Webhook
              </p>
              <Button
                variant="secondary"
                onClick={() =>
                  setAlertWebhook({
                    id: "",
                    project_id: projectId,
                    enabled: false,
                    url: "",
                    secret: "",
                    min_severity: "info",
                    on_new_asset: true,
                    on_asset_gone: true,
                    on_port_change: true,
                    on_service_change: true,
                    on_cert_expiry: true,
                  })
                }
              >
                <Plus className="h-4 w-4 mr-2" />
                配置 Webhook
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Notification Channels */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Bell className="h-5 w-5 text-amber-400" />
            <CardTitle>通知渠道</CardTitle>
          </div>
          <CardDescription>
            管理多个 Webhook 通知渠道，每个渠道独立接收扫描完成摘要和信号通知
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {channels.length > 0 && (
            <div className="space-y-2">
              {channels.map((ch) => (
                <div
                  key={ch.id}
                  className="flex items-center justify-between p-3 rounded-lg bg-muted/30 border border-border/50"
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-sm truncate">
                        {ch.name}
                      </span>
                      <span
                        className={`inline-block h-2 w-2 rounded-full ${
                          ch.enabled ? "bg-emerald-400" : "bg-muted-foreground"
                        }`}
                      />
                    </div>
                    <div className="text-xs text-muted-foreground mt-0.5 truncate">
                      {ch.url}
                    </div>
                  </div>
                  <div className="flex items-center gap-2 shrink-0 ml-4">
                    <Switch
                      checked={ch.enabled}
                      onCheckedChange={() => toggleChannelEnabled(ch)}
                    />
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-8 w-8 p-0 text-muted-foreground hover:text-destructive"
                      onClick={() => deleteChannel(ch.id)}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {channels.length === 0 && !addingChannel && (
            <div className="text-center py-4">
              <Bell className="h-8 w-8 text-muted-foreground mx-auto mb-2 opacity-30" />
              <p className="text-xs text-muted-foreground mb-4">
                尚未配置通知渠道，添加后会在每次自动扫描完成后推送通知
              </p>
            </div>
          )}

          <div className="flex items-end gap-3 pt-2 border-t border-border">
            <div className="flex-1 space-y-1">
              <label className="text-xs font-medium text-muted-foreground">
                名称
              </label>
              <Input
                placeholder="例如: 企业微信"
                value={newChannel.name}
                onChange={(e) =>
                  setNewChannel((prev) => ({ ...prev, name: e.target.value }))
                }
              />
            </div>
            <div className="flex-[3] space-y-1">
              <label className="text-xs font-medium text-muted-foreground">
                Webhook URL
              </label>
              <Input
                placeholder="https://hooks.example.com/anchor"
                value={newChannel.url}
                onChange={(e) =>
                  setNewChannel((prev) => ({ ...prev, url: e.target.value }))
                }
              />
            </div>
            <Button
              variant="primary"
              onClick={addChannel}
              loading={addingChannel}
            >
              <Plus className="h-4 w-4 mr-2" />
              添加
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
