import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { Project, Target, ScanTask, Asset, WebEndpoint, Port, Service, Finding, Evidence, Run } from "./api";

interface FindingStatusRecord {
  status: string;
  updatedAt: number;
}

interface AppState {
  projects: Project[];
  projectsLoading: boolean;
  projectsError: string | null;
  currentProjectId: string | null;
  currentProject: Project | null;
  targets: Target[];
  targetsLoading: boolean;
  targetsError: string | null;
  tasks: ScanTask[];
  assets: Asset[];
  assetsLoading: boolean;
  assetsError: string | null;
  webEndpoints: WebEndpoint[];
  ports: Record<string, Port[]>;
  services: Record<string, Service[]>;
  findings: Finding[];
  findingsLoading: boolean;
  findingsError: string | null;
  currentFinding: { finding: Finding; evidence: Evidence[] } | null;
  findingStatusHistory: Record<string, FindingStatusRecord>;
  runs: Run[];
  runsLoading: boolean;
  runsError: string | null;
  workersLoading: boolean;
  workersError: string | null;
  reportsLoading: boolean;
  reportsError: string | null;
  setProjects: (p: Project[]) => void;
  setProjectsLoading: (loading: boolean) => void;
  setProjectsError: (error: string | null) => void;
  setCurrentProjectId: (id: string | null) => void;
  setCurrentProject: (p: Project | null) => void;
  setTargets: (t: Target[]) => void;
  setTargetsLoading: (loading: boolean) => void;
  setTargetsError: (error: string | null) => void;
  addTask: (t: ScanTask) => void;
  updateTask: (t: ScanTask) => void;
  setAssets: (a: Asset[]) => void;
  setAssetsLoading: (loading: boolean) => void;
  setAssetsError: (error: string | null) => void;
  setWebEndpoints: (w: WebEndpoint[]) => void;
  setPorts: (assetId: string, p: Port[]) => void;
  setServices: (assetId: string, s: Service[]) => void;
  setFindings: (f: Finding[] | ((prev: Finding[]) => Finding[])) => void;
  setFindingsLoading: (loading: boolean) => void;
  setFindingsError: (error: string | null) => void;
  setCurrentFinding: (f: { finding: Finding; evidence: Evidence[] } | null) => void;
  recordStatusChange: (id: string, status: string) => void;
  setRuns: (runs: Run[]) => void;
  setRunsLoading: (loading: boolean) => void;
  setRunsError: (error: string | null) => void;
  setWorkersLoading: (loading: boolean) => void;
  setWorkersError: (error: string | null) => void;
  setReportsLoading: (loading: boolean) => void;
  setReportsError: (error: string | null) => void;
}

export const useStore = create<AppState>()(
  persist(
    (set) => ({
      projects: [],
      projectsLoading: false,
      projectsError: null,
      currentProjectId: null,
      currentProject: null,
      targets: [],
      targetsLoading: false,
      targetsError: null,
      tasks: [],
      assets: [],
      assetsLoading: false,
      assetsError: null,
      webEndpoints: [],
      ports: {},
      services: {},
      findings: [],
      findingsLoading: false,
      findingsError: null,
      currentFinding: null,
      findingStatusHistory: {},
      runs: [],
      runsLoading: false,
      runsError: null,
      workersLoading: false,
      workersError: null,
      reportsLoading: false,
      reportsError: null,

      setProjects: (projects) => set({ projects }),
      setProjectsLoading: (projectsLoading) => set({ projectsLoading }),
      setProjectsError: (projectsError) => set({ projectsError }),

      setCurrentProjectId: (currentProjectId) =>
        set({
          currentProjectId,
          currentProject: null,
          targets: [],
          targetsLoading: false,
          targetsError: null,
          tasks: [],
          assets: [],
          assetsLoading: false,
          assetsError: null,
          webEndpoints: [],
          ports: {},
          services: {},
          findings: [],
          findingsLoading: false,
          findingsError: null,
          currentFinding: null,
          runs: [],
          runsLoading: false,
          runsError: null,
        }),

      setCurrentProject: (currentProject) => set({ currentProject }),

      setTargets: (targets) => set({ targets }),
      setTargetsLoading: (targetsLoading) => set({ targetsLoading }),
      setTargetsError: (targetsError) => set({ targetsError }),

      addTask: (task) => set((state) => ({ tasks: [...state.tasks, task] })),
      updateTask: (task) =>
        set((state) => ({
          tasks: state.tasks.map((t) => (t.id === task.id ? task : t)),
        })),

      setAssets: (assets) => set({ assets }),
      setAssetsLoading: (assetsLoading) => set({ assetsLoading }),
      setAssetsError: (assetsError) => set({ assetsError }),

      setWebEndpoints: (webEndpoints) => set({ webEndpoints }),
      setPorts: (assetId, ports) =>
        set((state) => ({ ports: { ...state.ports, [assetId]: ports } })),
      setServices: (assetId, services) =>
        set((state) => ({ services: { ...state.services, [assetId]: services } })),

      setFindings: (findings) =>
        set((state) => ({
          findings:
            typeof findings === "function"
              ? findings(state.findings)
              : findings,
        })),
      setFindingsLoading: (findingsLoading) => set({ findingsLoading }),
      setFindingsError: (findingsError) => set({ findingsError }),
      setCurrentFinding: (currentFinding) => set({ currentFinding }),
      recordStatusChange: (id, status) =>
        set((state) => ({
          findingStatusHistory: {
            ...state.findingStatusHistory,
            [id]: { status, updatedAt: Date.now() },
          },
        })),

      setRuns: (runs) => set({ runs }),
      setRunsLoading: (runsLoading) => set({ runsLoading }),
      setRunsError: (runsError) => set({ runsError }),
      setWorkersLoading: (workersLoading) => set({ workersLoading }),
      setWorkersError: (workersError) => set({ workersError }),
      setReportsLoading: (reportsLoading) => set({ reportsLoading }),
      setReportsError: (reportsError) => set({ reportsError }),
    }),
    {
      name: "app-store",
      partialize: (state) => ({ currentProjectId: state.currentProjectId }),
    }
  )
);
