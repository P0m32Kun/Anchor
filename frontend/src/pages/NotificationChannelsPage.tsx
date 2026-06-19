import { useEffect, useState, useCallback } from "react";
import { useProjectId } from "../components/ProjectLayout";
import { api, NotificationChannel, AlertWebhook } from "../lib/api";
import {
  useToast,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
  Button,
  SkeletonCard,
  EmptyState,
} from "../components";
import {
  Bell,
  BellRing,
  RefreshCw,
  Plus,
  Trash2,
  Webhook,
  Link,
  Key,
  Shield,
  Box,
  Wifi,
  Server,
  Power,
  PowerOff,
  Save,
} from "lucide-react";

const SEVERITY_OPTIONS = [
  { value: "info", label: "信息" },
  { value: "low", label: "低" },
  { value: "medium", label: "中" },
  { value: "high", label: "高" },
  { value: "critical", label: "严重" },
];

interface ChannelForm {
  name: string;
  url: string;
  enabled: boolean;
}

interface WebhookForm {
  url: string;
  secret: string;
  min_severity: string;
  on_new_asset: boolean;
  on_asset_gone: boolean;
  on_port_change: boolean;
  on_service_change: boolean;
  on_cert_expiry: boolean;
  enabled: boolean;
}

export default function NotificationChannelsPage() {
  const projectId = useProjectId();
  const toast = useToast();
  const [channels, setChannels] = useState<NotificationChannel[]>([]);
  const [webhook, setWebhook] = useState<AlertWebhook | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showChannelForm, setShowChannelForm] = useState(false);
  const [channelForm, setChannelForm] = useState<ChannelForm>({
    name: "",
    url: "",
    enabled: true,
  });
  const [webhookForm, setWebhookForm] = useState<WebhookForm>({
    url: "",
    secret: "",
    min_severity: "medium",
    on_new_asset: true,
    on_asset_gone: true,
    on_port_change: true,
    on_service_change: true,
    on_cert_expiry: true,
    enabled: false,
  });
  const [saving, setSaving] = useState(false);

  const fetchData = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    setError(null);
    try {
      const [ch, wh] = await Promise.all([
        api.listNotificationChannels(projectId).catch(() => []),
        api.getAlertWebhook(projectId).catch(() => null),
      ]);
      setChannels(ch);
      if (wh) {
        setWebhook(wh);
        setWebhookForm({
          url: wh.url || "",
          secret: wh.secret || "",
          min_severity: wh.min_severity || "medium",
          on_new_asset: wh.on_new_asset,
          on_asset_gone: wh.on_asset_gone,
          on_port_change: wh.on_port_change,
          on_service_change: wh.on_service_change,
          on_cert_expiry: wh.on_cert_expiry,
          enabled: wh.enabled,
        });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleCreateChannel = async () => {
    if (!projectId || !channelForm.name || !channelForm.url) return;
    setSaving(true);
    try {
      await api.createNotificationChannel(projectId, channelForm);
      toast("通道已创建", "success");
      setShowChannelForm(false);
      setChannelForm({ name: "", url: "", enabled: true });
      fetchData();
    } catch (err) {
      toast("创建失败", "error");
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteChannel = async (channelId: string) => {
    if (!projectId) return;
    try {
      await api.deleteNotificationChannel(projectId, channelId);
      toast("已删除", "success");
      setChannels((prev) => prev.filter((c) => c.id !== channelId));
    } catch (err) {
      toast("删除失败", "error");
    }
  };

  const handleToggleChannel = async (channel: NotificationChannel) => {
    if (!projectId) return;
    try {
      const updated = await api.updateNotificationChannel(projectId, channel.id, {
        enabled: !channel.enabled,
      });
      setChannels((prev) =>
        prev.map((c) => (c.id === channel.id ? updated : c)),
      );
      toast(updated.enabled ? "已启用" : "已停用", "success");
    } catch (err) {
      toast("操作失败", "error");
    }
  };

  const handleSaveWebhook = async () => {
    if (!projectId || !webhookForm.url) return;
    setSaving(true);
    try {
      const result = await api.upsertAlertWebhook(projectId, {
        url: webhookForm.url,
        secret: webhookForm.secret || undefined,
        min_severity: webhookForm.min_severity,
        on_new_asset: webhookForm.on_new_asset,
        on_asset_gone: webhookForm.on_asset_gone,
        on_port_change: webhookForm.on_port_change,
        on_service_change: webhookForm.on_service_change,
        on_cert_expiry: webhookForm.on_cert_expiry,
        enabled: webhookForm.enabled,
      });
      setWebhook(result);
      toast("Webhook 已保存", "success");
    } catch (err) {
      toast("保存失败", "error");
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteWebhook = async () => {
    if (!projectId) return;
    try {
      await api.deleteAlertWebhook(projectId);
      setWebhook(null);
      toast("Webhook 已删除", "success");
    } catch (err) {
      toast("删除失败", "error");
    }
  };

  if (!projectId) {
    return <EmptyState title="未选择项目" description="请先选择一个项目" />;
  }

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <BellRing className="h-6 w-6 text-amber-400" />
            通知配置
          </h1>
          <p className="text-muted-foreground mt-1">
            配置 Webhook 通知通道，当资产发生变化时自动推送告警
          </p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          onClick={fetchData}
          loading={loading}
          className="gap-1"
        >
          <RefreshCw className="h-4 w-4" />
          刷新
        </Button>
      </div>

      {loading ? (
        <>
          <SkeletonCard />
          <SkeletonCard />
        </>
      ) : error ? (
        <Card>
          <CardContent className="py-12 text-center">
            <BellRing className="mx-auto h-10 w-10 text-muted-foreground/50" />
            <p className="mt-4 text-muted-foreground">{error}</p>
            <Button
              variant="secondary"
              size="sm"
              className="mt-4"
              onClick={fetchData}
            >
              重试
            </Button>
          </CardContent>
        </Card>
      ) : (
        <>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle className="flex items-center gap-2">
                <Webhook className="h-5 w-5 text-violet-400" />
                告警 Webhook
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <p className="text-sm text-muted-foreground">
                  项目级别的告警 Webhook，支持按事件类型过滤推送。
                </p>
                <Badge
                  variant={webhook?.enabled ? "default" : "outline"}
                  className={
                    webhook?.enabled
                      ? "bg-emerald-500/20 text-emerald-400 border-emerald-500/30"
                      : ""
                  }
                >
                  {webhook?.enabled ? "已启用" : "未启用"}
                </Badge>
              </div>

              <div className="space-y-3">
                <div>
                  <label className="text-xs font-medium text-muted-foreground flex items-center gap-1">
                    <Link className="h-3 w-3" />
                    Webhook URL
                  </label>
                  <input
                    type="url"
                    className="mt-1 w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-violet-500/30"
                    placeholder="https://hooks.slack.com/..."
                    value={webhookForm.url}
                    onChange={(e) =>
                      setWebhookForm((f) => ({ ...f, url: e.target.value }))
                    }
                  />
                </div>

                <div>
                  <label className="text-xs font-medium text-muted-foreground flex items-center gap-1">
                    <Key className="h-3 w-3" />
                    Secret（可选，用于 HMAC 签名验证）
                  </label>
                  <input
                    type="text"
                    className="mt-1 w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-violet-500/30"
                    placeholder="your-webhook-secret"
                    value={webhookForm.secret}
                    onChange={(e) =>
                      setWebhookForm((f) => ({ ...f, secret: e.target.value }))
                    }
                  />
                </div>

                <div>
                  <label className="text-xs font-medium text-muted-foreground flex items-center gap-1">
                    <Shield className="h-3 w-3" />
                    最低告警等级
                  </label>
                  <div className="flex gap-2 mt-1 flex-wrap">
                    {SEVERITY_OPTIONS.map((opt) => (
                      <button
                        key={opt.value}
                        className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors ${
                          webhookForm.min_severity === opt.value
                            ? "bg-violet-500/20 text-violet-400 border border-violet-500/30"
                            : "bg-white/5 text-muted-foreground hover:bg-white/10 border border-transparent"
                        }`}
                        onClick={() =>
                          setWebhookForm((f) => ({
                            ...f,
                            min_severity: opt.value,
                          }))
                        }
                      >
                        {opt.label}
                      </button>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="text-xs font-medium text-muted-foreground">
                    事件类型
                  </label>
                  <div className="grid grid-cols-2 gap-2 mt-1">
                    {[
                      {
                        key: "on_new_asset" as const,
                        label: "新资产",
                        icon: Box,
                      },
                      {
                        key: "on_asset_gone" as const,
                        label: "资产消失",
                        icon: PowerOff,
                      },
                      {
                        key: "on_port_change" as const,
                        label: "端口变化",
                        icon: Wifi,
                      },
                      {
                        key: "on_service_change" as const,
                        label: "服务变化",
                        icon: Server,
                      },
                      {
                        key: "on_cert_expiry" as const,
                        label: "证书过期",
                        icon: Shield,
                      },
                    ].map(({ key, label, icon: Icon }) => (
                      <label
                        key={key}
                        className="flex items-center gap-2 p-2 rounded-lg border border-white/5 bg-white/5 cursor-pointer hover:bg-white/10 transition-colors"
                      >
                        <input
                          type="checkbox"
                          className="rounded border-white/20 bg-white/5 text-violet-500 focus:ring-violet-500/30"
                          checked={webhookForm[key]}
                          onChange={(e) =>
                            setWebhookForm((f) => ({
                              ...f,
                              [key]: e.target.checked,
                            }))
                          }
                        />
                        <Icon className="h-3.5 w-3.5 text-muted-foreground" />
                        <span className="text-xs text-foreground">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>

                <div className="flex items-center justify-between pt-2 border-t border-border">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      className="rounded border-white/20 bg-white/5 text-emerald-500 focus:ring-emerald-500/30"
                      checked={webhookForm.enabled}
                      onChange={(e) =>
                        setWebhookForm((f) => ({
                          ...f,
                          enabled: e.target.checked,
                        }))
                      }
                    />
                    <span className="text-sm text-foreground">启用 Webhook</span>
                  </label>
                  <div className="flex gap-2">
                    {webhook && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={handleDeleteWebhook}
                        className="text-red-400 hover:text-red-300"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    )}
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={handleSaveWebhook}
                      loading={saving}
                      className="gap-1"
                    >
                      <Save className="h-4 w-4" />
                      保存
                    </Button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle className="flex items-center gap-2">
                <Bell className="h-5 w-5 text-blue-400" />
                通知通道
              </CardTitle>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => setShowChannelForm(true)}
                className="gap-1"
              >
                <Plus className="h-4 w-4" />
                添加通道
              </Button>
            </CardHeader>
            <CardContent className="p-0">
              {showChannelForm && (
                <div className="p-4 border-b border-border bg-muted/30 space-y-3">
                  <input
                    type="text"
                    className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                    placeholder="通道名称（如：Slack #alerts）"
                    value={channelForm.name}
                    onChange={(e) =>
                      setChannelForm((f) => ({ ...f, name: e.target.value }))
                    }
                  />
                  <input
                    type="url"
                    className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                    placeholder="Webhook URL"
                    value={channelForm.url}
                    onChange={(e) =>
                      setChannelForm((f) => ({ ...f, url: e.target.value }))
                    }
                  />
                  <div className="flex justify-end gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowChannelForm(false)}
                    >
                      取消
                    </Button>
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={handleCreateChannel}
                      loading={saving}
                      disabled={!channelForm.name || !channelForm.url}
                      className="gap-1"
                    >
                      <Save className="h-4 w-4" />
                      保存
                    </Button>
                  </div>
                </div>
              )}

              {channels.length === 0 && !showChannelForm ? (
                <div className="py-12 text-center">
                  <Bell className="mx-auto h-10 w-10 text-muted-foreground/30" />
                  <p className="mt-4 text-muted-foreground">暂无通知通道</p>
                  <Button
                    variant="secondary"
                    size="sm"
                    className="mt-4"
                    onClick={() => setShowChannelForm(true)}
                  >
                    <Plus className="h-4 w-4 mr-1" />
                    添加第一个通道
                  </Button>
                </div>
              ) : (
                <div className="divide-y divide-border">
                  {channels.map((ch) => (
                    <div
                      key={ch.id}
                      className="flex items-center justify-between p-4 hover:bg-muted/30 transition-colors"
                    >
                      <div className="flex items-center gap-3 min-w-0">
                        <div
                          className={`p-1.5 rounded-lg ${
                            ch.enabled ? "bg-emerald-500/20" : "bg-white/10"
                          }`}
                        >
                          <Webhook
                            className={`h-3.5 w-3.5 ${
                              ch.enabled ? "text-emerald-400" : "text-muted-foreground"
                            }`}
                          />
                        </div>
                        <div className="min-w-0">
                          <p className="text-sm font-medium text-foreground truncate">
                            {ch.name}
                          </p>
                          <p className="text-xs text-muted-foreground truncate font-mono">
                            {ch.url}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <Badge
                          variant="outline"
                          className={`text-[10px] ${
                            ch.enabled
                              ? "text-emerald-400 border-emerald-500/30"
                              : "text-muted-foreground"
                          }`}
                        >
                          {ch.enabled ? "已启用" : "已停用"}
                        </Badge>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleToggleChannel(ch)}
                          className="h-7 w-7"
                        >
                          {ch.enabled ? (
                            <Power className="h-3.5 w-3.5 text-amber-400" />
                          ) : (
                            <PowerOff className="h-3.5 w-3.5 text-muted-foreground" />
                          )}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDeleteChannel(ch.id)}
                          className="h-7 w-7 text-red-400 hover:text-red-300"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}
