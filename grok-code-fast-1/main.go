package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Create terminal
	terminal := &Terminal{
		FS:      NewFileSystem(),
		History: []string{},
		Running: true,
	}

	fmt.Println("Welcome to Virtual Terminal Emulator!")
	fmt.Println("Type 'help' for available commands, 'exit' to quit.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for terminal.Running {
		// Display prompt
		prompt := terminal.FS.GetPath(terminal.FS.CurrentDir) + " $ "
		fmt.Print(prompt)

		// Read input
		input, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			fmt.Println("Error reading input:", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Add to history
		terminal.History = append(terminal.History, input)

		// Parse and execute command
		cmd := ParseCommand(input)
		result := terminal.ExecuteCommand(cmd)

		// Display output
		if result.Output != "" {
			fmt.Print(result.Output)
		}

		// Display error
		if result.Error != nil {
			fmt.Println(result.Error)
		}

		// Check if should exit
		if result.Exit {
			terminal.Running = false
		}
	}

	fmt.Println("Terminal session ended.")
}
