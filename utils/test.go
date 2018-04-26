package utils

import "testing"

// CheckNum make int assertion
func CheckNum(num, expected int, t *testing.T) {
	if num != expected {
		t.Errorf("got %d, expected %d", num, expected)
	}
}

// ChechStr make string assertion
func CheckStr(str, expected string, t *testing.T) {
	if str != expected {
		t.Errorf("got %v, expected %v", str, expected)
	}
}
