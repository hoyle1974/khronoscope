package temporal

import (
	"bytes"
	"fmt"
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
	if !bytes.Equal(store.QueryValue(t1.Add(time.Second*2)), []byte{1, 2, 3}) {
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
	if !bytes.Equal(store.QueryValue(t1.Add(time.Second*2)), []byte{1, 2, 3}) {
		t.Fatalf("Expected a value")
	}
	if !bytes.Equal(store.QueryValue(t1.Add(time.Second*6)), []byte{2, 3, 4}) {
		t.Fatalf("Expected a value")
	}
}

func Test_ThreeValue(t *testing.T) {
	store := NewTimeValueStore()

	t1 := time.Now()
	store.AddValue(t1.Add(time.Second), []byte{1, 2, 3})
	store.AddValue(t1.Add(time.Second*5), []byte{2, 3, 4})
	store.AddValue(t1.Add(time.Second*10), []byte{3, 4, 5})

	if store.QueryValue(t1) != nil {
		t.Fatalf("Expected nil value")
	}
	if !bytes.Equal(store.QueryValue(t1.Add(time.Second*2)), []byte{1, 2, 3}) {
		t.Fatalf("Expected a value")
	}
	if !bytes.Equal(store.QueryValue(t1.Add(time.Second*6)), []byte{2, 3, 4}) {
		t.Fatalf("Expected a value")
	}
	if !bytes.Equal(store.QueryValue(t1.Add(time.Second*11)), []byte{3, 4, 5}) {
		t.Fatalf("Expected a value")
	}
}

func Test_ManyValues(t *testing.T) {
	store := NewTimeValueStore()

	t1 := time.Now()
	for i := 0; i < 1000; i++ {
		store.AddValue(t1.Add(time.Second+(time.Duration(i)*time.Second)), []byte{byte(i)})
	}

	if store.QueryValue(t1) != nil {
		t.Fatalf("Expected nil value")
	}

	for i := 0; i < 100; i++ {
		offset := time.Second + (time.Duration(i) * time.Second) + time.Millisecond
		if !bytes.Equal(store.QueryValue(t1.Add(offset)), []byte{byte(i)}) {
			t.Fatalf("Expected a value")
		}
	}
}

func Test_Next(t *testing.T) {
	store := NewTimeValueStore()

	// Create arrays of times
	start := time.Now()
	times := []time.Time{start}
	for idx := 1; idx < 200; idx++ {
		times = append(times, start.Add(time.Duration(idx)*time.Second))
		store.AddValue(times[idx], []byte(fmt.Sprintf("value:%v", idx)))
	}

	for idx := 2; idx < len(times); idx++ {
		temp, err := store.FindNextTimeKey(times[idx-1], 1)
		if err != nil {
			t.Fatalf("FindNextTimeKey %v -> %v failed, got %v",
				times[idx-1].Sub(start),
				times[idx].Sub(start),
				err)
		}
		if err != nil || !times[idx].Equal(temp) {
			t.Fatalf("FindNextTimeKey %v -> %v failed, got %v",
				times[idx-1].Sub(start),
				times[idx].Sub(start),
				temp.Sub(start))
		}

		temp, err = store.FindNextTimeKey(times[idx], -1)
		if err != nil {
			t.Fatalf("FindNextTimeKey %v -> %v failed, got %v",
				times[idx].Sub(start),
				times[idx-1].Sub(start),
				err)
		}
		if !times[idx-1].Equal(temp) {
			t.Fatalf("FindNextTimeKey %v -> %v failed, got %v",
				times[idx].Sub(start),
				times[idx-1].Sub(start),
				temp.Sub(start))
		}
	}

}

// func Test_Crash(t *testing.T) {
// 	gob.Register(resources.PodExtra{})
// 	resources.RegisterResourceRenderer("Pod", resources.PodRenderer{})

// 	// Read the file into a byte array
// 	data, err := ioutil.ReadFile("/Users/jstrohm/code/khronoscope/keyframe.bin")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	var k keyFrame
// 	buf := bytes.NewBuffer(data)
// 	dec := gob.NewDecoder(buf)
// 	err = dec.Decode(&k)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	nk := keyFrame{
// 		k.Timestamp,
// 		k.Value,
// 		[]diffFrame{},
// 	}

// 	for t := 0; t < len(k.DiffFrames); t += 2 {
// 		nk.addDiffFrame(k.DiffFrames[t].Timestamp.Time, k.DiffFrames[t].Original)
// 	}
// 	for t := 1; t < len(k.DiffFrames); t += 2 {
// 		nk.addDiffFrame(k.DiffFrames[t].Timestamp.Time, k.DiffFrames[t].Original)
// 	}

// 	nk.check()

// 	/*
// 		k.check()

// 		fmt.Println(k)
// 	*/

// }
