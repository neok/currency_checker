package validator

import (
	"regexp"
	"testing"
)

func TestValidator_Check(t *testing.T) {
	v := New()
	v.Check(true, "ok", "should not appear")
	v.Check(false, "limit", "must be positive")
	v.Check(false, "limit", "duplicate ignored")
	v.Check(false, "order", "must be asc or desc")

	if v.Valid() {
		t.Fatal("expected invalid")
	}
	if got, want := v.Errors["limit"], "must be positive"; got != want {
		t.Errorf("limit = %q, want %q (first message wins)", got, want)
	}
	if _, ok := v.Errors["ok"]; ok {
		t.Error("passing checks should not add errors")
	}
	if len(v.Errors) != 2 {
		t.Errorf("errors = %d, want 2", len(v.Errors))
	}
}

func TestValidator_Valid(t *testing.T) {
	v := New()
	if !v.Valid() {
		t.Error("new validator should be valid")
	}
	v.Check(false, "x", "boom")
	if v.Valid() {
		t.Error("validator with error should be invalid")
	}
}

func TestIn(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		allowed []string
		want    bool
	}{
		{"present", "asc", []string{"asc", "desc"}, true},
		{"missing", "sideways", []string{"asc", "desc"}, false},
		{"empty allowed", "asc", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := In(tt.value, tt.allowed...); got != tt.want {
				t.Errorf("In(%q, %v) = %v, want %v", tt.value, tt.allowed, got, tt.want)
			}
		})
	}
}

func TestMatches(t *testing.T) {
	re := regexp.MustCompile(`^[A-Z]{3}$`)
	if !Matches("USD", re) {
		t.Error("USD should match")
	}
	if Matches("usd", re) {
		t.Error("lowercase should not match")
	}
}
