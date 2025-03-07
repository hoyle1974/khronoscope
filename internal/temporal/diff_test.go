package temporal

import (
	"bytes"
	"testing"
)

func TestApplyDiff(t *testing.T) {
	a := []byte("hello world")
	b := []byte("hello go")

	diff, err := generateDiff(a, b)
	if err != nil {
		t.Fatalf("generateDiff failed: %v", err)
	}

	patched, err := applyDiff(a, diff)
	if err != nil {
		t.Fatalf("applyDiff failed: %v", err)
	}

	if !bytes.Equal(patched, b) {
		t.Fatalf("applyDiff did not produce expected result. Expected: %s, Got: %s", b, patched)
	}

	//Test with empty diff
	emptyDiff := Diff{}
	patched, err = applyDiff(a, emptyDiff)
	if err != nil {
		t.Fatalf("applyDiff failed with empty diff: %v", err)
	}
	if !bytes.Equal(patched, a) {
		t.Fatalf("applyDiff with empty diff did not return original data. Expected: %s, Got: %s", a, patched)
	}

	//Test with different data types
	a = []byte{1, 2, 3, 4, 5}
	b = []byte{1, 2, 6, 7, 8}
	diff, err = generateDiff(a, b)
	if err != nil {
		t.Fatalf("generateDiff failed: %v", err)
	}
	patched, err = applyDiff(a, diff)
	if err != nil {
		t.Fatalf("applyDiff failed: %v", err)
	}
	if !bytes.Equal(patched, b) {
		t.Fatalf("applyDiff did not produce expected result. Expected: %v, Got: %v", b, patched)
	}
}

func TestApplyDiffError(t *testing.T) {
	a := []byte("hello world")
	diff := Diff([]byte("invalid diff data"))

	_, err := applyDiff(a, diff)
	if err == nil {
		t.Fatalf("The code did not error as expected")
	}
}
