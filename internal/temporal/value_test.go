package temporal

import (
	"bytes"
	"testing"
	"time"
)

func Test_NoValues(t *testing.T) {
	store := NewTimeValueStore()

	if store.QueryValue(time.Now()) != nil {
		t.Fatalf("Expected nil value")
	}

}

func Test_OneValue(t *testing.T) {
	store := NewTimeValueStore()

	t1 := time.Now()
	store.AddValue(t1, []byte{1, 2, 3})

	if store.QueryValue(t1) != nil {
		t.Fatalf("Expected nil value")
	}
	if bytes.Compare(store.QueryValue(time.Now()), []byte{1, 2, 3}) != 0 {
		t.Fatalf("Expected a value")
	}

}
