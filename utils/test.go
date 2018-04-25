package utils

import "testing"

func CheckNum(num, expected int, t *testing.T) {
	if num != expected {
		t.Errorf("got %d, expected %d", num, expected)
	}
}

func CheckStr(str, expected string, t *testing.T) {
	if str != expected {
		t.Errorf("got %v, expected %v", str, expected)
	}
}

