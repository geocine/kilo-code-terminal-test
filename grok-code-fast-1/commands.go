package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ExecuteCommand executes a parsed command and returns the result
func (t *Terminal) ExecuteCommand(cmd *ParsedCommand) *CommandResult {
	switch cmd.Command {
	case "pwd":
		return t.cmdPwd(cmd.Args)
	case "cd":
		return t.cmdCd(cmd.Args)
	case "ls":
		return t.cmdLs(cmd.Args)
	case "mkdir":
		return t.cmdMkdir(cmd.Args)
	case "rmdir":
		return t.cmdRmdir(cmd.Args)
	case "touch":
		return t.cmdTouch(cmd.Args)
	case "rm":
		return t.cmdRm(cmd.Args)
	case "cp":
		return t.cmdCp(cmd.Args)
	case "mv":
		return t.cmdMv(cmd.Args)
	case "cat":
		return t.cmdCat(cmd.Args)
	case "echo":
		return t.cmdEcho(cmd.Args)
	case "edit":
		return t.cmdEdit(cmd.Args)
	case "clear":
		return t.cmdClear(cmd.Args)
	case "exit", "quit":
		return t.cmdExit(cmd.Args)
	case "help":
		return t.cmdHelp(cmd.Args)
	case "":
		return &CommandResult{Output: "", Error: nil, Exit: false}
	default:
		return &CommandResult{Output: "", Error: fmt.Errorf("command not found: %s", cmd.Command), Exit: false}
	}
}

// cmdPwd implements the pwd command
func (t *Terminal) cmdPwd(args []string) *CommandResult {
	if len(args) > 0 {
		return &CommandResult{Output: "", Error: fmt.Errorf("pwd: too many arguments"), Exit: false}
	}
	return &CommandResult{Output: t.FS.GetPath(t.FS.CurrentDir), Error: nil, Exit: false}
}

// cmdCd implements the cd command
func (t *Terminal) cmdCd(args []string) *CommandResult {
	var path string
	if len(args) == 0 {
		path = "~"
	} else if len(args) == 1 {
		path = args[0]
	} else {
		return &CommandResult{Output: "", Error: fmt.Errorf("cd: too many arguments"), Exit: false}
	}

	var target *VirtualFile
	var err error

	switch path {
	case "-":
		if t.FS.PrevDir == nil {
			return &CommandResult{Output: "", Error: fmt.Errorf("cd: no previous directory"), Exit: false}
		}
		target = t.FS.PrevDir
	case "~":
		target = t.FS.Root.Children["home"].Children["user"]
	default:
		target, err = t.FS.ResolvePath(path)
		if err != nil {
			return &CommandResult{Output: "", Error: err, Exit: false}
		}
	}

	if target.Type != Directory {
		return &CommandResult{Output: "", Error: fmt.Errorf("cd: not a directory: %s", path), Exit: false}
	}

	t.FS.PrevDir = t.FS.CurrentDir
	t.FS.CurrentDir = target
	return &CommandResult{Output: "", Error: nil, Exit: false}
}

// cmdLs implements the ls command
func (t *Terminal) cmdLs(args []string) *CommandResult {
	longFormat := false
	showHidden := false
	path := "."

	// Parse arguments
	for _, arg := range args {
		if arg == "-l" {
			longFormat = true
		} else if arg == "-a" {
			showHidden = true
		} else if !strings.HasPrefix(arg, "-") {
			path = arg
		}
	}

	target, err := t.FS.ResolvePath(path)
	if err != nil {
		return &CommandResult{Output: "", Error: err, Exit: false}
	}

	if target.Type != Directory {
		if longFormat {
			return &CommandResult{Output: t.formatFileLong(target), Error: nil, Exit: false}
		}
		return &CommandResult{Output: target.Name, Error: nil, Exit: false}
	}

	var output strings.Builder
	files := make([]*VirtualFile, 0, len(target.Children))

	for _, file := range target.Children {
		if !showHidden && strings.HasPrefix(file.Name, ".") {
			continue
		}
		files = append(files, file)
	}

	if longFormat {
		// Add total line
		total := 0
		for _, file := range files {
			total += int(file.Size)
		}
		output.WriteString(fmt.Sprintf("total %d\n", total))

		for _, file := range files {
			output.WriteString(t.formatFileLong(file))
			output.WriteString("\n")
		}
	} else {
		for i, file := range files {
			if i > 0 {
				output.WriteString("  ")
			}
			output.WriteString(file.Name)
		}
		if len(files) > 0 {
			output.WriteString("\n")
		}
	}

	return &CommandResult{Output: output.String(), Error: nil, Exit: false}
}

// formatFileLong formats a file in long format like ls -l
func (t *Terminal) formatFileLong(file *VirtualFile) string {
	var perms string
	if file.Type == Directory {
		perms = "d"
	} else {
		perms = "-"
	}

	// Simple permission string (just rwxrwxrwx for now)
	perms += "rwxrwxrwx"

	size := strconv.FormatInt(file.Size, 10)
	modTime := file.ModTime.Format("Jan 02 15:04")

	return fmt.Sprintf("%s 1 user user %s %s %s", perms, size, modTime, file.Name)
}

// cmdMkdir implements the mkdir command
func (t *Terminal) cmdMkdir(args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Output: "", Error: fmt.Errorf("mkdir: missing operand"), Exit: false}
	}

	createParents := false
	var paths []string

	// Parse arguments
	for _, arg := range args {
		if arg == "-p" {
			createParents = true
		} else {
			paths = append(paths, arg)
		}
	}

	for _, path := range paths {
		err := t.createDirectory(path, createParents)
		if err != nil {
			return &CommandResult{Output: "", Error: err, Exit: false}
		}
	}

	return &CommandResult{Output: "", Error: nil, Exit: false}
}

// createDirectory creates a directory at the given path
func (t *Terminal) createDirectory(path string, createParents bool) error {
	// Check if directory already exists
	if _, err := t.FS.ResolvePath(path); err == nil {
		return fmt.Errorf("mkdir: cannot create directory '%s': File exists", path)
	}

	// For now, implement simple case without -p
	if !createParents {
		parentPath := t.getParentPath(path)
		parent, err := t.FS.ResolvePath(parentPath)
		if err != nil {
			return fmt.Errorf("mkdir: cannot create directory '%s': No such file or directory", path)
		}
		if parent.Type != Directory {
			return fmt.Errorf("mkdir: cannot create directory '%s': Not a directory", path)
		}

		dirName := t.getBaseName(path)
		newDir := &VirtualFile{
			Name:        dirName,
			Type:        Directory,
			Children:    make(map[string]*VirtualFile),
			Parent:      parent,
			Permissions: 0755,
			ModTime:     time.Now(),
			Size:        0,
		}
		parent.Children[dirName] = newDir
		return nil
	}

	// TODO: Implement -p flag (create parents)
	return fmt.Errorf("mkdir: -p flag not implemented yet")
}

// getParentPath returns the parent path of a given path
func (t *Terminal) getParentPath(path string) string {
	if path == "/" {
		return "/"
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) <= 1 {
		return "."
	}
	return strings.Join(parts[:len(parts)-1], "/")
}

// getBaseName returns the base name of a path
func (t *Terminal) getBaseName(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return parts[len(parts)-1]
}

// cmdRmdir implements the rmdir command
func (t *Terminal) cmdRmdir(args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Output: "", Error: fmt.Errorf("rmdir: missing operand"), Exit: false}
	}

	for _, path := range args {
		target, err := t.FS.ResolvePath(path)
		if err != nil {
			return &CommandResult{Output: "", Error: err, Exit: false}
		}

		if target.Type != Directory {
			return &CommandResult{Output: "", Error: fmt.Errorf("rmdir: failed to remove '%s': Not a directory", path), Exit: false}
		}

		if len(target.Children) > 0 {
			return &CommandResult{Output: "", Error: fmt.Errorf("rmdir: failed to remove '%s': Directory not empty", path), Exit: false}
		}

		// Remove from parent
		delete(target.Parent.Children, target.Name)
	}

	return &CommandResult{Output: "", Error: nil, Exit: false}
}

// cmdTouch implements the touch command
func (t *Terminal) cmdTouch(args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Output: "", Error: fmt.Errorf("touch: missing file operand"), Exit: false}
	}

	for _, path := range args {
		// Check if file already exists
		if file, err := t.FS.ResolvePath(path); err == nil {
			// Update modification time
			file.ModTime = time.Now()
			continue
		}

		// Create new file
		parentPath := t.getParentPath(path)
		parent, err := t.FS.ResolvePath(parentPath)
		if err != nil {
			return &CommandResult{Output: "", Error: err, Exit: false}
		}
		if parent.Type != Directory {
			return &CommandResult{Output: "", Error: fmt.Errorf("touch: cannot touch '%s': Not a directory", path), Exit: false}
		}

		fileName := t.getBaseName(path)
		newFile := &VirtualFile{
			Name:        fileName,
			Type:        RegularFile,
			Content:     []byte{},
			Parent:      parent,
			Permissions: 0644,
			ModTime:     time.Now(),
			Size:        0,
		}
		parent.Children[fileName] = newFile
	}

	return &CommandResult{Output: "", Error: nil, Exit: false}
}

// cmdRm implements the rm command
func (t *Terminal) cmdRm(args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Output: "", Error: fmt.Errorf("rm: missing operand"), Exit: false}
	}

	recursive := false
	var paths []string

	// Parse arguments
	for _, arg := range args {
		if arg == "-r" || arg == "-rf" {
			recursive = true
		} else {
			paths = append(paths, arg)
		}
	}

	for _, path := range paths {
		target, err := t.FS.ResolvePath(path)
		if err != nil {
			return &CommandResult{Output: "", Error: err, Exit: false}
		}

		if target.Type == Directory && !recursive {
			return &CommandResult{Output: "", Error: fmt.Errorf("rm: cannot remove '%s': Is a directory", path), Exit: false}
		}

		// Remove from parent
		delete(target.Parent.Children, target.Name)
	}

	return &CommandResult{Output: "", Error: nil, Exit: false}
}

// cmdCp implements the cp command
func (t *Terminal) cmdCp(args []string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Output: "", Error: fmt.Errorf("cp: missing file operand"), Exit: false}
	}

	recursive := false
	source := args[len(args)-2]
	dest := args[len(args)-1]

	// Check for -r flag
	if len(args) > 2 && args[0] == "-r" {
		recursive = true
		source = args[1]
		dest = args[2]
	}

	sourceFile, err := t.FS.ResolvePath(source)
	if err != nil {
		return &CommandResult{Output: "", Error: err, Exit: false}
	}

	if sourceFile.Type == Directory && !recursive {
		return &CommandResult{Output: "", Error: fmt.Errorf("cp: -r not specified; omitting directory '%s'", source), Exit: false}
	}

	// For now, implement simple file copy
	destParentPath := t.getParentPath(dest)
	destParent, err := t.FS.ResolvePath(destParentPath)
	if err != nil {
		return &CommandResult{Output: "", Error: err, Exit: false}
	}

	destName := t.getBaseName(dest)
	newFile := &VirtualFile{
		Name:        destName,
		Type:        sourceFile.Type,
		Content:     make([]byte, len(sourceFile.Content)),
		Parent:      destParent,
		Permissions: sourceFile.Permissions,
		ModTime:     time.Now(),
		Size:        sourceFile.Size,
	}

	copy(newFile.Content, sourceFile.Content)
	destParent.Children[destName] = newFile

	return &CommandResult{Output: "", Error: nil, Exit: false}
}

// cmdMv implements the mv command
func (t *Terminal) cmdMv(args []string) *CommandResult {
	if len(args) != 2 {
		return &CommandResult{Output: "", Error: fmt.Errorf("mv: missing file operand"), Exit: false}
	}

	source := args[0]
	dest := args[1]

	sourceFile, err := t.FS.ResolvePath(source)
	if err != nil {
		return &CommandResult{Output: "", Error: err, Exit: false}
	}

	// Remove from old location
	delete(sourceFile.Parent.Children, sourceFile.Name)

	// Add to new location
	destParentPath := t.getParentPath(dest)
	destParent, err := t.FS.ResolvePath(destParentPath)
	if err != nil {
		return &CommandResult{Output: "", Error: err, Exit: false}
	}

	destName := t.getBaseName(dest)
	sourceFile.Name = destName
	sourceFile.Parent = destParent
	sourceFile.ModTime = time.Now()
	destParent.Children[destName] = sourceFile

	return &CommandResult{Output: "", Error: nil, Exit: false}
}

// cmdCat implements the cat command
func (t *Terminal) cmdCat(args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Output: "", Error: fmt.Errorf("cat: missing file operand"), Exit: false}
	}

	var output strings.Builder
	for _, path := range args {
		file, err := t.FS.ResolvePath(path)
		if err != nil {
			return &CommandResult{Output: "", Error: err, Exit: false}
		}

		if file.Type != RegularFile {
			return &CommandResult{Output: "", Error: fmt.Errorf("cat: %s: Is a directory", path), Exit: false}
		}

		output.Write(file.Content)
		if len(args) > 1 {
			output.WriteString("\n")
		}
	}

	return &CommandResult{Output: output.String(), Error: nil, Exit: false}
}

// cmdEcho implements the echo command
func (t *Terminal) cmdEcho(args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Output: "\n", Error: nil, Exit: false}
	}

	// Check for redirection
	output := strings.Join(args, " ") + "\n"

	// Simple implementation - no redirection yet
	return &CommandResult{Output: output, Error: nil, Exit: false}
}

// cmdEdit implements the edit command
func (t *Terminal) cmdEdit(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Output: "", Error: fmt.Errorf("edit: missing file operand"), Exit: false}
	}

	path := args[0]
	file, err := t.FS.ResolvePath(path)
	if err != nil {
		// Create new file if it doesn't exist
		parentPath := t.getParentPath(path)
		parent, err := t.FS.ResolvePath(parentPath)
		if err != nil {
			return &CommandResult{Output: "", Error: err, Exit: false}
		}

		fileName := t.getBaseName(path)
		file = &VirtualFile{
			Name:        fileName,
			Type:        RegularFile,
			Content:     []byte{},
			Parent:      parent,
			Permissions: 0644,
			ModTime:     time.Now(),
			Size:        0,
		}
		parent.Children[fileName] = file
	}

	if file.Type != RegularFile {
		return &CommandResult{Output: "", Error: fmt.Errorf("edit: %s: Is a directory", path), Exit: false}
	}

	// Simple editor implementation
	return t.simpleEditor(file)
}

// simpleEditor provides a basic text editor
func (t *Terminal) simpleEditor(file *VirtualFile) *CommandResult {
	fmt.Println("Simple Editor - Type ':w' to save, ':q' to quit, ':wq' to save and quit")
	fmt.Println("Current content:")
	fmt.Print(string(file.Content))

	// For now, just return - full editor implementation would need interactive input
	return &CommandResult{Output: "Editor not fully implemented yet", Error: nil, Exit: false}
}

// cmdClear implements the clear command
func (t *Terminal) cmdClear(args []string) *CommandResult {
	// In a real terminal, this would clear the screen
	return &CommandResult{Output: "\033[2J\033[H", Error: nil, Exit: false}
}

// cmdExit implements the exit command
func (t *Terminal) cmdExit(args []string) *CommandResult {
	return &CommandResult{Output: "Goodbye!", Error: nil, Exit: true}
}

// cmdHelp implements the help command
func (t *Terminal) cmdHelp(args []string) *CommandResult {
	helpText := `Available commands:
pwd              - Print working directory
cd [dir]         - Change directory
ls [-l|-a] [dir] - List directory contents
mkdir [-p] dir   - Create directory
rmdir dir        - Remove empty directory
touch file       - Create empty file or update timestamp
rm [-r] file     - Remove file or directory
cp [-r] src dst  - Copy file or directory
mv src dst       - Move/rename file or directory
cat file         - Display file contents
echo [text]      - Display text
edit file        - Simple text editor
clear            - Clear terminal screen
exit/quit        - Exit terminal
help             - Show this help`

	return &CommandResult{Output: helpText, Error: nil, Exit: false}
}
