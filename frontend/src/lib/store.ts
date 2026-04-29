import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { Project, Target, ScanTask, Asset, WebEndpoint, Port, Service, Finding, Evidence } from "./api";

interface AppState {
  projects: Project[];
  currentProjectId: string | null;
  currentProject: Project | null;
  targets: Target[];
  tasks: ScanTask[];
  assets: Asset[];
  webEndpoints: WebEndpoint[];
  ports: Record<string, Port[]>;
  services: Record<string, Service[]>;
  findings: Finding[];
  currentFinding: { finding: Finding; evidence: Evidence[] } | null;
  setProjects: (p: Project[]) => void;
  setCurrentProjectId: (id: string | null) => void;
  setCurrentProject: (p: Project | null) => void;
  setTargets: (t: Target[]) => void;
  addTask: (t: ScanTask) => void;
  updateTask: (t: ScanTask) => void;
  setAssets: (a: Asset[]) => void;
  setWebEndpoints: (w: WebEndpoint[]) => void;
  setPorts: (assetId: string, p: Port[]) => void;
  setServices: (assetId: string, s: Service[]) => void;
  setFindings: (f: Finding[]) => void;
  setCurrentFinding: (f: { finding: Finding; evidence: Evidence[] } | null) => void;
}

export const useStore = create<AppState>()(
  persist(
    (set) => ({
      projects: [],
      currentProjectId: null,
      currentProject: null,
      targets: [],
      tasks: [],
      assets: [],
      webEndpoints: [],
      ports: {},
      services: {},
      findings: [],
      currentFinding: null,
      setProjects: (projects) => set({ projects }),
      setCurrentProjectId: (currentProjectId) => set({ currentProjectId }),
      setCurrentProject: (currentProject) =>
        set({
          currentProject,
          targets: [],
          tasks: [],
          assets: [],
          webEndpoints: [],
          ports: {},
          services: {},
          findings: [],
          currentFinding: null,
        }),
      setTargets: (targets) => set({ targets }),
      addTask: (task) => set((state) => ({ tasks: [...state.tasks, task] })),
      updateTask: (task) =>
        set((state) => ({
          tasks: state.tasks.map((t) => (t.id === task.id ? task : t)),
        })),
      setAssets: (assets) => set({ assets }),
      setWebEndpoints: (webEndpoints) => set({ webEndpoints }),
      setPorts: (assetId, ports) =>
        set((state) => ({ ports: { ...state.ports, [assetId]: ports } })),
      setServices: (assetId, services) =>
        set((state) => ({ services: { ...state.services, [assetId]: services } })),
      setFindings: (findings) => set({ findings }),
      setCurrentFinding: (currentFinding) => set({ currentFinding }),
    }),
    {
      name: "app-store",
      partialize: (state) => ({ currentProject: state.currentProject, currentProjectId: state.currentProjectId }),
    }
  )
);
