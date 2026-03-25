package companion

import "testing"

func TestAssessComplexityHigh(t *testing.T) {
	report := AssessComplexity("请把这个项目重构成面向 Go 工程的研发智能体，加入 RAG、MCP、Docker、LangGraph 和多智能体自动执行链路。")
	if report.Level != ComplexityHigh {
		t.Fatalf("expected high complexity, got %s (%d)", report.Level, report.Score)
	}
	if len(report.Reasons) < 4 {
		t.Fatalf("expected multiple reasons, got %v", report.Reasons)
	}
}

func TestDecideRouteFallsBackToMock(t *testing.T) {
	decision := DecideRoute("离线演示任务", ComplexityReport{Level: ComplexityMedium}, map[string]ProviderStatus{
		"mock":   {Name: "mock", Ready: true},
		"groq":   {Name: "groq", Ready: false},
		"ollama": {Name: "ollama", Ready: false},
	})

	if decision.Primary != "mock" {
		t.Fatalf("expected mock as primary, got %s", decision.Primary)
	}
	if len(decision.Attempts) != 1 || decision.Attempts[0] != "mock" {
		t.Fatalf("unexpected attempts: %+v", decision.Attempts)
	}
}

func TestDecideRoutePrefersOnlineProviderForLowComplexity(t *testing.T) {
	decision := DecideRoute("帮我基于当前页面做个简单项目分析并导出文档", ComplexityReport{Level: ComplexityLow}, map[string]ProviderStatus{
		"mock": {Name: "mock", Ready: true},
		"groq": {Name: "groq", Ready: true},
	})

	if decision.Primary != "groq" {
		t.Fatalf("expected groq as primary, got %s", decision.Primary)
	}
	if len(decision.Attempts) == 0 || decision.Attempts[0] != "groq" {
		t.Fatalf("unexpected attempts: %+v", decision.Attempts)
	}
}
