package main

import (
	"strings"
	"time"
)

// ValidationMode defines how to validate test output
type ValidationMode int

const (
	ExactMatch ValidationMode = iota
	Contains
	RegexMatch
	NoError
	HasError
)

// TestCase represents a single test case
type TestCase struct {
	ID          string
	Category    string
	Description string
	Commands    []string
	Expected    []string
	Validation  []ValidationMode
	Setup       []string // Commands to run before test
	Cleanup     []string // Commands to run after test
	Timeout     time.Duration
}

// TestSuite contains all test cases organized by category
type TestSuite struct {
	Navigation  []TestCase
	FileOps     []TestCase
	DirOps      []TestCase
	Content     []TestCase
	System      []TestCase
	EdgeCases   []TestCase
	Integration []TestCase
	Performance []TestCase
}

// GetAllTestCases returns a comprehensive list of all test cases with configurable timeout
func GetAllTestCases(timeout time.Duration) TestSuite {
	return TestSuite{
		Navigation:  getNavigationTests(timeout),
		FileOps:     getFileOpTests(timeout),
		DirOps:      getDirOpTests(timeout),
		Content:     getContentTests(timeout),
		System:      getSystemTests(timeout),
		EdgeCases:   getEdgeCaseTests(timeout),
		Integration: getIntegrationTests(timeout),
		Performance: getPerformanceTests(timeout),
	}
}

// Navigation Tests (pwd, cd)
func getNavigationTests(timeout time.Duration) []TestCase {
	return []TestCase{
		{
			ID:          "1.1.1",
			Category:    "Navigation",
			Description: "Initial pwd",
			Commands:    []string{"pwd"},
			Expected:    []string{"/home/user"},
			Validation:  []ValidationMode{Contains},
			Timeout:     timeout,
		},
		{
			ID:          "1.1.2",
			Category:    "Navigation",
			Description: "pwd after cd",
			Commands:    []string{"cd /home", "pwd"},
			Expected:    []string{"", "/home"},
			Validation:  []ValidationMode{NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "1.1.3",
			Category:    "Navigation",
			Description: "pwd in known directory",
			Commands:    []string{"cd /home", "pwd"},
			Expected:    []string{"", "/home"},
			Validation:  []ValidationMode{NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "1.2.1",
			Category:    "Navigation",
			Description: "cd absolute path",
			Commands:    []string{"cd /home", "pwd"},
			Expected:    []string{"", "/home"},
			Validation:  []ValidationMode{NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "1.2.2",
			Category:    "Navigation",
			Description: "cd relative path",
			Commands:    []string{"cd /", "cd home", "pwd"},
			Expected:    []string{"", "", "/home"},
			Validation:  []ValidationMode{NoError, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "1.2.3",
			Category:    "Navigation",
			Description: "cd with ..",
			Commands:    []string{"cd /home/user", "cd ..", "pwd"},
			Expected:    []string{"", "", "/home"},
			Validation:  []ValidationMode{NoError, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "1.2.4",
			Category:    "Navigation",
			Description: "cd multiple ..",
			Commands:    []string{"cd /home/user", "cd ../..", "pwd"},
			Expected:    []string{"", "", "/"},
			Validation:  []ValidationMode{NoError, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "1.2.5",
			Category:    "Navigation",
			Description: "cd to home (~)",
			Commands:    []string{"cd /home", "cd ~", "pwd"},
			Expected:    []string{"", "", "/home/user"},
			Validation:  []ValidationMode{NoError, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "1.2.6",
			Category:    "Navigation",
			Description: "cd previous (-)",
			Commands:    []string{"cd /home", "cd /home/user", "cd -", "pwd"},
			Expected:    []string{"", "", "", "/home"},
			Validation:  []ValidationMode{NoError, NoError, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "1.2.7",
			Category:    "Navigation",
			Description: "cd non-existent",
			Commands:    []string{"cd /nonexistent"},
			Expected:    []string{"not found"},
			Validation:  []ValidationMode{HasError},
			Timeout:     timeout,
		},
		{
			ID:          "1.2.8",
			Category:    "Navigation",
			Description: "cd no args",
			Commands:    []string{"cd /home", "cd", "pwd"},
			Expected:    []string{"", "", "/home/user"},
			Validation:  []ValidationMode{NoError, NoError, Contains},
			Timeout:     timeout,
		},
	}
}

// File Operations Tests (touch, rm, cp, mv)
func getFileOpTests(timeout time.Duration) []TestCase {
	return []TestCase{
		{
			ID:          "2.1.1",
			Category:    "File Ops",
			Description: "Create new file",
			Commands:    []string{"touch new.txt", "ls"},
			Expected:    []string{"", "new.txt"},
			Validation:  []ValidationMode{NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "2.1.2",
			Category:    "File Ops",
			Description: "Touch existing",
			Commands:    []string{"touch file.txt", "touch file.txt", "ls"},
			Expected:    []string{"", "", "file.txt"},
			Validation:  []ValidationMode{NoError, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "2.1.3",
			Category:    "File Ops",
			Description: "Create with path",
			Commands:    []string{"mkdir dir", "touch dir/file.txt", "ls dir"},
			Expected:    []string{"", "", "file.txt"},
			Validation:  []ValidationMode{NoError, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "2.1.4",
			Category:    "File Ops",
			Description: "Touch invalid path",
			Commands:    []string{"touch /nonexistent/file.txt"},
			Expected:    []string{"not found"},
			Validation:  []ValidationMode{HasError},
			Timeout:     timeout,
		},
		{
			ID:          "2.2.1",
			Category:    "File Ops",
			Description: "Remove file",
			Commands:    []string{"touch test.txt", "ls", "rm test.txt", "ls"},
			Expected:    []string{"", "test.txt", "", ""},
			Validation:  []ValidationMode{NoError, Contains, NoError, NoError},
			Timeout:     timeout,
		},
		{
			ID:          "2.2.2",
			Category:    "File Ops",
			Description: "Remove non-existent",
			Commands:    []string{"rm nonexistent.txt"},
			Expected:    []string{"not found"},
			Validation:  []ValidationMode{HasError},
			Timeout:     timeout,
		},
		{
			ID:          "2.2.3",
			Category:    "File Ops",
			Description: "Remove dir no flag",
			Commands:    []string{"mkdir dir", "rm dir"},
			Expected:    []string{"", "directory"},
			Validation:  []ValidationMode{NoError, HasError},
			Timeout:     timeout,
		},
		{
			ID:          "2.2.4",
			Category:    "File Ops",
			Description: "Remove dir with -r",
			Commands:    []string{"mkdir dir", "ls", "rm -r dir", "ls"},
			Expected:    []string{"", "dir", "", ""},
			Validation:  []ValidationMode{NoError, Contains, NoError, NoError},
			Timeout:     timeout,
		},
		{
			ID:          "2.3.1",
			Category:    "File Ops",
			Description: "Copy file",
			Commands:    []string{"echo test > src.txt", "cp src.txt dst.txt", "cat dst.txt"},
			Expected:    []string{"", "", "test"},
			Validation:  []ValidationMode{NoError, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "2.3.2",
			Category:    "File Ops",
			Description: "Copy to dir",
			Commands:    []string{"touch file.txt", "mkdir dir", "cp file.txt dir/", "ls dir"},
			Expected:    []string{"", "", "", "file.txt"},
			Validation:  []ValidationMode{NoError, NoError, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "2.3.3",
			Category:    "File Ops",
			Description: "Copy non-existent",
			Commands:    []string{"cp nofile.txt dst.txt"},
			Expected:    []string{"not found"},
			Validation:  []ValidationMode{HasError},
			Timeout:     timeout,
		},
		{
			ID:          "2.3.4",
			Category:    "File Ops",
			Description: "Copy overwrite",
			Commands:    []string{"echo content2 > file2.txt", "echo content1 > file1.txt", "cp file1.txt file2.txt", "cat file2.txt"},
			Expected:    []string{"", "", "", "content1"},
			Validation:  []ValidationMode{NoError, NoError, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "2.3.5",
			Category:    "File Ops",
			Description: "Copy dir no flag",
			Commands:    []string{"mkdir dir", "cp dir dir2"},
			Expected:    []string{"", "directory"},
			Validation:  []ValidationMode{NoError, HasError},
			Timeout:     timeout,
		},
		{
			ID:          "2.3.6",
			Category:    "File Ops",
			Description: "Copy dir with -r",
			Commands:    []string{"mkdir dir", "ls", "cp -r dir dir2", "ls"},
			Expected:    []string{"", "dir", "", "dir2"},
			Validation:  []ValidationMode{NoError, Contains, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "2.4.1",
			Category:    "File Ops",
			Description: "Rename file",
			Commands:    []string{"touch old.txt", "ls", "mv old.txt new.txt", "ls"},
			Expected:    []string{"", "old.txt", "", "new.txt"},
			Validation:  []ValidationMode{NoError, Contains, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "2.4.2",
			Category:    "File Ops",
			Description: "Move to dir",
			Commands:    []string{"mkdir dir", "touch f.txt", "ls", "mv f.txt dir/", "ls dir"},
			Expected:    []string{"", "", "f.txt", "", "f.txt"},
			Validation:  []ValidationMode{NoError, NoError, Contains, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "2.4.3",
			Category:    "File Ops",
			Description: "Move non-existent",
			Commands:    []string{"mv nofile.txt dst.txt"},
			Expected:    []string{"not found"},
			Validation:  []ValidationMode{HasError},
			Timeout:     timeout,
		},
		{
			ID:          "2.4.4",
			Category:    "File Ops",
			Description: "Move directory",
			Commands:    []string{"mkdir dir1", "ls", "mv dir1 dir2", "ls"},
			Expected:    []string{"", "dir1", "", "dir2"},
			Validation:  []ValidationMode{NoError, Contains, NoError, Contains},
			Timeout:     timeout,
		},
	}
}

// Directory Operations Tests (mkdir, rmdir, ls)
func getDirOpTests(timeout time.Duration) []TestCase {
	return []TestCase{
		{
			ID:          "3.1.1",
			Category:    "Dir Ops",
			Description: "Create directory",
			Commands:    []string{"mkdir testdir", "ls"},
			Expected:    []string{"", "testdir"},
			Validation:  []ValidationMode{NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "3.1.2",
			Category:    "Dir Ops", 
			Description: "mkdir no parent",
			Commands:    []string{"mkdir parent/child"},
			Expected:    []string{"cannot create directory|not found|No such file"},
			Validation:  []ValidationMode{Contains},
			Timeout:     timeout,
		},
		{
			ID:          "3.1.3",
			Category:    "Dir Ops",
			Description: "mkdir with -p (if supported)",
			Commands:    []string{"mkdir -p a/b", "ls"},
			Expected:    []string{"", "a"},
			Validation:  []ValidationMode{NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "3.1.4",
			Category:    "Dir Ops",
			Description: "mkdir existing",
			Commands:    []string{"mkdir dir", "mkdir dir"},
			Expected:    []string{"", "exists"},
			Validation:  []ValidationMode{NoError, HasError},
			Timeout:     timeout,
		},
		{
			ID:          "3.2.1",
			Category:    "Dir Ops",
			Description: "Remove empty dir",
			Commands:    []string{"mkdir empty", "ls", "rmdir empty", "ls"},
			Expected:    []string{"", "empty", "", ""},
			Validation:  []ValidationMode{NoError, Contains, NoError, NoError},
			Timeout:     timeout,
		},
		{
			ID:          "3.2.2",
			Category:    "Dir Ops",
			Description: "rmdir non-empty",
			Commands:    []string{"mkdir dir", "touch dir/file", "rmdir dir"},
			Expected:    []string{"", "", "not empty"},
			Validation:  []ValidationMode{NoError, NoError, HasError},
			Timeout:     timeout,
		},
		{
			ID:          "3.2.3",
			Category:    "Dir Ops",
			Description: "rmdir non-existent",
			Commands:    []string{"rmdir nodir"},
			Expected:    []string{"not found"},
			Validation:  []ValidationMode{HasError},
			Timeout:     timeout,
		},
		{
			ID:          "3.3.1",
			Category:    "Dir Ops",
			Description: "List empty dir",
			Commands:    []string{"mkdir empty", "ls", "cd empty", "ls"},
			Expected:    []string{"", "empty", "", ""},
			Validation:  []ValidationMode{NoError, Contains, NoError, NoError},
			Timeout:     timeout,
		},
		{
			ID:          "3.3.2",
			Category:    "Dir Ops",
			Description: "List files/dirs",
			Commands:    []string{"touch f1", "touch f2", "mkdir d1", "mkdir d2", "ls"},
			Expected:    []string{"", "", "", "", "f1"},
			Validation:  []ValidationMode{NoError, NoError, NoError, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "3.3.3",
			Category:    "Dir Ops",
			Description: "List with path",
			Commands:    []string{"mkdir dir", "ls", "touch dir/f", "ls dir"},
			Expected:    []string{"", "dir", "", "f"},
			Validation:  []ValidationMode{NoError, Contains, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "3.3.4",
			Category:    "Dir Ops",
			Description: "List non-existent",
			Commands:    []string{"ls /nodir"},
			Expected:    []string{"not found"},
			Validation:  []ValidationMode{HasError},
			Timeout:     timeout,
		},
		{
			ID:          "3.3.5",
			Category:    "Dir Ops",
			Description: "List with -a",
			Commands:    []string{"touch .hidden", "touch visible", "ls -a"},
			Expected:    []string{"", "", ".hidden"},
			Validation:  []ValidationMode{NoError, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "3.3.6",
			Category:    "Dir Ops",
			Description: "List with -l",
			Commands:    []string{"touch file", "ls -l"},
			Expected:    []string{"", "-"},
			Validation:  []ValidationMode{NoError, Contains},
			Timeout:     timeout,
		},
	}
}

// Content Operations Tests (cat, echo, edit)
func getContentTests(timeout time.Duration) []TestCase {
	return []TestCase{
		{
			ID:          "4.1.1",
			Category:    "Content",
			Description: "Cat single line",
			Commands:    []string{"echo Hello > f.txt", "cat f.txt"},
			Expected:    []string{"", "Hello"},
			Validation:  []ValidationMode{NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "4.1.2",
			Category:    "Content",
			Description: "Cat multi-line",
			Commands:    []string{"echo L1 > f.txt", "echo L2 >> f.txt", "cat f.txt"},
			Expected:    []string{"", "", "L1"},
			Validation:  []ValidationMode{NoError, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "4.1.3",
			Category:    "Content",
			Description: "Cat non-existent",
			Commands:    []string{"cat nofile.txt"},
			Expected:    []string{"not found"},
			Validation:  []ValidationMode{HasError},
			Timeout:     timeout,
		},
		{
			ID:          "4.1.4",
			Category:    "Content",
			Description: "Cat directory",
			Commands:    []string{"mkdir dir", "cat dir"},
			Expected:    []string{"", "directory"},
			Validation:  []ValidationMode{NoError, HasError},
			Timeout:     timeout,
		},
		{
			ID:          "4.2.1",
			Category:    "Content",
			Description: "Echo to file",
			Commands:    []string{"echo Hi > f.txt", "ls", "cat f.txt"},
			Expected:    []string{"", "f.txt", "Hi"},
			Validation:  []ValidationMode{NoError, Contains, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "4.2.2",
			Category:    "Content",
			Description: "Echo overwrite",
			Commands:    []string{"echo Old > f.txt", "cat f.txt", "echo New > f.txt", "cat f.txt"},
			Expected:    []string{"", "Old", "", "New"},
			Validation:  []ValidationMode{NoError, Contains, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "4.2.3",
			Category:    "Content",
			Description: "Echo append",
			Commands:    []string{"echo L1 > f", "cat f", "echo L2 >> f", "cat f"},
			Expected:    []string{"", "L1", "", "L1"},
			Validation:  []ValidationMode{NoError, Contains, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "4.3.1",
			Category:    "Content",
			Description: "Create file with content",
			Commands:    []string{"echo Hello World > newfile.txt", "ls", "cat newfile.txt"},
			Expected:    []string{"", "newfile.txt", "Hello World"},
			Validation:  []ValidationMode{NoError, Contains, Contains},
			Timeout:     timeout,
		},
		// Echo Redirection Tests - Variant Capability Validation
		{
			ID:          "4.4.1",
			Category:    "Content",
			Description: "Echo redirection basic",
			Commands:    []string{"echo test > redirect1.txt", "cat redirect1.txt"},
			Expected:    []string{"", "test"},
			Validation:  []ValidationMode{NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "4.4.2",
			Category:    "Content",
			Description: "Echo redirection special chars",
			Commands:    []string{"echo 'Hello World!' > special.txt", "cat special.txt"},
			Expected:    []string{"", "Hello World!"},
			Validation:  []ValidationMode{NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "4.4.3",
			Category:    "Content",
			Description: "Echo redirection numbers",
			Commands:    []string{"echo 12345 > numbers.txt", "cat numbers.txt"},
			Expected:    []string{"", "12345"},
			Validation:  []ValidationMode{NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "4.4.4",
			Category:    "Content",
			Description: "Echo redirection overwrite",
			Commands:    []string{"echo first > overwrite.txt", "cat overwrite.txt", "echo second > overwrite.txt", "cat overwrite.txt"},
			Expected:    []string{"", "first", "", "second"},
			Validation:  []ValidationMode{NoError, Contains, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "4.4.5",
			Category:    "Content",
			Description: "Echo append basic",
			Commands:    []string{"echo line1 > append.txt", "echo line2 >> append.txt", "cat append.txt"},
			Expected:    []string{"", "", "line1"},
			Validation:  []ValidationMode{NoError, NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "4.4.6",
			Category:    "Content",
			Description: "Echo redirection empty content",
			Commands:    []string{"echo '' > empty.txt", "ls", "cat empty.txt"},
			Expected:    []string{"", "empty.txt", ""},
			Validation:  []ValidationMode{NoError, Contains, NoError},
			Timeout:     timeout,
		},
		{
			ID:          "4.4.7",
			Category:    "Content",
			Description: "Echo redirection multiple files",
			Commands:    []string{"echo file1 > f1.txt", "echo file2 > f2.txt", "ls", "cat f1.txt", "cat f2.txt"},
			Expected:    []string{"", "", "f1.txt", "file1", "file2"},
			Validation:  []ValidationMode{NoError, NoError, Contains, Contains, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "4.4.8",
			Category:    "Content",
			Description: "Echo redirection with spaces in filename",
			Commands:    []string{"echo content > \"file with spaces.txt\"", "ls", "cat \"file with spaces.txt\""},
			Expected:    []string{"", "file with spaces.txt", "content"},
			Validation:  []ValidationMode{NoError, Contains, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "4.4.9",
			Category:    "Content",
			Description: "Echo redirection long content",
			Commands:    []string{"echo This is a very long line of text that should be handled properly > long.txt", "cat long.txt"},
			Expected:    []string{"", "This is a very long line"},
			Validation:  []ValidationMode{NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "4.4.10",
			Category:    "Content",
			Description: "Echo redirection verification",
			Commands:    []string{"echo verification > verify.txt", "ls verify.txt", "cat verify.txt", "rm verify.txt", "ls verify.txt"},
			Expected:    []string{"", "verify.txt", "verification", "", "not found"},
			Validation:  []ValidationMode{NoError, Contains, Contains, NoError, Contains},
			Timeout:     timeout,
		},
	}
}

// System Commands Tests (clear, exit, help)
func getSystemTests(timeout time.Duration) []TestCase {
	return []TestCase{
		{
			ID:          "5.1.1",
			Category:    "System",
			Description: "Clear screen",
			Commands:    []string{"clear"},
			Expected:    []string{""},
			Validation:  []ValidationMode{NoError},
			Timeout:     timeout,
		},
		{
			ID:          "5.2.1",
			Category:    "System",
			Description: "Help command",
			Commands:    []string{"help"},
			Expected:    []string{"Available commands"},
			Validation:  []ValidationMode{Contains},
			Timeout:     timeout,
		},
	}
}

// Edge Cases Tests
func getEdgeCaseTests(timeout time.Duration) []TestCase {
	return []TestCase{
		{
			ID:          "6.1.1",
			Category:    "Edge Cases",
			Description: "Simple filename",
			Commands:    []string{"touch file.txt", "ls"},
			Expected:    []string{"", "file.txt"},
			Validation:  []ValidationMode{NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "6.1.2",
			Category:    "Edge Cases",
			Description: "Create and verify file exists",
			Commands:    []string{"touch test.txt", "ls", "cat test.txt"},
			Expected:    []string{"", "test.txt", ""},
			Validation:  []ValidationMode{NoError, Contains, NoError},
			Timeout:     timeout,
		},
		{
			ID:          "6.2.1",
			Category:    "Edge Cases",
			Description: "Multiple slashes",
			Commands:    []string{"cd //home///user//", "pwd"},
			Expected:    []string{"", "/home/user"},
			Validation:  []ValidationMode{NoError, Contains},
			Timeout:     timeout,
		},
		{
			ID:          "6.2.2",
			Category:    "Edge Cases",
			Description: "Empty path",
			Commands:    []string{"cd \"\""},
			Expected:    []string{""},
			Validation:  []ValidationMode{NoError},
			Timeout:     timeout,
		},
	}
}

// Integration Tests
func getIntegrationTests(timeout time.Duration) []TestCase {
	return []TestCase{
		{
			ID:          "7.1.1",
			Category:    "Integration",
			Description: "Basic structure",
			Commands: []string{
				"mkdir project",
				"ls",
				"touch project/README.md",
				"ls project",
				"cd project",
				"pwd",
			},
			Expected:   []string{"", "project", "", "README.md", "", "/home/user/project"},
			Validation: []ValidationMode{NoError, Contains, NoError, Contains, NoError, Contains},
			Timeout:    timeout,
		},
		{
			ID:          "7.1.2",
			Category:    "Integration",
			Description: "Copy and move",
			Commands: []string{
				"mkdir source",
				"mkdir dest",
				"ls",
				"echo content > source/file.txt",
				"ls source",
				"cp source/file.txt dest/",
				"ls dest",
				"mv dest/file.txt dest/renamed.txt",
				"ls dest",
				"cat dest/renamed.txt",
			},
			Expected:   []string{"", "", "source", "", "file.txt", "", "file.txt", "", "renamed.txt", "content"},
			Validation: []ValidationMode{NoError, NoError, Contains, NoError, Contains, NoError, Contains, NoError, Contains, Contains},
			Timeout:    timeout,
		},
	}
}

// Performance Tests
func getPerformanceTests(timeout time.Duration) []TestCase {
	return []TestCase{
		{
			ID:          "8.1.1",
			Category:    "Performance",
			Description: "Create test directory",
			Commands:    []string{"mkdir test_perf", "ls", "cd test_perf", "pwd"},
			Expected:    []string{"", "test_perf", "", "/home/user/test_perf"},
			Validation:  []ValidationMode{NoError, Contains, NoError, Contains},
			Timeout:     15 * time.Second,
		},
		{
			ID:          "8.1.2",
			Category:    "Performance",
			Description: "Multiple operations",
			Commands:    []string{"mkdir a", "mkdir b", "mkdir c", "ls"},
			Expected:    []string{"", "", "", "a"},
			Validation:  []ValidationMode{NoError, NoError, NoError, Contains},
			Timeout:     15 * time.Second,
		},
	}
}

// ValidateOutput validates the output against expected result using the specified mode
func ValidateOutput(actual, expected string, mode ValidationMode) bool {
	actual = strings.TrimSpace(actual)
	expected = strings.TrimSpace(expected)

	switch mode {
	case ExactMatch:
		return actual == expected
	case Contains:
		if expected == "" {
			return true // Empty expected means any output is valid
		}
		// Support multiple patterns separated by |
		if strings.Contains(expected, "|") {
			patterns := strings.Split(expected, "|")
			for _, pattern := range patterns {
				pattern = strings.TrimSpace(pattern)
				if strings.Contains(strings.ToLower(actual), strings.ToLower(pattern)) {
					return true
				}
			}
			return false
		}
		return strings.Contains(strings.ToLower(actual), strings.ToLower(expected))
	case RegexMatch:
		// Implementation for regex matching would go here
		return strings.Contains(actual, expected)
	case NoError:
		// Check if output doesn't contain common error indicators
		lowerActual := strings.ToLower(actual)
		errorKeywords := []string{"error", "not found", "permission denied", "cannot", "failed"}
		for _, keyword := range errorKeywords {
			if strings.Contains(lowerActual, keyword) {
				return false
			}
		}
		return true
	case HasError:
		// Check if output contains error indicators
		lowerActual := strings.ToLower(actual)
		lowerExpected := strings.ToLower(expected)
		errorKeywords := []string{"error", "not found", "permission denied", "cannot", "failed", "directory", "exists", "not empty"}
		if lowerExpected != "" {
			return strings.Contains(lowerActual, lowerExpected)
		}
		for _, keyword := range errorKeywords {
			if strings.Contains(lowerActual, keyword) {
				return true
			}
		}
		return false
	default:
		return false
	}
}
