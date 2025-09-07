package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TerminalProcess represents a running terminal emulator process
type TerminalProcess struct {
	Process *exec.Cmd
	Stdin   io.WriteCloser
	Stdout  io.ReadCloser
	Stderr  io.ReadCloser
	Name    string // Will be the folder name (glm, fast, sky, dusk1)
}

// TestResult represents the result of a single test case
type TestResult struct {
	TestCase    TestCase
	Variant     string
	Passed      bool
	Output      []string
	Expected    []string
	Error       string
	Duration    time.Duration
	Timestamp   time.Time
}

// VariantResults holds all test results for a single variant
type VariantResults struct {
	Name          string
	BuildSuccess  bool
	BuildError    string
	TestResults   []TestResult
	TotalTests    int
	PassedTests   int
	FailedTests   int
	TotalDuration time.Duration
	PassRate      float64
}

// TestSummary holds the overall test summary
type TestSummary struct {
	Variants      []VariantResults
	TotalTests    int
	TotalPassed   int
	TotalFailed   int
	TotalDuration time.Duration
	Timestamp     time.Time
	PassRate      float64
}

// BuildVariant builds a specific variant and returns the executable path
func BuildVariant(variantPath string) (string, error) {
	variantName := filepath.Base(variantPath)
	
	// Get absolute paths to avoid confusion
	absVariantPath, err := filepath.Abs(variantPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %v", err)
	}
	
	// Build in the current directory (test/)
	outputPath := variantName + ".exe"
	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute output path: %v", err)
	}
	
	// Clean previous build
	os.Remove(outputPath)
	
	// Build the variant
	cmd := exec.Command("go", "build", "-o", absOutputPath)
	cmd.Dir = absVariantPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %v\nOutput: %s", err, string(output))
	}
	
	// Verify the executable exists
	if _, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("executable not found after build: %v", err)
	}
	
	return outputPath, nil
}

// StartTerminal starts a terminal emulator process
func StartTerminal(executablePath string) (*TerminalProcess, error) {
	variantName := filepath.Base(strings.TrimSuffix(executablePath, ".exe"))
	
	// Use absolute path for the command
	absPath, err := filepath.Abs(executablePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for executable: %v", err)
	}
	
	cmd := exec.Command(absPath)
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
	}
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}
	
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %v", err)
	}
	
	return &TerminalProcess{
		Process: cmd,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		Name:    variantName,
	}, nil
}

// ExecuteCommand sends a command to the terminal and captures the output
func (tp *TerminalProcess) ExecuteCommand(command string, timeout time.Duration) (string, error) {
	// Send command
	_, err := tp.Stdin.Write([]byte(command + "\n"))
	if err != nil {
		return "", fmt.Errorf("failed to write command: %v", err)
	}
	
	// Wait for command to process
	time.Sleep(200 * time.Millisecond)
	
	// Read output with a simple approach
	outputChan := make(chan string, 1)
	errorChan := make(chan error, 1)
	
	go func() {
		buffer := make([]byte, 4096)
		var output strings.Builder
		
		// Try to read output with timeout
		for attempts := 0; attempts < 5; attempts++ {
			n, err := tp.Stdout.Read(buffer)
			if err != nil {
				if err == io.EOF {
					break
				}
				// For other errors, try a few more times
				time.Sleep(100 * time.Millisecond)
				continue
			}
			
			if n > 0 {
				output.Write(buffer[:n])
				// If we got some output, wait a bit more for any additional output
				time.Sleep(100 * time.Millisecond)
			} else {
				break
			}
		}
		
		outputChan <- output.String()
	}()
	
	select {
	case output := <-outputChan:
		cleaned := cleanTerminalOutput(output)
		return cleaned, nil
	case err := <-errorChan:
		return "", err
	case <-time.After(timeout):
		return "", fmt.Errorf("command timeout after %v", timeout)
	}
}

// cleanTerminalOutput removes terminal control characters and prompts
func cleanTerminalOutput(output string) string {
	lines := strings.Split(output, "\n")
	var cleanLines []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines
		if line == "" {
			continue
		}
		
		// Skip prompt lines
		if strings.Contains(line, "$") && len(line) < 20 {
			continue
		}
		
		// Skip error messages about EOF (these are from our testing)
		if strings.Contains(line, "Error reading input") {
			continue
		}
		
		// Skip terminal startup messages
		if strings.Contains(line, "Terminal Emulator") || strings.Contains(line, "Type 'help'") {
			continue
		}
		
		cleanLines = append(cleanLines, line)
	}
	
	return strings.Join(cleanLines, "\n")
}

// Close terminates the terminal process
func (tp *TerminalProcess) Close() error {
	if tp.Stdin != nil {
		tp.Stdin.Close()
	}
	if tp.Stdout != nil {
		tp.Stdout.Close()
	}
	if tp.Stderr != nil {
		tp.Stderr.Close()
	}
	
	if tp.Process != nil && tp.Process.Process != nil {
		// Try to terminate gracefully
		tp.Stdin.Write([]byte("exit\n"))
		time.Sleep(100 * time.Millisecond)
		
		// Force kill if still running
		return tp.Process.Process.Kill()
	}
	
	return nil
}

// RunTestCase executes a single test case against a terminal process
func RunTestCase(tp *TerminalProcess, testCase TestCase) TestResult {
	result := TestResult{
		TestCase:  testCase,
		Variant:   tp.Name,
		Timestamp: time.Now(),
	}
	
	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()
	
	// Execute setup commands
	for _, setupCmd := range testCase.Setup {
		_, err := tp.ExecuteCommand(setupCmd, testCase.Timeout)
		if err != nil {
			result.Error = fmt.Sprintf("Setup command failed: %v", err)
			result.Passed = false
			return result
		}
	}
	
	// Execute test commands
	result.Output = make([]string, len(testCase.Commands))
	result.Expected = testCase.Expected
	
	for i, cmd := range testCase.Commands {
		output, err := tp.ExecuteCommand(cmd, testCase.Timeout)
		if err != nil {
			result.Error = err.Error()
			result.Passed = false
			return result
		}
		result.Output[i] = output
	}
	
	// Validate outputs
	result.Passed = true
	for i, output := range result.Output {
		if i < len(testCase.Expected) && i < len(testCase.Validation) {
			expected := testCase.Expected[i]
			validation := testCase.Validation[i]
			
			if !ValidateOutput(output, expected, validation) {
				result.Passed = false
				if result.Error == "" {
					result.Error = fmt.Sprintf("Validation failed for command %d: expected '%s', got '%s'", i+1, expected, output)
				}
			}
		}
	}
	
	// Execute cleanup commands
	for _, cleanupCmd := range testCase.Cleanup {
		tp.ExecuteCommand(cleanupCmd, testCase.Timeout)
		// Ignore errors in cleanup
	}
	
	return result
}

// GetVariantPaths returns the paths to all variant directories
func GetVariantPaths() ([]string, error) {
	baseDir := ".."
	variants := []string{}
	
	// Read directory entries
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read base directory: %v", err)
	}
	
	// Look for directories that contain go.mod and main.go
	for _, entry := range entries {
		if entry.IsDir() {
			dirPath := filepath.Join(baseDir, entry.Name())
			
			// Skip the test directory itself
			if entry.Name() == "test" {
				continue
			}
			
			// Check if it has go.mod and main.go (or any .go file)
			if hasGoFiles(dirPath) {
				variants = append(variants, dirPath)
			}
		}
	}
	
	if len(variants) == 0 {
		return nil, fmt.Errorf("no valid Go projects found in parent directories")
	}
	
	return variants, nil
}

// hasGoFiles checks if a directory contains Go source files
func hasGoFiles(dirPath string) bool {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}
	
	hasGoMod := false
	hasGoSource := false
	
	for _, entry := range entries {
		if !entry.IsDir() {
			if entry.Name() == "go.mod" {
				hasGoMod = true
			}
			if strings.HasSuffix(entry.Name(), ".go") {
				hasGoSource = true
			}
		}
	}
	
	return hasGoMod || hasGoSource // At least one of these should be true
}

// CreateFreshTerminal creates a new terminal process for testing
func CreateFreshTerminal(executablePath string) (*TerminalProcess, error) {
	tp, err := StartTerminal(executablePath)
	if err != nil {
		return nil, err
	}
	
	// Give the terminal more time to start
	time.Sleep(500 * time.Millisecond)
	
	// Try a simple verification command with longer timeout
	_, err = tp.ExecuteCommand("pwd", 5*time.Second)
	if err != nil {
		// Don't fail immediately - the terminal might still work for other commands
		fmt.Printf("Warning: Initial pwd command failed for %s: %v\n", tp.Name, err)
		// Reset the terminal state by waiting a bit more
		time.Sleep(300 * time.Millisecond)
	}
	
	return tp, nil
}

// LogTestProgress logs test progress to console
func LogTestProgress(variant string, testCase TestCase, result TestResult) {
	status := "[PASS]"
	if !result.Passed {
		status = "[FAIL]"
	}
	
	fmt.Printf("[%s] %s %s.%s - %s (%v)\n", 
		variant, 
		status, 
		testCase.Category, 
		testCase.ID, 
		testCase.Description,
		result.Duration.Truncate(time.Millisecond))
	
	if !result.Passed && result.Error != "" {
		fmt.Printf("    Error: %s\n", result.Error)
	}
}

// CalculateSummary calculates the test summary from variant results
func CalculateSummary(variants []VariantResults) TestSummary {
	summary := TestSummary{
		Variants:  variants,
		Timestamp: time.Now(),
	}
	
	for _, variant := range variants {
		summary.TotalTests += variant.TotalTests
		summary.TotalPassed += variant.PassedTests
		summary.TotalFailed += variant.FailedTests
		summary.TotalDuration += variant.TotalDuration
	}
	
	return summary
}