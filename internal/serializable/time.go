package serializable

import (
	"bytes"
	"encoding/gob"
	"time"
)

// Time wraps time.Time and implements GobEncoder and GobDecoder
type Time struct {
	Time     time.Time
	Location string
}

// GobEncode implements the gob.GobEncoder interface
func (st Time) GobEncode() ([]byte, error) {
	// Serialize time and location
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	// Serialize time as binary and location as a string
	timeData, err := st.Time.MarshalBinary()
	if err != nil {
		return nil, err
	}

	if err := enc.Encode(timeData); err != nil {
		return nil, err
	}

	if err := enc.Encode(st.Time.Location().String()); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

var timeZoneMapping = map[string]string{
	"PST": "America/Los_Angeles",
	"EST": "America/New_York",
	"CST": "America/Chicago",
	"MST": "America/Denver",
	"UTC": "UTC",
}

// GobDecode implements the gob.GobDecoder interface
func (st *Time) GobDecode(data []byte) error {
	// Deserialize time and location
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)

	var timeData []byte
	if err := dec.Decode(&timeData); err != nil {
		return err
	}

	var locationName string
	if err := dec.Decode(&locationName); err != nil {
		return err
	}

	// Unmarshal the time
	var t time.Time
	if err := t.UnmarshalBinary(timeData); err != nil {
		return err
	}

	// Resolve the location
	normalizedLocation := locationName
	if mappedLocation, ok := timeZoneMapping[locationName]; ok {
		normalizedLocation = mappedLocation
	}

	// Set the location
	loc, err := time.LoadLocation(normalizedLocation)
	if err != nil {
		return err
	}

	st.Time = t.In(loc)
	st.Location = locationName
	return nil
}
