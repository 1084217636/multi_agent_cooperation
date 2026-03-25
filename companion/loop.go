package companion

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ScreenContext 保存最近一次桌面屏幕捕捉信息。
type ScreenContext struct {
	Available         bool      `json:"available"`
	Mode              string    `json:"mode"`
	SourceLabel       string    `json:"source_label"`
	Width             int       `json:"width"`
	Height            int       `json:"height"`
	CapturedAt        time.Time `json:"captured_at"`
	ImagePath         string    `json:"image_path"`
	ImageURL          string    `json:"image_url"`
	CaptureCount      int       `json:"capture_count"`
	OCRText           string    `json:"ocr_text,omitempty"`
	VisionSummary     string    `json:"vision_summary,omitempty"`
	AppHint           string    `json:"app_hint,omitempty"`
	NextActions       []string  `json:"next_actions,omitempty"`
	AnalysisStatus    string    `json:"analysis_status,omitempty"`
	AnalysisError     string    `json:"analysis_error,omitempty"`
	AnalysisTarget    string    `json:"analysis_target,omitempty"`
	AnalysisUpdatedAt time.Time `json:"analysis_updated_at,omitempty"`
	LastError         string    `json:"last_error,omitempty"`
}

// WorkflowState 描述当前闭环执行状态。
type WorkflowState struct {
	Status     string    `json:"status"`
	Phase      string    `json:"phase"`
	Message    string    `json:"message"`
	ActiveGoal string    `json:"active_goal,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ExecutionStep 描述一次运行中的关键阶段。
type ExecutionStep struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Summary     string    `json:"summary"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// ScreenCaptureInput 是浏览器上传的屏幕帧。
type ScreenCaptureInput struct {
	DataURL     string `json:"data_url"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	SourceLabel string `json:"source_label"`
}

// SaveScreenCapture 保存一帧屏幕截图并更新上下文。
func (s *Service) SaveScreenCapture(input ScreenCaptureInput) (*ScreenContext, error) {
	if strings.TrimSpace(input.DataURL) == "" {
		return nil, fmt.Errorf("capture data cannot be empty")
	}

	extension, rawData, err := decodeDataURL(input.DataURL)
	if err != nil {
		s.setScreenError(err.Error())
		return nil, err
	}

	captureDir := filepath.Join(s.config.App.DataDir, "captures")
	if err := os.MkdirAll(captureDir, 0o755); err != nil {
		return nil, err
	}

	filename := time.Now().Format("20060102150405.000") + extension
	fullPath := filepath.Join(captureDir, filename)
	if err := os.WriteFile(fullPath, rawData, 0o644); err != nil {
		return nil, err
	}
	_ = cleanupOldCaptures(captureDir, 24)

	s.mu.Lock()
	s.screen.Available = true
	s.screen.Mode = "browser-screen-capture"
	s.screen.SourceLabel = strings.TrimSpace(input.SourceLabel)
	s.screen.Width = input.Width
	s.screen.Height = input.Height
	s.screen.CapturedAt = time.Now()
	s.screen.ImagePath = fullPath
	s.screen.ImageURL = "/captures/" + filename
	s.screen.CaptureCount++
	s.screen.OCRText = ""
	s.screen.VisionSummary = ""
	s.screen.AppHint = ""
	s.screen.NextActions = nil
	s.screen.AnalysisStatus = "pending"
	s.screen.AnalysisError = ""
	s.screen.AnalysisTarget = fullPath
	s.screen.AnalysisUpdatedAt = time.Time{}
	s.screen.LastError = ""
	currentWorkflowStatus := s.workflow.Status
	snapshot := s.screen
	s.mu.Unlock()

	s.persistRuntimeState()
	if s.config.Vision.Enabled && s.config.Vision.AnalyzeOnCapture {
		s.analyzeScreenAsync(fullPath)
	}

	if currentWorkflowStatus != "running" {
		s.updateWorkflow("capturing", "desktop-sense", "已捕捉最新屏幕数据", "")
	}
	return &snapshot, nil
}

// ClearScreenCapture 清空当前屏幕截图上下文，避免旧截图继续参与后续任务。
func (s *Service) ClearScreenCapture() *ScreenContext {
	s.mu.Lock()
	currentWorkflowStatus := s.workflow.Status
	s.screen = ScreenContext{}
	snapshot := s.screen
	s.mu.Unlock()

	s.persistRuntimeState()
	if currentWorkflowStatus != "running" {
		s.updateWorkflow("idle", "ready", "屏幕截图已清空，等待任务", "")
	}
	return &snapshot
}

func (s *Service) screenSnapshot() ScreenContext {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.screen
}

func (s *Service) workflowSnapshot() WorkflowState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workflow
}

func (s *Service) updateWorkflow(status, phase, message, goal string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workflow.Status = status
	s.workflow.Phase = phase
	s.workflow.Message = message
	if goal != "" {
		s.workflow.ActiveGoal = goal
	}
	s.workflow.UpdatedAt = time.Now()
	go s.persistRuntimeState()
}

func (s *Service) setScreenError(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.screen.LastError = message
}

func decodeDataURL(dataURL string) (string, []byte, error) {
	header, encoded, ok := strings.Cut(dataURL, ",")
	if !ok {
		return "", nil, fmt.Errorf("invalid data url")
	}

	extension := ".bin"
	switch {
	case strings.Contains(header, "image/png"):
		extension = ".png"
	case strings.Contains(header, "image/jpeg"):
		extension = ".jpg"
	case strings.Contains(header, "image/webp"):
		extension = ".webp"
	}

	rawData, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, fmt.Errorf("invalid base64 payload: %w", err)
	}

	return extension, rawData, nil
}

func cleanupOldCaptures(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	type item struct {
		path string
		time time.Time
	}
	var captures []item
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		captures = append(captures, item{
			path: filepath.Join(dir, entry.Name()),
			time: info.ModTime(),
		})
	}

	sort.Slice(captures, func(i, j int) bool {
		return captures[i].time.After(captures[j].time)
	})

	for idx := keep; idx < len(captures); idx++ {
		_ = os.Remove(captures[idx].path)
	}
	return nil
}
