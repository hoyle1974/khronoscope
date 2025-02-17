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

	start := time.Now()

	m1 := start.Add(time.Second)
	store.AddValue(m1, []byte("value1"))

	m2 := m1.Add(time.Second)
	store.AddValue(m2, []byte("value2"))

	m3 := m2.Add(time.Second)
	store.AddValue(m3, []byte("value3"))

	temp, err := store.FindNextTimeKey(start, 1)
	if err != nil || !m1.Equal(temp) {
		t.Fatalf("FindNextTimeKey start -> m1 failed")
	}

	temp, err = store.FindNextTimeKey(m1, 1)
	if err != nil || !m2.Equal(temp) {
		t.Fatalf("FindNextTimeKey m1 -> m2 failed")
	}

	temp, err = store.FindNextTimeKey(m2, 1)
	if err != nil || !m3.Equal(temp) {
		t.Fatalf("FindNextTimeKey m2 -> m3 failed")
	}

	_, err = store.FindNextTimeKey(m3, 1)
	if err == nil {
		t.Fatalf("FindNextTimeKey m3 -> err failed")
	}

	temp, err = store.FindNextTimeKey(m3, -1)
	if err != nil || !m2.Equal(temp) {
		t.Fatalf("FindNextTimeKey m3 -> m2 failed")
	}

	temp, err = store.FindNextTimeKey(m2, -1)
	if err != nil || !m1.Equal(temp) {
		t.Fatalf("FindNextTimeKey m2 -> m1 failed")
	}

	_, err = store.FindNextTimeKey(m1, -1)
	if err == nil {
		t.Fatalf("FindNextTimeKey m1 -> err failed")
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
