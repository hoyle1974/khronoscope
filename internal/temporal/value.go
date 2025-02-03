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
	cur := frame.Value
	for i := 0; i < index; i++ {
		cur, _ = applyDiff(cur, frame.DiffFrames[i].Diff)
	}
	return cur
}

// Add a Diff frame to the array of DiffFrames attached to this keyframe.  Return false if not possible.
// This may need to recalculate diffs if a value is set inside this
func (frame *keyFrame) addDiffFrame(timestamp time.Time, value []byte) bool {
	if len(frame.DiffFrames) > 10 {
		return false // We have enough frames
	}
	curr := frame.Value
	original := make([][]byte, len(frame.DiffFrames))
	for idx, diffFrame := range frame.DiffFrames {
		if actual, err := applyDiff(curr, diffFrame.Diff); err != nil {
			original[idx] = actual
			curr = actual
		} else {
			return false
		}
	}

	index := sort.Search(len(frame.DiffFrames), func(j int) bool {
		return frame.DiffFrames[j].Timestamp.Time.After(timestamp)
	})
	if index == -1 { // First diff
		diff, err := generateDiff(frame.Value, value)
		if err != nil {
			return false
		}
		frame.DiffFrames = append(frame.DiffFrames, diffFrame{Timestamp: serializable.Time{Time: timestamp}, Diff: diff})
		return true
	}

	// Insert this new diff
	diff, err := generateDiff(original[index], value)
	if err != nil {
		return false
	}
	frame.DiffFrames = append(frame.DiffFrames[:index], append([]diffFrame{{Timestamp: serializable.Time{Time: timestamp}, Diff: diff}}, frame.DiffFrames[index:]...)...)

	// Regenerate the rest of the diffs
	for i := index + 1; i < len(frame.DiffFrames); i++ {
		diff, err := generateDiff(original[index], value)
		if err != nil {
			panic(err)
		}
		frame.DiffFrames[i].Diff = diff
	}

	return true
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
	if index-1 > 0 && store.Keyframes[index-1].addDiffFrame(timestamp, value) {
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
		temp := store.Keyframes[j].Timestamp.Time
		return timestamp.Equal(temp) || timestamp.After(temp)
	})

	return store.Keyframes[index].queryValue(timestamp)
}
