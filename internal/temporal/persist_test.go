package temporal

import (
	"testing"
	"time"

	"github.com/hoyle1974/khronoscope/internal/resources"
	"github.com/hoyle1974/khronoscope/internal/serializable"
)

// Helper function to create a sample Resource
func createResource(uid, name string) resources.Resource {
	return resources.Resource{
		Uid:       uid,
		Timestamp: serializable.NewTime(time.Now()),
		Kind:      "Pod",
		Namespace: "default",
		Name:      name,
		RawJSON:   map[string]any{"key": "value"},
		Extra:     map[string]any{"key": "value"},
	}
}

func TestNewCompressedChunk(t *testing.T) {
	r1 := createResource("r1", "resource1")
	r2 := createResource("r2", "resource2")
	r3 := createResource("r3", "resource3")

	f1 := frame{Timestamp: time.Date(2023, 10, 27, 10, 30, 0, 0, time.UTC), Resources: map[string]resources.Resource{"r1": r1}}
	f2 := frame{Timestamp: time.Date(2023, 10, 27, 10, 30, 1, 0, time.UTC), Resources: map[string]resources.Resource{"r1": r1, "r2": r2}}
	f3 := frame{Timestamp: time.Date(2023, 10, 27, 10, 30, 2, 0, time.UTC), Resources: map[string]resources.Resource{"r1": r1, "r2": r2, "r3": r3}}
	frames := []frame{f1, f2, f3}

	c := newCompressedChunk(frames)

	if !c.KeyFrame.Timestamp.Equal(f1.Timestamp) {
		t.Errorf("Expected KeyFrame timestamp to be %v, but got %v", f1.Timestamp, c.KeyFrame.Timestamp)
	}

	if len(c.Diffs) != 2 {
		t.Errorf("Expected 2 diffs, but got %d", len(c.Diffs))
	}

	if !c.Range.Min.Time.Equal(f1.Timestamp) || !c.Range.Max.Time.Equal(f1.Timestamp) {
		t.Errorf("Range incorrect, expected %v got %v, %v", f1.Timestamp, c.Range.Min.Time, c.Range.Max.Time)
	}

	s1 := c.GetStateAtTime(f1.Timestamp)
	if len(s1) != 1 {
		t.Errorf("Expected 1 state, but got %d", len(s1))
	}

	s2 := c.GetStateAtTime(f2.Timestamp)
	if len(s2) != 2 {
		t.Errorf("Expected 2 state, but got %d", len(s2))
	}

	s3 := c.GetStateAtTime(f3.Timestamp)
	if len(s3) != 3 {
		t.Errorf("Expected 3 state, but got %d", len(s3))
	}

}
