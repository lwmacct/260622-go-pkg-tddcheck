package tddcheck

import "testing"

func TestAnalysisPassedRejectsLoadErrors(t *testing.T) {
	analysis := Analysis{LoadErrors: []LoadError{{PackagePath: "example.com/app/internal/service", Message: "undefined: missing"}}}
	if analysis.Passed() {
		t.Fatal("expected package load errors to fail analysis")
	}
	if text := analysis.Text(); text == "" || !contains(text, "undefined: missing") {
		t.Fatalf("expected load error in analysis text, got %q", text)
	}
}

func contains(value string, needle string) bool {
	for index := 0; index+len(needle) <= len(value); index++ {
		if value[index:index+len(needle)] == needle {
			return true
		}
	}
	return false
}
