import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import ScanModal from "./ScanModal";
import { DEFAULT_FFUF_DICTIONARY_ID } from "../lib/api";

const mockDictionary = {
  id: DEFAULT_FFUF_DICTIONARY_ID,
  name: "top100",
  category: "dirscan" as const,
  file_path: "/opt/dict/top100.txt",
  line_count: 100,
  size_bytes: 1024,
  builtin: true,
  enabled: true,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

vi.mock("../lib/api", async () => {
  const actual: any = await vi.importActual("../lib/api");
  return {
    ...actual,
    api: {
      ...actual.api,
      listDictionaries: vi.fn(() => Promise.resolve([mockDictionary])),
    },
  };
});

function openAdvanced() {
  fireEvent.click(screen.getByRole("button", { name: /高级选项/ }));
}

function goToStep2() {
  fireEvent.click(screen.getByRole("button", { name: /高级配置/ }));
}

async function ensureFfufMisconfigured() {
  goToStep2();
  openAdvanced();
  const ffufBtn = screen.getByText("Ffuf").closest("button");
  expect(ffufBtn).not.toBeNull();
  if (!screen.queryByLabelText("Ffuf 字典")) {
    fireEvent.click(ffufBtn!);
  }
  const select = await screen.findByLabelText("Ffuf 字典");
  fireEvent.change(select, { target: { value: "" } });
  await waitFor(() => {
    expect(screen.getByText(/请选择 Ffuf 字典,或关闭 Ffuf/)).toBeInTheDocument();
  });
}

describe("ScanModal — ffuf dictionary guard (Fix 3 frontend)", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("disables the start button when ffuf is enabled but no dictionary is selected", async () => {
    render(<ScanModal open onClose={() => {}} onStart={() => {}} />);
    await ensureFfufMisconfigured();

    const startBtn = screen.getByRole("button", { name: "立即启动扫描" });
    await waitFor(() => expect(startBtn).toBeDisabled());
    expect(startBtn).toHaveAttribute("title", expect.stringMatching(/请选择 Ffuf 字典/));
  });

  it("shows the inline misconfiguration message", async () => {
    render(<ScanModal open onClose={() => {}} onStart={() => {}} />);
    await ensureFfufMisconfigured();
    expect(screen.getByText(/请选择 Ffuf 字典,或关闭 Ffuf/)).toBeInTheDocument();
  });

  it("re-enables the start button when ffuf is disabled", async () => {
    render(<ScanModal open onClose={() => {}} onStart={() => {}} />);
    await ensureFfufMisconfigured();

    const ffufLabel = screen.getByText("Ffuf");
    const toggleBtn = ffufLabel.closest("button");
    expect(toggleBtn).not.toBeNull();
    fireEvent.click(toggleBtn!);

    const startBtn = screen.getByRole("button", { name: "立即启动扫描" });
    await waitFor(() => expect(startBtn).not.toBeDisabled());
    expect(startBtn).not.toHaveAttribute("title");
  });
});
