package temporal

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-test/deep"
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
}

// keyFrame represents a snapshot of a value at a specific timestamp
// It also contains up a set number of DiffFrames that hold the diffs of the data
type keyFrame struct {
	Timestamp  serializable.Time
	Value      []byte
	DiffFrames []diffFrame
	Last       []byte
}

func decodeFromBytes(data []byte, resource *resources.Resource) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(resource)
	return err
}

func (frame *keyFrame) check() {
	var r resources.Resource
	orig := frame.queryValue(frame.Timestamp.Time)
	decodeFromBytes(orig, &r)
	fmt.Printf("------------------ ORIG\n%s\n", strings.Join(r.GetDetails(), "\n"))

	for idx, d := range frame.DiffFrames {
		b := frame.queryValue(d.Timestamp.Time)

		decodeFromBytes(b, &r)
		fmt.Printf("------------------ %d\n%s\n", idx, strings.Join(r.GetDetails(), "\n"))

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

	cur := frame.Value
	for i := 0; i < index; i++ {
		cur, _ = applyDiff(cur, frame.DiffFrames[i].Diff)
		if bytes.Compare(cur, frame.DiffFrames[i].Original) != 0 {
			var r1, r2 resources.Resource
			decodeFromBytes(cur, &r1)
			decodeFromBytes(frame.DiffFrames[i].Original, &r2)

			fmt.Printf("------------------\n%s\n", strings.Join(r2.GetDetails(), "\n"))

			diff := deep.Equal(r1, r2)
			fmt.Println(strings.Join(diff, "\n"))

			// file, err := os.OpenFile("output.txt", os.O_CREATE|os.O_WRONLY, 0644)
			// if err != nil {
			// 	log.Fatal(err)
			// }
			// defer file.Close()

			// b, e := EncodeToBytes(frame)
			// if e != nil {
			// 	fmt.Println(e)
			// }
			// file.Write(b)

			// HELP! Why does this compare sometimes fail?
			/*
				var r1, r2 resources.Resource
				decodeFromBytes(cur, &r1)
				decodeFromBytes(frame.DiffFrames[i].Original, &r2)

				file, err := os.OpenFile("output.txt", os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					log.Fatal(err)
				}
				defer file.Close()

				file.WriteString(fmt.Sprintf("i=%d\n", i))
				file.WriteString(fmt.Sprintf("Diff Frames: %d\n", len(frame.DiffFrames)))
				for idx, d := range frame.DiffFrames {
					file.WriteString(
						fmt.Sprintf("%d) %d\n", idx, len(d.Diff)),
					)
				}

				file.WriteString("-------------------------")
				file.WriteString(r1.GetUID() + " ")
				file.WriteString(r1.GetTimestamp().String() + "\n")
				file.WriteString(fmt.Sprintf(" Cur:\n%s\n", strings.Join(r1.GetDetails(), "\n")))
				file.WriteString("-------------------------")
				file.WriteString(r2.GetUID() + " ")
				file.WriteString(r2.GetTimestamp().String() + "\n")
				file.WriteString(fmt.Sprintf("Orig:\n%s\n", strings.Join(r2.GetDetails(), "\n")))
				file.WriteString("-------------------------")
			*/

			panic("Original did not match diffed version")
		}
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
		diff, err := generateDiff(frame.Last, value) // Diff against full last value
		if err != nil {
			return false
		}

		frame.DiffFrames = append(frame.DiffFrames, diffFrame{
			Timestamp: serializable.Time{Time: timestamp},
			Diff:      diff,
			Original:  value,
		})

		frame.Last = value // Update stored full value for next append
		return true
	}

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
			Original:  original[idx],
		})

		last = original[idx]
	}

	return true
}

/*
func (frame *keyFrame) addDiffFrame(timestamp time.Time, value []byte) bool {
	originalCopy := make([]byte, len(value))
	copy(originalCopy, value)

	if len(frame.DiffFrames) == KEYFRAME_RATE {
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
		frame.DiffFrames = append(frame.DiffFrames, diffFrame{Timestamp: serializable.Time{Time: timestamp}, Diff: diff, Original: originalCopy})
		return true
	}

	diff, err := generateDiff(original[max(0, index-1)], value)
	if err != nil {
		return false
	}
	newDiffFrame := diffFrame{Timestamp: serializable.Time{Time: timestamp}, Diff: diff, Original: originalCopy}

	if len(frame.DiffFrames) == KEYFRAME_RATE {
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
*/

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type diffFrame struct {
	Timestamp serializable.Time
	Diff      Diff
	Original  []byte
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
