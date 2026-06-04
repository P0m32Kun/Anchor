import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { srcApi, SubmissionPack, BountyCandidate } from "../lib/api";
import { Button, Badge, Card, CardHeader, CardTitle, CardContent, useToast } from "../components";
import { Loader2, Save, Download, Shield, ArrowLeft, Check, Copy } from "lucide-react";

const REDACTION_STATUS_LABELS: Record<string, string> = {
  raw: "未脱敏",
  reviewed: "已审核",
  redacted: "已脱敏",
};

const REDACTION_STATUS_COLORS: Record<string, string> = {
  raw: "bg-yellow-500",
  reviewed: "bg-blue-500",
  redacted: "bg-green-500",
};

export default function SubmissionPackPage() {
  const { packId } = useParams<{ projectId: string; packId: string }>();
  const navigate = useNavigate();
  const toast = useToast();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [redacting, setRedacting] = useState(false);
  const [pack, setPack] = useState<SubmissionPack | null>(null);
  const [candidate, setCandidate] = useState<BountyCandidate | null>(null);
  const [content, setContent] = useState("");
  const [checklist, setChecklist] = useState<Array<{ id: string; label: string; checked: boolean; note: string }>>([]);

  useEffect(() => {
    if (packId) {
      loadPack();
    }
  }, [packId]);

  const loadPack = async () => {
    try {
      setLoading(true);
      const data = await srcApi.getPack(packId!);
      setPack(data);
      setContent(data.content);

      // 解析检查清单
      try {
        const parsed = JSON.parse(data.checklist_json);
        setChecklist(Array.isArray(parsed) ? parsed : []);
      } catch {
        setChecklist([]);
      }

      // 加载候选信息
      if (data.candidate_id) {
        const candidateData = await srcApi.getCandidate(data.candidate_id);
        setCandidate(candidateData);
      }
    } catch (error) {
      console.error("Failed to load pack:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      await srcApi.updatePack(packId!, {
        content,
        checklist_json: JSON.stringify(checklist),
      });
      toast("保存成功", "success");
    } catch (error) {
      toast("保存失败", "error");
    } finally {
      setSaving(false);
    }
  };

  const handleRedact = async () => {
    if (!confirm("确定要对内容进行脱敏处理吗？")) return;
    try {
      setRedacting(true);
      const result = await srcApi.redactPack(packId!);
      setPack(result);
      setContent(result.content);
      toast("脱敏完成", "success");
    } catch (error) {
      toast("脱敏失败", "error");
    } finally {
      setRedacting(false);
    }
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content);
      toast("已复制到剪贴板", "success");
    } catch (error) {
      toast("复制失败", "error");
    }
  };

  const handleDownload = () => {
    const blob = new Blob([content], { type: "text/markdown" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `submission-${pack?.id || "report"}.md`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const toggleChecklistItem = (id: string) => {
    setChecklist((prev) =>
      prev.map((item) =>
        item.id === id ? { ...item, checked: !item.checked } : item
      )
    );
  };

  const updateChecklistNote = (id: string, note: string) => {
    setChecklist((prev) =>
      prev.map((item) =>
        item.id === id ? { ...item, note } : item
      )
    );
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    );
  }

  if (!pack) {
    return (
      <div className="text-center py-8">
        <p className="text-muted-foreground">提交包不存在</p>
        <Button variant="outline" onClick={() => navigate(-1)} className="mt-4">
          返回
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold">提交包</h1>
            {candidate && (
              <p className="text-muted-foreground">{candidate.title}</p>
            )}
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handleCopy}>
            <Copy className="h-4 w-4 mr-2" />
            复制
          </Button>
          <Button variant="outline" onClick={handleDownload}>
            <Download className="h-4 w-4 mr-2" />
            下载
          </Button>
          <Button variant="outline" onClick={handleRedact} disabled={redacting}>
            {redacting ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Shield className="h-4 w-4 mr-2" />}
            脱敏
          </Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Save className="h-4 w-4 mr-2" />}
            保存
          </Button>
        </div>
      </div>

      {/* 元信息 */}
      <div className="flex items-center gap-4">
        <Badge variant="outline">模板: {pack.template}</Badge>
        <Badge className={REDACTION_STATUS_COLORS[pack.redaction_status]}>
          {REDACTION_STATUS_LABELS[pack.redaction_status]}
        </Badge>
        <span className="text-sm text-muted-foreground">
          创建于 {new Date(pack.created_at).toLocaleString()}
        </span>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* 内容编辑器 */}
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <CardTitle>报告内容</CardTitle>
            </CardHeader>
            <CardContent>
              <textarea
                value={content}
                onChange={(e) => setContent(e.target.value)}
                className="w-full min-h-[500px] p-3 border rounded-md font-mono text-sm resize-y"
                placeholder="Markdown 内容..."
              />
            </CardContent>
          </Card>
        </div>

        {/* 检查清单 */}
        <div>
          <Card>
            <CardHeader>
              <CardTitle>检查清单</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {checklist.map((item) => (
                  <div key={item.id} className="space-y-2">
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => toggleChecklistItem(item.id)}
                        className={`w-5 h-5 rounded border-2 flex items-center justify-center ${
                          item.checked
                            ? "bg-primary border-primary"
                            : "border-muted-foreground"
                        }`}
                      >
                        {item.checked && <Check className="h-3 w-3 text-primary-foreground" />}
                      </button>
                      <span className={item.checked ? "line-through text-muted-foreground" : ""}>
                        {item.label}
                      </span>
                    </div>
                    <input
                      type="text"
                      value={item.note}
                      onChange={(e) => updateChecklistNote(item.id, e.target.value)}
                      placeholder="备注..."
                      className="text-sm w-full px-2 py-1 border rounded"
                    />
                  </div>
                ))}
              </div>

              {/* 进度 */}
              <div className="mt-4 pt-4 border-t">
                <div className="flex items-center justify-between text-sm">
                  <span>完成进度</span>
                  <span>
                    {checklist.filter((i) => i.checked).length} / {checklist.length}
                  </span>
                </div>
                <div className="mt-2 h-2 bg-muted rounded-full overflow-hidden">
                  <div
                    className="h-full bg-primary transition-all"
                    style={{
                      width: `${
                        checklist.length > 0
                          ? (checklist.filter((i) => i.checked).length / checklist.length) * 100
                          : 0
                      }%`,
                    }}
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
