package temporal

import (
	"testing"
	"time"
)

func createTestMap() (start, end, t1, t2, t3 time.Time, m Map) {
	m = New()

	start = time.Now().Add(-time.Second)
	t1 = time.Now()
	m.Add(t1, "key1", "key1value1")

	t2 = time.Now()
	m.Add(t2, "key1", "key1value2")
	m.Add(t2, "key2", "key2value1")

	t3 = time.Now()
	m.Add(t3, "key1", "key1value3")

	end = time.Now().Add(time.Second)

	return start, end, t1, t2, t3, m
}

func validateMap(t *testing.T, start, end, t1, t2, t3 time.Time, m Map) {

	min, max := m.GetTimeRange()

	if !min.Equal(t1) {
		t.Fatalf("Expected min time to be %v, not %v", t1, min)
	}
	if !max.Equal(t3) {
		t.Fatalf("Expected min time to be %v, not %v", t3, max)
	}

	if a := m.GetStateAtTime(start); len(a) != 0 {
		t.Fatalf("Expected empty map")
	}
	if a := m.GetStateAtTime(end); len(a) != 2 {
		t.Fatalf("Expected empty map")
	}

	if a := m.GetStateAtTime(t1); a["key1"] != "key1value1" {
		t.Fatalf("Unexpected value for key1 at t1")
	}

	if a := m.GetStateAtTime(t2); a["key1"] != "key1value2" || a["key2"] != "key2value1" {
		t.Fatalf("Unexpected value for key1 at t2 or key2 at t2")
	}

	if a := m.GetStateAtTime(t3); a["key1"] != "key1value3" || a["key2"] != "key2value1" {
		t.Fatalf("Unexpected value for key1 at t3 or key2 at t3")
	}

	if a := m.GetStateAtTime(end); a["key1"] != "key1value3" || a["key2"] != "key2value1" {
		t.Fatalf("Unexpected value for key1 at end or key2 at end")
	}
}

func TestBasicSerialization(t *testing.T) {
	start, end, t1, t2, t3, m := createTestMap()
	validateMap(t, start, end, t1, t2, t3, m)

	m2 := FromBytes(m.ToBytes())

	validateMap(t, start, end, t1, t2, t3, m2)

}

func TestBasicMap(t *testing.T) {
	start, end, t1, t2, t3, m := createTestMap()
	validateMap(t, start, end, t1, t2, t3, m)
}
