import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { srcApi, BountyCandidate, BountyCandidateStats } from "../lib/api";
import { Button, Badge, Card, CardContent, Select, useToast } from "../components";
import { Loader2, RefreshCw, FileText, CheckCircle, Clock, TrendingUp } from "lucide-react";

const VERIFY_STATUS_LABELS: Record<string, string> = {
  pending: "待验证",
  verifying: "验证中",
  confirmed: "已确认",
  false_positive: "误报",
  not_applicable: "不适用",
};

const SUBMISSION_STATUS_LABELS: Record<string, string> = {
  not_ready: "未就绪",
  ready: "可提交",
  submitted: "已提交",
  duplicate: "重复",
  accepted: "已接受",
  rejected: "已拒绝",
  paid: "已支付",
};

const DUPLICATE_RISK_LABELS: Record<string, string> = {
  low: "低",
  medium: "中",
  high: "高",
  unknown: "未知",
};

const SEVERITY_COLORS: Record<string, string> = {
  critical: "bg-red-500",
  high: "bg-orange-500",
  medium: "bg-yellow-500",
  low: "bg-blue-500",
  info: "bg-gray-500",
};

export default function BountyQueuePage() {
  const { projectId } = useParams<{ projectId: string }>();
  const navigate = useNavigate();
  const toast = useToast();
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [candidates, setCandidates] = useState<BountyCandidate[]>([]);
  const [stats, setStats] = useState<BountyCandidateStats | null>(null);
  const [verifyFilter, setVerifyFilter] = useState<string>("");
  const [submissionFilter, setSubmissionFilter] = useState<string>("");

  const loadCandidates = useCallback(async () => {
    try {
      setLoading(true);
      const data = await srcApi.listCandidates(projectId!, {
        verify_status: verifyFilter || undefined,
        submission_status: submissionFilter || undefined,
      });
      setCandidates(data.candidates || []);
      setStats(data.stats);
    } catch (error) {
      console.error("Failed to load candidates:", error);
    } finally {
      setLoading(false);
    }
  }, [projectId, verifyFilter, submissionFilter]);

  useEffect(() => {
    if (projectId) {
      loadCandidates();
    }
  }, [projectId, loadCandidates]);

  const handleRefresh = async () => {
    try {
      setRefreshing(true);
      const result = await srcApi.refreshCandidates(projectId!);
      toast(`新增 ${result.created} 个候选`, "success");
      await loadCandidates();
    } catch (error) {
      toast("请重试", "error");
    } finally {
      setRefreshing(false);
    }
  };

  const handleUpdateStatus = async (id: string, field: string, value: string) => {
    try {
      await srcApi.updateCandidate(id, { [field]: value });
      toast("更新成功", "success");
      await loadCandidates();
    } catch (error) {
      toast("更新失败", "error");
    }
  };

  const handleGeneratePack = async (candidateId: string) => {
    try {
      const pack = await srcApi.createPack(candidateId);
      toast("提交包已生成", "success");
      navigate(`/projects/${projectId}/submission-packs/${pack.id}`);
    } catch (error) {
      toast("生成失败", "error");
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">赏金候选队列</h1>
          <p className="text-muted-foreground">
            按价值评分排序的漏洞候选列表
          </p>
        </div>
        <Button onClick={handleRefresh} disabled={refreshing}>
          {refreshing ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <RefreshCw className="h-4 w-4 mr-2" />}
          刷新候选
        </Button>
      </div>

      {/* 统计卡片 */}
      {stats && (
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">总候选</p>
                  <p className="text-2xl font-bold">{stats.total}</p>
                </div>
                <FileText className="h-8 w-8 text-muted-foreground" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">待验证</p>
                  <p className="text-2xl font-bold">{stats.pending}</p>
                </div>
                <Clock className="h-8 w-8 text-yellow-500" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">已确认</p>
                  <p className="text-2xl font-bold">{stats.verified}</p>
                </div>
                <CheckCircle className="h-8 w-8 text-green-500" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">总价值</p>
                  <p className="text-2xl font-bold">{stats.total_value}</p>
                </div>
                <TrendingUp className="h-8 w-8 text-blue-500" />
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* 筛选器 */}
      <div className="flex gap-4">
        <Select value={verifyFilter} onChange={(e) => setVerifyFilter(e.target.value)}>
          <option value="">所有验证状态</option>
          {Object.entries(VERIFY_STATUS_LABELS).map(([value, label]) => (
            <option key={value} value={value}>{label}</option>
          ))}
        </Select>
        <Select value={submissionFilter} onChange={(e) => setSubmissionFilter(e.target.value)}>
          <option value="">所有提交状态</option>
          {Object.entries(SUBMISSION_STATUS_LABELS).map(([value, label]) => (
            <option key={value} value={value}>{label}</option>
          ))}
        </Select>
      </div>

      {/* 候选列表 */}
      <div className="space-y-4">
        {candidates.length === 0 ? (
          <Card>
            <CardContent className="p-8 text-center">
              <p className="text-muted-foreground">暂无候选，点击"刷新候选"生成</p>
            </CardContent>
          </Card>
        ) : (
          candidates.map((candidate) => (
            <Card key={candidate.id}>
              <CardContent className="p-4">
                <div className="flex items-start justify-between">
                  <div className="space-y-2 flex-1">
                    <div className="flex items-center gap-2">
                      <Badge className={SEVERITY_COLORS[candidate.severity] || "bg-gray-500"}>
                        {candidate.severity}
                      </Badge>
                      <h3 className="font-semibold">{candidate.title}</h3>
                    </div>
                    <div className="flex items-center gap-4 text-sm text-muted-foreground">
                      <span>类型: {candidate.vuln_type}</span>
                      <span>来源: {candidate.source_kind}</span>
                      <span>重复风险: {DUPLICATE_RISK_LABELS[candidate.duplicate_risk] || candidate.duplicate_risk}</span>
                    </div>
                    {candidate.ranking_reason && (
                      <p className="text-sm text-muted-foreground">{candidate.ranking_reason}</p>
                    )}
                  </div>

                  <div className="flex items-center gap-4">
                    {/* 分数 */}
                    <div className="text-center">
                      <div className="text-3xl font-bold text-primary">{candidate.value_score}</div>
                      <div className="text-xs text-muted-foreground">价值评分</div>
                    </div>

                    {/* 状态操作 */}
                    <div className="flex flex-col gap-2">
                      <Select
                        value={candidate.verify_status}
                        onChange={(e) => handleUpdateStatus(candidate.id, "verify_status", e.target.value)}
                      >
                        {Object.entries(VERIFY_STATUS_LABELS).map(([value, label]) => (
                          <option key={value} value={value}>{label}</option>
                        ))}
                      </Select>
                      <Select
                        value={candidate.submission_status}
                        onChange={(e) => handleUpdateStatus(candidate.id, "submission_status", e.target.value)}
                      >
                        {Object.entries(SUBMISSION_STATUS_LABELS).map(([value, label]) => (
                          <option key={value} value={value}>{label}</option>
                        ))}
                      </Select>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleGeneratePack(candidate.id)}
                      >
                        <FileText className="h-4 w-4 mr-1" />
                        生成报告
                      </Button>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))
        )}
      </div>
    </div>
  );
}
