package memory

import (
	"os"
	"testing"
)

func TestAddDecision(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	decision, err := store.AddDecision(AddDecisionRequest{
		Title:        "Use PostgreSQL for persistence",
		Context:      "We need a reliable database",
		Decision:     "We will use PostgreSQL",
		Consequences: "Need to manage migrations. Good performance expected.",
		Alternatives: []string{"MySQL", "SQLite"},
	})
	if err != nil {
		t.Fatalf("AddDecision failed: %v", err)
	}

	if decision.ID == "" {
		t.Error("Decision ID is empty")
	}
	if decision.Status != DecisionStatusProposed {
		t.Errorf("Status = %q, want %q", decision.Status, DecisionStatusProposed)
	}
	if len(decision.Alternatives) != 2 {
		t.Errorf("Alternatives count = %d, want 2", len(decision.Alternatives))
	}
}

func TestAddDecisionWithAcceptedStatus(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	decision, err := store.AddDecision(AddDecisionRequest{
		Title:    "Accepted Decision",
		Context:  "Context",
		Decision: "Decision",
		Status:   string(DecisionStatusAccepted),
	})
	if err != nil {
		t.Fatal(err)
	}

	if decision.Status != DecisionStatusAccepted {
		t.Errorf("Status = %q, want %q", decision.Status, DecisionStatusAccepted)
	}
}

func TestAddDecisionValidation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	// Missing title
	_, err = store.AddDecision(AddDecisionRequest{
		Context:  "Context",
		Decision: "Decision",
	})
	if err == nil {
		t.Error("Expected error for missing title")
	}

	// Missing context
	_, err = store.AddDecision(AddDecisionRequest{
		Title:    "Title",
		Decision: "Decision",
	})
	if err == nil {
		t.Error("Expected error for missing context")
	}

	// Missing decision
	_, err = store.AddDecision(AddDecisionRequest{
		Title:   "Title",
		Context: "Context",
	})
	if err == nil {
		t.Error("Expected error for missing decision")
	}
}

func TestGetDecision(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	decision, err := store.AddDecision(AddDecisionRequest{
		Title:    "Test Decision",
		Context:  "Test context",
		Decision: "Test decision text",
	})
	if err != nil {
		t.Fatal(err)
	}

	retrieved, err := store.GetDecision(decision.ID)
	if err != nil {
		t.Fatalf("GetDecision failed: %v", err)
	}

	if retrieved.ID != decision.ID {
		t.Errorf("ID = %q, want %q", retrieved.ID, decision.ID)
	}

	// Non-existent
	_, err = store.GetDecision("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent decision")
	}
}

func TestAcceptDecision(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	decision, err := store.AddDecision(AddDecisionRequest{
		Title:    "Test Decision",
		Context:  "Test context",
		Decision: "Test decision",
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := store.AcceptDecision(decision.ID)
	if err != nil {
		t.Fatalf("AcceptDecision failed: %v", err)
	}

	if updated.Status != DecisionStatusAccepted {
		t.Errorf("Status = %q, want %q", updated.Status, DecisionStatusAccepted)
	}
}

func TestDeprecateDecision(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	decision, err := store.AddDecision(AddDecisionRequest{
		Title:    "Test Decision",
		Context:  "Test context",
		Decision: "Test decision",
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := store.DeprecateDecision(decision.ID)
	if err != nil {
		t.Fatalf("DeprecateDecision failed: %v", err)
	}

	if updated.Status != DecisionStatusDeprecated {
		t.Errorf("Status = %q, want %q", updated.Status, DecisionStatusDeprecated)
	}
}

func TestListDecisions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	_, _ = store.AddDecision(AddDecisionRequest{
		Title:    "Decision 1",
		Context:  "Context 1",
		Decision: "Decision 1",
	})

	_, _ = store.AddDecision(AddDecisionRequest{
		Title:    "Decision 2",
		Context:  "Context 2",
		Decision: "Decision 2",
	})

	// List all
	decisions := store.ListDecisions(ListDecisionsRequest{})
	if len(decisions) != 2 {
		t.Errorf("ListDecisions returned %d, want 2", len(decisions))
	}

	// Filter by status
	decisions = store.ListDecisions(ListDecisionsRequest{
		Status: []string{string(DecisionStatusProposed)},
	})
	if len(decisions) != 2 {
		t.Errorf("ListDecisions with status returned %d, want 2", len(decisions))
	}

	// Search
	decisions = store.ListDecisions(ListDecisionsRequest{Search: "Decision 1"})
	if len(decisions) != 1 {
		t.Errorf("ListDecisions with search returned %d, want 1", len(decisions))
	}
}

func TestDeleteDecision(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	decision, err := store.AddDecision(AddDecisionRequest{
		Title:    "To Delete",
		Context:  "Context",
		Decision: "Decision",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteDecision(decision.ID); err != nil {
		t.Fatalf("DeleteDecision failed: %v", err)
	}

	_, err = store.GetDecision(decision.ID)
	if err == nil {
		t.Error("Decision should have been deleted")
	}
}

func TestExportDecisionAsMarkdown(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	decision, err := store.AddDecision(AddDecisionRequest{
		Title:        "Use REST API",
		Context:      "We need an API design",
		Decision:     "Use REST with JSON",
		Consequences: "Standard approach, easy to understand",
		Alternatives: []string{"GraphQL", "gRPC"},
	})
	if err != nil {
		t.Fatal(err)
	}

	md, err := store.ExportDecisionAsMarkdown(decision.ID)
	if err != nil {
		t.Fatalf("ExportDecisionAsMarkdown failed: %v", err)
	}

	// Check that markdown contains key elements
	if !contains(md, "# Use REST API") {
		t.Error("Markdown missing title")
	}
	if !contains(md, "## Context") {
		t.Error("Markdown missing context section")
	}
	if !contains(md, "## Decision") {
		t.Error("Markdown missing decision section")
	}
	if !contains(md, "GraphQL") {
		t.Error("Markdown missing alternatives")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
