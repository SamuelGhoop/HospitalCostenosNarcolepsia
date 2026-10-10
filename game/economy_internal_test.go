package game

import (
	"testing"
	"time"
)

// §5.8: $450 hasta 15 s; después −$15 por cada segundo completo; mínimo $100.
func TestReviewFee_DependsOnTheWait(t *testing.T) {
	cases := []struct {
		waited time.Duration
		want   int
	}{
		{10 * time.Second, 450},
		{15 * time.Second, 450},
		{15900 * time.Millisecond, 450}, // todavía no se completa el segundo 16
		{16 * time.Second, 435},
		{25 * time.Second, 300},
		{38 * time.Second, 105},
		{39 * time.Second, 100},
		{60 * time.Second, 100},
	}
	for _, c := range cases {
		if got := reviewFee(c.waited); got != c.want {
			t.Errorf("reviewFee(%v) = $%d; se esperaban $%d", c.waited, got, c.want)
		}
	}
}
