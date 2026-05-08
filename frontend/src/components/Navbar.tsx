import { Link, useLocation, useNavigate } from "react-router-dom";
import type React from "react";
import { useStore } from "../lib/store";

type NavItem = {
  path: string;
  label: string;
  icon: "dashboard" | "projects" | "workers" | "settings" | "target" | "asset" | "runs" | "finding" | "report" | "config" | "search";
};

const globalNavItems: NavItem[] = [
  { path: "/", label: "总览", icon: "dashboard" },
  { path: "/projects", label: "项目", icon: "projects" },
  { path: "/engines", label: "搜索引擎", icon: "search" },
  { path: "/workers", label: "Workers", icon: "workers" },
  { path: "/settings", label: "设置", icon: "settings" },
];

const workflowLabels = ["目标", "资产", "扫描", "发现", "报告", "配置"];

function isItemActive(locationPathname: string, itemPath: string): boolean {
  if (itemPath === "/") {
    return locationPathname === "/";
  }
  return locationPathname.startsWith(itemPath);
}

function Icon({ name }: { name: NavItem["icon"] }) {
  const common = "h-4 w-4";
  const paths: Record<NavItem["icon"], React.ReactNode> = {
    dashboard: (
      <>
        <rect x="3" y="3" width="7" height="7" rx="1.5" />
        <rect x="14" y="3" width="7" height="7" rx="1.5" />
        <rect x="3" y="14" width="7" height="7" rx="1.5" />
        <rect x="14" y="14" width="7" height="7" rx="1.5" />
      </>
    ),
    projects: (
      <>
        <path d="M4 5.5A2.5 2.5 0 0 1 6.5 3h11A2.5 2.5 0 0 1 20 5.5v13A2.5 2.5 0 0 1 17.5 21h-11A2.5 2.5 0 0 1 4 18.5z" />
        <path d="M8 8h8M8 12h8M8 16h5" />
      </>
    ),
    workers: (
      <>
        <rect x="4" y="4" width="16" height="6" rx="1.5" />
        <rect x="4" y="14" width="16" height="6" rx="1.5" />
        <path d="M8 7h.01M8 17h.01" />
      </>
    ),
    settings: (
      <>
        <circle cx="12" cy="12" r="3" />
        <path d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.05.05a2 2 0 1 1-2.83 2.83l-.05-.05A1.7 1.7 0 0 0 15 19.4a1.7 1.7 0 0 0-1 .6 1.7 1.7 0 0 0-.4 1.1V21a2 2 0 1 1-4 0v-.08A1.7 1.7 0 0 0 8 19.4a1.7 1.7 0 0 0-1.88.34l-.05.05a2 2 0 1 1-2.83-2.83l.05-.05A1.7 1.7 0 0 0 4.6 15a1.7 1.7 0 0 0-.6-1 1.7 1.7 0 0 0-1.1-.4H2.8a2 2 0 1 1 0-4h.08A1.7 1.7 0 0 0 4.6 8a1.7 1.7 0 0 0-.34-1.88l-.05-.05a2 2 0 1 1 2.83-2.83l.05.05A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1-.6 1.7 1.7 0 0 0 .4-1.1V2.8a2 2 0 1 1 4 0v.08A1.7 1.7 0 0 0 16 4.6a1.7 1.7 0 0 0 1.88-.34l.05-.05a2 2 0 1 1 2.83 2.83l-.05.05A1.7 1.7 0 0 0 19.4 9c.31.18.57.39.6.6.03.2.4.4 1.1.4h.1a2 2 0 1 1 0 4h-.08A1.7 1.7 0 0 0 19.4 15Z" />
      </>
    ),
    target: <path d="M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Zm0-4a5 5 0 1 0 0-10 5 5 0 0 0 0 10Zm0-2a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z" />,
    asset: (
      <>
        <path d="M12 3 4 7v10l8 4 8-4V7z" />
        <path d="M4 7l8 4 8-4M12 11v10" />
      </>
    ),
    runs: (
      <>
        <path d="M5 4v16" />
        <path d="m9 8 4 4-4 4" />
        <path d="M14 12h6" />
      </>
    ),
    finding: (
      <>
        <path d="M12 3 3 19h18z" />
        <path d="M12 9v4M12 17h.01" />
      </>
    ),
    report: (
      <>
        <path d="M7 3h7l4 4v14H7z" />
        <path d="M14 3v5h5M10 13h6M10 17h4" />
      </>
    ),
    config: (
      <>
        <path d="M4 7h16M4 17h16" />
        <circle cx="8" cy="7" r="2" />
        <circle cx="16" cy="17" r="2" />
      </>
    ),
    search: (
      <>
        <circle cx="11" cy="11" r="8" />
        <path d="m21 21-4.3-4.3" />
      </>
    ),
  };

  return (
    <svg className={common} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      {paths[name]}
    </svg>
  );
}

function NavLinkItem({ item, active }: { item: NavItem; active: boolean }) {
  return (
    <Link
      to={item.path}
      aria-current={active ? "page" : undefined}
      className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors ${
        active
          ? "bg-brand-primary/12 text-text-primary ring-1 ring-brand-primary/30 shadow-[0_0_16px_rgba(0,212,255,0.10)]"
          : "text-text-tertiary hover:bg-brand-primary/[0.055] hover:text-text-secondary"
      }`}
    >
      <Icon name={item.icon} />
      <span>{item.label}</span>
    </Link>
  );
}

export function Navbar() {
  const location = useLocation();
  const navigate = useNavigate();
  const currentProjectId = useStore((s) => s.currentProjectId);
  const currentProject = useStore((s) => s.currentProject);
  const projects = useStore((s) => s.projects);
  const setCurrentProjectId = useStore((s) => s.setCurrentProjectId);

  const projectLinks: NavItem[] = currentProjectId
    ? [
        { path: `/projects/${currentProjectId}/targets`, label: "目标与 Scope", icon: "target" },
        { path: `/projects/${currentProjectId}/assets`, label: "资产清单", icon: "asset" },
        { path: `/projects/${currentProjectId}/runs`, label: "扫描执行", icon: "runs" },
        { path: `/projects/${currentProjectId}/findings`, label: "发现审核", icon: "finding" },
        { path: `/projects/${currentProjectId}/reports`, label: "报告交付", icon: "report" },
      ]
    : [];

  const handleProjectSwitch = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const id = e.target.value;
    if (id) {
      setCurrentProjectId(id);
      navigate(`/projects/${id}/targets`);
    }
  };

  return (
    <aside className="app-sidebar fixed inset-y-0 left-0 z-50 flex w-72 flex-col">
      <div className="flex h-16 items-center gap-3 border-b border-white/[0.08] px-5">
        <Link to="/" className="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-primary/12 text-brand-primary ring-1 ring-brand-primary/30 shadow-[0_0_18px_rgba(0,212,255,0.16)]">
          <Icon name="asset" />
        </Link>
        <div>
          <div className="text-sm font-semibold text-text-primary">Anchor</div>
          <div className="text-xs text-text-quaternary">Security operations workspace</div>
        </div>
      </div>

      <div className="flex-1 space-y-6 overflow-y-auto px-4 py-5">
        <section>
          <div className="mb-2 px-2 text-[11px] font-semibold uppercase tracking-wider text-text-quaternary">Workspace</div>
          <div className="space-y-1">
            {globalNavItems.map((item) => (
              <NavLinkItem
                key={item.path}
                item={item}
                active={isItemActive(location.pathname, item.path)}
              />
            ))}
          </div>
        </section>

        <section className="space-y-3">
          <div className="px-2">
            <div className="text-[11px] font-semibold uppercase tracking-wider text-text-quaternary">Current Project</div>
            {projects.length > 0 ? (
              <select
                value={currentProjectId ?? ""}
                onChange={handleProjectSwitch}
                className="mt-2 input-dark"
                aria-label="Switch project"
              >
                <option value="" disabled>选择项目</option>
                {projects.map((project) => (
                  <option key={project.id} value={project.id}>
                    {project.name}
                  </option>
                ))}
              </select>
            ) : (
              <div className="mt-2 surface-item px-3 py-2 text-sm text-text-secondary">
                {currentProject?.name ?? "未选择项目"}
              </div>
            )}
          </div>

          {projectLinks.length > 0 && (
            <div className="space-y-1">
              {projectLinks.map((item) => (
                <NavLinkItem
                  key={item.path}
                  item={item}
                  active={isItemActive(location.pathname, item.path)}
                />
              ))}
            </div>
          )}
        </section>
      </div>

      <div className="border-t border-white/[0.08] px-4 py-4">
        <div className="mb-3 flex items-center justify-between text-[11px] uppercase tracking-wider text-text-quaternary">
          <span>Workflow</span>
          <span>{currentProjectId ? "Ready" : "Select project"}</span>
        </div>
        <div className="grid grid-cols-3 gap-1.5">
          {workflowLabels.map((label, index) => (
            <div key={label} className="rounded-md border border-white/[0.08] bg-white/[0.03] px-2 py-1.5 text-center">
              <div className="text-[10px] text-text-quaternary">{index + 1}</div>
              <div className="text-[11px] text-text-secondary">{label}</div>
            </div>
          ))}
        </div>
      </div>
    </aside>
    </>
  );
}
