package companion

import "testing"

func TestLangGraphOperationURL(t *testing.T) {
	bridge := newLangGraphBridge("http://127.0.0.1:8000/plan", 0)
	if bridge == nil {
		t.Fatal("expected bridge")
	}

	if got := bridge.operationURL("plan"); got != "http://127.0.0.1:8000/plan" {
		t.Fatalf("unexpected plan url: %s", got)
	}
	if got := bridge.operationURL("codegen"); got != "http://127.0.0.1:8000/codegen" {
		t.Fatalf("unexpected codegen url: %s", got)
	}
}
