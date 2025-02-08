package ui

import (
	"fmt"
)

func ResetTerminal() {
	// Optionally reset the screen and cursor position
	fmt.Print("\033[H\033[2J") // Clear the screen and reset cursor
	fmt.Print("\033[H")        // Move cursor to top-left
	fmt.Print("\r")            // Move cursor to the start of the line
}
