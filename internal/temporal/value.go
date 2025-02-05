package temporal

import (
	"sort"
	"time"

	"github.com/hoyle1974/khronoscope/internal/metrics"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/resources"
	"github.com/hoyle1974/khronoscope/internal/serializable"
)

const KEYFRAME_RATE = 16

func NewTimeValueStore() *TimeValueStore {
	store := &TimeValueStore{
		Keyframes: []keyFrame{},
	}
	return store
}

// TimeValueStore holds the values and diffs with associated timestamps
type TimeValueStore struct {
	Keyframes []keyFrame
	// lastTimestamp time.Time
	// lastValue     Diff
}

// keyFrame represents a snapshot of a value at a specific timestamp
// It also contains up a set number of DiffFrames that hold the diffs of the data
type keyFrame struct {
	Timestamp  serializable.Time
	Value      []byte
	DiffFrames []diffFrame
	Last       []byte
}

func (frame *keyFrame) check() {
	var r resources.Resource
	orig := frame.queryValue(frame.Timestamp.Time)
	misc.DecodeFromBytes(orig, &r)
	// fmt.Printf("------------------ ORIG\n%s\n", strings.Join(r.GetDetails(), "\n"))

	for _, d := range frame.DiffFrames {
		b := frame.queryValue(d.Timestamp.Time)

		misc.DecodeFromBytes(b, &r)
		// fmt.Printf("------------------ %d\n%s\n", idx, strings.Join(r.GetDetails(), "\n"))
	}
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

	if index > 0 {
		if frame.DiffFrames[index-1].original != nil {
			metrics.Count("DiffFrame.Original", 1)
			return frame.DiffFrames[index-1].original
		} else {
			metrics.Count("DiffFrame.Original.Nil", 1)
		}
	}
	metrics.Count("DiffFrame.Diff", 1)

	cur := frame.Value
	for i := 0; i < index; i++ {
		cur, _ = applyDiff(cur, frame.DiffFrames[i].Diff)
		/*
			if bytes.Compare(cur, frame.DiffFrames[i].Original) != 0 {
				var r1, r2 resources.Resource
				decodeFromBytes(cur, &r1)
				decodeFromBytes(frame.DiffFrames[i].Original, &r2)

				fmt.Printf("------------------\n%s\n", strings.Join(r2.GetDetails(), "\n"))

				diff := deep.Equal(r1, r2)
				fmt.Println(strings.Join(diff, "\n"))

				panic("Original did not match diffed version")
			}
		*/
	}
	return cur
}

func (frame *keyFrame) addDiffFrame(timestamp time.Time, value []byte) bool {
	if len(frame.DiffFrames) == KEYFRAME_RATE {
		if len(frame.DiffFrames) > 0 && frame.DiffFrames[len(frame.DiffFrames)-1].Timestamp.Time.Before(timestamp) {
			return false // We have enough frames, and adding at the end
		}
	}

	if len(frame.DiffFrames) == 0 || (len(frame.DiffFrames) > 0 && frame.DiffFrames[len(frame.DiffFrames)-1].Timestamp.Time.Before(timestamp)) {
		metrics.Count("addDiffFrame.Append", 1)
		diff, err := generateDiff(frame.Last, value) // Diff against full last value
		if err != nil {
			return false
		}

		frame.DiffFrames = append(frame.DiffFrames, diffFrame{
			Timestamp: serializable.Time{Time: timestamp},
			Diff:      diff,
			original:  value,
		})

		frame.Last = value // Update stored full value for next append
		return true
	}

	metrics.Count("addDiffFrame.Insert", 1)

	// Decompress all frames
	curr := frame.Value
	times := make([]serializable.Time, len(frame.DiffFrames))
	original := make([][]byte, len(frame.DiffFrames))
	for idx, diffFrame := range frame.DiffFrames {
		actual, err := applyDiff(curr, diffFrame.Diff)
		if err != nil {
			return false // Stop if any diff application fails
		}
		original[idx] = actual
		times[idx] = diffFrame.Timestamp
		curr = actual
	}

	index := sort.Search(len(times), func(j int) bool {
		return times[j].Time.After(timestamp)
	})

	original = append(original, nil)           // Extend the slice by one
	times = append(times, serializable.Time{}) // Extend the slice by one

	copy(original[index+1:], original[index:]) // Shift elements to the right
	copy(times[index+1:], times[index:])       // Shift elements to the right

	original[index] = value
	times[index] = serializable.Time{Time: timestamp}

	frame.DiffFrames = []diffFrame{}

	last := frame.Value
	for idx := 0; idx < len(original); idx++ {
		diff, err := generateDiff(last, original[idx])
		if err != nil {
			panic(err)
		}
		frame.DiffFrames = append(frame.DiffFrames, diffFrame{
			Timestamp: times[idx],
			Diff:      diff,
			original:  original[idx],
		})

		last = original[idx]
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
	original  []byte
}

// Add a value that is valid @timestamp and after
func (store *TimeValueStore) AddValue(timestamp time.Time, value []byte) {
	// Find the insertion point to maintain chronological order.
	index := sort.Search(len(store.Keyframes), func(j int) bool {
		return store.Keyframes[j].Timestamp.Time.After(timestamp)
	})

	if len(store.Keyframes) == 0 {
		store.Keyframes = append(store.Keyframes, keyFrame{
			Timestamp:  serializable.Time{Time: timestamp},
			Value:      value,
			DiffFrames: make([]diffFrame, 0, KEYFRAME_RATE),
			Last:       value,
		})
		return
	}

	// index is where we would put this keyframe.  Backup one keyframe and see if we just want to append diff
	if index > 0 && store.Keyframes[index-1].addDiffFrame(timestamp, value) { // Corrected: Check index > 0
		return
	}

	// Insert the new value at the correct position.
	store.Keyframes = append(store.Keyframes[:index], append([]keyFrame{{
		Timestamp:  serializable.Time{Time: timestamp},
		Value:      value,
		DiffFrames: make([]diffFrame, 0, KEYFRAME_RATE),
		Last:       value,
	}}, store.Keyframes[index:]...)...)

}

func (store *TimeValueStore) QueryValue(timestamp time.Time) []byte {
	return store.queryValue(timestamp)
}

// Return the most recent value that was set on or before timestamp.
func (store *TimeValueStore) queryValue(timestamp time.Time) []byte {
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
