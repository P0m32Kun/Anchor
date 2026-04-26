import { create } from "zustand";
import type { Project, Target, ScanTask } from "./api";

interface AppState {
  projects: Project[];
  currentProject: Project | null;
  targets: Target[];
  tasks: ScanTask[];
  setProjects: (p: Project[]) => void;
  setCurrentProject: (p: Project | null) => void;
  setTargets: (t: Target[]) => void;
  addTask: (t: ScanTask) => void;
  updateTask: (t: ScanTask) => void;
}

export const useStore = create<AppState>((set) => ({
  projects: [],
  currentProject: null,
  targets: [],
  tasks: [],
  setProjects: (projects) => set({ projects }),
  setCurrentProject: (currentProject) => set({ currentProject }),
  setTargets: (targets) => set({ targets }),
  addTask: (task) => set((state) => ({ tasks: [...state.tasks, task] })),
  updateTask: (task) =>
    set((state) => ({
      tasks: state.tasks.map((t) => (t.id === task.id ? task : t)),
    })),
}));
