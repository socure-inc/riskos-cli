package ui

import (
	"strings"
	"testing"
)

func TestResolveOutput_ForcesJSONWhenForceJSONTrue(t *testing.T) {
	orig := ForceJSON
	ForceJSON = true
	defer func() { ForceJSON = orig }()

	out, err := ResolveOutput("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "json" {
		t.Errorf("expected json, got %s", out)
	}
}

func TestResolveOutput_ReturnsExplicitFlagValue(t *testing.T) {
	orig := ForceJSON
	ForceJSON = false
	defer func() { ForceJSON = orig }()

	out, err := ResolveOutput("json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "json" {
		t.Errorf("expected json, got %s", out)
	}

	out, err = ResolveOutput("plain")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "plain" {
		t.Errorf("expected plain, got %s", out)
	}
}

func TestResolveOutput_RejectsInvalidValue(t *testing.T) {
	_, err := ResolveOutput("potato")
	if err == nil {
		t.Error("expected error for invalid output format")
	}
}

func TestSuggestTypo_ExactMatch(t *testing.T) {
	choices := []string{"Valid_Consumer_Accept", "Watchlist_Hit_Reject"}
	got := SuggestTypo("Valid_Consumer_Accept", choices)
	if got != "Valid_Consumer_Accept" {
		t.Errorf("expected exact match, got %s", got)
	}
}

func TestSuggestTypo_CloseMatch(t *testing.T) {
	choices := []string{"Valid_Consumer_Accept", "Watchlist_Hit_Reject"}
	// One character off
	got := SuggestTypo("Valid_Consumer_Accep", choices)
	if got != "Valid_Consumer_Accept" {
		t.Errorf("expected Valid_Consumer_Accept, got %s", got)
	}
}

func TestSuggestTypo_EmptyWhenTooFar(t *testing.T) {
	choices := []string{"Valid_Consumer_Accept"}
	got := SuggestTypo("completely_different_thing_xyz", choices)
	if got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestDecisionBadge_Accept(t *testing.T) {
	badge := DecisionBadge("ACCEPT")
	if !strings.Contains(badge, "ACCEPT") {
		t.Errorf("badge should contain ACCEPT, got %s", badge)
	}
}

func TestDecisionBadge_Reject(t *testing.T) {
	badge := DecisionBadge("REJECT")
	if !strings.Contains(badge, "REJECT") {
		t.Errorf("badge should contain REJECT, got %s", badge)
	}
}

func TestDecisionBadge_Review(t *testing.T) {
	badge := DecisionBadge("REVIEW")
	if !strings.Contains(badge, "REVIEW") {
		t.Errorf("badge should contain REVIEW, got %s", badge)
	}
}
