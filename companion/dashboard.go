package companion

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

// Dashboard 封装本地桌面工作台。
type Dashboard struct {
	service   *Service
	addr      string
	autoOpen  bool
	dataDir   string
	indexTmpl *template.Template
}

// NewDashboard 创建桌面工作台。
func NewDashboard(service *Service, addr string, autoOpen bool) *Dashboard {
	return &Dashboard{
		service:   service,
		addr:      addr,
		autoOpen:  autoOpen,
		dataDir:   service.config.App.DataDir,
		indexTmpl: template.Must(template.New("index").Parse(indexHTML)),
	}
}

// Run 启动桌面工作台并监听到上下文结束。
func (d *Dashboard) Run(ctx context.Context) error {
	if err := d.service.Bootstrap(ctx); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", d.handleIndex)
	mux.HandleFunc("/api/state", d.handleState)
	mux.HandleFunc("/api/run", d.handleRun)
	mux.HandleFunc("/api/action/execute", d.handleActionExecute)
	mux.HandleFunc("/api/capture/frame", d.handleCaptureFrame)
	mux.HandleFunc("/api/capture/clear", d.handleCaptureClear)
	mux.Handle("/reports/", http.StripPrefix("/reports/", http.FileServer(http.Dir(d.dataDir+"/reports"))))
	mux.Handle("/runs/", http.StripPrefix("/runs/", http.FileServer(http.Dir(d.dataDir+"/runs"))))
	mux.Handle("/exports/", http.StripPrefix("/exports/", http.FileServer(http.Dir(d.dataDir+"/exports"))))
	mux.Handle("/captures/", http.StripPrefix("/captures/", http.FileServer(http.Dir(d.dataDir+"/captures"))))
	mux.Handle("/project-files/", http.StripPrefix("/project-files/", http.FileServer(http.Dir(d.service.projectRoot()))))
	mux.Handle("/generated-files/", http.StripPrefix("/generated-files/", http.FileServer(http.Dir(d.service.generatedRoot()))))

	server := &http.Server{
		Addr:              d.addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	if d.autoOpen {
		go func() {
			time.Sleep(500 * time.Millisecond)
			_ = openBrowser("http://" + d.addr)
		}()
	}

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title string
	}{
		Title: d.service.config.App.Name,
	}
	_ = d.indexTmpl.Execute(w, data)
}

func (d *Dashboard) handleState(w http.ResponseWriter, r *http.Request) {
	state, err := d.service.State(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, state)
}

func (d *Dashboard) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Goal string `json:"goal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	report, err := d.service.Execute(r.Context(), payload.Goal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, report)
}

func (d *Dashboard) handleActionExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		RunID    string `json:"run_id"`
		ActionID string `json:"action_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	report, err := d.service.ExecuteAction(r.Context(), payload.RunID, payload.ActionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, report)
}

func (d *Dashboard) handleCaptureFrame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input ScreenCaptureInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	screen, err := d.service.SaveScreenCapture(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, screen)
}

func (d *Dashboard) handleCaptureClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	screen := d.service.ClearScreenCapture()
	writeJSON(w, screen)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}

func openBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", rawURL)
	case "darwin":
		cmd = exec.Command("open", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}

const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    :root {
      --ink: #203040;
      --muted: #5f6f7d;
      --line: rgba(32,48,64,0.12);
      --card: rgba(255,255,255,0.84);
      --accent: #0c6c67;
      --accent-soft: rgba(12,108,103,0.10);
      --success: #2f855a;
      --warning: #b7791f;
      --danger: #c53030;
      --shadow: 0 24px 80px rgba(45, 57, 72, 0.14);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      color: var(--ink);
      font-family: "PingFang SC","Hiragino Sans GB","Microsoft YaHei","Segoe UI",sans-serif;
      background:
        radial-gradient(circle at 12% 8%, rgba(12,108,103,0.16), transparent 22%),
        radial-gradient(circle at 90% 2%, rgba(209,99,61,0.14), transparent 18%),
        linear-gradient(180deg, #fffaf3 0%, #efe4d4 100%);
    }
    a { color: #b8542f; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .shell {
      width: min(1240px, calc(100% - 28px));
      margin: 20px auto 44px;
    }
    .hero, .panel, .result-card {
      background: var(--card);
      backdrop-filter: blur(12px);
      border: 1px solid rgba(255,255,255,0.74);
      border-radius: 28px;
      box-shadow: var(--shadow);
    }
    .hero {
      padding: 30px;
      display: grid;
      grid-template-columns: 1.15fr .85fr;
      gap: 18px;
      align-items: start;
    }
    .hero h1 {
      margin: 0 0 12px;
      font-size: clamp(36px, 6vw, 62px);
      line-height: 0.92;
      letter-spacing: -0.05em;
    }
    .hero p {
      margin: 0;
      color: var(--muted);
      line-height: 1.8;
      max-width: 760px;
    }
    .meta-stack {
      display: grid;
      gap: 12px;
    }
    .meta-card, .metric, .result-box, .provider-item, .run-item, .action-item {
      border-radius: 20px;
      border: 1px solid var(--line);
      background: rgba(255,255,255,0.62);
    }
    .meta-card {
      padding: 16px 18px;
    }
    .meta-card strong {
      display: block;
      margin-bottom: 4px;
      font-size: 14px;
      letter-spacing: .05em;
      text-transform: uppercase;
    }
    .muted { color: var(--muted); line-height: 1.7; }
    .grid {
      margin-top: 18px;
      display: grid;
      grid-template-columns: repeat(12, minmax(0, 1fr));
      gap: 18px;
    }
    .panel { padding: 22px; }
    .span-12 { grid-column: span 12; }
    .span-7 { grid-column: span 7; }
    .span-5 { grid-column: span 5; }
    .span-4 { grid-column: span 4; }
    h2 {
      margin: 0 0 16px;
      font-size: 14px;
      letter-spacing: .16em;
      text-transform: uppercase;
      color: #536171;
    }
    textarea {
      width: 100%;
      min-height: 220px;
      resize: vertical;
      border-radius: 20px;
      border: 1px solid var(--line);
      padding: 18px;
      font: inherit;
      font-size: 18px;
      color: var(--ink);
      background: rgba(255,255,255,0.64);
      outline: none;
    }
    textarea:focus {
      border-color: rgba(12,108,103,0.42);
      box-shadow: 0 0 0 4px rgba(12,108,103,0.08);
    }
    .toolbar {
      margin-top: 14px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 14px;
      flex-wrap: wrap;
    }
    .hint {
      color: var(--muted);
      line-height: 1.7;
      max-width: 720px;
    }
    button {
      border: none;
      border-radius: 999px;
      padding: 13px 24px;
      font: inherit;
      font-weight: 700;
      color: white;
      background: linear-gradient(135deg, var(--accent), #17807a);
      cursor: pointer;
      transition: transform .18s ease, box-shadow .18s ease, opacity .18s ease;
      box-shadow: 0 14px 32px rgba(12,108,103,0.26);
    }
    button:hover:not(:disabled) { transform: translateY(-1px); }
    button:disabled { opacity: .7; cursor: wait; }
    button.secondary {
      color: var(--ink);
      background: rgba(255,255,255,0.82);
      border: 1px solid var(--line);
      box-shadow: none;
    }
    .cards, .result-grid {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 14px;
    }
    .metric {
      padding: 16px;
      min-height: 122px;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
    }
    .metric strong {
      font-size: 28px;
      line-height: 1;
    }
    .metric span {
      font-size: 18px;
      line-height: 1.35;
    }
    .provider-list, .run-list, .box-list, .action-list {
      list-style: none;
      margin: 0;
      padding: 0;
      display: grid;
      gap: 12px;
    }
    .provider-item, .run-item {
      padding: 16px 18px;
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 12px;
    }
    .provider-item strong, .run-item strong {
      display: block;
      margin-bottom: 4px;
      font-size: 16px;
    }
    .status {
      border-radius: 999px;
      padding: 7px 13px;
      font-size: 12px;
      font-weight: 700;
      letter-spacing: .08em;
      text-transform: uppercase;
      white-space: nowrap;
    }
    .status.ready, .status-completed, .status-passed { color: var(--success); background: rgba(47,133,90,0.12); }
    .status.down, .status-failed, .status-error { color: var(--danger); background: rgba(197,48,48,0.12); }
    .status.status-running { color: var(--accent); background: rgba(12,108,103,0.12); }
    .status.status-pending, .status.status-waiting, .status.status-idle { color: var(--muted); background: rgba(98,112,125,0.12); }
    .status.status-skipped { color: var(--warning); background: rgba(183,121,31,0.14); }
    .progress-meter {
      width: 100%;
      height: 10px;
      border-radius: 999px;
      background: rgba(98,112,125,0.12);
      overflow: hidden;
    }
    .progress-fill {
      height: 100%;
      width: 0%;
      background: linear-gradient(135deg, var(--accent), #1a8a83);
      transition: width .28s ease;
    }
    .inline-result {
      margin-top: 14px;
      display: none;
      border-radius: 20px;
      border: 1px solid rgba(12,108,103,0.14);
      background: var(--accent-soft);
      padding: 16px 18px;
    }
    .inline-result.visible { display: block; }
    .inline-result strong {
      display: block;
      font-size: 14px;
      letter-spacing: .12em;
      text-transform: uppercase;
      color: var(--accent);
      margin-bottom: 10px;
    }
    .inline-result p {
      margin: 0;
      line-height: 1.75;
      font-size: 16px;
    }
    .result-card {
      margin-top: 18px;
      padding: 22px;
      display: none;
    }
    .result-card.visible { display: block; }
    .result-head {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 14px;
      margin-bottom: 16px;
      flex-wrap: wrap;
    }
    .result-box {
      padding: 16px 18px;
      min-height: 148px;
    }
    .result-box h3 {
      margin: 0 0 12px;
      font-size: 14px;
      letter-spacing: .12em;
      text-transform: uppercase;
      color: #5d6d7d;
    }
    .result-box ul {
      list-style: none;
      margin: 0;
      padding: 0;
      display: grid;
      gap: 10px;
    }
    .result-box li {
      padding: 12px 14px;
      border-radius: 16px;
      border: 1px solid var(--line);
      background: rgba(255,255,255,0.54);
    }
    .result-box li strong {
      display: block;
      margin-bottom: 4px;
      font-size: 15px;
    }
    .mono {
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      font-size: 12px;
      line-height: 1.55;
      white-space: pre-wrap;
      word-break: break-word;
    }
    .action-stack {
      margin: 22px 0 26px;
      padding: 16px;
      border-radius: 24px;
      border: 1px solid var(--line);
      background: rgba(255,255,255,0.56);
      display: none;
    }
    .action-stack h3 {
      margin: 0 0 14px;
      font-size: 14px;
      letter-spacing: .14em;
      text-transform: uppercase;
      color: #5a6878;
    }
    .action-item { padding: 16px; }
    .action-row {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 14px;
    }
    .action-copy { flex: 1; }
    .action-controls {
      display: flex;
      flex-direction: column;
      align-items: flex-end;
      gap: 10px;
    }
    .action-output {
      margin-top: 12px;
      padding: 12px 14px;
      border-radius: 16px;
      background: rgba(34,49,63,0.06);
      color: var(--muted);
      white-space: pre-wrap;
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      font-size: 12px;
      line-height: 1.5;
      max-height: 220px;
      overflow: auto;
    }
    @media (max-width: 980px) {
      .hero { grid-template-columns: 1fr; }
      .span-7, .span-5, .span-4, .span-12 { grid-column: span 12; }
      .cards, .result-grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div>
        <h1>Go Project<br>Agent</h1>
        <p>现在这页只保留面向 Go 项目的核心闭环：输入问题，限定分析边界，读取项目上下文，输出分析结果、生成代码、预检结论和可导出的产物。生成代码和被分析项目会分区展示，不再混成一块。</p>
      </div>
      <div class="meta-stack">
        <div class="meta-card"><strong>Project Root</strong><div class="muted" id="projectRootValue">loading...</div></div>
        <div class="meta-card"><strong>Generated Root</strong><div class="muted" id="generatedRootValue">loading...</div></div>
        <div class="meta-card"><strong>Analysis Boundary</strong><div class="muted" id="boundaryMeta">loading...</div></div>
        <div class="meta-card"><strong>Workflow Backend</strong><div class="muted" id="workflowBackendValue">builtin</div></div>
      </div>
    </section>

    <section class="grid">
      <section class="panel span-7">
        <h2>Mission Input</h2>
        <textarea id="goalInput">请基于当前 Go 项目给出技术分析，或在生成目录里创建一个最小可运行的 Go 子项目，并明确输出哪些文件、验证是否通过。</textarea>
        <div class="toolbar">
          <div class="hint">分析只会发生在 app.workspace 指定的项目根目录内；新生成的项目默认写到 app.generated_root。如果只是问答或文档模式，不会随便改当前项目。</div>
          <button id="runButton">提交任务</button>
        </div>
        <div class="inline-result" id="inlineResult">
          <strong>Latest Output</strong>
          <p id="inlineResultText">任务完成后，最新摘要会先显示在这里。</p>
          <div class="muted" id="inlineResultMeta" style="margin-top:8px;"></div>
        </div>
      </section>

      <section class="panel span-5">
        <h2>Project Boundary</h2>
        <ul class="box-list" id="boundaryList"></ul>
      </section>

      <section class="panel span-4">
        <h2>System Snapshot</h2>
        <div class="cards">
          <div class="metric"><strong id="providerCount">0</strong><span>Configured Providers</span></div>
          <div class="metric"><strong id="symbolCount">0</strong><span>Indexed Symbols</span></div>
          <div class="metric"><strong id="knowledgeCount">0</strong><span>Knowledge Chunks</span></div>
        </div>
        <div style="margin-top:14px;" class="muted" id="dockerStatus">loading...</div>
        <div style="margin-top:8px;" class="muted" id="redisStatus">loading...</div>
        <div style="margin-top:8px;" class="muted" id="visionStatus">loading...</div>
      </section>

      <section class="panel span-4">
        <h2>Loop Engine</h2>
        <div class="provider-item">
          <div>
            <strong id="workflowStatusText">idle</strong>
            <div class="muted" id="workflowPhaseText">ready</div>
            <div class="muted" id="workflowMessageText">工作台就绪，等待任务</div>
          </div>
          <span id="workflowBadge" class="status status-idle">idle</span>
        </div>
        <div class="progress-meter"><div class="progress-fill" id="workflowProgressFill"></div></div>
        <div class="muted" id="workflowProgressText" style="margin-top:8px;">0% · waiting</div>
      </section>

      <section class="panel span-4">
        <h2>Provider Ladder</h2>
        <ul class="provider-list" id="providerList"></ul>
      </section>

      <section class="panel span-12">
        <h2>Recent Runs</h2>
        <ul class="run-list" id="runList"></ul>
      </section>
    </section>

    <section class="result-card" id="resultCard">
      <div class="result-head">
        <div>
          <h2 style="margin:0;font-size:28px;">Latest Run</h2>
          <div class="muted" id="resultMeta"></div>
        </div>
        <div class="muted" id="resultLinks"></div>
      </div>
      <p id="overviewText" style="font-size:16px;line-height:1.8;"></p>
      <div class="action-stack" id="manualActionStack">
        <h3>Manual Recovery Actions</h3>
        <ul id="approvalActionList" class="action-list"></ul>
      </div>
      <div class="result-grid">
        <div class="result-box"><h3>Task Summary</h3><ul id="summaryList"></ul></div>
        <div class="result-box"><h3>Generated Output</h3><ul id="generatedOutputList"></ul></div>
        <div class="result-box"><h3>Generated Files</h3><ul id="generatedFilesList"></ul></div>
        <div class="result-box"><h3>Validation Checks</h3><ul id="validationChecksList"></ul></div>
        <div class="result-box"><h3>Validation Scope</h3><ul id="validationScopeList"></ul></div>
        <div class="result-box"><h3>Exported Files</h3><ul id="artifactList"></ul></div>
        <div class="result-box"><h3>Execution Steps</h3><ul id="stepList"></ul></div>
        <div class="result-box"><h3>Snapshot</h3><ul id="snapshotList"></ul></div>
        <div class="result-box"><h3>Troubleshooting</h3><ul id="troubleshootList"></ul></div>
      </div>
    </section>
  </div>

  <script>
    const providerList = document.getElementById("providerList");
    const runList = document.getElementById("runList");
    const runButton = document.getElementById("runButton");
    const resultCard = document.getElementById("resultCard");
    const workflowBackendValue = document.getElementById("workflowBackendValue");
    const workflowBadge = document.getElementById("workflowBadge");
    const workflowStatusText = document.getElementById("workflowStatusText");
    const workflowPhaseText = document.getElementById("workflowPhaseText");
    const workflowMessageText = document.getElementById("workflowMessageText");
    const workflowProgressFill = document.getElementById("workflowProgressFill");
    const workflowProgressText = document.getElementById("workflowProgressText");
    const manualActionStack = document.getElementById("manualActionStack");
    const inlineResult = document.getElementById("inlineResult");
    const inlineResultText = document.getElementById("inlineResultText");
    const inlineResultMeta = document.getElementById("inlineResultMeta");
    const approvalActionList = document.getElementById("approvalActionList");

    function escapeHTML(value) {
      return String(value || "")
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
    }

    function statusClass(status) {
      const value = String(status || "idle").toLowerCase();
      if (value === "completed" || value === "passed" || value === "ready") return "status-completed";
      if (value === "failed" || value === "error" || value === "down") return "status-failed";
      if (value === "running") return "status-running";
      if (value === "skipped") return "status-skipped";
      return "status-idle";
    }

    function renderEntries(el, items, emptyText) {
      el.innerHTML = "";
      if (!items || items.length === 0) {
        const li = document.createElement("li");
        li.innerHTML = "<span class='muted'>" + escapeHTML(emptyText || "暂无数据") + "</span>";
        el.appendChild(li);
        return;
      }
      items.forEach(function(item) {
        const li = document.createElement("li");
        li.innerHTML = item;
        el.appendChild(li);
      });
    }

    function renderBoundary(state) {
      document.getElementById("projectRootValue").textContent = state.project_root || state.workspace || "n/a";
      document.getElementById("generatedRootValue").textContent = state.generated_root || "n/a";
      document.getElementById("boundaryMeta").textContent =
        "include " + (state.analysis_scope || []).length + " 项 / exclude " + (state.excluded_names || []).length + " 项";
      renderEntries(document.getElementById("boundaryList"), [
        "<strong>分析项目根目录</strong><div class='muted mono'>" + escapeHTML(state.project_root || state.workspace || "n/a") + "</div>",
        "<strong>生成输出目录</strong><div class='muted mono'>" + escapeHTML(state.generated_root || "n/a") + "</div>",
        "<strong>纳入上下文的范围</strong><div class='muted'>" + escapeHTML((state.analysis_scope || []).join("、") || "未配置") + "</div>",
        "<strong>默认忽略目录</strong><div class='muted'>" + escapeHTML((state.excluded_names || []).join("、") || "无") + "</div>"
      ], "暂无边界配置");
    }

    function renderWorkflow(workflow) {
      if (!workflow) return;
      const status = workflow.status || "idle";
      const progress = inferWorkflowProgress(status, workflow.phase || "ready");
      workflowBadge.className = "status " + statusClass(status);
      workflowBadge.textContent = status;
      workflowStatusText.textContent = status;
      workflowPhaseText.textContent = workflow.phase || "ready";
      workflowMessageText.textContent = workflow.message || "工作台就绪，等待任务";
      workflowProgressFill.style.width = progress + "%";
      workflowProgressText.textContent = progress + "% · " + (workflow.phase || "ready");
    }

    function inferWorkflowProgress(status, phase) {
      if (status === "error") return 100;
      if (status !== "running") return 0;
      switch (phase) {
        case "snapshot": return 12;
        case "context": return 28;
        case "planning": return 52;
        case "codegen": return 66;
        case "sandbox": return 78;
        case "preflight": return 88;
        case "persisting": return 96;
        default: return 18;
      }
    }

    function buildResultLinks(run) {
      const links = [
        "<a href='" + escapeHTML(run.markdown_url) + "' target='_blank'>Markdown</a>",
        "<a href='" + escapeHTML(run.json_url) + "' target='_blank'>JSON</a>"
      ];
      (run.artifacts || []).forEach(function(item) {
        if (item.url) {
          links.push("<a href='" + escapeHTML(item.url) + "' target='_blank'>" + escapeHTML(item.name) + "</a>");
        }
      });
      return links.join(" · ");
    }

    function buildSummaryItems(run) {
      const plan = run.plan || {};
      const items = [
        "<strong>任务模式</strong><div class='muted'>" + escapeHTML(plan.mode || "general") + "</div>"
      ];
      if ((plan.actions || []).length > 0) {
        items.push("<strong>执行动作</strong><div class='muted'>" + escapeHTML(plan.actions.join("；")) + "</div>");
      }
      if ((plan.deliverables || []).length > 0) {
        items.push("<strong>交付物</strong><div class='muted'>" + escapeHTML(plan.deliverables.join("、")) + "</div>");
      }
      if ((plan.progress_signals || []).length > 0) {
        items.push("<strong>进度信号</strong><div class='muted'>" + escapeHTML(plan.progress_signals.join("；")) + "</div>");
      }
      return items;
    }

    function buildGeneratedOutputItems(run) {
      const codegen = run.codegen || {};
      const items = [
        "<strong>代码生成状态</strong><div class='muted'>" + escapeHTML(codegen.status || "not-run") + " · " + escapeHTML(codegen.summary || "本次未生成代码") + "</div>",
        "<strong>目标模式</strong><div class='muted'>" + escapeHTML(codegen.target_mode || "analysis-only") + "</div>"
      ];
      if (codegen.output_dir) {
        items.push("<strong>输出目录</strong><div class='muted mono'>" + escapeHTML(codegen.output_dir) + "</div>");
      }
      if ((codegen.run_commands || []).length > 0) {
        items.push("<strong>建议运行命令</strong><div class='muted mono'>" + escapeHTML(codegen.run_commands.join("\n")) + "</div>");
      }
      if ((codegen.notes || []).length > 0) {
        items.push("<strong>备注</strong><div class='muted'>" + escapeHTML(codegen.notes.join("；")) + "</div>");
      }
      return items;
    }

    function buildGeneratedFileItems(run) {
      const codegen = run.codegen || {};
      const files = Array.isArray(codegen.files) ? codegen.files : [];
      if (files.length === 0) {
        return ["<span class='muted'>本次运行没有写出代码文件；如果只是问答或文档模式，这是正常结果。</span>"];
      }
      return files.map(function(file) {
        const label = file.url
          ? "<a href='" + escapeHTML(file.url) + "' target='_blank'>" + escapeHTML(file.path) + "</a>"
          : escapeHTML(file.path);
        const meta = [];
        if (file.language) meta.push(file.language);
        if (file.bytes) meta.push(file.bytes + " bytes");
        if (file.purpose) meta.push(file.purpose);
        return "<strong>" + label + "</strong><div class='muted'>" + escapeHTML(meta.join(" · ")) + "</div>";
      });
    }

    function buildValidationCheckItems(run) {
      const report = run.preflight || {};
      const checks = Array.isArray(report.checks) ? report.checks : [];
      if (checks.length === 0) {
        return ["<span class='muted'>本次没有预检结果。</span>"];
      }
      return checks.map(function(check) {
        let output = "";
        if (check.output) {
          output = "<div class='muted mono' style='margin-top:6px;'>" + escapeHTML(check.output.slice(0, 260)) + "</div>";
        }
        return "<strong>" + escapeHTML(check.name) + " <span class='status " + statusClass(check.status) + "'>" + escapeHTML(check.status) + "</span></strong><div class='muted'>" + escapeHTML(check.summary || "") + "</div>" + output;
      });
    }

    function buildValidationScopeItems(run) {
      const items = [
        "<strong>Workflow Backend</strong><div class='muted'>" + escapeHTML(run.workflow_backend || "builtin") + "</div>"
      ];
      const targets = Array.isArray(run.validation_targets) ? run.validation_targets : [];
      items.push("<strong>验证范围</strong><div class='muted'>" + escapeHTML(targets.length > 0 ? targets.join("、") : "./...") + "</div>");
      if (run.workflow_trace && Array.isArray(run.workflow_trace.nodes) && run.workflow_trace.nodes.length > 0) {
        const trace = run.workflow_trace.nodes.map(function(node) {
          return node.name + ":" + node.status;
        }).join(" → ");
        items.push("<strong>阶段轨迹</strong><div class='muted'>" + escapeHTML(trace) + "</div>");
      }
      return items;
    }

    function buildArtifactItems(run) {
      const artifacts = Array.isArray(run.artifacts) ? run.artifacts : [];
      if (artifacts.length === 0) {
        return ["<span class='muted'>暂无导出文件。</span>"];
      }
      return artifacts.map(function(item) {
        const name = item.url
          ? "<a href='" + escapeHTML(item.url) + "' target='_blank'>" + escapeHTML(item.name) + "</a>"
          : escapeHTML(item.name);
        return "<strong>" + name + "</strong><div class='muted'>" + escapeHTML(item.summary || "") + "</div>";
      });
    }

    function buildStepItems(run) {
      const steps = Array.isArray(run.steps) ? run.steps : [];
      if (steps.length === 0) {
        return ["<span class='muted'>当前没有执行步骤。</span>"];
      }
      return steps.map(function(step) {
        return "<strong>" + escapeHTML(step.name) + " <span class='status " + statusClass(step.status) + "'>" + escapeHTML(step.status) + "</span></strong><div class='muted'>" + escapeHTML(step.summary || "") + "</div>";
      });
    }

    function buildSnapshotItems(snapshot) {
      if (!snapshot || !snapshot.enabled) {
        return ["<span class='muted'>快照未启用。</span>"];
      }
      const items = [
        "<strong>变更文件数</strong><div class='muted'>" + escapeHTML(String(snapshot.changed_files || 0)) + "</div>"
      ];
      if (snapshot.before_path) items.push("<strong>Before</strong><div class='muted mono'>" + escapeHTML(snapshot.before_path) + "</div>");
      if (snapshot.after_path) items.push("<strong>After</strong><div class='muted mono'>" + escapeHTML(snapshot.after_path) + "</div>");
      if (snapshot.rollback_advice) items.push("<strong>回滚建议</strong><div class='muted'>" + escapeHTML(snapshot.rollback_advice) + "</div>");
      return items;
    }

    function buildTroubleshootItems(report) {
      if (!report) {
        return ["<span class='muted'>暂无排障报告。</span>"];
      }
      const items = ["<strong>状态</strong><div class='muted'>" + escapeHTML(report.status || "unknown") + "</div>"];
      (report.issues || []).forEach(function(item) {
        items.push("<strong>问题</strong><div class='muted'>" + escapeHTML(item) + "</div>");
      });
      (report.recommendations || []).forEach(function(item) {
        items.push("<strong>建议</strong><div class='muted'>" + escapeHTML(item) + "</div>");
      });
      return items;
    }

    function renderApprovalActions(run) {
      approvalActionList.innerHTML = "";
      const actions = run.execution_actions || [];
      if (actions.length === 0) {
        manualActionStack.style.display = "none";
        return;
      }
      manualActionStack.style.display = "block";
      actions.forEach(function(action) {
        const li = document.createElement("li");
        li.className = "action-item";
        const commandLine = action.command
          ? "<div class='muted mono' style='margin-top:6px;'>" + escapeHTML(action.command) + "</div>"
          : "";
        const outputBlock = action.output
          ? "<div class='action-output'>" + escapeHTML(action.output) + "</div>"
          : "";
        const executedLine = action.last_executed_at
          ? "<div class='muted' style='margin-top:6px;'>上次执行: " + new Date(action.last_executed_at).toLocaleString() + "</div>"
          : "";
        const disabled = action.status === "running" ? "disabled" : "";
        li.innerHTML =
          "<div class='action-row'>" +
            "<div class='action-copy'>" +
              "<strong>" + escapeHTML(action.title) + "</strong>" +
              "<div class='muted'>" + escapeHTML(action.summary) + "</div>" +
              commandLine +
              executedLine +
            "</div>" +
            "<div class='action-controls'>" +
              "<span class='status " + statusClass(action.status || "pending") + "'>" + escapeHTML(action.status || "pending") + "</span>" +
              "<button class='secondary action-trigger' data-run-id='" + escapeHTML(run.id) + "' data-action-id='" + escapeHTML(action.id) + "' " + disabled + ">执行修复</button>" +
            "</div>" +
          "</div>" +
          outputBlock;
        approvalActionList.appendChild(li);
      });
    }

    function renderResult(run, options) {
      if (!run) return;
      const plan = run.plan || {};
      const mode = plan.mode || "general";
      const overview = plan.overview || "本次运行已完成，但没有返回结构化摘要，请打开 Markdown 或 JSON 查看详情。";
      resultCard.classList.add("visible");
      document.getElementById("overviewText").textContent = overview;
      document.getElementById("resultMeta").textContent =
        "Provider: " + run.used_provider + " | Mode: " + mode + " | Complexity: " + run.complexity.level + " (" + run.complexity.score + ")";
      document.getElementById("resultLinks").innerHTML = buildResultLinks(run);
      inlineResult.classList.add("visible");
      inlineResultText.textContent = overview;
      inlineResultMeta.innerHTML =
        "Provider: " + escapeHTML(run.used_provider) +
        " · Mode: " + escapeHTML(mode) +
        " · Complexity: " + escapeHTML(run.complexity.level) +
        " · <a href='" + escapeHTML(run.markdown_url) + "' target='_blank'>Markdown</a> · <a href='" + escapeHTML(run.json_url) + "' target='_blank'>JSON</a>";
      renderApprovalActions(run);
      renderEntries(document.getElementById("summaryList"), buildSummaryItems(run), "暂无任务摘要");
      renderEntries(document.getElementById("generatedOutputList"), buildGeneratedOutputItems(run), "暂无生成信息");
      renderEntries(document.getElementById("generatedFilesList"), buildGeneratedFileItems(run), "暂无生成文件");
      renderEntries(document.getElementById("validationChecksList"), buildValidationCheckItems(run), "暂无预检结果");
      renderEntries(document.getElementById("validationScopeList"), buildValidationScopeItems(run), "暂无验证范围");
      renderEntries(document.getElementById("artifactList"), buildArtifactItems(run), "暂无导出文件");
      renderEntries(document.getElementById("stepList"), buildStepItems(run), "暂无执行步骤");
      renderEntries(document.getElementById("snapshotList"), buildSnapshotItems(run.snapshot), "暂无快照");
      renderEntries(document.getElementById("troubleshootList"), buildTroubleshootItems(run.troubleshoot), "暂无排障信息");
      if (options && options.focus) {
        window.setTimeout(function() {
          resultCard.scrollIntoView({ behavior: "smooth", block: "start" });
        }, 60);
      }
    }

    async function loadState() {
      const res = await fetch("/api/state");
      const state = await res.json();
      renderBoundary(state);
      document.getElementById("providerCount").textContent = state.providers.length;
      document.getElementById("symbolCount").textContent = state.symbols.function_count + state.symbols.struct_count;
      document.getElementById("knowledgeCount").textContent = state.knowledge.chunk_count;
      document.getElementById("dockerStatus").textContent = state.docker_status;
      document.getElementById("redisStatus").textContent = state.redis_status;
      document.getElementById("visionStatus").textContent = state.vision_status;
      workflowBackendValue.textContent = state.workflow_backend || "builtin";

      providerList.innerHTML = "";
      state.providers.forEach(function(provider) {
        const li = document.createElement("li");
        li.className = "provider-item";
        const statusText = provider.ready ? "ready" : "down";
        li.innerHTML =
          "<div>" +
            "<strong>" + escapeHTML(provider.name) + "</strong>" +
            "<div class='muted'>" + escapeHTML(provider.default_model || "n/a") + "</div>" +
            "<div class='muted'>" + escapeHTML(provider.reason || "") + "</div>" +
          "</div>" +
          "<span class='status " + statusClass(statusText) + "'>" + statusText + "</span>";
        providerList.appendChild(li);
      });

      runList.innerHTML = "";
      if (!state.recent_runs || state.recent_runs.length === 0) {
        runList.innerHTML = "<li class='run-item'><span class='muted'>还没有运行记录。提交一次任务后，这里会显示最近的报告和生成结果。</span></li>";
      } else {
        state.recent_runs.forEach(function(run) {
          const li = document.createElement("li");
          li.className = "run-item";
          li.innerHTML =
            "<div>" +
              "<strong>" + escapeHTML(run.goal) + "</strong>" +
              "<div class='muted'>" + new Date(run.created_at).toLocaleString() + " · " + escapeHTML(run.used_provider) + " · " + escapeHTML(run.complexity.level) + "</div>" +
            "</div>" +
            "<a href='" + escapeHTML(run.markdown_url) + "' target='_blank'>Report</a>";
          runList.appendChild(li);
        });
      }

      renderWorkflow(state.workflow);
      renderResult(state.latest_run, { focus: false });
    }

    runButton.addEventListener("click", async function() {
      const goal = document.getElementById("goalInput").value.trim();
      if (!goal) {
        alert("请输入任务目标");
        return;
      }
      runButton.disabled = true;
      runButton.textContent = "执行中...";
      try {
        const res = await fetch("/api/run", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ goal: goal })
        });
        if (!res.ok) {
          throw new Error(await res.text());
        }
        const run = await res.json();
        renderResult(run, { focus: true });
        await loadState();
      } catch (err) {
        alert("执行失败: " + err.message);
      } finally {
        runButton.disabled = false;
        runButton.textContent = "提交任务";
      }
    });

    approvalActionList.addEventListener("click", async function(event) {
      const button = event.target.closest(".action-trigger");
      if (!button) return;
      const runId = button.getAttribute("data-run-id");
      const actionId = button.getAttribute("data-action-id");
      button.disabled = true;
      button.textContent = "执行中...";
      try {
        const res = await fetch("/api/action/execute", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ run_id: runId, action_id: actionId })
        });
        if (!res.ok) {
          throw new Error(await res.text());
        }
        const run = await res.json();
        renderResult(run, { focus: true });
        await loadState();
      } catch (err) {
        alert("执行修复失败: " + err.message);
      } finally {
        button.disabled = false;
        button.textContent = "执行修复";
      }
    });

    loadState().catch(function(err) {
      document.getElementById("projectRootValue").textContent = "state load failed: " + err.message;
    });
    window.setInterval(function() {
      loadState().catch(function() {});
    }, 6000);
  </script>
</body>
</html>`
