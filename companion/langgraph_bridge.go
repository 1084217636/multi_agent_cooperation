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
	Goal       string           `json:"goal"`
	Complexity ComplexityReport `json:"complexity"`
	Route      RouteDecision    `json:"route"`
	Symbols    SymbolSnapshot   `json:"symbols"`
	Screen     ScreenContext    `json:"screen"`
	Knowledge  []KnowledgeMatch `json:"knowledge"`
}

type langGraphResponse struct {
	Plan         Plan   `json:"plan"`
	UsedProvider string `json:"used_provider"`
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
	payload, err := json.Marshal(req)
	if err != nil {
		return Plan{}, "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint, bytes.NewReader(payload))
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
