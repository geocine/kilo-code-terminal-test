package main

import (
	"strings"
)

// ParsedCommand represents a parsed command with its arguments
type ParsedCommand struct {
	Command string
	Args    []string
}

// ParseCommand parses a command string into command and arguments
func ParseCommand(input string) *ParsedCommand {
	input = strings.TrimSpace(input)
	if input == "" {
		return &ParsedCommand{Command: "", Args: []string{}}
	}

	// Simple parsing - split by spaces, handle quoted strings later if needed
	parts := strings.Fields(input)

	if len(parts) == 0 {
		return &ParsedCommand{Command: "", Args: []string{}}
	}

	return &ParsedCommand{
		Command: parts[0],
		Args:    parts[1:],
	}
}

// CommandResult represents the result of executing a command
type CommandResult struct {
	Output string
	Error  error
	Exit   bool // true if command should exit the terminal
}
