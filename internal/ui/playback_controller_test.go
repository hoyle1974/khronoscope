package ui

import (
	"testing"
	"time"
)

func TestTimeRenderer(t *testing.T) {
	renderer := PlaybackRenderer{}

	tests := []struct {
		name     string
		model    playbackModel
		expected string
	}{
		{"Paused", playbackModel{useVirtualTime: true, playSpeed: 0}, "⏸"},
		{"Play", playbackModel{useVirtualTime: true, playSpeed: time.Second}, "▶"},
		{"Fast Forward 2x", playbackModel{useVirtualTime: true, playSpeed: 2 * time.Second}, "▶▶"},
		{"Rewind", playbackModel{useVirtualTime: true, playSpeed: -1 * time.Second}, "◀"},
		{"Rewind 3x", playbackModel{useVirtualTime: true, playSpeed: -3 * time.Second}, "◀◀3"},
		{"Disabled", playbackModel{useVirtualTime: false}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.Render(tt.model)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
