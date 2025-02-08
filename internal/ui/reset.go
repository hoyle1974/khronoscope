package ui

import "fmt"

func ResetTerminal() {
	fmt.Print("\033[H\033[2J") // Reset terminal
	fmt.Print("\033[H")        // Move cursor to top-left
}
