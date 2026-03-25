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
      --bg: #f3ebdd;
      --ink: #22313f;
      --muted: #62707d;
      --line: rgba(34,49,63,0.12);
      --card: rgba(255,255,255,0.82);
      --accent: #0c6c67;
      --accent-2: #d1633d;
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
        radial-gradient(circle at 14% 12%, rgba(12,108,103,0.18), transparent 24%),
        radial-gradient(circle at 88% 2%, rgba(209,99,61,0.17), transparent 20%),
        linear-gradient(180deg, #fff9f2 0%, #efe4d4 100%);
    }
    .shell {
      width: min(1240px, calc(100% - 28px));
      margin: 20px auto 44px;
      animation: rise .45s ease both;
    }
    .hero, .panel, .result-card {
      background: var(--card);
      backdrop-filter: blur(12px);
      border: 1px solid rgba(255,255,255,0.7);
      border-radius: 28px;
      box-shadow: var(--shadow);
    }
    .hero {
      padding: 30px;
      display: grid;
      grid-template-columns: 1.2fr .8fr;
      gap: 18px;
      align-items: start;
    }
    .hero h1 {
      margin: 0 0 10px;
      font-size: clamp(34px, 6vw, 60px);
      line-height: .92;
      letter-spacing: -0.05em;
    }
    .hero p {
      margin: 0;
      color: var(--muted);
      line-height: 1.8;
      max-width: 720px;
    }
    .meta-stack {
      display: grid;
      gap: 12px;
    }
    .meta-card {
      padding: 14px 16px;
      border-radius: 18px;
      border: 1px solid var(--line);
      background: rgba(255,255,255,0.68);
    }
    .grid {
      margin-top: 18px;
      display: grid;
      grid-template-columns: repeat(12, minmax(0, 1fr));
      gap: 18px;
    }
    .panel {
      padding: 22px;
    }
    .panel h2 {
      margin: 0 0 14px;
      font-size: 14px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: .14em;
    }
    .span-7 { grid-column: span 7; }
    .span-5 { grid-column: span 5; }
    .span-4 { grid-column: span 4; }
    .span-8 { grid-column: span 8; }
    .span-12 { grid-column: span 12; }
    .cards {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 12px;
    }
    .metric {
      padding: 16px;
      border-radius: 18px;
      border: 1px solid var(--line);
      background: rgba(255,255,255,0.72);
    }
    .metric strong {
      display: block;
      font-size: 28px;
      margin-bottom: 4px;
    }
    textarea {
      width: 100%;
      min-height: 196px;
      resize: vertical;
      border: 1px solid var(--line);
      background: rgba(255,255,255,0.92);
      border-radius: 18px;
      padding: 16px;
      font: inherit;
      color: var(--ink);
      outline: none;
    }
    textarea:focus {
      border-color: rgba(12,108,103,0.45);
      box-shadow: 0 0 0 4px rgba(12,108,103,0.12);
    }
    .toolbar, .capture-actions {
      display: flex;
      gap: 12px;
      align-items: center;
      justify-content: space-between;
      flex-wrap: wrap;
      margin-top: 12px;
    }
    .hint, .muted {
      color: var(--muted);
      font-size: 13px;
    }
    .inline-result {
      display: none;
      margin-top: 14px;
      padding: 16px 18px;
      border-radius: 18px;
      border: 1px solid var(--line);
      background: rgba(12,108,103,0.07);
    }
    .inline-result.visible {
      display: block;
    }
    .inline-result strong {
      display: block;
      margin-bottom: 6px;
      font-size: 13px;
      letter-spacing: .08em;
      text-transform: uppercase;
      color: var(--accent);
    }
    .inline-result p {
      margin: 0;
      line-height: 1.8;
    }
    button {
      border: 0;
      border-radius: 999px;
      padding: 13px 22px;
      background: linear-gradient(135deg, var(--accent), #0b5a55);
      color: #fff;
      font: inherit;
      cursor: pointer;
      box-shadow: 0 10px 28px rgba(12,108,103,0.24);
    }
    button.secondary {
      background: linear-gradient(135deg, #d7e8e7, #c6dedc);
      color: var(--ink);
      box-shadow: none;
    }
    button:disabled {
      opacity: .65;
      cursor: wait;
    }
    .provider-list, .run-list, .bullet-list {
      list-style: none;
      margin: 0;
      padding: 0;
      display: grid;
      gap: 10px;
    }
    .provider-item, .run-item, .step-item {
      display: flex;
      justify-content: space-between;
      gap: 14px;
      padding: 14px 16px;
      border-radius: 16px;
      border: 1px solid var(--line);
      background: rgba(255,255,255,0.74);
    }
    .status {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-width: 72px;
      padding: 4px 10px;
      border-radius: 999px;
      font-size: 12px;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: .06em;
      white-space: nowrap;
    }
    .status-failed {
      background: rgba(197,48,48,0.12);
      color: var(--danger);
    }
    .ready, .status-idle, .status-completed { background: rgba(47,133,90,0.12); color: var(--success); }
    .down, .status-error { background: rgba(197,48,48,0.12); color: var(--danger); }
    .status-running, .status-capturing { background: rgba(12,108,103,0.12); color: var(--accent); }
    .status-waiting { background: rgba(183,121,31,0.12); color: var(--warning); }
    .capture-card {
      overflow: hidden;
    }
    .capture-preview {
      width: 100%;
      aspect-ratio: 16 / 10;
      object-fit: cover;
      display: block;
      border-radius: 20px;
      border: 1px solid var(--line);
      background:
        linear-gradient(135deg, rgba(12,108,103,0.08), rgba(209,99,61,0.08)),
        #f4f6f8;
    }
    .capture-meta {
      margin-top: 10px;
      line-height: 1.7;
      white-space: pre-line;
    }
    .result-card {
      margin-top: 18px;
      padding: 24px;
      display: none;
    }
    .result-card.visible {
      display: block;
      animation: rise .35s ease both;
    }
    .result-head {
      display: flex;
      justify-content: space-between;
      gap: 16px;
      align-items: flex-start;
      margin-bottom: 16px;
    }
    .result-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 18px;
    }
    .action-stack {
      margin: 18px 0 0;
      padding: 18px;
      border-radius: 18px;
      background: rgba(255,255,255,0.78);
      border: 1px solid var(--line);
    }
    .action-stack h3 {
      margin: 0 0 12px;
      font-size: 14px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: .1em;
    }
    .action-list {
      list-style: none;
      margin: 0;
      padding: 0;
      display: grid;
      gap: 12px;
    }
    .action-item {
      padding: 14px 16px;
      border-radius: 18px;
      border: 1px solid var(--line);
      background: rgba(255,255,255,0.86);
    }
    .action-row {
      display: flex;
      justify-content: space-between;
      gap: 14px;
      align-items: flex-start;
      flex-wrap: wrap;
    }
    .action-copy {
      flex: 1 1 360px;
    }
    .action-copy strong {
      display: block;
      margin-bottom: 4px;
    }
    .action-controls {
      display: flex;
      gap: 10px;
      align-items: center;
      flex-wrap: wrap;
    }
    .action-output {
      margin-top: 10px;
      padding: 12px;
      border-radius: 14px;
      background: rgba(12,108,103,0.06);
      font-size: 12px;
      line-height: 1.6;
      white-space: pre-wrap;
      word-break: break-word;
    }
    .result-box {
      padding: 18px;
      border-radius: 18px;
      background: rgba(255,255,255,0.78);
      border: 1px solid var(--line);
    }
    .result-box h3 {
      margin: 0 0 10px;
      font-size: 14px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: .1em;
    }
    .result-box ul {
      margin: 0;
      padding-left: 18px;
      line-height: 1.7;
    }
    .progress-meter {
      margin-top: 12px;
      width: 100%;
      height: 10px;
      border-radius: 999px;
      background: rgba(34,49,63,0.08);
      overflow: hidden;
    }
    .progress-fill {
      width: 0%;
      height: 100%;
      border-radius: 999px;
      background: linear-gradient(135deg, var(--accent), #0b5a55);
      transition: width .25s ease;
    }
    a {
      color: var(--accent-2);
      text-decoration: none;
    }
    code {
      padding: 2px 8px;
      border-radius: 999px;
      background: rgba(12,108,103,0.08);
      color: var(--accent);
    }
    @keyframes rise {
      from { opacity: 0; transform: translateY(8px); }
      to { opacity: 1; transform: translateY(0); }
    }
    @media (max-width: 980px) {
      .hero { grid-template-columns: 1fr; }
      .span-7, .span-5, .span-4, .span-8, .span-12 { grid-column: span 12; }
      .cards, .result-grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div>
        <h1>Go R&amp;D Agent<br>Workbench</h1>
        <p>这不是普通聊天页，而是面向 Go 工程的研发智能体工作台。它把上下文采集、RAG、符号快照、模型路由、工程预检和运行报告串成一条更稳定的本地闭环。</p>
      </div>
      <div class="meta-stack">
        <div class="meta-card"><strong>Workspace</strong><div class="muted" id="workspaceValue">loading...</div></div>
        <div class="meta-card"><strong>Loop Engine</strong><div class="muted" id="workflowMeta">准备中...</div></div>
        <div class="meta-card"><strong>Screen Context</strong><div class="muted" id="screenMetaInline">等待屏幕授权</div></div>
        <div class="meta-card"><strong>Workflow Backend</strong><div class="muted" id="workflowBackendValue">builtin</div></div>
      </div>
    </section>

    <section class="grid">
      <section class="panel span-7">
        <h2>Mission Input</h2>
        <textarea id="goalInput">围绕当前 Go 项目生成技术分析、交付文档或闭环开发方案，要求结合屏幕上下文、RAG、AST 符号快照和工程预检。</textarea>
        <div class="toolbar">
          <div class="hint">如果当前任务依赖浏览器页面或外部界面信息，先截图一次再发起任务；否则可以直接运行。</div>
          <button id="runButton">启动闭环任务</button>
        </div>
        <div class="inline-result" id="inlineResult">
          <strong>Latest Output</strong>
          <p id="inlineResultText">任务完成后，最新摘要会先显示在这里。</p>
          <div class="muted" id="inlineResultMeta" style="margin-top:8px;"></div>
        </div>
      </section>

      <section class="panel span-5">
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

      <section class="panel span-4 capture-card">
        <h2>Screen Context</h2>
        <img id="capturePreview" class="capture-preview" alt="screen preview">
        <div class="capture-meta muted" id="captureMeta">尚未捕捉屏幕。点击下方按钮时会请求浏览器共享权限，并只抓取当前这一帧，不会自动轮询。</div>
        <div class="capture-actions">
          <button id="captureButton">截图一次</button>
          <button id="clearCaptureButton" class="secondary" disabled>清空截图</button>
        </div>
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
        <ul class="bullet-list" id="stepList" style="margin-top:10px;"></ul>
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
          <h2 style="margin:0;font-size:28px;">Latest Companion Run</h2>
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
        <div class="result-box"><h3>Task Mode</h3><ul id="modeList"></ul></div>
        <div class="result-box"><h3>Actions</h3><ul id="actionsList"></ul></div>
        <div class="result-box"><h3>Deliverables</h3><ul id="deliverablesList"></ul></div>
        <div class="result-box"><h3>Progress Signals</h3><ul id="progressSignalsList"></ul></div>
        <div class="result-box"><h3>Exported Files</h3><ul id="artifactList"></ul></div>
        <div class="result-box"><h3>Innovations</h3><ul id="innovationsList"></ul></div>
        <div class="result-box"><h3>RAG Use Cases</h3><ul id="ragList"></ul></div>
        <div class="result-box"><h3>Current Gaps</h3><ul id="gapsList"></ul></div>
        <div class="result-box"><h3>Risks</h3><ul id="risksList"></ul></div>
        <div class="result-box"><h3>Next Steps</h3><ul id="nextStepsList"></ul></div>
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
    const captureButton = document.getElementById("captureButton");
    const clearCaptureButton = document.getElementById("clearCaptureButton");
    const capturePreview = document.getElementById("capturePreview");
    const captureMeta = document.getElementById("captureMeta");
    const workflowMeta = document.getElementById("workflowMeta");
    const screenMetaInline = document.getElementById("screenMetaInline");
    const workflowBackendValue = document.getElementById("workflowBackendValue");
    const workflowBadge = document.getElementById("workflowBadge");
    const workflowStatusText = document.getElementById("workflowStatusText");
    const workflowPhaseText = document.getElementById("workflowPhaseText");
    const workflowMessageText = document.getElementById("workflowMessageText");
    const workflowProgressFill = document.getElementById("workflowProgressFill");
    const workflowProgressText = document.getElementById("workflowProgressText");
    const stepList = document.getElementById("stepList");
    const manualActionStack = document.getElementById("manualActionStack");
    const inlineResult = document.getElementById("inlineResult");
    const inlineResultText = document.getElementById("inlineResultText");
    const inlineResultMeta = document.getElementById("inlineResultMeta");
    const approvalActionList = document.getElementById("approvalActionList");

    let captureStream = null;
    let captureBusy = false;
    const captureVideo = document.createElement("video");
    captureVideo.muted = true;
    captureVideo.playsInline = true;

    function renderSimpleList(el, items) {
      el.innerHTML = "";
      if (!items || items.length === 0) {
        const li = document.createElement("li");
        li.className = "step-item";
        li.innerHTML = "<span class='muted'>暂无数据</span>";
        el.appendChild(li);
        return;
      }
      items.forEach(function(item) {
        const li = document.createElement("li");
        li.className = "step-item";
        li.textContent = item;
        el.appendChild(li);
      });
    }

    function renderArtifactList(el, artifacts) {
      el.innerHTML = "";
      if (!artifacts || artifacts.length === 0) {
        const li = document.createElement("li");
        li.className = "step-item";
        li.innerHTML = "<span class='muted'>暂无导出文件</span>";
        el.appendChild(li);
        return;
      }
      artifacts.forEach(function(item) {
        const li = document.createElement("li");
        li.className = "step-item";
        const link = item.url
          ? "<a href='" + escapeHTML(item.url) + "' target='_blank'>" + escapeHTML(item.name) + "</a>"
          : escapeHTML(item.name);
        li.innerHTML = "<span>" + link + " · " + escapeHTML(item.summary || "") + "</span>";
        el.appendChild(li);
      });
    }

    function escapeHTML(value) {
      return String(value || "")
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
    }

    function setCaptureBusy(active) {
      captureBusy = active;
      captureButton.disabled = active;
      captureButton.textContent = active ? "截图中..." : "截图一次";
    }

    function setClearCaptureEnabled(enabled) {
      clearCaptureButton.disabled = !enabled;
    }

    function releaseCaptureStream() {
      if (!captureStream) {
        return;
      }
      captureStream.getTracks().forEach(function(track) {
        track.stop();
      });
      captureStream = null;
      captureVideo.pause();
      captureVideo.srcObject = null;
    }

    function renderResult(run, options) {
      if (!run) return;
      const plan = run.plan || {};
      const mode = plan.mode || "general";
      const overview = plan.overview || "本次运行已完成，但没有返回结构化摘要，请打开 Markdown 或 JSON 查看详情。";
      const actions = Array.isArray(plan.actions) ? plan.actions : [];
      const deliverables = Array.isArray(plan.deliverables) ? plan.deliverables : [];
      const progressSignals = Array.isArray(plan.progress_signals) ? plan.progress_signals : [];
      const innovations = Array.isArray(plan.innovations) ? plan.innovations : [];
      const ragUseCases = Array.isArray(plan.rag_use_cases) ? plan.rag_use_cases : [];
      const gaps = Array.isArray(plan.desktop_pet_gaps) ? plan.desktop_pet_gaps : [];
      const risks = Array.isArray(plan.risks) ? plan.risks : [];
      const nextSteps = Array.isArray(plan.next_steps) ? plan.next_steps : [];

      resultCard.classList.add("visible");
      document.getElementById("overviewText").textContent = overview;
      document.getElementById("resultMeta").textContent =
        "Provider: " + run.used_provider + " | Mode: " + mode + " | Complexity: " + run.complexity.level + " (" + run.complexity.score + ")";
      const artifactLinks = Array.isArray(run.artifacts) ? run.artifacts.map(function(item) {
        return "<a href='" + item.url + "' target='_blank'>" + escapeHTML(item.name) + "</a>";
      }) : [];
      document.getElementById("resultLinks").innerHTML =
        "<a href='" + run.markdown_url + "' target='_blank'>Markdown</a> · <a href='" + run.json_url + "' target='_blank'>JSON</a>" +
        (artifactLinks.length > 0 ? " · " + artifactLinks.join(" · ") : "");
      inlineResult.classList.add("visible");
      inlineResultText.textContent = overview;
      inlineResultMeta.innerHTML =
        "Provider: " + escapeHTML(run.used_provider) +
        " · Mode: " + escapeHTML(mode) +
        " · Complexity: " + escapeHTML(run.complexity.level) +
        " · <a href='" + run.markdown_url + "' target='_blank'>Markdown</a> · <a href='" + run.json_url + "' target='_blank'>JSON</a>";
      renderApprovalActions(run);
      renderSimpleList(document.getElementById("modeList"), ["任务模式: " + mode]);
      renderSimpleList(document.getElementById("actionsList"), actions);
      renderSimpleList(document.getElementById("deliverablesList"), deliverables);
      renderSimpleList(document.getElementById("progressSignalsList"), progressSignals);
      renderArtifactList(document.getElementById("artifactList"), run.artifacts);
      renderSimpleList(document.getElementById("innovationsList"), innovations);
      renderSimpleList(document.getElementById("ragList"), ragUseCases);
      renderSimpleList(document.getElementById("gapsList"), gaps);
      renderSimpleList(document.getElementById("risksList"), risks);
      renderSimpleList(document.getElementById("nextStepsList"), nextSteps);
      renderSimpleList(document.getElementById("snapshotList"), buildSnapshotItems(run.snapshot));
      renderSimpleList(document.getElementById("troubleshootList"), buildTroubleshootItems(run.troubleshoot));
      renderSteps(run.steps || []);

      if (options && options.focus) {
        window.setTimeout(function() {
          resultCard.scrollIntoView({ behavior: "smooth", block: "start" });
        }, 60);
      }
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
          ? "<div class='muted' style='margin-top:6px;'><code>" + escapeHTML(action.command) + "</code></div>"
          : "";
        const outputBlock = action.output
          ? "<div class='action-output'>" + escapeHTML(action.output) + "</div>"
          : "";
        const executedLine = action.last_executed_at
          ? "<div class='muted' style='margin-top:6px;'>上次执行: " + new Date(action.last_executed_at).toLocaleString() + "</div>"
          : "";
        const buttonLabel = action.status === "completed"
          ? "再次执行"
          : (action.requires_approval ? "执行修复" : "立即执行");
        const actionHint = action.requires_approval
          ? "<div class='muted' style='margin-top:6px;'>这会修改当前工作区或触发额外执行，因此保留人工触发。</div>"
          : "";
        const disabled = action.status === "running" ? "disabled" : "";

        li.innerHTML =
          "<div class='action-row'>" +
            "<div class='action-copy'>" +
              "<strong>" + escapeHTML(action.title) + "</strong>" +
              "<div class='muted'>" + escapeHTML(action.summary) + "</div>" +
              actionHint +
              commandLine +
              executedLine +
            "</div>" +
            "<div class='action-controls'>" +
              "<span class='status status-" + escapeHTML(action.status || "waiting") + "'>" + escapeHTML(action.status || "pending") + "</span>" +
              "<button class='secondary action-trigger' data-run-id='" + escapeHTML(run.id) + "' data-action-id='" + escapeHTML(action.id) + "' " + disabled + ">" + buttonLabel + "</button>" +
            "</div>" +
          "</div>" +
          outputBlock;

        approvalActionList.appendChild(li);
      });
    }

    function buildSnapshotItems(snapshot) {
      if (!snapshot || !snapshot.enabled) {
        return ["快照未启用"];
      }
      const items = [];
      items.push("变更文件数: " + (snapshot.changed_files || 0));
      if (snapshot.before_path) items.push("Before: " + snapshot.before_path);
      if (snapshot.after_path) items.push("After: " + snapshot.after_path);
      if (snapshot.rollback_advice) items.push(snapshot.rollback_advice);
      return items;
    }

    function buildTroubleshootItems(report) {
      if (!report) {
        return ["暂无排障报告"];
      }
      const items = ["状态: " + (report.status || "unknown")];
      (report.issues || []).forEach(function(item) { items.push("问题: " + item); });
      (report.recommendations || []).forEach(function(item) { items.push("建议: " + item); });
      return items;
    }

    function renderSteps(steps) {
      stepList.innerHTML = "";
      if (!steps || steps.length === 0) {
        stepList.innerHTML = "<li class='step-item'><span class='muted'>当前没有执行步骤。</span></li>";
        return;
      }
      steps.forEach(function(step) {
        const badgeClass = "status status-" + step.status;
        const li = document.createElement("li");
        li.className = "step-item";
        li.innerHTML =
          "<div>" +
            "<strong>" + step.name + "</strong>" +
            "<div class='muted'>" + step.summary + "</div>" +
          "</div>" +
          "<span class='" + badgeClass + "'>" + step.status + "</span>";
        stepList.appendChild(li);
      });
    }

    function renderScreen(screen) {
      if (screen && screen.available && screen.image_url) {
        capturePreview.src = screen.image_url + "?t=" + Date.now();
        let metaText =
          "来源: " + (screen.source_label || "shared-screen") +
          " | 分辨率: " + screen.width + "x" + screen.height +
          " | 已捕捉: " + screen.capture_count + " 帧";
        if (screen.analysis_status) {
          metaText += " | 分析: " + screen.analysis_status;
        }
        if (screen.app_hint) {
          metaText += "\n应用猜测: " + screen.app_hint;
        }
        if (screen.vision_summary) {
          metaText += "\n视觉摘要: " + screen.vision_summary;
        }
        if (screen.ocr_text) {
          metaText += "\nOCR: " + screen.ocr_text.slice(0, 180);
        }
        captureMeta.textContent = metaText;
        screenMetaInline.textContent = "最近屏幕帧: " + (screen.source_label || "shared-screen");
        setClearCaptureEnabled(true);
      } else {
        capturePreview.removeAttribute("src");
        captureMeta.textContent = "尚未捕捉屏幕。点击下方按钮时会请求浏览器共享权限，并只抓取当前这一帧，不会自动轮询。";
        screenMetaInline.textContent = "等待屏幕授权";
        setClearCaptureEnabled(false);
      }
    }

    function renderWorkflow(workflow) {
      if (!workflow) {
        return;
      }
      const status = workflow.status || "idle";
      const progress = inferWorkflowProgress(status, workflow.phase || "ready");
      workflowBadge.className = "status status-" + status;
      workflowBadge.textContent = status;
      workflowStatusText.textContent = status;
      workflowPhaseText.textContent = workflow.phase || "ready";
      workflowMessageText.textContent = workflow.message || "工作台就绪，等待任务";
      workflowMeta.textContent = workflow.phase + " · " + workflow.message;
      workflowProgressFill.style.width = progress + "%";
      workflowProgressText.textContent = progress + "% · " + (workflow.phase || "ready");
    }

    function inferWorkflowProgress(status, phase) {
      if (status === "capturing") return 10;
      if (status === "error") return 100;
      if (status !== "running") return 0;
      switch (phase) {
        case "snapshot":
          return 12;
        case "context":
          return 28;
        case "planning":
          return 56;
        case "sandbox":
          return 70;
        case "preflight":
          return 84;
        case "persisting":
          return 96;
        case "approval-action":
          return 75;
        default:
          return 18;
      }
    }

    async function loadState() {
      const res = await fetch("/api/state");
      const state = await res.json();

      document.getElementById("workspaceValue").textContent = state.workspace;
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
        const statusClass = provider.ready ? "ready" : "down";
        const statusText = provider.ready ? "ready" : "down";
        li.innerHTML =
          "<div>" +
            "<strong>" + provider.name + "</strong>" +
            "<div class='muted'>" + (provider.default_model || "n/a") + "</div>" +
            "<div class='muted'>" + provider.reason + "</div>" +
          "</div>" +
          "<span class='status " + statusClass + "'>" + statusText + "</span>";
        providerList.appendChild(li);
      });

      runList.innerHTML = "";
      if (!state.recent_runs || state.recent_runs.length === 0) {
        runList.innerHTML = "<li class='run-item'><span class='muted'>还没有运行记录。授权屏幕后发起一次任务，这里就会形成完整闭环记录。</span></li>";
      } else {
        state.recent_runs.forEach(function(run) {
          const li = document.createElement("li");
          li.className = "run-item";
          li.innerHTML =
            "<div>" +
              "<strong>" + run.goal + "</strong>" +
              "<div class='muted'>" + new Date(run.created_at).toLocaleString() + " · " + run.used_provider + " · " + run.complexity.level + "</div>" +
            "</div>" +
            "<a href='" + run.markdown_url + "' target='_blank'>Report</a>";
          runList.appendChild(li);
        });
      }

      renderScreen(state.screen);
      renderWorkflow(state.workflow);
      renderResult(state.latest_run, { focus: false });
    }

    async function uploadCaptureFrame() {
      if (!captureStream) {
        return;
      }

      try {
        const track = captureStream.getVideoTracks()[0];
        if (!track) {
          return;
        }

        const nativeWidth = captureVideo.videoWidth || 1280;
        const nativeHeight = captureVideo.videoHeight || 720;
        const maxWidth = 1280;
        const ratio = nativeWidth > maxWidth ? (maxWidth / nativeWidth) : 1;
        const width = Math.max(320, Math.round(nativeWidth * ratio));
        const height = Math.max(180, Math.round(nativeHeight * ratio));
        const canvas = document.createElement("canvas");
        canvas.width = width;
        canvas.height = height;
        const ctx = canvas.getContext("2d");
        ctx.drawImage(captureVideo, 0, 0, width, height);
        const dataURL = canvas.toDataURL("image/jpeg", 0.76);
        capturePreview.src = dataURL;

        const res = await fetch("/api/capture/frame", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            data_url: dataURL,
            width: nativeWidth,
            height: nativeHeight,
            source_label: track.label || "shared-screen"
          })
        });
        if (!res.ok) {
          throw new Error(await res.text());
        }

        const screen = await res.json();
        renderScreen(screen);
      } catch (err) {
        captureMeta.textContent = "屏幕捕捉失败: " + err.message;
      }
    }

    async function startCapture() {
      if (!navigator.mediaDevices || !navigator.mediaDevices.getDisplayMedia) {
        alert("当前浏览器不支持屏幕捕捉 API");
        return;
      }
      if (captureBusy) {
        return;
      }

      setCaptureBusy(true);
      try {
        releaseCaptureStream();
        captureStream = await navigator.mediaDevices.getDisplayMedia({
          video: { frameRate: 1 },
          audio: false
        });
        captureVideo.srcObject = captureStream;
        await captureVideo.play();
        await uploadCaptureFrame();
      } catch (err) {
        alert("开启屏幕捕捉失败: " + err.message);
      } finally {
        releaseCaptureStream();
        setCaptureBusy(false);
      }
    }

    async function clearCapture() {
      clearCaptureButton.disabled = true;
      try {
        const res = await fetch("/api/capture/clear", {
          method: "POST"
        });
        if (!res.ok) {
          throw new Error(await res.text());
        }
        const screen = await res.json();
        renderScreen(screen);
      } catch (err) {
        alert("清空截图失败: " + err.message);
        setClearCaptureEnabled(true);
      }
    }

    runButton.addEventListener("click", async function() {
      runButton.disabled = true;
      runButton.textContent = "闭环执行中...";
      try {
        const res = await fetch("/api/run", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ goal: document.getElementById("goalInput").value.trim() })
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
        runButton.textContent = "启动闭环任务";
      }
    });

    approvalActionList.addEventListener("click", async function(event) {
      const button = event.target.closest(".action-trigger");
      if (!button) {
        return;
      }

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
        alert("审批动作执行失败: " + err.message);
      } finally {
        button.disabled = false;
        button.textContent = "执行修复";
      }
    });

    captureButton.addEventListener("click", startCapture);
    clearCaptureButton.addEventListener("click", clearCapture);

    loadState().catch(function(err) {
      document.getElementById("workspaceValue").textContent = "state load failed: " + err.message;
    });
    window.setInterval(function() {
      loadState().catch(function() {});
    }, 6000);
  </script>
</body>
</html>`
