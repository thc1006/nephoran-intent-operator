package kpm

import "testing"

func TestSample_Range(t *testing.T) {
s := Sample("utilization", 1.5)
if s.Value < 0 || s.Value > 1 {
t.Fatalf("value out of range: %v", s.Value)
}
}
