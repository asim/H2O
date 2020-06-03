package controlplane

import (
	"fmt"
	"testing"
)

func TestHashSampleOnEmptyString(t *testing.T) {
	for i := 0; i < 1000; i++ {
		if !hashSample("", 0.5) {
			t.Fatal("HashSample on empty string should always default to true")
		}
	}

	for i := 0; i < 1000; i++ {
		if hashSample("foobar", 0.0) {
			t.Fatal("HashSample on string with 0 proportion should always be false")
		}
	}

	for i := 0; i < 1000; i++ {
		if !hashSample("foobar", 1.0) {
			t.Fatal("HashSample on string with 1.0 proportion should always be true")
		}
	}

	for i := 0; i < 1000; i++ {
		if hashSample("foobar", 0.5) {
			t.Fatal("HashSample on foobar with proportion 0.5 should be consistently false")
		}
	}

	ct := 0
	for i := 0; i < 1000; i++ {
		if hashSample(fmt.Sprintf("foobarbaz%v", i), 0.5) {
			ct++
		}
	}
	if ct != 480 {
		t.Fatalf("HashSample on known set with proportion 0.5 should be exactly 480 true results out of 1000, got %v", ct)
	}
}

func TestSpecificity(t *testing.T) {
	routePath := &Rule{
		Match: &Match{
			Path: "/foo/bar",
		},
		Action: ActionThrottle,
	}
	if routePath.Specificity() != 10 {
		t.Errorf("Path should equate to specificity of 10, got %v", routePath.Specificity())
	}

	routeSource := &Rule{
		Match: &Match{
			Source: "foobar",
		},
		Action: ActionThrottle,
	}
	if routeSource.Specificity() != 5 {
		t.Errorf("Source should equate to specificity of 5, got %v", routeSource.Specificity())
	}

	routeRegArea := &Rule{
		Match: &Match{
			Hob: "LON",
		},
		Action: ActionThrottle,
	}
	if routeRegArea.Specificity() != 5 {
		t.Errorf("RegArea should equate to specificity of 5, got %v", routeRegArea.Specificity())
	}

	routePathRegArea := &Rule{
		Match: &Match{
			Path: "/foo/bar",
			Hob:  "LON",
		},
		Action: ActionThrottle,
	}
	if routePathRegArea.Specificity() != 15 {
		t.Errorf("RegArea should equate to specificity of 15, got %v", routePathRegArea.Specificity())
	}
}

func TestSourceMatch(t *testing.T) {
	driverSource := &Rule{
		Match:  &Match{Source: "driver", Proportion: 1.0},
		Action: ActionThrottle,
	}
	customerSource := &Rule{
		Match:  &Match{Source: "customer", Proportion: 1.0},
		Action: ActionThrottle,
	}

	ext := &testExtractor{
		hob:    "LON",
		source: "driver",
		path:   "/v1/foo/bar",
	}

	if !driverSource.Matches(ext) {
		t.Error("Expecting driver source match")
	}
	if customerSource.Matches(ext) {
		t.Error("Not expecting customer source match")
	}
}

func TestHobMatch(t *testing.T) {
	lonRule := &Rule{
		Match:  &Match{Hob: "LON", Proportion: 1.0},
		Action: ActionThrottle,
	}
	dubRule := &Rule{
		Match:  &Match{Hob: "DUB", Proportion: 1.0},
		Action: ActionThrottle,
	}
	lonDubFooBarRule := &Rule{
		Match:  &Match{Hob: "DUB,FOO,LON,BAR", Proportion: 1.0},
		Action: ActionThrottle,
	}
	ext := &testExtractor{
		hob:    "LON",
		source: "customer",
		path:   "/v1/foo/bar",
	}

	if !lonRule.Matches(ext) {
		t.Error("Expecting LON match")
	}
	if dubRule.Matches(ext) {
		t.Error("Not expecting DUB match")
	}
	if !lonDubFooBarRule.Matches(ext) {
		t.Error("Expecting LON CSV match")
	}
}

// Test that nil pointers in a Rule don't cause panics in the methods.
// See: https://github.com/hailo-platform/H2O/api-proxy/pull/32
func TestRegression_RuleNilPointers(t *testing.T) {
	ext := &testExtractor{
		hob:    "LON",
		source: "customer",
		path:   "/v1/foo/bar",
	}

	var rule *Rule
	rule.Specificity()
	rule.Matches(ext)

	rule = &Rule{}
	rule.Specificity()
	rule.Matches(ext)
}
