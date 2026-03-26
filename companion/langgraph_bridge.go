package companion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type langGraphBridge struct {
	endpoint string
	client   *http.Client
}

type langGraphRequest struct {
	Operation       string           `json:"operation,omitempty"`
	Goal            string           `json:"goal"`
	Complexity      ComplexityReport `json:"complexity"`
	Route           RouteDecision    `json:"route"`
	Symbols         SymbolSnapshot   `json:"symbols"`
	Screen          ScreenContext    `json:"screen"`
	Knowledge       []KnowledgeMatch `json:"knowledge"`
	Plan            Plan             `json:"plan,omitempty"`
	TargetMode      string           `json:"target_mode,omitempty"`
	PatchCandidates []string         `json:"patch_candidates,omitempty"`
}

type langGraphResponse struct {
	Plan         Plan              `json:"plan"`
	CodeBundle   codeBundlePayload `json:"code_bundle"`
	UsedProvider string            `json:"used_provider"`
	Trace        []string          `json:"trace,omitempty"`
}

func newLangGraphBridge(endpoint string, timeout time.Duration) *langGraphBridge {
	if strings.TrimSpace(endpoint) == "" {
		return nil
	}
	if timeout <= 0 {
		timeout = 25 * time.Second
	}
	return &langGraphBridge{
		endpoint: strings.TrimSpace(endpoint),
		client:   &http.Client{Timeout: timeout},
	}
}

func (b *langGraphBridge) GeneratePlan(ctx context.Context, req langGraphRequest) (Plan, string, error) {
	req.Operation = "plan"
	return b.generatePlan(ctx, b.operationURL("plan"), req)
}

func (b *langGraphBridge) GenerateCodeBundle(ctx context.Context, req langGraphRequest) (codeBundlePayload, string, error) {
	req.Operation = "codegen"
	payload, usedProvider, err := b.generateCodeBundle(ctx, b.operationURL("codegen"), req)
	if err == nil {
		return payload, usedProvider, nil
	}
	if b.operationURL("codegen") != b.endpoint {
		return payload, usedProvider, err
	}
	return b.generateCodeBundle(ctx, b.endpoint, req)
}

func (b *langGraphBridge) generatePlan(ctx context.Context, targetURL string, req langGraphRequest) (Plan, string, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return Plan{}, "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return Plan{}, "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return Plan{}, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Plan{}, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return Plan{}, "", fmt.Errorf("langgraph bridge status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed langGraphResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Plan{}, "", err
	}
	if parsed.Plan.Overview == "" {
		return Plan{}, "", fmt.Errorf("langgraph bridge returned empty plan")
	}
	return normalizePlan(parsed.Plan), strings.TrimSpace(parsed.UsedProvider), nil
}

func (b *langGraphBridge) generateCodeBundle(ctx context.Context, targetURL string, req langGraphRequest) (codeBundlePayload, string, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return codeBundlePayload{}, "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return codeBundlePayload{}, "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return codeBundlePayload{}, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return codeBundlePayload{}, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return codeBundlePayload{}, "", fmt.Errorf("langgraph bridge status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed langGraphResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return codeBundlePayload{}, "", err
	}
	normalized, err := normalizeCodeBundlePayload(parsed.CodeBundle)
	if err != nil {
		return codeBundlePayload{}, "", fmt.Errorf("langgraph bridge returned invalid code bundle: %w", err)
	}
	return normalized, strings.TrimSpace(parsed.UsedProvider), nil
}

func (b *langGraphBridge) operationURL(operation string) string {
	if strings.TrimSpace(operation) == "" {
		return b.endpoint
	}
	if strings.Contains(b.endpoint, "{operation}") {
		return strings.ReplaceAll(b.endpoint, "{operation}", operation)
	}
	if operation == "plan" {
		return b.endpoint
	}
	if strings.HasSuffix(b.endpoint, "/plan") {
		return strings.TrimSuffix(b.endpoint, "/plan") + "/" + operation
	}
	if strings.HasSuffix(b.endpoint, "/") {
		return b.endpoint + operation
	}
	return b.endpoint + "/" + operation
}
