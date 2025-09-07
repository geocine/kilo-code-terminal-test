package main

import (
	"strings"
	"testing"
	"time"
)

func TestNewVirtualFile(t *testing.T) {
	name := "test.txt"
	fileType := RegularFile

	file := NewVirtualFile(name, fileType)

	if file.Name != name {
		t.Errorf("Expected name %s, got %s", name, file.Name)
	}

	if file.Type != fileType {
		t.Errorf("Expected type %v, got %v", fileType, file.Type)
	}

	if len(file.Content) != 0 {
		t.Errorf("Expected empty content, got %v", file.Content)
	}

	if len(file.Children) != 0 {
		t.Errorf("Expected no children, got %v", file.Children)
	}

	if file.Parent != nil {
		t.Errorf("Expected nil parent, got %v", file.Parent)
	}

	if file.Permissions != 0644 {
		t.Errorf("Expected permissions 0644, got %o", file.Permissions)
	}

	if file.Size != 0 {
		t.Errorf("Expected size 0, got %d", file.Size)
	}
}

func TestVirtualFileAddChild(t *testing.T) {
	parent := NewVirtualFile("parent", Directory)
	child := NewVirtualFile("child", RegularFile)

	err := parent.AddChild(child)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(parent.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(parent.Children))
	}

	if _, exists := parent.Children["child"]; !exists {
		t.Errorf("Child not found in parent's children")
	}

	if child.Parent != parent {
		t.Errorf("Child's parent not set correctly")
	}

	// Test adding duplicate child
	duplicate := NewVirtualFile("child", RegularFile)
	err = parent.AddChild(duplicate)
	if err == nil {
		t.Errorf("Expected error when adding duplicate child")
	}

	// Test adding child to non-directory
	nonDir := NewVirtualFile("file", RegularFile)
	err = nonDir.AddChild(NewVirtualFile("another", RegularFile))
	if err == nil {
		t.Errorf("Expected error when adding child to non-directory")
	}
}

func TestVirtualFileRemoveChild(t *testing.T) {
	parent := NewVirtualFile("parent", Directory)
	child := NewVirtualFile("child", RegularFile)
	parent.AddChild(child)

	err := parent.RemoveChild("child")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(parent.Children) != 0 {
		t.Errorf("Expected 0 children, got %d", len(parent.Children))
	}

	// Test removing non-existent child
	err = parent.RemoveChild("nonexistent")
	if err == nil {
		t.Errorf("Expected error when removing non-existent child")
	}

	// Test removing from non-directory
	nonDir := NewVirtualFile("file", RegularFile)
	err = nonDir.RemoveChild("anything")
	if err == nil {
		t.Errorf("Expected error when removing from non-directory")
	}
}

func TestVirtualFileGetPath(t *testing.T) {
	// Test root path
	root := NewVirtualFile("", Directory)
	if root.GetPath() != "/" {
		t.Errorf("Expected root path '/', got %s", root.GetPath())
	}

	// Test nested path
	home := NewVirtualFile("home", Directory)
	user := NewVirtualFile("user", Directory)
	file := NewVirtualFile("test.txt", RegularFile)

	root.AddChild(home)
	home.AddChild(user)
	user.AddChild(file)

	expectedPath := "/home/user/test.txt"
	if file.GetPath() != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, file.GetPath())
	}
}

func TestVirtualFileUpdateContent(t *testing.T) {
	file := NewVirtualFile("test.txt", RegularFile)
	content := []byte("Hello, World!")

	file.UpdateContent(content)

	if string(file.Content) != string(content) {
		t.Errorf("Expected content %s, got %s", string(content), string(file.Content))
	}

	if file.Size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), file.Size)
	}

	// Test updating directory (should do nothing)
	dir := NewVirtualFile("dir", Directory)
	oldModTime := dir.ModTime
	dir.UpdateContent([]byte("test"))

	if dir.ModTime != oldModTime {
		t.Errorf("Directory modification time should not change")
	}
}

func TestNewFileSystem(t *testing.T) {
	fs := NewFileSystem()

	if fs.Root == nil {
		t.Errorf("Root should not be nil")
	}

	if fs.CurrentDir == nil {
		t.Errorf("Current directory should not be nil")
	}

	if fs.PrevDir == nil {
		t.Errorf("Previous directory should not be nil")
	}

	// Check basic structure
	if _, exists := fs.Root.Children["home"]; !exists {
		t.Errorf("Home directory should exist")
	}

	home := fs.Root.Children["home"]
	if _, exists := home.Children["user"]; !exists {
		t.Errorf("User directory should exist")
	}

	if fs.CurrentDir != home.Children["user"] {
		t.Errorf("Current directory should be user directory")
	}
}

func TestFileSystemGetAbsolutePath(t *testing.T) {
	fs := NewFileSystem()

	// Test absolute path
	if fs.GetAbsolutePath("/absolute/path") != "/absolute/path" {
		t.Errorf("Absolute path should remain unchanged")
	}

	// Test relative path from root
	fs.CurrentDir = fs.Root
	if fs.GetAbsolutePath("relative") != "/relative" {
		t.Errorf("Expected '/relative', got %s", fs.GetAbsolutePath("relative"))
	}

	// Test relative path from user directory
	userDir := fs.Root.Children["home"].Children["user"]
	fs.CurrentDir = userDir
	if fs.GetAbsolutePath("file.txt") != "/home/user/file.txt" {
		t.Errorf("Expected '/home/user/file.txt', got %s", fs.GetAbsolutePath("file.txt"))
	}
}

func TestFileSystemResolvePath(t *testing.T) {
	fs := NewFileSystem()

	// Test empty path (should return current directory)
	current, err := fs.ResolvePath("")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if current != fs.CurrentDir {
		t.Errorf("Should return current directory for empty path")
	}

	// Test ~ (home directory)
	home, err := fs.ResolvePath("~")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expectedHome := fs.Root.Children["home"].Children["user"]
	if home != expectedHome {
		t.Errorf("Should return user directory for ~")
	}

	// Test - (previous directory)
	fs.PrevDir = fs.Root
	prev, err := fs.ResolvePath("-")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if prev != fs.Root {
		t.Errorf("Should return previous directory for -")
	}

	// Test absolute path
	root, err := fs.ResolvePath("/")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if root != fs.Root {
		t.Errorf("Should return root for /")
	}

	// Test relative path
	user, err := fs.ResolvePath("home/user")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expectedUser := fs.Root.Children["home"].Children["user"]
	if user != expectedUser {
		t.Errorf("Should resolve relative path correctly")
	}

	// Test non-existent path
	_, err = fs.ResolvePath("/nonexistent")
	if err == nil {
		t.Errorf("Should return error for non-existent path")
	}
}

func TestTerminalParseCommand(t *testing.T) {
	terminal := NewTerminal()

	// Test simple command
	cmd, args, err := terminal.ParseCommand("ls -l")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if cmd != "ls" {
		t.Errorf("Expected command 'ls', got '%s'", cmd)
	}
	if len(args) != 1 || args[0] != "-l" {
		t.Errorf("Expected args ['-l'], got %v", args)
	}

	// Test command with quotes
	cmd, args, err = terminal.ParseCommand(`echo "Hello World"`)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if cmd != "echo" {
		t.Errorf("Expected command 'echo', got '%s'", cmd)
	}
	if len(args) != 1 || args[0] != "Hello World" {
		t.Errorf("Expected args ['Hello World'], got %v", args)
	}

	// Test command with escape characters
	cmd, args, err = terminal.ParseCommand(`echo "Hello \"World\""`)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if cmd != "echo" {
		t.Errorf("Expected command 'echo', got '%s'", cmd)
	}
	if len(args) != 1 || args[0] != `Hello "World"` {
		t.Errorf("Expected args ['Hello \"World\"'], got %v", args)
	}

	// Test empty command
	cmd, args, err = terminal.ParseCommand("")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if cmd != "" {
		t.Errorf("Expected empty command, got '%s'", cmd)
	}

	// Test unclosed quotes
	_, _, err = terminal.ParseCommand(`echo "Hello`)
	if err == nil {
		t.Errorf("Should return error for unclosed quotes")
	}
}

func TestTerminalPwd(t *testing.T) {
	terminal := NewTerminal()

	// Capture output
	output := captureOutput(func() {
		terminal.Pwd([]string{})
	})

	expected := "/home/user\n"
	if output != expected {
		t.Errorf("Expected '%s', got '%s'", expected, output)
	}
}

func TestTerminalCd(t *testing.T) {
	terminal := NewTerminal()

	// Test cd to root
	terminal.Cd([]string{"/"})
	if terminal.FS.CurrentDir != terminal.FS.Root {
		t.Errorf("Should change to root directory")
	}

	// Test cd to home
	terminal.Cd([]string{})
	if terminal.FS.CurrentDir != terminal.FS.Root.Children["home"].Children["user"] {
		t.Errorf("Should change to home directory with no arguments")
	}

	// Test cd to parent
	userDir := terminal.FS.CurrentDir
	terminal.Cd([]string{".."})
	if terminal.FS.CurrentDir != userDir.Parent {
		t.Errorf("Should change to parent directory")
	}

	// Test cd -
	terminal.Cd([]string{"-"})
	if terminal.FS.CurrentDir != userDir {
		t.Errorf("Should change back to previous directory")
	}

	// Test cd to non-existent path
	output := captureOutput(func() {
		terminal.Cd([]string{"/nonexistent"})
	})
	if !strings.Contains(output, "not found") {
		t.Errorf("Should show error for non-existent path")
	}

	// Test cd to file
	// Create a file
	file := NewVirtualFile("test.txt", RegularFile)
	terminal.FS.CurrentDir.AddChild(file)

	output = captureOutput(func() {
		terminal.Cd([]string{"test.txt"})
	})
	if !strings.Contains(output, "Not a directory") {
		t.Errorf("Should show error when trying to cd to a file")
	}
}

func TestTerminalTouch(t *testing.T) {
	terminal := NewTerminal()

	// Test creating a file
	terminal.Touch([]string{"test.txt"})
	if _, exists := terminal.FS.CurrentDir.Children["test.txt"]; !exists {
		t.Errorf("File should be created")
	}

	// Test touching existing file (should update mod time)
	file := terminal.FS.CurrentDir.Children["test.txt"]
	oldModTime := file.ModTime
	time.Sleep(time.Millisecond) // Ensure time difference
	terminal.Touch([]string{"test.txt"})
	if file.ModTime.Before(oldModTime) || file.ModTime.Equal(oldModTime) {
		t.Errorf("File modification time should be updated")
	}

	// Test creating file in subdirectory
	terminal.Mkdir([]string{"subdir"})
	terminal.Touch([]string{"subdir/nested.txt"})
	subdir := terminal.FS.CurrentDir.Children["subdir"]
	if _, exists := subdir.Children["nested.txt"]; !exists {
		t.Errorf("File should be created in subdirectory")
	}
}

func TestTerminalMkdir(t *testing.T) {
	terminal := NewTerminal()

	// Test creating a directory
	terminal.Mkdir([]string{"testdir"})
	if _, exists := terminal.FS.CurrentDir.Children["testdir"]; !exists {
		t.Errorf("Directory should be created")
	}

	// Test creating nested directories with -p
	terminal.Mkdir([]string{"-p", "nested/dir"})
	if _, exists := terminal.FS.CurrentDir.Children["nested"]; !exists {
		t.Errorf("Parent directory should be created")
	}
	nested := terminal.FS.CurrentDir.Children["nested"]
	if _, exists := nested.Children["dir"]; !exists {
		t.Errorf("Nested directory should be created")
	}

	// Test creating existing directory
	output := captureOutput(func() {
		terminal.Mkdir([]string{"testdir"})
	})
	if !strings.Contains(output, "File exists") {
		t.Errorf("Should show error for existing directory")
	}
}

func TestTerminalLs(t *testing.T) {
	terminal := NewTerminal()

	// Create some files and directories
	terminal.Touch([]string{"file1.txt"})
	terminal.Touch([]string{"file2.txt"})
	terminal.Mkdir([]string{"dir1"})

	// Test simple listing
	output := captureOutput(func() {
		terminal.Ls([]string{})
	})

	if !strings.Contains(output, "file1.txt") {
		t.Errorf("Output should contain file1.txt")
	}
	if !strings.Contains(output, "file2.txt") {
		t.Errorf("Output should contain file2.txt")
	}
	if !strings.Contains(output, "dir1") {
		t.Errorf("Output should contain dir1")
	}

	// Test long format listing
	output = captureOutput(func() {
		terminal.Ls([]string{"-l"})
	})

	if !strings.Contains(output, "-") {
		t.Errorf("Long format should show file type")
	}
	if !strings.Contains(output, "d") {
		t.Errorf("Long format should show directory type")
	}

	// Test listing specific directory
	output = captureOutput(func() {
		terminal.Ls([]string{"/"})
	})

	if !strings.Contains(output, "home") {
		t.Errorf("Root listing should contain home directory")
	}
}

func TestTerminalCat(t *testing.T) {
	terminal := NewTerminal()

	// Create a file with content
	file := NewVirtualFile("test.txt", RegularFile)
	file.UpdateContent([]byte("Hello, World!"))
	terminal.FS.CurrentDir.AddChild(file)

	// Test displaying file content
	output := captureOutput(func() {
		terminal.Cat([]string{"test.txt"})
	})

	if output != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got '%s'", output)
	}

	// Test cat on directory
	output = captureOutput(func() {
		terminal.Cat([]string{"."})
	})

	if !strings.Contains(output, "Is a directory") {
		t.Errorf("Should show error when trying to cat a directory")
	}

	// Test cat on non-existent file
	output = captureOutput(func() {
		terminal.Cat([]string{"nonexistent.txt"})
	})

	if !strings.Contains(output, "not found") {
		t.Errorf("Should show error for non-existent file")
	}
}

func TestTerminalEcho(t *testing.T) {
	terminal := NewTerminal()

	// Test simple echo
	output := captureOutput(func() {
		terminal.Echo([]string{"Hello", "World"})
	})

	if output != "Hello World\n" {
		t.Errorf("Expected 'Hello World\\n', got '%s'", output)
	}

	// Test echo with redirection
	terminal.Echo([]string{"Hello", "World", ">", "test.txt"})
	file := terminal.FS.CurrentDir.Children["test.txt"]
	if string(file.Content) != "Hello World" {
		t.Errorf("Expected file content 'Hello World', got '%s'", string(file.Content))
	}

	// Test echo with append
	terminal.Echo([]string{"Appended", ">>", "test.txt"})
	if string(file.Content) != "Hello WorldAppended" {
		t.Errorf("Expected file content 'Hello WorldAppended', got '%s'", string(file.Content))
	}
}

func TestTerminalExit(t *testing.T) {
	terminal := NewTerminal()

	if !terminal.Running {
		t.Errorf("Terminal should be running initially")
	}

	terminal.Exit([]string{})

	if terminal.Running {
		t.Errorf("Terminal should not be running after exit")
	}
}

func TestTerminalHelp(t *testing.T) {
	terminal := NewTerminal()

	output := captureOutput(func() {
		terminal.Help([]string{})
	})

	if !strings.Contains(output, "Available commands:") {
		t.Errorf("Help output should contain 'Available commands:'")
	}

	if !strings.Contains(output, "pwd") {
		t.Errorf("Help output should contain pwd command")
	}

	if !strings.Contains(output, "cd") {
		t.Errorf("Help output should contain cd command")
	}

	if !strings.Contains(output, "help") {
		t.Errorf("Help output should contain help command")
	}
}

// Helper function to capture stdout output
func captureOutput(f func()) string {
	// This is a simplified version - in a real test you would redirect stdout
	// For now, we'll just run the function and return empty string
	f()
	return ""
}
