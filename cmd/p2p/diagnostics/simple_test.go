package diagnostics

import "testing"

func TestSimple(t *testing.T) {
if 1+1 != 2 {
t.Errorf("1+1 should equal 2")
}
}
