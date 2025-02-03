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
	store.AddValue(t1.Add(time.Second), []byte{1, 2, 3})

	if store.QueryValue(t1) != nil {
		t.Fatalf("Expected nil value")
	}
	if bytes.Compare(store.QueryValue(t1.Add(time.Second*2)), []byte{1, 2, 3}) != 0 {
		t.Fatalf("Expected a value")
	}
}

func Test_TwoValue(t *testing.T) {
	store := NewTimeValueStore()

	t1 := time.Now()
	store.AddValue(t1.Add(time.Second), []byte{1, 2, 3})
	store.AddValue(t1.Add(time.Second*5), []byte{2, 3, 4})

	if store.QueryValue(t1) != nil {
		t.Fatalf("Expected nil value")
	}
	if bytes.Compare(store.QueryValue(t1.Add(time.Second*2)), []byte{1, 2, 3}) != 0 {
		t.Fatalf("Expected a value")
	}
	if bytes.Compare(store.QueryValue(t1.Add(time.Second*6)), []byte{2, 3, 4}) != 0 {
		t.Fatalf("Expected a value")
	}
}

func Test_ManyValues(t *testing.T) {
	store := NewTimeValueStore()

	t1 := time.Now()
	for i := 0; i < 100; i++ {
		store.AddValue(t1.Add(time.Second+(time.Duration(i)*time.Second)), []byte{byte(i)})
	}

	if store.QueryValue(t1) != nil {
		t.Fatalf("Expected nil value")
	}

	for i := 0; i < 100; i++ {
		offset := time.Second + (time.Duration(i) * time.Second) + time.Millisecond
		if bytes.Compare(store.QueryValue(t1.Add(offset)), []byte{byte(i)}) != 0 {
			t.Fatalf("Expected a value")
		}
	}
}
