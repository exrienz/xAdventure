package config

import "testing"

func TestIsValidModelNameAllowsCommonProviderFormats(t *testing.T) {
	valid := []string{
		"Malay",
		"gpt-4.1-mini",
		"openai/gpt-4o-mini",
		"llama3.1:8b",
		"provider/model:tag-1.2",
	}
	for _, name := range valid {
		if !isValidModelName(name) {
			t.Fatalf("expected %q to be valid", name)
		}
	}
}

func TestIsValidModelNameRejectsUnsafeValues(t *testing.T) {
	invalid := []string{"", "bad model", "../secret", "model$", "model?x=1"}
	for _, name := range invalid {
		if isValidModelName(name) {
			t.Fatalf("expected %q to be invalid", name)
		}
	}
}
