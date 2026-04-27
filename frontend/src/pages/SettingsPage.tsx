export default function SettingsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold">Settings</h1>
        <p className="text-sm text-text-tertiary mt-1">
          应用配置和偏好设置
        </p>
      </div>

      <div className="liquid-glass rounded-xl p-5 space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <div className="text-sm font-medium">本地 Worker 自动启动</div>
            <div className="text-xs text-text-tertiary mt-0.5">
              应用启动时自动启动本地 Worker
            </div>
          </div>
          <div className="w-10 h-5 bg-brand-primary rounded-full relative cursor-pointer">
            <div className="w-4 h-4 bg-white rounded-full absolute right-0.5 top-0.5" />
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
