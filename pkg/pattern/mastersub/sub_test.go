package mastersub

import (
	"testing"
)

func TestNewSubReactor(t *testing.T) {
	t.Run("nil reactor", func(t *testing.T) {
		sub := NewSubReactor(nil)
		if sub == nil {
			t.Fatal("Expected non-nil sub reactor")
		}
	})
}