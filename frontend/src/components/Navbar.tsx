import { Link, useLocation, useNavigate } from "react-router-dom";
import { useStore } from "../lib/store";

const globalNavItems = [
  { path: "/", label: "Dashboard" },
  { path: "/projects", label: "Projects" },
  { path: "/workers", label: "Workers" },
  { path: "/settings", label: "Settings" },
];

function isItemActive(locationPathname: string, itemPath: string): boolean {
  if (itemPath === "/") {
    return locationPathname === "/";
  }
  return locationPathname.startsWith(itemPath);
}

export function Navbar() {
  const location = useLocation();
  const navigate = useNavigate();
  const currentProjectId = useStore((s) => s.currentProjectId);
  const projects = useStore((s) => s.projects);
  const setCurrentProjectId = useStore((s) => s.setCurrentProjectId);

  const projectLinks = currentProjectId
    ? [
        { path: `/projects/${currentProjectId}/targets`, label: "Targets" },
        { path: `/projects/${currentProjectId}/assets`, label: "Assets" },
        { path: `/projects/${currentProjectId}/runs`, label: "Runs" },
        { path: `/projects/${currentProjectId}/findings`, label: "Findings" },
        { path: `/projects/${currentProjectId}/reports`, label: "Reports" },
      ]
    : [];

  const hasProjectSection = projects.length > 0 && currentProjectId;

  const handleProjectSwitch = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const id = e.target.value;
    if (id) {
      setCurrentProjectId(id);
      navigate(`/projects/${id}/targets`);
    }
  };

  return (
    <nav className="nav-glass-dark sticky top-0 z-50">
      <div className="flex items-center h-14 px-5 gap-1 max-w-6xl mx-auto">
        {/* Logo */}
        <Link
          to="/"
          className="flex items-center gap-2.5 mr-4 text-text-primary font-semibold text-[15px] tracking-tight group"
        >
          <div className="p-1.5 rounded-lg bg-brand-primary/10 border border-brand-primary/20 transition-all duration-300 group-hover:bg-brand-primary/20 group-hover:shadow-[0_0_15px_rgba(47,129,247,0.25)]">
            <svg
              className="w-4.5 h-4.5 text-brand-primary"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M12 2L2 7l10 5 10-5-10-5z" />
              <path d="M2 17l10 5 10-5" />
              <path d="M2 12l10 5 10-5" />
            </svg>
          </div>
          Anchor
        </Link>

        {/* Global nav items */}
        <div className="flex items-center gap-0.5 overflow-x-auto">
          {globalNavItems.map((item) => {
            const isActive = isItemActive(location.pathname, item.path);
            return (
              <Link
                key={item.path}
                to={item.path}
                aria-current={isActive ? "page" : undefined}
                className={`relative px-3 py-2 text-[13px] font-medium rounded-lg transition-all duration-200 whitespace-nowrap ${
                  isActive
                    ? "text-text-primary bg-white/[0.06]"
                    : "text-text-tertiary hover:text-text-secondary hover:bg-white/[0.04]"
                }`}
              >
                {item.label}
                {isActive && (
                  <span className="absolute bottom-0 left-1/2 -translate-x-1/2 w-4 h-[2px] bg-brand-primary rounded-full shadow-[0_0_8px_rgba(47,129,247,0.6)]" />
                )}
              </Link>
            );
          })}
        </div>

        {/* Divider */}
        {hasProjectSection && (
          <div className="w-px h-6 bg-white/10 mx-2 shrink-0" />
        )}

        {/* Project section */}
        {hasProjectSection && (
          <div className="flex items-center gap-0.5 overflow-x-auto">
            {/* Project Switcher */}
            <select
              value={currentProjectId}
              onChange={handleProjectSwitch}
              className="mr-2 h-8 px-2 text-[13px] font-medium rounded-lg bg-white/[0.06] text-text-primary border border-white/10 hover:bg-white/[0.08] focus:outline-none focus:ring-1 focus:ring-brand-primary cursor-pointer"
              aria-label="Switch project"
            >
              {projects.map((project) => (
                <option key={project.id} value={project.id}>
                  {project.name}
                </option>
              ))}
            </select>

            {/* Project nav items */}
            {projectLinks.map((item) => {
              const isActive = isItemActive(location.pathname, item.path);
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  aria-current={isActive ? "page" : undefined}
                  className={`relative px-3 py-2 text-[13px] font-medium rounded-lg transition-all duration-200 whitespace-nowrap ${
                    isActive
                      ? "text-text-primary bg-white/[0.06]"
                      : "text-text-tertiary hover:text-text-secondary hover:bg-white/[0.04]"
                  }`}
                >
                  {item.label}
                  {isActive && (
                    <span className="absolute bottom-0 left-1/2 -translate-x-1/2 w-4 h-[2px] bg-brand-primary rounded-full shadow-[0_0_8px_rgba(47,129,247,0.6)]" />
                  )}
                </Link>
              );
            })}
          </div>
        )}
      </div>
    </nav>
  );
}
