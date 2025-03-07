package temporal

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/serializable"
	"github.com/rs/zerolog/log"
)

/*
	Frame - a datastructure that stores a timestamp and a map[string]Resource
	Chunk - a set of Frames
	CompressedChunk - a chunk consisting of a keyframe, diffs, and a range of times
*/

const CHUNK_SIZE = 60

// Converts a time to a UTC version of that time at a 60 second granularity
func timeToKey60(t time.Time) string {
	t = t.Truncate(CHUNK_SIZE * time.Second)
	return t.Format(time.RFC3339)
}

func timeToKey1(t time.Time) string {
	t = t.Truncate(time.Second)
	return t.Format(time.RFC3339)
}

type Resources []byte

func NewResource(r map[string][]byte) Resources {
	b, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}
	return b
}
func (r Resources) ToMap() map[string][]byte {
	v := map[string][]byte{}
	err := json.Unmarshal(r, &v)
	if err != nil {
		panic(err)
	}
	return v
}

type frame struct {
	Timestamp time.Time
	Resources map[string][]byte
}

func (f frame) contains(timestamp time.Time) bool {
	return timeToKey1(timestamp) == timeToKey1(f.Timestamp)
}

func (f frame) Freeze() frozen_frame {
	return frozen_frame{
		Timestamp: f.Timestamp,
		Resources: NewResource(f.Resources),
	}
}

type frozen_frame struct {
	Timestamp time.Time
	Resources Resources
}

// func (f frozen_frame) contains(timestamp time.Time) bool {
// 	return timeToKey1(timestamp) == timeToKey1(f.Timestamp)
// }

type diffFrame2 struct {
	Timestamp time.Time
	Patch     Diff
}

func newDiffFrame2(f1 frozen_frame, f2 frozen_frame) diffFrame2 {
	diff, err := generateDiff(f1.Resources, f2.Resources)
	if err != nil {
		panic(err)
	}
	return diffFrame2{
		Timestamp: f2.Timestamp,
		Patch:     diff,
	}
}

type chunk struct {
	Frames      []frame
	Range       TimeRange
	CurrentTime time.Time
}

func newChunk() chunk {
	n := time.Now().Add(-time.Second)
	return chunk{
		Frames:      []frame{},
		Range:       TimeRange{serializable.NewTime(n), serializable.NewTime(n)},
		CurrentTime: n.Truncate(time.Second),
	}
}

func (c chunk) GetStateAtTime(timestamp time.Time) map[string][]byte {
	for _, frame := range c.Frames {
		if frame.contains(timestamp) {
			return frame.Resources
		}
	}
	return map[string][]byte{}
}

func (c chunk) Remove(timestamp time.Time, key string) {
	for idx, frame := range c.Frames {
		if frame.contains(timestamp) {
			delete(c.Frames[idx].Resources, key)
		}
	}
}

func (c *chunk) Add(timestamp time.Time, key string, value []byte) {
	defer c.Range.Adjust(timestamp)

	tt := timestamp.Truncate(time.Second)
	for c.CurrentTime.Before(tt) {
		c.CurrentTime = c.CurrentTime.Add(time.Second)
		lastResource := map[string][]byte{}
		if len(c.Frames) > 0 {
			lastResource = misc.DeepCopyMap(c.Frames[len(c.Frames)-1].Resources)
		}
		c.Frames = append(c.Frames, frame{
			Timestamp: c.CurrentTime.Truncate(time.Second),
			Resources: lastResource,
		})
	}

	for idx, frame := range c.Frames {
		if frame.contains(timestamp) {
			c.Frames[idx].Resources[key] = value
			return
		}
	}

	panic("what?")
}

type compressedChunk struct {
	KeyFrame frozen_frame
	Diffs    []diffFrame2
	Range    TimeRange
	size     int
}

func newCompressedChunk(f []frame) compressedChunk {
	ff := make([]frozen_frame, len(f))
	for idx, frame := range f {
		ff[idx] = frame.Freeze()
	}

	size := len(ff[0].Resources)

	c := compressedChunk{
		KeyFrame: f[0].Freeze(),
		Diffs:    make([]diffFrame2, len(f)-1),
		Range:    TimeRange{serializable.NewTime(f[0].Timestamp), serializable.NewTime(f[0].Timestamp)},
	}

	for idx := 1; idx < len(ff); idx++ {
		c.Diffs[idx-1] = newDiffFrame2(ff[idx-1], ff[idx])
		size += len(c.Diffs[idx-1].Patch)
	}

	c.size = size

	return c
}

func (c compressedChunk) GetStateAtTime(timestamp time.Time) map[string][]byte {
	if timestamp.Truncate(time.Second).Equal(c.KeyFrame.Timestamp) {
		return c.KeyFrame.Resources.ToMap()
	}

	last := c.KeyFrame.Resources
	var err error
	for _, d := range c.Diffs {
		last, err = applyDiff(last, d.Patch)
		if err != nil {
			panic(err)
		}
		if timestamp.Truncate(time.Second).Equal(d.Timestamp.Truncate(time.Second)) {
			v := map[string][]byte{}
			err = json.Unmarshal(last, &v)
			if err != nil {
				panic(err)
			}
			return v
		}
	}

	panic("not found!")
}

// Map represents a map-like data structure with time-ordered items.
type DiskMapImpl struct {
	lock       sync.RWMutex
	Chunk      chunk
	Compressed []compressedChunk
	Range      TimeRange
	Index      map[string]int
}

func NewDiskMap() *DiskMapImpl {
	return &DiskMapImpl{
		Chunk:      newChunk(),
		Compressed: []compressedChunk{},
		Range: TimeRange{
			serializable.NewTime(time.Now()),
			serializable.NewTime(time.Now()),
		},
		Index: map[string]int{},
	}
}

func (m *DiskMapImpl) GetTimeRange() (time.Time, time.Time) {
	return m.Range.Min.Time, m.Range.Max.Time
}

func (m *DiskMapImpl) Update(timestamp time.Time, key string, value []byte) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.Chunk.Add(timestamp, key, value)
	m.Range.Adjust(timestamp)

	// We have 60 seconds of frames, let's take them and compress them
	if len(m.Chunk.Frames) > CHUNK_SIZE {
		timeIdx := timeToKey60(m.Chunk.Frames[0].Timestamp)
		toCompress := []frame{}
		for idx, f := range m.Chunk.Frames {
			if timeToKey60(f.Timestamp) != timeIdx {
				copy(m.Chunk.Frames, m.Chunk.Frames[idx:]) // Shift elements left
				m.Chunk.Frames = m.Chunk.Frames[:len(m.Chunk.Frames)-idx]
				break
			} else {
				toCompress = append(toCompress, f)
			}
		}
		m.Compressed = append(m.Compressed, newCompressedChunk(toCompress))
		m.Index[timeIdx] = len(m.Compressed) - 1
		m.Chunk.Range = TimeRange{
			Min: serializable.NewTime(m.Chunk.Frames[0].Timestamp),
			Max: serializable.NewTime(m.Chunk.Frames[len(m.Chunk.Frames)-1].Timestamp),
		}
	}
}

func (m *DiskMapImpl) Remove(timestamp time.Time, key string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.Chunk.Remove(timestamp, key)
	m.Range.Adjust(timestamp)

}
func (m *DiskMapImpl) GetStateAtTime(timestamp time.Time) map[string][]byte {
	m.lock.Lock()
	defer m.lock.Unlock()
	timestamp = timestamp.Add(-time.Second)

	if m.Chunk.Range.Min.Time.Before(timestamp) {
		temp := m.Chunk.GetStateAtTime(timestamp)
		if len(temp) == 0 {
			temp = m.Chunk.GetStateAtTime(timestamp)
			log.Info().Any("timestamp", timestamp).Any("size", len(temp)).Msg("chunk size")
		}
		return temp
	}
	if idx, ok := m.Index[timeToKey60(timestamp)]; ok {
		return m.Compressed[idx].GetStateAtTime(timestamp)
	}

	return map[string][]byte{}
}

func (m *DiskMapImpl) SetMeta(timestamp time.Time, value []byte) {
	m.lock.Lock()
	defer m.lock.Unlock()
}

func (m *DiskMapImpl) GetMeta(timestamp time.Time) ([]byte, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	return []byte{}, false
}

func (m *DiskMapImpl) FindMetaTime(timestamp time.Time, dir int) (time.Time, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	return timestamp, nil
}
