package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

// FileBasedTerminal represents a terminal that uses file-based communication
type FileBasedTerminal struct {
	Name           string
	ExecutablePath string
	WorkDir        string
	InputFile      string
	OutputFile     string
}

// NewFileBasedTerminal creates a new file-based terminal tester
func NewFileBasedTerminal(executablePath string) (*FileBasedTerminal, error) {
	name := filepath.Base(strings.TrimSuffix(executablePath, ".exe"))
	workDir := filepath.Join("temp", name)

	// Create work directory
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %v", err)
	}

	return &FileBasedTerminal{
		Name:           name,
		ExecutablePath: executablePath,
		WorkDir:        workDir,
		InputFile:      filepath.Join(workDir, "input.txt"),
		OutputFile:     filepath.Join(workDir, "output.txt"),
	}, nil
}

// ExecuteCommand executes a command using file-based communication
func (fbt *FileBasedTerminal) ExecuteCommand(command string, timeout time.Duration) (string, error) {
	// Write command to input file
	commands := []string{command, "exit"}
	input := strings.Join(commands, "\n") + "\n"

	if err := os.WriteFile(fbt.InputFile, []byte(input), 0644); err != nil {
		return "", fmt.Errorf("failed to write input file: %v", err)
	}

	// Clean output file
	os.Remove(fbt.OutputFile)

	// Get absolute paths
	absExec, _ := filepath.Abs(fbt.ExecutablePath)
	absInput, _ := filepath.Abs(fbt.InputFile)
	absOutput, _ := filepath.Abs(fbt.OutputFile)

	// Run terminal with input/output redirection
	cmd := exec.Command(absExec)
	cmd.Dir = fbt.WorkDir

	// Set up input from file
	inputFile, err := os.Open(absInput)
	if err != nil {
		return "", fmt.Errorf("failed to open input file: %v", err)
	}
	defer inputFile.Close()
	cmd.Stdin = inputFile

	// Set up output to file
	outputFile, err := os.Create(absOutput)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()
	cmd.Stdout = outputFile
	cmd.Stderr = outputFile

	// Run command with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		// Read output file regardless of success/failure
		output, readErr := os.ReadFile(absOutput)
		if readErr != nil {
			return "", fmt.Errorf("failed to read output: %v", readErr)
		}

		cleaned := fbt.cleanOutput(string(output))
		if err != nil {
			return cleaned, fmt.Errorf("command failed: %v", err)
		}

		return cleaned, nil

	case <-time.After(timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}

		// Try to read partial output
		if output, err := os.ReadFile(absOutput); err == nil {
			return fbt.cleanOutput(string(output)), fmt.Errorf("timeout after %v", timeout)
		}
		return "", fmt.Errorf("timeout after %v", timeout)
	}
}

// transformCommandsForVariant transforms echo-based commands into appropriate sequences per variant
func transformCommandsForVariant(variantName string, commands []string) []string {
	switch variantName {
	case "sonoma-dusk-alpha":
		return transformForDusk1(commands)
	case "sonoma-sky-alpha":
		return transformForSky(commands)
	case "glm-4.5":
		// GLM supports echo redirection, but it fails due to concatenation issues
		// Let it fail naturally to indicate source code needs fixing
		return commands
	case "grok-code-fast-1":
		// Fast doesn't support echo redirection or edit - skip content creation tests
		return commands
	default:
		return commands
	}
}

// transformForDusk1 converts echo > file commands to touch + edit sequences
func transformForDusk1(commands []string) []string {
	var result []string

	for _, cmd := range commands {
		if strings.Contains(cmd, " > ") && strings.HasPrefix(cmd, "echo ") {
			// Transform "echo Hello > file.txt" to touch + edit sequence
			parts := strings.Split(cmd, " > ")
			if len(parts) == 2 {
				content := strings.TrimPrefix(parts[0], "echo ")
				filename := strings.TrimSpace(parts[1])

				// Create touch + edit + content + :wq sequence
				result = append(result, "touch "+filename)
				// Note: The edit sequence will be handled by special input processing
				result = append(result, "edit_with_content:"+filename+":"+content)
			}
		} else {
			result = append(result, cmd)
		}
	}

	return result
}

// transformForSky converts echo > file commands to touch + edit sequences
func transformForSky(commands []string) []string {
	var result []string

	for _, cmd := range commands {
		if strings.Contains(cmd, " > ") && strings.HasPrefix(cmd, "echo ") {
			// Transform "echo Hello > file.txt" to touch + edit sequence
			parts := strings.Split(cmd, " > ")
			if len(parts) == 2 {
				content := strings.TrimPrefix(parts[0], "echo ")
				filename := strings.TrimSpace(parts[1])

				// Create touch + edit + content + :wq sequence
				result = append(result, "touch "+filename)
				// Note: The edit sequence will be handled by special input processing
				result = append(result, "edit_with_content:"+filename+":"+content)
			}
		} else {
			result = append(result, cmd)
		}
	}

	return result
}

// ExecuteCommands executes multiple commands in sequence
func (fbt *FileBasedTerminal) ExecuteCommands(commands []string, timeout time.Duration) ([]string, error) {
	// Transform commands based on variant capabilities
	variantName := filepath.Base(strings.TrimSuffix(fbt.ExecutablePath, ".exe"))
	transformedCommands := transformCommandsForVariant(variantName, commands)

	// Generate input with special handling for edit commands
	input := fbt.generateInputForCommands(transformedCommands)

	// Clean output file
	os.Remove(fbt.OutputFile)

	// Get absolute paths
	absExec, _ := filepath.Abs(fbt.ExecutablePath)
	absOutput, _ := filepath.Abs(fbt.OutputFile)

	// Run terminal with piped input
	cmd := exec.Command(absExec)
	cmd.Dir = fbt.WorkDir

	// Set up input as string reader (pipe)
	cmd.Stdin = strings.NewReader(input)

	// Set up output to file
	outputFile, err := os.Create(absOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()
	cmd.Stdout = outputFile
	cmd.Stderr = outputFile

	// Run command with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		// Read output file regardless of success/failure
		output, readErr := os.ReadFile(absOutput)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read output: %v", readErr)
		}

		// Parse output into individual command responses
		results := fbt.parseMultiCommandOutput(string(output), len(commands))

		if err != nil {
			return results, fmt.Errorf("command failed: %v", err)
		}

		return results, nil

	case <-time.After(timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}

		// Try to read partial output
		if output, err := os.ReadFile(absOutput); err == nil {
			results := fbt.parseMultiCommandOutput(string(output), len(commands))
			return results, fmt.Errorf("timeout after %v", timeout)
		}
		return make([]string, len(commands)), fmt.Errorf("timeout after %v", timeout)
	}
}

// generateInputForCommands generates input string with special handling for edit commands
func (fbt *FileBasedTerminal) generateInputForCommands(commands []string) string {
	var inputLines []string

	for _, cmd := range commands {
		if strings.HasPrefix(cmd, "edit_with_content:") {
			// Parse edit_with_content:filename:content
			parts := strings.SplitN(cmd, ":", 3)
			if len(parts) == 3 {
				filename := parts[1]
				content := parts[2]

				// Generate edit sequence based on variant
				variantName := filepath.Base(strings.TrimSuffix(fbt.ExecutablePath, ".exe"))
				editSequence := fbt.generateEditSequence(variantName, filename, content)
				inputLines = append(inputLines, editSequence...)
			}
		} else {
			inputLines = append(inputLines, cmd)
		}
	}

	// Add exit command
	inputLines = append(inputLines, "exit")

	return strings.Join(inputLines, "\n") + "\n"
}

// generateEditSequence generates the appropriate edit command sequence for each variant
func (fbt *FileBasedTerminal) generateEditSequence(variantName, filename, content string) []string {
	switch variantName {
	case "sonoma-dusk-alpha":
		return []string{
			"edit " + filename,
			content, // Add the content line
			"",      // Empty line to continue in dusk1's editor
			":wq",   // Save and quit
		}
	case "sonoma-sky-alpha":
		return []string{
			"edit " + filename,
			content, // Sky appends lines directly
			":wq",   // Save and quit
		}
	default:
		// For other variants, just return the edit command
		return []string{"edit " + filename}
	}
}

// isCommandEcho checks if a string looks like a command echo rather than actual output
func isCommandEcho(s string) bool {
	// Commands typically start with common command names
	commonCommands := []string{"touch", "ls", "mkdir", "cd", "pwd", "cat", "echo", "rm", "rmdir", "cp", "mv", "grep", "find", "chmod", "chown"}

	// Split by spaces to get potential command name
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return false
	}

	firstWord := parts[0]
	for _, cmd := range commonCommands {
		if firstWord == cmd {
			return true
		}
	}

	return false
}

// parseMultiCommandOutput parses the output of multiple commands with variant-specific handling
func (fbt *FileBasedTerminal) parseMultiCommandOutput(output string, numCommands int) []string {
	variantName := filepath.Base(strings.TrimSuffix(fbt.ExecutablePath, ".exe"))

	// Use variant-specific parsing
	switch variantName {
	case "glm-4.5":
		return fbt.parseGlmOutput(output, numCommands)
	case "sonoma-dusk-alpha":
		return fbt.parseDusk1Output(output, numCommands)
	case "groke-code-fast-1":
		return fbt.parseFastOutput(output, numCommands)
	case "sonoma-sky-alpha":
		return fbt.parseSkyOutput(output, numCommands)
	default:
		return fbt.parseDefaultOutput(output, numCommands)
	}
}

// parseGlmOutput handles GLM's specific output format
func (fbt *FileBasedTerminal) parseGlmOutput(output string, numCommands int) []string {
	// GLM concatenates output with prompts - this should fail naturally to indicate
	// that GLM needs to be fixed at the source code level to output on separate lines
	return fbt.parseDefaultOutput(output, numCommands)
}

// parseDefaultOutput handles standard output format
func (fbt *FileBasedTerminal) parseDefaultOutput(output string, numCommands int) []string {
	// Split raw output into lines first without cleaning
	lines := strings.Split(output, "\n")

	// Process each line and separate prompts from actual output
	var actualOutput []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip startup messages and terminal noise
		if strings.Contains(line, "Welcome to") ||
			strings.Contains(line, "Type 'help'") ||
			strings.Contains(line, "Terminal Emulator") ||
			strings.Contains(line, "session ended") ||
			strings.Contains(line, "available commands") ||
			strings.Contains(line, "Virtual Terminal") {
			continue
		}

		// Skip prompts - identify by pattern: "/home/user$" with no spaces after $
		if strings.HasSuffix(line, "$") && strings.Contains(line, "/home/user") && !strings.Contains(strings.TrimPrefix(line, "/home/user"), " ") {
			continue // This is just a prompt
		}

		// Handle lines that contain prompts followed by actual output
		if strings.Contains(line, "/home/user$") {
			// Split by prompt and extract the last non-empty part as actual output
			parts := strings.Split(line, "/home/user$")
			var extractedOutput string
			for i := len(parts) - 1; i >= 0; i-- {
				trimmed := strings.TrimSpace(parts[i])
				if trimmed != "" {
					extractedOutput = trimmed
					break
				}
			}
			// Only add to output if it's not a command echo
			// Command echoes typically contain command names like "touch", "ls", "mkdir", etc.
			if extractedOutput != "" && !isCommandEcho(extractedOutput) {
				actualOutput = append(actualOutput, extractedOutput)
			}
		} else {
			// This is actual command output without prompts
			actualOutput = append(actualOutput, line)
		}
	}

	return fbt.distributeOutput(actualOutput, numCommands)
}

// parseDusk1Output handles dusk1-specific output format
func (fbt *FileBasedTerminal) parseDusk1Output(output string, numCommands int) []string {
	return fbt.parseDefaultOutput(output, numCommands)
}

// parseFastOutput handles fast-specific output format
func (fbt *FileBasedTerminal) parseFastOutput(output string, numCommands int) []string {
	return fbt.parseDefaultOutput(output, numCommands)
}

// parseSkyOutput handles sky-specific output format
func (fbt *FileBasedTerminal) parseSkyOutput(output string, numCommands int) []string {
	return fbt.parseDefaultOutput(output, numCommands)
}

// distributeOutput distributes parsed output across commands
func (fbt *FileBasedTerminal) distributeOutput(actualOutput []string, numCommands int) []string {
	results := make([]string, numCommands)

	// Distribute the actual output across commands
	// For simple cases where we have one output per command
	if len(actualOutput) == numCommands {
		for i := 0; i < numCommands && i < len(actualOutput); i++ {
			results[i] = actualOutput[i]
		}
	} else if len(actualOutput) == 1 && numCommands > 1 {
		// If we have one output but multiple commands, assume it belongs to the last command
		// This handles cases like: touch file.txt; ls -> "file.txt" output belongs to ls
		results[numCommands-1] = actualOutput[0]
	} else {
		// Default distribution
		for i := 0; i < numCommands && i < len(actualOutput); i++ {
			results[i] = actualOutput[i]
		}
	}

	return results
}

// cleanOutput cleans terminal output
func (fbt *FileBasedTerminal) cleanOutput(output string) string {
	lines := strings.Split(output, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and terminal noise
		if line == "" ||
			strings.Contains(line, "Terminal Emulator") ||
			strings.Contains(line, "Type 'help'") ||
			strings.Contains(line, "Welcome to") ||
			strings.Contains(line, "Goodbye") ||
			strings.Contains(line, "session ended") ||
			strings.Contains(line, "available commands") ||
			strings.Contains(line, "Virtual Terminal") ||
			strings.HasSuffix(line, "$ ") ||
			strings.HasSuffix(line, "$") ||
			(strings.Contains(line, "/home/user") && strings.Contains(line, "$") && !strings.Contains(line, " ")) {
			continue
		}

		cleanLines = append(cleanLines, line)
	}

	return strings.Join(cleanLines, "\n")
}

// Close cleans up temporary files
func (fbt *FileBasedTerminal) Close() error {
	// Clean up temp files
	os.Remove(fbt.InputFile)
	os.Remove(fbt.OutputFile)
	os.Remove(fbt.WorkDir)
	return nil
}

// RunFileBasedTest runs a test case using file-based communication
func RunFileBasedTest(executablePath string, testCase TestCase) TestResult {
	result := TestResult{
		TestCase:  testCase,
		Variant:   filepath.Base(strings.TrimSuffix(executablePath, ".exe")),
		Timestamp: time.Now(),
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// Create file-based terminal
	fbt, err := NewFileBasedTerminal(executablePath)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create file-based terminal: %v", err)
		result.Passed = false
		return result
	}
	defer fbt.Close()

	// Execute setup commands
	for _, setupCmd := range testCase.Setup {
		_, err := fbt.ExecuteCommand(setupCmd, testCase.Timeout)
		if err != nil && !strings.Contains(err.Error(), "timeout") {
			result.Error = fmt.Sprintf("Setup command failed: %v", err)
			result.Passed = false
			return result
		}
	}

	// Execute test commands
	if len(testCase.Commands) == 1 {
		// Single command
		output, err := fbt.ExecuteCommand(testCase.Commands[0], testCase.Timeout)
		if err != nil && !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "command failed") {
			result.Error = err.Error()
			result.Passed = false
			return result
		}
		result.Output = []string{output}
	} else {
		// Multiple commands
		outputs, err := fbt.ExecuteCommands(testCase.Commands, testCase.Timeout)
		if err != nil && !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "command failed") {
			result.Error = err.Error()
			result.Passed = false
			return result
		}
		result.Output = outputs
	}

	result.Expected = testCase.Expected

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
				break // Stop at first validation failure
			}
		}
	}

	// Execute cleanup commands (ignore errors)
	for _, cleanupCmd := range testCase.Cleanup {
		fbt.ExecuteCommand(cleanupCmd, testCase.Timeout)
	}

	return result
}

func main() {
	fmt.Printf(" Terminal Emulator Test Suite (File-Based)\n")

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("[ERROR] Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	variants := config.Variants.Names
	fmt.Printf("Found %d variants to test: %v\n\n", len(variants), variants)

	// Create temp and reports directories
	os.MkdirAll(config.Paths.TempDir, 0755)
	os.MkdirAll(config.Paths.ReportsDir, 0755)
	defer os.RemoveAll(config.Paths.TempDir) // Clean up temp dir at the end

	testSuite := GetAllTestCases(config.GetTimeout())
	var allResults []VariantResults

	// Test each variant
	for _, variantPath := range variants {
		variantName := filepath.Base(variantPath)
		startTime := time.Now()

		result := VariantResults{
			Name:        variantName,
			TestResults: []TestResult{},
		}

		// Use pre-built executable from bin directory
		executablePath := filepath.Join(config.Paths.BinDir, variantName+".exe")
		absExecPath, _ := filepath.Abs(executablePath)
		if _, err := os.Stat(absExecPath); err != nil {
			result.BuildSuccess = false
			result.BuildError = fmt.Sprintf("Executable not found: %s (abs: %s)", executablePath, absExecPath)
			color.Red("[ERROR] Executable not found for %s: %s\n", variantName, absExecPath)
			allResults = append(allResults, result)
			continue
		}

		result.BuildSuccess = true
		color.Green("[OK] Found executable for %s\n", variantName)

		// Run a subset of tests (first few from each category for demo)
		categories := []struct {
			name  string
			tests []TestCase
		}{
			{"Navigation", testSuite.Navigation[:min(3, len(testSuite.Navigation))]},
			{"File Operations", testSuite.FileOps[:min(3, len(testSuite.FileOps))]},
			{"Directory Operations", testSuite.DirOps[:min(2, len(testSuite.DirOps))]},
			{"Content Operations", testSuite.Content[:min(2, len(testSuite.Content))]},
			{"System Commands", testSuite.System},
		}

		for _, category := range categories {
			if len(category.tests) == 0 {
				continue
			}

			fmt.Printf("\n* Running %s tests for %s...\n", category.name, variantName)

			for _, testCase := range category.tests {
				testResult := RunFileBasedTest(absExecPath, testCase)
				result.TestResults = append(result.TestResults, testResult)
				result.TotalTests++

				if testResult.Passed {
					result.PassedTests++
				} else {
					result.FailedTests++
				}

				LogTestProgress(variantName, testCase, testResult)
			}
		}

		result.TotalDuration = time.Since(startTime)
		if result.TotalTests > 0 {
			result.PassRate = float64(result.PassedTests) / float64(result.TotalTests) * 100
		}
		allResults = append(allResults, result)
	}

	// Generate summary
	summary := CalculateSummary(allResults)

	// Print summary
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf(" TEST SUMMARY (FILE-BASED)\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n\n")

	// Generate HTML report
	fmt.Printf(" Generating HTML report...\n")
	reportPath := filepath.Join(config.Paths.ReportsDir, "test_report.html")
	reportErr := GenerateHTMLReport(summary, reportPath)
	if reportErr != nil {
		color.Red("[ERROR] Failed to generate HTML report: %v\n", reportErr)
	} else {
		color.Green("[OK] HTML report generated successfully!\n")
	}

	fmt.Printf(" Open %s in your browser to view the detailed report\n", reportPath)
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
