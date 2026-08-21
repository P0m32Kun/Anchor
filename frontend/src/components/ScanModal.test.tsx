import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import ScanModal from "./ScanModal";
import { api } from "../lib/api";

vi.mock("../lib/api", async () => {
  const actual: any = await vi.importActual("../lib/api");
  return {
    ...actual,
    api: {
      ...actual.api,
      listDictionaries: vi.fn(() => Promise.resolve([])),
      getPipelineConfig: vi.fn(() => Promise.reject(new Error("no project config"))),
      getScanDefaults: vi.fn(() =>
        Promise.resolve({
          high_risk_ports: actual.DEFAULT_HIGH_RISK_PORTS,
          ffuf_dictionary_default: "",
          presets: {},
          junk_keyword_count: 0,
          exclude_domain_count: 0,
        })
      ),
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
    vi.clearAllMocks();
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

  it("uses deployment scan defaults for quick start", async () => {
    vi.mocked(api.getScanDefaults).mockResolvedValue({
      high_risk_ports: "80,443",
      ffuf_dictionary_default: "",
      presets: {
        internal: {
          enable_cdn_filter: true,
          port_range: "full",
          naabu_rate: 321,
          naabu_threads: 88,
          naabu_timeout: 5000,
          enable_nmap_service: true,
          nmap_service_timeout: 180,
          enable_httpx: true,
          httpx_rate_limit: 150,
          httpx_threads: 50,
          enable_nuclei: true,
          nuclei_rate_limit: 100,
          nuclei_rate_limit_per_min: 0,
          nuclei_concurrency: 25,
          nuclei_scan_depth: "tags",
          enable_ffuf: true,
          ffuf_rate_limit: 6,
          ffuf_timeout: 30,
          katana_timeout: 10,
          enable_katana: true,
          katana_max_depth: 2,
          katana_rate_limit: 10,
          skip_portscan_on_cdn_host: false,
          nuclei_require_fingerprint: false,
          passive_search_result_limit: 500,
          passive_search_concurrency: 3,
          enable_passive_junk_filter: false,
          enable_spoor: false,
        },
      },
      junk_keyword_count: 0,
      exclude_domain_count: 0,
    });

    const onStart = vi.fn();
    render(<ScanModal open onClose={() => {}} onStart={onStart} projectId="proj-1" />);

    await waitFor(() => expect(api.getScanDefaults).toHaveBeenCalled());
    fireEvent.click(screen.getByRole("button", { name: "内网扫描" }));
    fireEvent.click(screen.getByRole("button", { name: "快速启动" }));

    expect(onStart).toHaveBeenCalledWith(
      "internal",
      expect.objectContaining({
        port_range: "full",
        naabu_rate: 321,
        naabu_threads: 88,
      })
    );
  });
});
