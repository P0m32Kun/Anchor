import { create } from "zustand";
import type { Project, Target, ScanTask, Asset, WebEndpoint, Port, Service } from "./api";

interface AppState {
  projects: Project[];
  currentProject: Project | null;
  targets: Target[];
  tasks: ScanTask[];
  assets: Asset[];
  webEndpoints: WebEndpoint[];
  ports: Record<string, Port[]>;
  services: Record<string, Service[]>;
  setProjects: (p: Project[]) => void;
  setCurrentProject: (p: Project | null) => void;
  setTargets: (t: Target[]) => void;
  addTask: (t: ScanTask) => void;
  updateTask: (t: ScanTask) => void;
  setAssets: (a: Asset[]) => void;
  setWebEndpoints: (w: WebEndpoint[]) => void;
  setPorts: (assetId: string, p: Port[]) => void;
  setServices: (assetId: string, s: Service[]) => void;
}

export const useStore = create<AppState>((set) => ({
  projects: [],
  currentProject: null,
  targets: [],
  tasks: [],
  assets: [],
  webEndpoints: [],
  ports: {},
  services: {},
  setProjects: (projects) => set({ projects }),
  setCurrentProject: (currentProject) => set({ currentProject }),
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
}));
