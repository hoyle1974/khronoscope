package temporal

import (
	"sort"
	"time"

	"github.com/hoyle1974/khronoscope/internal/serializable"
)

func NewTimeValueStore() *TimeValueStore {
	store := &TimeValueStore{
		Keyframes: []keyFrame{},
	}
	return store
}

// TimeValueStore holds the values and diffs with associated timestamps
type TimeValueStore struct {
	Keyframes []keyFrame
}

// keyFrame represents a snapshot of a value at a specific timestamp
// It also contains up a set number of DiffFrames that hold the diffs of the data
type keyFrame struct {
	Timestamp  serializable.Time
	Value      []byte
	DiffFrames []diffFrame
}

// This keyFrame is expected to hold this value.  Query it and it's diffs to find the
// one that was valid for when this timestamp is
func (frame *keyFrame) queryValue(timestamp time.Time) []byte {
	index := sort.Search(len(frame.DiffFrames), func(j int) bool {
		return frame.DiffFrames[j].Timestamp.Time.After(timestamp)
	})

	// Handle the case where the timestamp is before all diffs
	if index == 0 && len(frame.DiffFrames) > 0 && timestamp.Before(frame.DiffFrames[0].Timestamp.Time) {
		return frame.Value
	}

	cur := frame.Value
	for i := 0; i < index; i++ {
		cur, _ = applyDiff(cur, frame.DiffFrames[i].Diff)
	}
	return cur
}

func (frame *keyFrame) addDiffFrame(timestamp time.Time, value []byte) bool {
	if len(frame.DiffFrames) == 10 {
		if len(frame.DiffFrames) > 0 && frame.DiffFrames[len(frame.DiffFrames)-1].Timestamp.Time.Before(timestamp) {
			return false // We have enough frames, and adding at the end
		}
	}

	curr := frame.Value
	original := make([][]byte, len(frame.DiffFrames))
	for idx, diffFrame := range frame.DiffFrames {
		actual, err := applyDiff(curr, diffFrame.Diff)
		if err != nil {
			return false // Stop if any diff application fails
		}
		original[idx] = actual
		curr = actual
	}

	index := sort.Search(len(frame.DiffFrames), func(j int) bool {
		return frame.DiffFrames[j].Timestamp.Time.After(timestamp)
	})

	// Handle the case where DiffFrames is empty
	if len(frame.DiffFrames) == 0 {
		diff, err := generateDiff(frame.Value, value) // Use frame.Value for the first diff
		if err != nil {
			return false
		}
		frame.DiffFrames = append(frame.DiffFrames, diffFrame{Timestamp: serializable.Time{Time: timestamp}, Diff: diff})
		return true
	}

	diff, err := generateDiff(original[max(0, index-1)], value)
	if err != nil {
		return false
	}
	newDiffFrame := diffFrame{Timestamp: serializable.Time{Time: timestamp}, Diff: diff}

	if len(frame.DiffFrames) == 10 {
		if index < len(frame.DiffFrames) {
			frame.DiffFrames = append(frame.DiffFrames[:index], append([]diffFrame{newDiffFrame}, frame.DiffFrames[index+1:]...)...)
		} else { // if it is full and we are inserting at the end.
			frame.DiffFrames[index-1] = newDiffFrame
		}
	} else {
		frame.DiffFrames = append(frame.DiffFrames[:index], append([]diffFrame{newDiffFrame}, frame.DiffFrames[index:]...)...)
	}

	// Regenerate the rest of the diffs
	for i := index + 1; i < len(frame.DiffFrames); i++ {
		diff, err := generateDiff(original[i-1], value)
		if err != nil {
			panic(err) // Consider a less drastic error handling here
		}
		frame.DiffFrames[i].Diff = diff
	}

	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type diffFrame struct {
	Timestamp serializable.Time
	Diff      Diff
}

// Add a value that is valid @timestamp and after
func (store *TimeValueStore) AddValue(timestamp time.Time, value []byte) {

	// Find the insertion point to maintain chronological order.
	index := sort.Search(len(store.Keyframes), func(j int) bool {
		return store.Keyframes[j].Timestamp.Time.After(timestamp)
	})

	if len(store.Keyframes) == 0 {
		store.Keyframes = append(store.Keyframes, keyFrame{
			Timestamp: serializable.Time{Time: timestamp},
			Value:     value,
		})
		return
	}

	// index is where we would put this keyframe.  Backup one keyframe and see if we just want to append diff
	if index > 0 && store.Keyframes[index-1].addDiffFrame(timestamp, value) { // Corrected: Check index > 0
		return
	}

	// Insert the new value at the correct position.
	store.Keyframes = append(store.Keyframes[:index], append([]keyFrame{{Timestamp: serializable.Time{Time: timestamp}, Value: value}}, store.Keyframes[index:]...)...)

}

// Return the most recent value that was set on or before timestamp.
func (store *TimeValueStore) QueryValue(timestamp time.Time) []byte {
	if len(store.Keyframes) == 0 { // No data
		return nil
	}

	index := sort.Search(len(store.Keyframes), func(j int) bool {
		return store.Keyframes[j].Timestamp.Time.After(timestamp)
	})

	if index == 0 { // Timestamp is before the first keyframe
		return nil
	} else if index == len(store.Keyframes) { // Timestamp is after the last keyframe
		return store.Keyframes[index-1].queryValue(timestamp) // Use the *last* keyframe
	} else { // Timestamp is within the keyframes
		return store.Keyframes[index-1].queryValue(timestamp) // Use the keyframe *before*
	}
}
