# 前端编码约定

> Anchor 前端（React + TypeScript + Tailwind）编码规范。
> 最后更新：2026-06-01

---

## 1. 技术栈

- **框架**: React 18 + TypeScript
- **构建**: Vite
- **样式**: Tailwind CSS v3 + shadcn/ui
- **状态**: Zustand
- **路由**: React Router（如需要）
- **HTTP**: 原生 fetch（通过 api.ts 封装）
- **SSE**: 原生 EventSource
- **语法高亮**: Prism.js

## 2. 目录结构

```
frontend/src/
├── lib/                    # 工具层（不依赖 UI）
│   ├── api.ts             # HTTP API 客户端（唯一与后端通信入口）
│   ├── store.ts           # Zustand store 定义
│   └── utils.ts           # 通用工具函数
├── components/            # 共享组件
│   ├── ui/               # shadcn/ui 组件
│   ├── common/           # 自定义共享组件
│   └── index.ts          # 统一导出
├── pages/                # 页面组件（路由级别）
│   ├── ProjectPage.tsx
│   ├── TargetPage.tsx
│   ├── RunsPage.tsx
│   ├── AssetPage.tsx
│   ├── FindingsPage.tsx
│   └── ReportsPage.tsx
├── hooks/                # 自定义 hooks
├── types/                # 共享类型定义
└── App.tsx              # 根组件（路由 + 布局）
```

## 3. 组件规范

### 3.1 文件命名
- 组件文件: `PascalCase.tsx`
- Hook 文件: `use-camel-case.ts`
- 工具文件: `camel-case.ts`
- 类型文件: `camel-case.ts`

### 3.2 组件结构
```tsx
// 展示组件（纯 UI，无数据获取）
export function TaskCard({ task, onDelete }: TaskCardProps) {
  return (
    <div className="rounded-lg border p-4">
      <h3 className="text-lg font-medium">{task.name}</h3>
      <Button variant="ghost" onClick={() => onDelete(task.id)}>删除</Button>
    </div>
  );
}

// 容器组件（数据获取 + 状态管理）
export function TaskListContainer() {
  const { tasks, isLoading, error } = useTasks();
  
  if (isLoading) return <TaskListSkeleton />;
  if (error) return <ErrorState message={error.message} />;
  if (tasks.length === 0) return <EmptyState />;
  
  return <TaskList tasks={tasks} />;
}
```

### 3.3 Props 规范
- 使用 interface 定义 Props，不要 inline type
- 事件 handler 命名：`on + 动词`（如 `onDelete`、`onSubmit`）
- boolean props 命名：用形容词（如 `isOpen`、`isLoading`）

## 4. 状态管理

### 4.1 Zustand Store 拆分
```typescript
// store.ts — 主入口
export const useProjectStore = create<ProjectStore>((set) => ({
  projects: [],
  setProjects: (projects) => set({ projects }),
  addProject: (project) => set((state) => ({
    projects: [...state.projects, project]
  })),
}));
```

### 4.2 规则
- 按 domain 拆分 store（project、target、task 各一个）
- 不要在组件内直接调用 fetch，通过 store action 封装
- 异步状态（loading/error）必须在 store 中管理

## 5. API 调用规范

### 5.1 唯一入口
所有后端通信必须通过 `api.ts`：
```typescript
// api.ts — 默认通过 Nginx 反向代理 /api/ → server:17421
const API_BASE = getApiBase(); // 默认 "/api"，可通过 localStorage 覆盖

export async function createProject(data: CreateProjectInput) {
  const res = await fetch(`${API_BASE}/projects`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new ApiError(err.code, err.message);
  }
  return res.json();
}
```

### 5.2 错误处理
- API 错误统一抛出 `ApiError`，包含 code 和 message
- UI 层使用 try/catch 捕获，展示用户友好的错误信息
- 网络错误（fetch 失败）单独处理，提示检查服务是否运行

### 5.3 SSE 连接
```typescript
// api.ts
export function connectEvents(onMessage: (data: EventData) => void) {
  const source = new EventSource(`${API_BASE}/events`);
  source.onmessage = (e) => onMessage(JSON.parse(e.data));
  source.onerror = () => {
    console.error('SSE 连接断开，3秒后重连...');
    setTimeout(() => connectEvents(onMessage), 3000);
  };
  return () => source.close();
}
```

## 6. 样式规范

### 6.1 Tailwind
- 使用 Tailwind 的 utility classes，不写自定义 CSS
- 复杂组件提取为 `cn()` 工具函数（来自 shadcn/ui）
- 响应式：优先移动端，使用 `md:`、`lg:` 断点

### 6.2 shadcn/ui
- 使用 shadcn/ui 组件库作为基础
- 自定义样式通过 `className` 覆盖
- 新增组件优先从 shadcn/ui 安装，其次自建

## 7. 类型安全

- 所有组件 props、函数参数必须加类型
- API 返回数据必须定义接口
- 后端模型变更后同步更新前端类型

## 8. 安全

- **NEVER** 在前端代码中嵌入 API key、token、密码
- **NEVER** 在 URL 中传递敏感参数（用 POST body）
- 所有用户输入在提交前做前端校验（但后端仍需二次校验）
- **NEVER** 使用 `dangerouslySetInnerHTML` 渲染用户可控内容；如需展示富文本，使用纯文本 `<pre>` 或经过净化的 HTML
