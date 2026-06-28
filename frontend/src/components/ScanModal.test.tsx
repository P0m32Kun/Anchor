import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import ScanModal from "./ScanModal";

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

function openAdvanced() {
  fireEvent.click(screen.getByRole("button", { name: /高级选项/ }));
}

function goToStep2() {
  fireEvent.click(screen.getByRole("button", { name: /高级配置/ }));
}

describe("ScanModal — ffuf toggle", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("allows starting a scan when ffuf is enabled without dictionary selection", async () => {
    render(<ScanModal open onClose={() => {}} onStart={() => {}} />);
    goToStep2();
    openAdvanced();
    const ffufBtn = screen.getByText("Ffuf").closest("button");
    expect(ffufBtn).not.toBeNull();
    fireEvent.click(ffufBtn!);

    const startBtn = screen.getByRole("button", { name: "立即启动扫描" });
    expect(startBtn).not.toBeDisabled();
  });

  it("allows starting a scan when ffuf is disabled", async () => {
    render(<ScanModal open onClose={() => {}} onStart={() => {}} />);
    goToStep2();
    openAdvanced();

    const startBtn = screen.getByRole("button", { name: "立即启动扫描" });
    expect(startBtn).not.toBeDisabled();
  });

  it("toggles ffuf settings visibility", async () => {
    render(<ScanModal open onClose={() => {}} onStart={() => {}} />);
    goToStep2();
    openAdvanced();

    const ffufBtn = screen.getByText("Ffuf").closest("button");
    expect(ffufBtn).not.toBeNull();
    fireEvent.click(ffufBtn!);

    expect(screen.getByText("目录与文件爆破")).toBeInTheDocument();
    expect(screen.getAllByRole("spinbutton").length).toBeGreaterThan(0);
  });
});
