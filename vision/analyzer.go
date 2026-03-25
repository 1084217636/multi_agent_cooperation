package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Insight 是一次屏幕 OCR / 多模态分析结果。
type Insight struct {
	OCRText       string   `json:"ocr_text"`
	Summary       string   `json:"summary"`
	AppHint       string   `json:"app_hint"`
	NextActions   []string `json:"next_actions"`
	RawOutputText string   `json:"raw_output_text,omitempty"`
}

// Analyzer 使用 Groq Responses API 对屏幕截图做多模态分析。
type Analyzer struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewAnalyzer 创建新的多模态分析器。
func NewAnalyzer(apiKey, baseURL, model string) (*Analyzer, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("vision api key is empty")
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}
	if strings.TrimSpace(model) == "" {
		model = "meta-llama/llama-4-scout-17b-16e-instruct"
	}

	return &Analyzer{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: 35 * time.Second},
	}, nil
}

// AnalyzeImage 对截图进行 OCR / UI 理解。
func (a *Analyzer) AnalyzeImage(ctx context.Context, imagePath string) (Insight, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return Insight{}, err
	}

	dataURL := "data:" + mimeTypeForPath(imagePath) + ";base64," + base64.StdEncoding.EncodeToString(data)
	requestBody := map[string]any{
		"model": a.model,
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": "你是桌面 AI 开发伴侣的视觉分析器。请读取截图中的主要可见文字，并判断当前界面大概是什么应用或任务场景。只返回 JSON，对象字段必须包含：ocr_text, summary, app_hint, next_actions。",
					},
					{
						"type":      "input_image",
						"detail":    "auto",
						"image_url": dataURL,
					},
				},
			},
		},
		"text": map[string]any{
			"format": map[string]any{
				"type": "json_schema",
				"name": "screen_insight",
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"ocr_text": map[string]any{
							"type": "string",
						},
						"summary": map[string]any{
							"type": "string",
						},
						"app_hint": map[string]any{
							"type": "string",
						},
						"next_actions": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "string",
							},
						},
					},
					"required": []string{"ocr_text", "summary", "app_hint", "next_actions"},
				},
			},
		},
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return Insight{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/responses", bytes.NewReader(payload))
	if err != nil {
		return Insight{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return Insight{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return a.fallbackOCR(ctx, imagePath, err)
	}
	if resp.StatusCode != http.StatusOK {
		return a.fallbackOCR(ctx, imagePath, fmt.Errorf("vision api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body))))
	}

	var parsed struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return a.fallbackOCR(ctx, imagePath, err)
	}

	rawText := strings.TrimSpace(parsed.OutputText)
	if rawText == "" {
		for _, item := range parsed.Output {
			for _, content := range item.Content {
				if strings.HasSuffix(content.Type, "text") && strings.TrimSpace(content.Text) != "" {
					rawText = strings.TrimSpace(content.Text)
					break
				}
			}
			if rawText != "" {
				break
			}
		}
	}
	if rawText == "" {
		return a.fallbackOCR(ctx, imagePath, fmt.Errorf("vision output is empty"))
	}

	var insight Insight
	if err := json.Unmarshal([]byte(rawText), &insight); err != nil {
		insight = Insight{
			OCRText:       "",
			Summary:       rawText,
			AppHint:       "unknown",
			NextActions:   nil,
			RawOutputText: rawText,
		}
		return insight, nil
	}
	insight.RawOutputText = rawText
	return insight, nil
}

func (a *Analyzer) fallbackOCR(ctx context.Context, imagePath string, cause error) (Insight, error) {
	ocrText, err := runOCR(ctx, imagePath)
	if err != nil {
		return Insight{}, cause
	}

	ocrText = strings.TrimSpace(ocrText)
	if ocrText == "" {
		return Insight{
			OCRText:     "",
			Summary:     "本地 OCR 已执行，但没有识别到稳定文本。",
			AppHint:     "unknown",
			NextActions: []string{"切换到文字更清晰的界面后重新捕捉屏幕"},
		}, nil
	}

	return Insight{
		OCRText:     ocrText,
		Summary:     "本地 OCR 回退成功，已提取到可见文字。",
		AppHint:     inferAppHint(ocrText),
		NextActions: inferNextActions(ocrText),
	}, nil
}

func mimeTypeForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

func runOCR(ctx context.Context, imagePath string) (string, error) {
	if tesseractPath, err := exec.LookPath("tesseract"); err == nil {
		cmd := exec.CommandContext(ctx, tesseractPath, imagePath, "stdout")
		output, runErr := cmd.CombinedOutput()
		return string(output), runErr
	}

	if _, err := exec.LookPath("docker"); err == nil {
		dir := filepath.Dir(imagePath)
		name := filepath.Base(imagePath)
		cmd := exec.CommandContext(
			ctx,
			"docker", "run", "--rm",
			"-v", dir+":/work",
			"debian:bookworm-slim",
			"sh", "-lc",
			"apt-get update >/dev/null 2>/dev/null && apt-get install -y --no-install-recommends tesseract-ocr >/dev/null 2>/dev/null && tesseract /work/"+name+" stdout 2>/dev/null",
		)
		output, runErr := cmd.CombinedOutput()
		return string(output), runErr
	}

	return "", fmt.Errorf("no local tesseract or docker fallback available")
}

func inferAppHint(ocrText string) string {
	lower := strings.ToLower(ocrText)
	switch {
	case strings.Contains(lower, "readme") || strings.Contains(lower, "go.mod") || strings.Contains(lower, "package "):
		return "code-editor"
	case strings.Contains(lower, "github") || strings.Contains(lower, "http://") || strings.Contains(lower, "https://"):
		return "browser"
	case strings.Contains(lower, "terminal") || strings.Contains(lower, "bash") || strings.Contains(lower, "go test"):
		return "terminal"
	default:
		return "unknown"
	}
}

func inferNextActions(ocrText string) []string {
	lower := strings.ToLower(ocrText)
	switch {
	case strings.Contains(lower, "error") || strings.Contains(lower, "failed"):
		return []string{"优先解释当前报错原因", "给出下一步修复动作"}
	case strings.Contains(lower, "readme") || strings.Contains(lower, "docs"):
		return []string{"总结当前文档内容", "提出补全文档建议"}
	default:
		return []string{"总结当前屏幕内容", "结合项目上下文给出下一步建议"}
	}
}
