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
        { path: `/projects/${currentProjectId}/scan-config`, label: "Scan Config" },
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
    <nav className="nav-cyber sticky top-0 z-50">
      <div className="flex items-center h-14 px-5 gap-1 max-w-6xl mx-auto">
        {/* Logo */}
        <Link
          to="/"
          className="flex items-center gap-2.5 mr-4 text-F0F6FC font-semibold text-[15px] tracking-tight group"
        >
          <div className="p-1.5 rounded-lg bg-38bdf8/10 border border-38bdf8/20 transition-all duration-300 group-hover:bg-38bdf8/20 group-hover:shadow-[0_0_15px_rgba(56,189,248,0.25)]">
            <svg
              className="w-4.5 h-4.5 text-38bdf8"
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
                className={`relative px-3 py-1.5 text-[13px] font-medium rounded-md transition-all duration-200 whitespace-nowrap ${
                  isActive
                    ? "text-38bdf8 bg-38bdf8/10 border-b-2 border-38bdf8"
                    : "text-94a3b8 hover:text-F0F6FC hover:bg-white/[0.04]"
                }`}
              >
                {item.label}
              </Link>
            );
          })}
        </div>

        {/* Divider */}
        {hasProjectSection && (
          <div className="w-px h-6 bg-38bdf8/20 mx-2 shrink-0" />
        )}

        {/* Project section */}
        {hasProjectSection && (
          <div className="flex items-center gap-0.5 overflow-x-auto">
            {/* Project Switcher */}
            <select
              value={currentProjectId}
              onChange={handleProjectSwitch}
              className="mr-2 h-8 px-2 text-[13px] font-medium rounded-lg bg-white/[0.06] text-F0F6FC border border-38bdf8/20 hover:bg-white/[0.08] focus:outline-none focus:ring-1 focus:ring-38bdf8 cursor-pointer"
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
                  className={`relative px-3 py-1.5 text-[13px] font-medium rounded-md transition-all duration-200 whitespace-nowrap ${
                    isActive
                      ? "text-38bdf8 bg-38bdf8/10 border-b-2 border-38bdf8"
                      : "text-94a3b8 hover:text-F0F6FC hover:bg-white/[0.04]"
                  }`}
                >
                  {item.label}
                </Link>
              );
            })}
          </div>
        )}
      </div>
    </nav>
  );
}
