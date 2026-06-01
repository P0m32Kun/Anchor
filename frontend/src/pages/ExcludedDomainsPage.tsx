import { useState, useEffect, useCallback } from 'react'
import { api } from '../lib/api'
import {
  useToast,
  Button,
  Input,
  Card,
  Badge,
  EmptyState,
} from '../components'
import { cn } from '../lib/utils'
import { Loader2, Plus, Trash2, RefreshCw, Search, Shield, Globe } from 'lucide-react'

interface ExcludedDomain {
  id: string
  domain: string
  reason: string
  builtin: boolean
  created_at: string
}

interface ExcludedDomainsResponse {
  builtin: ExcludedDomain[]
  custom: ExcludedDomain[]
  total: number
}

export default function ExcludedDomainsPage() {
  const [domains, setDomains] = useState<ExcludedDomainsResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [newDomain, setNewDomain] = useState('')
  const [newReason, setNewReason] = useState('')
  const [checkDomain, setCheckDomain] = useState('')
  const [checkResult, setCheckResult] = useState<{ domain: string; excluded: boolean; reason: string } | null>(null)
  const [adding, setAdding] = useState(false)
  const [checking, setChecking] = useState(false)
  const [activeTab, setActiveTab] = useState<'custom' | 'builtin'>('custom')
  const toast = useToast()

  const fetchDomains = useCallback(async () => {
    try {
      setLoading(true)
      const response = await api.listExcludedDomains()
      setDomains(response)
    } catch (error) {
      toast('获取排除域名列表失败', 'error')
    } finally {
      setLoading(false)
    }
  }, [toast])

  useEffect(() => {
    fetchDomains()
  }, [fetchDomains])

  const handleAddDomain = async () => {
    if (!newDomain.trim()) {
      toast('请输入域名', 'error')
      return
    }

    try {
      setAdding(true)
      await api.addExcludedDomain(newDomain.trim(), newReason.trim())
      toast(`已添加 ${newDomain.trim()} 到排除列表`, 'success')
      setNewDomain('')
      setNewReason('')
      fetchDomains()
    } catch (error: any) {
      const message = error?.message || '添加失败'
      toast(message, 'error')
    } finally {
      setAdding(false)
    }
  }

  const handleDeleteDomain = async (domain: string) => {
    if (!confirm(`确定要删除 ${domain} 吗？`)) {
      return
    }

    try {
      await api.deleteExcludedDomain(domain)
      toast(`已删除 ${domain}`, 'success')
      fetchDomains()
    } catch (error: any) {
      const message = error?.message || '删除失败'
      toast(message, 'error')
    }
  }

  const handleReset = async () => {
    if (!confirm('确定要重置为默认列表吗？这将删除所有自定义域名。')) {
      return
    }

    try {
      await api.resetExcludedDomains()
      toast('已重置为默认列表', 'success')
      fetchDomains()
    } catch (error) {
      toast('重置失败', 'error')
    }
  }

  const handleCheckDomain = async () => {
    if (!checkDomain.trim()) {
      toast('请输入要检查的域名', 'error')
      return
    }

    try {
      setChecking(true)
      const response = await api.checkExcludedDomain(checkDomain.trim())
      setCheckResult(response)
    } catch (error) {
      toast('检查失败', 'error')
    } finally {
      setChecking(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Shield className="h-8 w-8" />
            域名排除列表
          </h1>
          <p className="text-muted-foreground mt-2">
            管理扫描时需要排除的公共服务域名，避免误扫描非目标资产
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={fetchDomains}>
            <RefreshCw className="h-4 w-4 mr-2" />
            刷新
          </Button>
          <Button variant="danger" onClick={handleReset}>
            重置为默认
          </Button>
        </div>
      </div>

      {/* 域名检查工具 */}
      <Card>
        <div className="p-6">
          <h3 className="text-lg font-semibold flex items-center gap-2 mb-2">
            <Search className="h-5 w-5" />
            域名检查
          </h3>
          <p className="text-sm text-muted-foreground mb-4">
            检查某个域名是否在排除列表中
          </p>
          <div className="flex gap-2">
            <Input
              placeholder="输入域名，例如 github.com"
              value={checkDomain}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setCheckDomain(e.target.value)}
              onKeyDown={(e: React.KeyboardEvent) => e.key === 'Enter' && handleCheckDomain()}
            />
            <Button onClick={handleCheckDomain} disabled={checking}>
              {checking ? <Loader2 className="h-4 w-4 animate-spin" /> : '检查'}
            </Button>
          </div>
          {checkResult && (
            <div className={cn(
              "mt-4 p-4 rounded-lg",
              checkResult.excluded ? 'bg-red-500/10 border border-red-500/20' : 'bg-green-500/10 border border-green-500/20'
            )}>
              <div className="flex items-center gap-2">
                <Globe className="h-5 w-5" />
                <span className="font-medium">{checkResult.domain}</span>
                <Badge variant={checkResult.excluded ? 'destructive' : 'default'}>
                  {checkResult.excluded ? '已排除' : '未排除'}
                </Badge>
              </div>
              <p className="text-sm text-muted-foreground mt-1">
                {checkResult.reason}
              </p>
            </div>
          )}
        </div>
      </Card>

      {/* 添加自定义域名 */}
      <Card>
        <div className="p-6">
          <h3 className="text-lg font-semibold flex items-center gap-2 mb-2">
            <Plus className="h-5 w-5" />
            添加自定义域名
          </h3>
          <p className="text-sm text-muted-foreground mb-4">
            添加需要排除的自定义域名，支持子域名自动匹配
          </p>
          <div className="flex gap-2">
            <Input
              placeholder="域名，例如 evil.com"
              value={newDomain}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNewDomain(e.target.value)}
              className="flex-1"
            />
            <Input
              placeholder="原因（可选）"
              value={newReason}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNewReason(e.target.value)}
              className="flex-1"
            />
            <Button onClick={handleAddDomain} disabled={adding}>
              {adding ? <Loader2 className="h-4 w-4 animate-spin" /> : '添加'}
            </Button>
          </div>
        </div>
      </Card>

      {/* 域名列表 */}
      <div className="space-y-4">
        <div className="flex gap-2 border-b">
          <button
            className={cn(
              "px-4 py-2 font-medium text-sm border-b-2 transition-colors",
              activeTab === 'custom'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            )}
            onClick={() => setActiveTab('custom')}
          >
            自定义域名 ({domains?.custom?.length || 0})
          </button>
          <button
            className={cn(
              "px-4 py-2 font-medium text-sm border-b-2 transition-colors",
              activeTab === 'builtin'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            )}
            onClick={() => setActiveTab('builtin')}
          >
            内置域名 ({domains?.builtin?.length || 0})
          </button>
        </div>

        {activeTab === 'custom' ? (
          <Card>
            <div className="p-6">
              <h3 className="text-lg font-semibold mb-2">自定义排除域名</h3>
              <p className="text-sm text-muted-foreground mb-4">
                这些是您手动添加的域名，可以删除
              </p>
              {!domains?.custom || domains.custom.length === 0 ? (
                <EmptyState
                  icon={<Globe className="h-12 w-12" />}
                  title="暂无自定义域名"
                  description="点击上方按钮添加需要排除的域名"
                />
              ) : (
                <div className="space-y-2">
                  {domains.custom.map((domain) => (
                    <div
                      key={domain.id}
                      className="flex items-center justify-between p-3 border rounded-lg hover:bg-muted/50"
                    >
                      <div>
                        <div className="font-medium">{domain.domain}</div>
                        {domain.reason && (
                          <div className="text-sm text-muted-foreground">
                            {domain.reason}
                          </div>
                        )}
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDeleteDomain(domain.domain)}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </Card>
        ) : (
          <Card>
            <div className="p-6">
              <h3 className="text-lg font-semibold mb-2">内置排除域名</h3>
              <p className="text-sm text-muted-foreground mb-4">
                这些是系统内置的公共服务域名，不可删除
              </p>
              {!domains?.builtin || domains.builtin.length === 0 ? (
                <EmptyState
                  icon={<Globe className="h-12 w-12" />}
                  title="暂无内置域名"
                  description="系统启动时会自动加载内置域名列表"
                />
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
                  {domains.builtin.map((domain) => (
                    <div
                      key={domain.id}
                      className="p-2 border rounded text-sm"
                    >
                      {domain.domain}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </Card>
        )}
      </div>
    </div>
  )
}
