package workflow

// Tool execution helpers — moved to individual pipeline_tool_*.go files:
//   pipeline_tool_subfinder.go  — runSubfinder
//   pipeline_tool_dnsx.go       — runDNSx
//   pipeline_tool_httpx.go      — runHttpx, prepareHttpxFingerprints, mergeFingerprintFiles
//   pipeline_tool_naabu.go      — runNaabu
//   pipeline_tool_nmap.go       — runNmapAlive, runNmapServiceScan
//   pipeline_tool_cdncheck.go   — runCDNCheck
//   pipeline_tool_nuclei.go     — runNucleiWeb, runNucleiNonWeb, customWorkflowPaths
//   pipeline_tool_ffuf.go       — runFfuf
//   pipeline_tool_legacy.go     — createAndRunTask, legacyCreateAndRunTask, readTaskStdout
//   pipeline_tool_passive.go    — recordPassiveTask
