package companion

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"multi_agent_cooperation/vision"
)

func (s *Service) initEnhancements(ctx context.Context) {
	s.langGraph = newLangGraphBridge(
		s.config.Workflow.LangGraphEndpoint,
		time.Duration(s.config.Workflow.LangGraphTimeoutSec)*time.Second,
	)
	if s.langGraph != nil {
		s.config.Workflow.Backend = "langgraph_http"
	}

	s.store = newStateStore(ctx, s.config.Redis)
	s.redisStatus = s.store.Status()

	s.visionStatus = "vision disabled"
	if !s.config.Vision.Enabled {
		return
	}

	providerName := strings.TrimSpace(s.config.Vision.Provider)
	if providerName == "" {
		providerName = "groq"
	}

	providerConfig, ok := s.config.ProviderConfig(providerName)
	if !ok || strings.TrimSpace(providerConfig.APIKey) == "" {
		s.visionStatus = "vision unavailable: provider not configured"
		return
	}

	analyzer, err := vision.NewAnalyzer(providerConfig.APIKey, providerConfig.BaseURL, s.config.Vision.Model)
	if err != nil {
		s.visionStatus = "vision unavailable: " + err.Error()
		return
	}

	s.vision = analyzer
	s.visionStatus = "vision ready: " + providerName + " / " + s.config.Vision.Model
}

func (s *Service) persistRuntimeState() {
	if s.store == nil {
		return
	}

	s.mu.RLock()
	workflow := s.workflow
	screen := s.screen
	var latest *RunReport
	if len(s.runs) > 0 {
		latest = s.runs[0]
	}
	s.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	_ = s.store.SaveJSON(ctx, "workflow", workflow)
	_ = s.store.SaveJSON(ctx, "screen", screen)
	if latest != nil {
		_ = s.store.SaveJSON(ctx, "latest_run", latest)
	}
}

func (s *Service) restoreRuntimeState(ctx context.Context) {
	if s.store == nil {
		return
	}

	var workflow WorkflowState
	if err := s.store.LoadJSON(ctx, "workflow", &workflow); err == nil && !workflow.UpdatedAt.IsZero() {
		s.mu.Lock()
		s.workflow = workflow
		s.mu.Unlock()
	}

	var screen ScreenContext
	if err := s.store.LoadJSON(ctx, "screen", &screen); err == nil && (screen.Available || screen.ImagePath != "" || screen.CaptureCount > 0) {
		s.mu.Lock()
		s.screen = screen
		s.mu.Unlock()
	}

	var latest RunReport
	if err := s.store.LoadJSON(ctx, "latest_run", &latest); err == nil && latest.ID != "" {
		s.mu.Lock()
		exists := false
		for _, run := range s.runs {
			if run.ID == latest.ID {
				exists = true
				break
			}
		}
		if !exists {
			s.runs = append([]*RunReport{&latest}, s.runs...)
		}
		s.mu.Unlock()
	}
}

func (s *Service) ensureScreenAnalysis(ctx context.Context, screen ScreenContext) ScreenContext {
	if !screen.Available || s.vision == nil || strings.TrimSpace(screen.ImagePath) == "" {
		return screen
	}
	if screen.AnalysisStatus == "completed" && screen.AnalysisTarget == screen.ImagePath && strings.TrimSpace(screen.VisionSummary) != "" {
		return screen
	}

	timeout := time.Duration(s.config.Vision.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	analysisCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	insight, err := s.vision.AnalyzeImage(analysisCtx, screen.ImagePath)
	s.mu.Lock()

	if s.screen.ImagePath != screen.ImagePath {
		current := s.screen
		s.mu.Unlock()
		return current
	}

	s.screen.AnalysisTarget = screen.ImagePath
	s.screen.AnalysisUpdatedAt = time.Now()
	if err != nil {
		s.screen.AnalysisStatus = "failed"
		s.screen.AnalysisError = err.Error()
		current := s.screen
		s.mu.Unlock()
		s.persistRuntimeState()
		return current
	}

	s.screen.AnalysisStatus = "completed"
	s.screen.AnalysisError = ""
	s.screen.OCRText = strings.TrimSpace(insight.OCRText)
	s.screen.VisionSummary = strings.TrimSpace(insight.Summary)
	s.screen.AppHint = strings.TrimSpace(insight.AppHint)
	s.screen.NextActions = cleanItems(insight.NextActions)
	current := s.screen
	s.mu.Unlock()
	s.persistRuntimeState()
	return current
}

func (s *Service) analyzeScreenAsync(imagePath string) {
	if s.vision == nil || strings.TrimSpace(imagePath) == "" {
		return
	}

	go func(target string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Vision.TimeoutSec)*time.Second)
		if s.config.Vision.TimeoutSec <= 0 {
			cancel()
			ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
		}
		defer cancel()
		_ = s.ensureScreenAnalysis(ctx, ScreenContext{Available: true, ImagePath: target})
	}(imagePath)
}

func cleanItems(items []string) []string {
	var cleaned []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		cleaned = append(cleaned, item)
	}
	return cleaned
}

func isGitRepository(workdir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = workdir
	output, err := cmd.CombinedOutput()
	return err == nil && strings.TrimSpace(string(output)) == "true"
}

func marshalPretty(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}
