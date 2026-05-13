import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import ScanModal from "./ScanModal";

// Stub the dictionary list API. enable_ffuf=true triggers a fetch on mount of
// step 2; returning [] keeps the modal in "no dictionaries available" mode,
// which mirrors the production default seed and exercises the ffufMisconfigured
// branch without needing a real backend.
vi.mock("../lib/api", async () => {
  const actual: any = await vi.importActual("../lib/api");
  return {
    ...actual,
    api: {
      ...actual.api,
      listDictionaries: vi.fn(() => Promise.resolve([])),
    },
  };
});

function goToStep2() {
  fireEvent.click(screen.getByRole("button", { name: /配置参数/ }));
}

describe("ScanModal — ffuf dictionary guard (Fix 3 frontend)", () => {
  beforeEach(() => {
    // ScanModal persists state in localStorage; clear between cases to avoid
    // bleed-over.
    window.localStorage.clear();
  });

  it("disables the start button when ffuf is enabled but no dictionary is selected", async () => {
    render(<ScanModal open onClose={() => {}} onStart={() => {}} />);
    goToStep2();

    // Default config has enable_ffuf=true and ffuf_dictionary_id="", so the
    // guard should fire immediately on step 2.
    const startBtn = screen.getByRole("button", { name: /立即启动扫描/ });
    expect(startBtn).toBeDisabled();
    expect(startBtn).toHaveAttribute("title", expect.stringMatching(/请选择 Ffuf 字典/));
  });

  it("shows the inline misconfiguration message", () => {
    render(<ScanModal open onClose={() => {}} onStart={() => {}} />);
    goToStep2();
    expect(screen.getByText(/请选择 Ffuf 字典,或关闭 Ffuf/)).toBeInTheDocument();
  });

  it("re-enables the start button when ffuf is disabled", () => {
    render(<ScanModal open onClose={() => {}} onStart={() => {}} />);
    goToStep2();

    // The Ffuf toggle button has two child divs ("Ffuf" + "目录与文件爆破").
    // Find it by walking up from the visible "Ffuf" label to its button parent.
    const ffufLabel = screen.getByText("Ffuf");
    const toggleBtn = ffufLabel.closest("button");
    expect(toggleBtn).not.toBeNull();
    fireEvent.click(toggleBtn!);

    const startBtn = screen.getByRole("button", { name: /立即启动扫描/ });
    expect(startBtn).not.toBeDisabled();
    expect(startBtn).not.toHaveAttribute("title");
  });
});
