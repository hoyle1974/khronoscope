package ui

import (
	"fmt"
	"sync"
	"time"
)

type playbackModel struct {
	useVirtualTime bool
	virtualTime    time.Time
	playSpeed      time.Duration
}

type PlaybackRenderer struct {
	leftArrow   string
	rightArrow  string
	pauseSymbol string
}

func (tc PlaybackRenderer) Render(model playbackModel) string {
	if !model.useVirtualTime {
		return ""
	}

	// var leftArrow = fmt.Sprintf("%c", '\u25C0')
	// var rightArrow = fmt.Sprintf("%c", '\u25B6')
	// var pauseSymbol = fmt.Sprintf("%c", '\u23F8')

	symbol := tc.pauseSymbol

	if model.playSpeed == 0 {
		symbol = tc.pauseSymbol
	} else if model.playSpeed == 1*time.Second {
		symbol = tc.rightArrow
	} else if model.playSpeed == 2*time.Second {
		symbol = tc.rightArrow + tc.rightArrow
	} else if model.playSpeed > 2*time.Second {
		symbol = fmt.Sprintf("%d"+tc.rightArrow+tc.rightArrow, model.playSpeed/time.Second)
	} else if model.playSpeed == -1*time.Second {
		symbol = tc.leftArrow
	} else if model.playSpeed == -2*time.Second {
		symbol = tc.leftArrow + tc.leftArrow
	} else if model.playSpeed < -2*time.Second {
		symbol = fmt.Sprintf(tc.leftArrow+tc.leftArrow+"%d", -model.playSpeed/time.Second)
	}

	return symbol
}

// Toggle between getting a time value based on time.Now
// vs VCR like controls
type PlaybackController struct {
	lock     sync.Mutex
	limiter  Limiter
	onChange func()
	model    playbackModel
	renderer PlaybackRenderer
}

type Limiter interface {
	GetTimeRange() (time.Time, time.Time)
}

type ChangeListener interface {
	OnChange()
}

func NewTimeController(limiter Limiter, onChange func()) *PlaybackController {
	t := &PlaybackController{
		limiter:  limiter,
		onChange: onChange,
		model:    playbackModel{},
		renderer: PlaybackRenderer{
			leftArrow:   "◀",
			rightArrow:  "▶",
			pauseSymbol: "⏸",
		},
	}

	go t.Tick()

	return t
}

func (tc *PlaybackController) Tick() {
	for {
		time.Sleep(time.Second)

		tc.lock.Lock()
		if tc.model.useVirtualTime {
			minTime, maxTime := tc.limiter.GetTimeRange()
			tc.model.virtualTime = tc.model.virtualTime.Add(tc.model.playSpeed)

			if tc.model.virtualTime.Before(minTime) {
				tc.model.virtualTime = minTime
				tc.model.playSpeed = 0
			}
			if tc.model.virtualTime.After(maxTime) {
				tc.model.useVirtualTime = false
			}
		}

		tc.lock.Unlock()

		tc.onChange()
	}
}

func (tc *PlaybackController) IsEnabled() bool {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	return tc.model.useVirtualTime
}

func (tc *PlaybackController) GetTimeToUse() time.Time {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	if tc.model.useVirtualTime {
		return tc.model.virtualTime
	}
	return time.Now()
}

func (tc *PlaybackController) enableVirtualTime() {
	if !tc.model.useVirtualTime {
		tc.model.playSpeed = 0
		tc.model.virtualTime = time.Now().Add(-time.Millisecond)
	}
	tc.model.useVirtualTime = true
}

func (tc *PlaybackController) EnableVirtualTime() {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	tc.enableVirtualTime()
}

func (tc *PlaybackController) DisableVirtualTime() {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	tc.model.useVirtualTime = false
}

func (tc *PlaybackController) FastForward() {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	tc.enableVirtualTime()
	if tc.model.playSpeed < 0 {
		tc.model.playSpeed = 0
	} else if tc.model.playSpeed == 0 {
		tc.model.playSpeed = time.Second
	} else {
		tc.model.playSpeed *= 2
	}
}

func (tc *PlaybackController) Rewind() {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	tc.enableVirtualTime()
	if tc.model.playSpeed > 0 {
		tc.model.playSpeed = 0
	} else if tc.model.playSpeed == 0 {
		tc.model.playSpeed = -time.Second
	} else {
		tc.model.playSpeed *= 2
	}
}

func (tc *PlaybackController) Pause() {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	tc.enableVirtualTime()
	tc.model.playSpeed = 0
}

func (tc *PlaybackController) Play() {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	tc.enableVirtualTime()
	tc.model.playSpeed = time.Second
}

func (tc *PlaybackController) GetPlaySpeed() time.Duration {
	tc.lock.Lock()
	defer tc.lock.Unlock()
	return tc.model.playSpeed
}

// SetTime sets the VCR time to the specified time
func (v *PlaybackController) SetTime(t time.Time) {
	v.lock.Lock()
	defer v.lock.Unlock()
	v.model.virtualTime = t
}

func (v *PlaybackController) Render() string {
	v.lock.Lock()
	defer v.lock.Unlock()

	return v.renderer.Render(v.model)
}
