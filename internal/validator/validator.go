package validator

import (
	"regexp"
	"slices"
)

type Validator struct {
	Errors map[string]string
}

func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

func (v *Validator) Check(ok bool, key, message string) {
	if ok {
		return
	}
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

func Matches(s string, re *regexp.Regexp) bool {
	return re.MatchString(s)
}

func In[T comparable](value T, allowed ...T) bool {
	return slices.Contains(allowed, value)
}
