package build

import "testing"

func TestPrintInfo(t *testing.T) {
	PrintInfo("1.0.0", "10.08.2026", "abc")
	PrintInfo("", "", "")
}

func TestValue(t *testing.T) {
	if value("") != "N/A" {
		t.Fatal("empty should be N/A")
	}
	if value("x") != "x" {
		t.Fatal("value mismatch")
	}
}
