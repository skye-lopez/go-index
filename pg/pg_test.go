package pg

import (
	"testing"
)

func TestConnect(t *testing.T) {
	_, err := NewPg()
	if err != nil {
		t.Fatalf("Err creating db instance:\n%e", err)
	}
}
