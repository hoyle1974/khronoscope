package main

import (
	"fmt"
	"sync"
	"time"
)

// Toggle between getting a time value based on time.Now
// vs VCR like controls
type VCRControl struct {
	lock          sync.Mutex
	limiter       Limiter
	onChange      func()
	enableVCRFlag bool
	vcrTime       time.Time
	playSpeed     time.Duration
}

type Limiter interface {
	GetTimeRange() (time.Time, time.Time)
}

type ChangeListener interface {
	OnChange()
}

func NewVCRControl(limiter Limiter, onChange func()) *VCRControl {
	t := &VCRControl{
		limiter:       limiter,
		onChange:      onChange,
		enableVCRFlag: false,
	}

	go t.Tick()

	return t
}

func (tc *VCRControl) Tick() {
	for {
		time.Sleep(time.Second)

		tc.lock.Lock()
		if tc.enableVCRFlag {
			minTime, maxTime := tc.limiter.GetTimeRange()
			tc.vcrTime = tc.vcrTime.Add(tc.playSpeed)

			if tc.vcrTime.Before(minTime) {
				tc.vcrTime = minTime
				tc.playSpeed = 0
			}
			if tc.vcrTime.After(maxTime) {
				tc.enableVCRFlag = false
			}
		}

		tc.lock.Unlock()

		tc.onChange()
	}
}

func (tc *VCRControl) IsEnabled() bool {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	return tc.enableVCRFlag
}

func (tc *VCRControl) GetTimeToUse() time.Time {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	if tc.enableVCRFlag {
		return tc.vcrTime
	}
	return time.Now()
}

func (tc *VCRControl) enableVCR() {
	if !tc.enableVCRFlag {
		tc.playSpeed = 0
		tc.vcrTime = time.Now()
	}
	tc.enableVCRFlag = true
}

func (tc *VCRControl) EnableVCR() {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	tc.enableVCR()
}

func (tc *VCRControl) DisableVCR() {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	tc.enableVCRFlag = false
}

func (tc *VCRControl) FastForward() {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	tc.enableVCR()
	if tc.playSpeed < 0 {
		tc.playSpeed = 0
	} else if tc.playSpeed == 0 {
		tc.playSpeed = time.Second
	} else {
		tc.playSpeed *= 2
	}
}

func (tc *VCRControl) Rewind() {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	tc.enableVCR()
	if tc.playSpeed > 0 {
		tc.playSpeed = 0
	} else if tc.playSpeed == 0 {
		tc.playSpeed = -time.Second
	} else {
		tc.playSpeed *= 2
	}
}

func (tc *VCRControl) Pause() {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	tc.enableVCR()
	tc.playSpeed = 0
}

func (tc *VCRControl) Play() {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	tc.enableVCR()
	tc.playSpeed = time.Second
}

func (tc *VCRControl) Render() string {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	if !tc.enableVCRFlag {
		return ""
	}

	symbol := "||"

	if tc.playSpeed == 1*time.Second {
		symbol = ">"
	} else if tc.playSpeed == 2*time.Second {
		symbol = ">>"
	} else if tc.playSpeed > 2*time.Second {
		symbol = fmt.Sprintf("%d>>", tc.playSpeed/time.Second)
	} else if tc.playSpeed == -1*time.Second {
		symbol = "<"
	} else if tc.playSpeed == -2*time.Second {
		symbol = "<<"
	} else if tc.playSpeed < -2*time.Second {
		symbol = fmt.Sprintf("<<%d", -tc.playSpeed/time.Second)
	}

	return symbol
}
