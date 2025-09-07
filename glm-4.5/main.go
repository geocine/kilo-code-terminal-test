package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type FileType int

const (
	RegularFile FileType = iota
	Directory
)

type VirtualFile struct {
	Name        string
	Type        FileType
	Content     []byte
	Children    map[string]*VirtualFile
	Parent      *VirtualFile
	Permissions uint32
	ModTime     time.Time
	Size        int64
}

// NewVirtualFile creates a new VirtualFile with the given name and type
func NewVirtualFile(name string, fileType FileType) *VirtualFile {
	now := time.Now()
	return &VirtualFile{
		Name:        name,
		Type:        fileType,
		Content:     make([]byte, 0),
		Children:    make(map[string]*VirtualFile),
		Parent:      nil,
		Permissions: 0644, // Default permissions
		ModTime:     now,
		Size:        0,
	}
}

// AddChild adds a child file/directory to this directory
func (vf *VirtualFile) AddChild(child *VirtualFile) error {
	if vf.Type != Directory {
		return fmt.Errorf("cannot add child to non-directory")
	}

	if _, exists := vf.Children[child.Name]; exists {
		return fmt.Errorf("file or directory '%s' already exists", child.Name)
	}

	child.Parent = vf
	vf.Children[child.Name] = child
	vf.ModTime = time.Now()
	return nil
}

// RemoveChild removes a child file/directory from this directory
func (vf *VirtualFile) RemoveChild(name string) error {
	if vf.Type != Directory {
		return fmt.Errorf("cannot remove child from non-directory")
	}

	if _, exists := vf.Children[name]; !exists {
		return fmt.Errorf("file or directory '%s' not found", name)
	}

	delete(vf.Children, name)
	vf.ModTime = time.Now()
	return nil
}

// GetPath returns the absolute path of this file/directory
func (vf *VirtualFile) GetPath() string {
	if vf.Parent == nil {
		return "/"
	}

	path := vf.Name
	parent := vf.Parent

	for parent != nil && parent.Name != "" {
		path = parent.Name + "/" + path
		parent = parent.Parent
	}

	return "/" + path
}

// UpdateContent updates the content of a file and updates metadata
func (vf *VirtualFile) UpdateContent(content []byte) {
	if vf.Type != RegularFile {
		return
	}

	vf.Content = content
	vf.Size = int64(len(content))
	vf.ModTime = time.Now()
}

type FileSystem struct {
	Root       *VirtualFile
	CurrentDir *VirtualFile
	PrevDir    *VirtualFile
}

// NewFileSystem creates a new file system with basic structure
func NewFileSystem() *FileSystem {
	// Create root directory
	root := NewVirtualFile("", Directory)

	// Create home directory
	home := NewVirtualFile("home", Directory)
	root.AddChild(home)

	// Create user directory
	user := NewVirtualFile("user", Directory)
	home.AddChild(user)

	return &FileSystem{
		Root:       root,
		CurrentDir: user,
		PrevDir:    user,
	}
}

// GetAbsolutePath resolves a path to an absolute path
func (fs *FileSystem) GetAbsolutePath(path string) string {
	if strings.HasPrefix(path, "/") {
		// Already absolute
		return path
	}

	// Relative path, prepend current directory
	currentPath := fs.CurrentDir.GetPath()
	if currentPath == "/" {
		return "/" + path
	}
	return currentPath + "/" + path
}

// ResolvePath resolves a path to a VirtualFile
func (fs *FileSystem) ResolvePath(path string) (*VirtualFile, error) {
	// Handle special cases
	if path == "" {
		return fs.CurrentDir, nil
	}

	if path == "~" {
		// Navigate to home directory
		home, ok := fs.Root.Children["home"]
		if !ok {
			return nil, fmt.Errorf("home directory not found")
		}
		user, ok := home.Children["user"]
		if !ok {
			return nil, fmt.Errorf("user directory not found")
		}
		return user, nil
	}

	if path == "-" {
		// Navigate to previous directory
		return fs.PrevDir, nil
	}

	// Get absolute path
	absPath := fs.GetAbsolutePath(path)

	// Split path into components
	components := strings.Split(absPath, "/")

	// Start from root
	current := fs.Root

	// Traverse path
	for _, component := range components {
		if component == "" {
			continue
		}

		if component == "." {
			// Current directory, stay
			continue
		}

		if component == ".." {
			// Parent directory
			if current.Parent != nil {
				current = current.Parent
			}
			continue
		}

		// Look for child
		child, ok := current.Children[component]
		if !ok {
			return nil, fmt.Errorf("path not found: %s", path)
		}

		current = child
	}

	return current, nil
}

type Terminal struct {
	FS      *FileSystem
	History []string
	Running bool
}

func main() {
	// Initialize the terminal
	terminal := NewTerminal()

	// Start the command loop
	terminal.Run()
}

func NewTerminal() *Terminal {
	fs := NewFileSystem()

	return &Terminal{
		FS:      fs,
		History: make([]string, 0),
		Running: true,
	}
}

func (t *Terminal) Run() {
	reader := bufio.NewReader(os.Stdin)
	historyIndex := -1

	// Display welcome message
	fmt.Println("Terminal Emulator - Type 'help' for available commands")

	for t.Running {
		// Display prompt with current directory
		currentPath := t.FS.CurrentDir.GetPath()
		fmt.Printf("%s$ ", currentPath)

		// Read input
		input, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		// Trim whitespace
		input = strings.TrimSpace(input)
		if input == "" {
			historyIndex = -1
			continue
		}

		// Handle command history navigation
		if input == "\x1b[A" { // Up arrow
			if len(t.History) > 0 {
				if historyIndex == -1 {
					historyIndex = len(t.History) - 1
				} else if historyIndex > 0 {
					historyIndex--
				}
				fmt.Printf("\r%s$ %s", currentPath, t.History[historyIndex])
				continue
			}
		} else if input == "\x1b[B" { // Down arrow
			if historyIndex != -1 {
				if historyIndex < len(t.History)-1 {
					historyIndex++
					fmt.Printf("\r%s$ %s", currentPath, t.History[historyIndex])
				} else {
					historyIndex = -1
					fmt.Printf("\r%s$ ", currentPath)
				}
				continue
			}
		} else {
			// Add to history if not a navigation command
			if len(t.History) == 0 || input != t.History[len(t.History)-1] {
				t.History = append(t.History, input)
			}
			historyIndex = -1
		}

		// Parse and execute command
		t.ExecuteCommand(input)
	}

	fmt.Println("Goodbye!")
}

func (t *Terminal) ExecuteCommand(input string) {
	// Parse command with proper handling of quotes and escape characters
	command, args, err := t.ParseCommand(input)
	if err != nil {
		fmt.Printf("Error parsing command: %v\n", err)
		return
	}

	if command == "" {
		return
	}

	// Route to appropriate handler
	switch command {
	case "pwd":
		t.Pwd(args)
	case "cd":
		t.Cd(args)
	case "touch":
		t.Touch(args)
	case "rm":
		t.Rm(args)
	case "cp":
		t.Cp(args)
	case "mv":
		t.Mv(args)
	case "mkdir":
		t.Mkdir(args)
	case "rmdir":
		t.Rmdir(args)
	case "ls":
		t.Ls(args)
	case "cat":
		t.Cat(args)
	case "echo":
		t.Echo(args)
	case "edit":
		t.Edit(args)
	case "clear":
		t.Clear(args)
	case "exit", "quit":
		t.Exit(args)
	case "help":
		t.Help(args)
	default:
		fmt.Printf("Command not found: %s\n", command)
	}
}

// ParseCommand parses a command string into command and arguments
// with support for quoted strings and escape characters
func (t *Terminal) ParseCommand(input string) (string, []string, error) {
	var args []string
	var current strings.Builder
	inQuotes := false
	escapeNext := false

	for i, r := range input {
		if escapeNext {
			current.WriteRune(r)
			escapeNext = false
			continue
		}

		switch r {
		case '\\':
			escapeNext = true
		case '"':
			inQuotes = !inQuotes
		case ' ':
			if inQuotes {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}

		// If we're at the end and not in quotes, add the last argument
		if i == len(input)-1 && !inQuotes && current.Len() > 0 {
			args = append(args, current.String())
		}
	}

	if inQuotes {
		return "", nil, fmt.Errorf("unclosed quotes")
	}

	if escapeNext {
		return "", nil, fmt.Errorf("incomplete escape sequence")
	}

	if len(args) == 0 {
		return "", nil, nil
	}

	return args[0], args[1:], nil
}

// Pwd prints the current working directory
func (t *Terminal) Pwd(args []string) {
	if len(args) > 0 {
		fmt.Println("pwd: too many arguments")
		return
	}

	path := t.FS.CurrentDir.GetPath()
	fmt.Println(path)
}

// Cd changes the current directory
func (t *Terminal) Cd(args []string) {
	if len(args) == 0 {
		// Default to home directory
		t.FS.PrevDir = t.FS.CurrentDir
		home, ok := t.FS.Root.Children["home"]
		if !ok {
			fmt.Println("cd: home directory not found")
			return
		}
		user, ok := home.Children["user"]
		if !ok {
			fmt.Println("cd: user directory not found")
			return
		}
		t.FS.CurrentDir = user
		return
	}

	if len(args) > 1 {
		fmt.Println("cd: too many arguments")
		return
	}

	path := args[0]

	// Special case for "-"
	if path == "-" {
		if t.FS.PrevDir == nil {
			fmt.Println("cd: no previous directory")
			return
		}
		// Swap current and previous directories
		t.FS.CurrentDir, t.FS.PrevDir = t.FS.PrevDir, t.FS.CurrentDir
		fmt.Printf("%s\n", t.FS.CurrentDir.GetPath())
		return
	}

	// Resolve the path
	target, err := t.FS.ResolvePath(path)
	if err != nil {
		fmt.Printf("cd: %v\n", err)
		return
	}

	// Check if it's a directory
	if target.Type != Directory {
		fmt.Printf("cd: %s: Not a directory\n", path)
		return
	}

	// Update current directory
	t.FS.PrevDir = t.FS.CurrentDir
	t.FS.CurrentDir = target
}

// Touch creates a new empty file
func (t *Terminal) Touch(args []string) {
	if len(args) == 0 {
		fmt.Println("touch: missing file operand")
		return
	}

	for _, arg := range args {
		// Get the directory and filename
		dir := t.FS.CurrentDir
		filename := arg

		// If path contains a directory separator, split it
		if strings.Contains(arg, "/") {
			// Extract directory path
			lastSlash := strings.LastIndex(arg, "/")
			if lastSlash > 0 {
				dirPath := arg[:lastSlash]
				filename = arg[lastSlash+1:]

				// Resolve the directory
				var err error
				dir, err = t.FS.ResolvePath(dirPath)
				if err != nil {
					fmt.Printf("touch: %v\n", err)
					continue
				}

				if dir.Type != Directory {
					fmt.Printf("touch: %s: Not a directory\n", dirPath)
					continue
				}
			}
		}

		// Check if file already exists
		if _, exists := dir.Children[filename]; exists {
			// Update modification time
			dir.Children[filename].ModTime = time.Now()
			continue
		}

		// Create new file
		newFile := NewVirtualFile(filename, RegularFile)
		if err := dir.AddChild(newFile); err != nil {
			fmt.Printf("touch: %v\n", err)
		}
	}
}

// Rm removes files or directories
func (t *Terminal) Rm(args []string) {
	if len(args) == 0 {
		fmt.Println("rm: missing operand")
		return
	}

	recursive := false

	// Check for -r flag
	if len(args) > 0 && args[0] == "-r" {
		recursive = true
		args = args[1:]
	}

	if len(args) == 0 {
		fmt.Println("rm: missing operand after -r")
		return
	}

	for _, arg := range args {
		target, err := t.FS.ResolvePath(arg)
		if err != nil {
			fmt.Printf("rm: %v\n", err)
			continue
		}

		// Check if it's a directory and if recursive flag is set
		if target.Type == Directory && len(target.Children) > 0 && !recursive {
			fmt.Printf("rm: cannot remove '%s': Is a directory\n", arg)
			continue
		}

		// Remove from parent directory
		if target.Parent != nil {
			if err := target.Parent.RemoveChild(target.Name); err != nil {
				fmt.Printf("rm: %v\n", err)
			}
		} else {
			fmt.Printf("rm: cannot remove root directory\n")
		}
	}
}

// Cp copies files or directories
func (t *Terminal) Cp(args []string) {
	if len(args) < 2 {
		fmt.Println("cp: missing file operand")
		return
	}

	recursive := false

	// Check for -r flag
	if args[0] == "-r" {
		recursive = true
		args = args[1:]
	}

	if len(args) < 2 {
		fmt.Println("cp: missing file operand after -r")
		return
	}

	sourcePath := args[0]
	destPath := args[1]

	// Resolve source
	source, err := t.FS.ResolvePath(sourcePath)
	if err != nil {
		fmt.Printf("cp: %v\n", err)
		return
	}

	// Resolve destination directory or get parent of destination
	var destDir *VirtualFile
	var destName string

	if strings.Contains(destPath, "/") {
		lastSlash := strings.LastIndex(destPath, "/")
		dirPath := destPath[:lastSlash]
		destName = destPath[lastSlash+1:]

		destDir, err = t.FS.ResolvePath(dirPath)
		if err != nil {
			fmt.Printf("cp: %v\n", err)
			return
		}

		if destDir.Type != Directory {
			fmt.Printf("cp: %s: Not a directory\n", dirPath)
			return
		}
	} else {
		// Check if destPath is an existing directory
		dest, err := t.FS.ResolvePath(destPath)
		if err == nil && dest.Type == Directory {
			destDir = dest
			destName = source.Name
		} else {
			// Use current directory and destPath as filename
			destDir = t.FS.CurrentDir
			destName = destPath
		}
	}

	// Check if destination already exists
	if _, exists := destDir.Children[destName]; exists {
		fmt.Printf("cp: cannot create regular file '%s': File exists\n", destPath)
		return
	}

	// Copy the file/directory
	if err := t.copyFileOrDirectory(source, destDir, destName, recursive); err != nil {
		fmt.Printf("cp: %v\n", err)
	}
}

// Helper function to copy a file or directory recursively
func (t *Terminal) copyFileOrDirectory(source, destDir *VirtualFile, destName string, recursive bool) error {
	// Create new file/directory
	newFile := NewVirtualFile(destName, source.Type)
	newFile.Permissions = source.Permissions

	if err := destDir.AddChild(newFile); err != nil {
		return err
	}

	// If it's a regular file, copy content
	if source.Type == RegularFile {
		content := make([]byte, len(source.Content))
		copy(content, source.Content)
		newFile.UpdateContent(content)
		return nil
	}

	// If it's a directory and recursive is enabled, copy children
	if source.Type == Directory && recursive {
		for name, child := range source.Children {
			if err := t.copyFileOrDirectory(child, newFile, name, recursive); err != nil {
				return err
			}
		}
	} else if source.Type == Directory && !recursive {
		return fmt.Errorf("omitting directory '%s'", source.Name)
	}

	return nil
}

// Mv moves or renames files or directories
func (t *Terminal) Mv(args []string) {
	if len(args) < 2 {
		fmt.Println("mv: missing file operand")
		return
	}

	sourcePath := args[0]
	destPath := args[1]

	// Resolve source
	source, err := t.FS.ResolvePath(sourcePath)
	if err != nil {
		fmt.Printf("mv: %v\n", err)
		return
	}

	// Resolve destination directory or get parent of destination
	var destDir *VirtualFile
	var destName string

	if strings.Contains(destPath, "/") {
		lastSlash := strings.LastIndex(destPath, "/")
		dirPath := destPath[:lastSlash]
		destName = destPath[lastSlash+1:]

		destDir, err = t.FS.ResolvePath(dirPath)
		if err != nil {
			fmt.Printf("mv: %v\n", err)
			return
		}

		if destDir.Type != Directory {
			fmt.Printf("mv: %s: Not a directory\n", dirPath)
			return
		}
	} else {
		// Check if destPath is an existing directory
		dest, err := t.FS.ResolvePath(destPath)
		if err == nil && dest.Type == Directory {
			destDir = dest
			destName = source.Name
		} else {
			// Use current directory and destPath as filename
			destDir = t.FS.CurrentDir
			destName = destPath
		}
	}

	// Check if destination already exists
	if _, exists := destDir.Children[destName]; exists {
		fmt.Printf("mv: cannot move '%s' to '%s': File exists\n", sourcePath, destPath)
		return
	}

	// Remove from parent
	if source.Parent != nil {
		source.Parent.RemoveChild(source.Name)
	} else {
		fmt.Printf("mv: cannot move root directory\n")
		return
	}

	// Add to destination
	source.Name = destName
	if err := destDir.AddChild(source); err != nil {
		// Add back to parent if failed
		source.Parent.AddChild(source)
		fmt.Printf("mv: %v\n", err)
	}
}

// Mkdir creates directories
func (t *Terminal) Mkdir(args []string) {
	if len(args) == 0 {
		fmt.Println("mkdir: missing operand")
		return
	}

	createParents := false

	// Check for -p flag
	if args[0] == "-p" {
		createParents = true
		args = args[1:]
	}

	if len(args) == 0 {
		fmt.Println("mkdir: missing operand after -p")
		return
	}

	for _, arg := range args {
		if createParents {
			// Create parent directories as needed
			t.createDirectoryWithParents(arg)
		} else {
			// Create only the last directory
			t.createSingleDirectory(arg)
		}
	}
}

// Helper function to create a single directory
func (t *Terminal) createSingleDirectory(path string) {
	// Get the parent directory and directory name
	var parent *VirtualFile
	dirName := path

	if strings.Contains(path, "/") {
		// Extract parent path
		lastSlash := strings.LastIndex(path, "/")
		parentPath := path[:lastSlash]
		dirName = path[lastSlash+1:]

		// Resolve the parent directory
		var err error
		parent, err = t.FS.ResolvePath(parentPath)
		if err != nil {
			fmt.Printf("mkdir: %v\n", err)
			return
		}

		if parent.Type != Directory {
			fmt.Printf("mkdir: %s: Not a directory\n", parentPath)
			return
		}
	} else {
		// Use current directory as parent
		parent = t.FS.CurrentDir
	}

	// Check if directory already exists
	if _, exists := parent.Children[dirName]; exists {
		fmt.Printf("mkdir: cannot create directory '%s': File exists\n", path)
		return
	}

	// Create new directory
	newDir := NewVirtualFile(dirName, Directory)
	if err := parent.AddChild(newDir); err != nil {
		fmt.Printf("mkdir: %v\n", err)
	}
}

// Helper function to create directory with parent directories
func (t *Terminal) createDirectoryWithParents(path string) {
	// Split path into components
	components := strings.Split(path, "/")

	// Start from root if absolute path, otherwise from current directory
	var current *VirtualFile
	if strings.HasPrefix(path, "/") {
		current = t.FS.Root
	} else {
		current = t.FS.CurrentDir
	}

	// Create directories as needed
	for _, component := range components {
		if component == "" {
			continue
		}

		// Check if directory already exists
		if child, exists := current.Children[component]; exists {
			if child.Type != Directory {
				fmt.Printf("mkdir: cannot create directory '%s': File exists\n", path)
				return
			}
			current = child
			continue
		}

		// Create new directory
		newDir := NewVirtualFile(component, Directory)
		if err := current.AddChild(newDir); err != nil {
			fmt.Printf("mkdir: %v\n", err)
			return
		}
		current = newDir
	}
}

// Rmdir removes empty directories
func (t *Terminal) Rmdir(args []string) {
	if len(args) == 0 {
		fmt.Println("rmdir: missing operand")
		return
	}

	for _, arg := range args {
		target, err := t.FS.ResolvePath(arg)
		if err != nil {
			fmt.Printf("rmdir: %v\n", err)
			continue
		}

		// Check if it's a directory
		if target.Type != Directory {
			fmt.Printf("rmdir: failed to remove '%s': Not a directory\n", arg)
			continue
		}

		// Check if directory is empty
		if len(target.Children) > 0 {
			fmt.Printf("rmdir: failed to remove '%s': Directory not empty\n", arg)
			continue
		}

		// Remove from parent directory
		if target.Parent != nil {
			if err := target.Parent.RemoveChild(target.Name); err != nil {
				fmt.Printf("rmdir: %v\n", err)
			}
		} else {
			fmt.Printf("rmdir: cannot remove root directory\n")
		}
	}
}

// Ls lists directory contents
func (t *Terminal) Ls(args []string) {
	// Check for flags
	longFormat := false
	showHidden := false

	var path string

	// Parse arguments
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			if strings.Contains(arg, "l") {
				longFormat = true
			}
			if strings.Contains(arg, "a") {
				showHidden = true
			}
		} else {
			path = arg
		}
	}

	// Determine directory to list
	var target *VirtualFile
	if path == "" {
		target = t.FS.CurrentDir
	} else {
		var err error
		target, err = t.FS.ResolvePath(path)
		if err != nil {
			fmt.Printf("ls: %v\n", err)
			return
		}

		if target.Type != Directory {
			// If it's a file, just print the file name
			fmt.Println(target.Name)
			return
		}
	}

	// List contents
	var names []string
	for name := range target.Children {
		// Skip hidden files unless -a flag is set
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		names = append(names, name)
	}

	// Sort names (simple alphabetical sort)
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}

	if longFormat {
		// Long format listing
		for _, name := range names {
			file := target.Children[name]
			var fileType string
			if file.Type == Directory {
				fileType = "d"
			} else {
				fileType = "-"
			}

			permissions := fmt.Sprintf("%o", file.Permissions)
			if len(permissions) > 3 {
				permissions = permissions[len(permissions)-3:]
			}

			size := file.Size
			timeStr := file.ModTime.Format("Jan 02 15:04")

			fmt.Printf("%s%s%s %8d %s %s\n", fileType, permissions, "rwxrwxrwx", size, timeStr, name)
		}
	} else {
		// Simple listing
		for _, name := range names {
			fmt.Println(name)
		}
	}
}

// Cat displays file contents
func (t *Terminal) Cat(args []string) {
	if len(args) == 0 {
		fmt.Println("cat: missing file operand")
		return
	}

	for _, arg := range args {
		// Resolve the file path
		file, err := t.FS.ResolvePath(arg)
		if err != nil {
			fmt.Printf("cat: %v\n", err)
			continue
		}

		// Check if it's a regular file
		if file.Type != RegularFile {
			fmt.Printf("cat: %s: Is a directory\n", arg)
			continue
		}

		// Print file contents
		fmt.Printf("%s", string(file.Content))
	}
}

// Echo displays text or writes it to a file with redirection
func (t *Terminal) Echo(args []string) {
	if len(args) == 0 {
		fmt.Println()
		return
	}

	// Check for redirection
	var text string
	var redirectOp string
	var redirectFile string

	// Parse arguments to find redirection operators
	for i, arg := range args {
		if arg == ">" || arg == ">>" {
			redirectOp = arg
			if i+1 < len(args) {
				redirectFile = args[i+1]
			}
			// Text is everything before the redirection operator
			text = strings.Join(args[:i], " ")
			break
		}
	}

	if redirectOp == "" {
		// No redirection, just print the text
		fmt.Println(strings.Join(args, " "))
		return
	}

	if redirectFile == "" {
		fmt.Println("echo: syntax error near unexpected token 'newline'")
		return
	}

	// Resolve the file path
	file, err := t.FS.ResolvePath(redirectFile)
	if err != nil {
		// File doesn't exist, create it
		var dir *VirtualFile
		var filename string

		if strings.Contains(redirectFile, "/") {
			lastSlash := strings.LastIndex(redirectFile, "/")
			dirPath := redirectFile[:lastSlash]
			filename = redirectFile[lastSlash+1:]

			dir, err = t.FS.ResolvePath(dirPath)
			if err != nil {
				fmt.Printf("echo: %v\n", err)
				return
			}

			if dir.Type != Directory {
				fmt.Printf("echo: %s: Not a directory\n", dirPath)
				return
			}
		} else {
			dir = t.FS.CurrentDir
			filename = redirectFile
		}

		// Create new file
		file = NewVirtualFile(filename, RegularFile)
		if err := dir.AddChild(file); err != nil {
			fmt.Printf("echo: %v\n", err)
			return
		}
	} else if file.Type != RegularFile {
		fmt.Printf("echo: %s: Is a directory\n", redirectFile)
		return
	}

	// Update file content
	var content []byte
	if redirectOp == ">" {
		// Overwrite
		content = []byte(text)
	} else if redirectOp == ">>" {
		// Append
		content = append(file.Content, []byte(text)...)
	}

	file.UpdateContent(content)
}

// Edit opens a simple text editor for a file
func (t *Terminal) Edit(args []string) {
	if len(args) == 0 {
		fmt.Println("edit: missing file operand")
		return
	}

	if len(args) > 1 {
		fmt.Println("edit: too many arguments")
		return
	}

	filename := args[0]

	// Resolve the file path
	file, err := t.FS.ResolvePath(filename)
	if err != nil {
		// File doesn't exist, create it
		var dir *VirtualFile
		var name string

		if strings.Contains(filename, "/") {
			lastSlash := strings.LastIndex(filename, "/")
			dirPath := filename[:lastSlash]
			name = filename[lastSlash+1:]

			dir, err = t.FS.ResolvePath(dirPath)
			if err != nil {
				fmt.Printf("edit: %v\n", err)
				return
			}

			if dir.Type != Directory {
				fmt.Printf("edit: %s: Not a directory\n", dirPath)
				return
			}
		} else {
			dir = t.FS.CurrentDir
			name = filename
		}

		// Create new file
		file = NewVirtualFile(name, RegularFile)
		if err := dir.AddChild(file); err != nil {
			fmt.Printf("edit: %v\n", err)
			return
		}
	} else if file.Type != RegularFile {
		fmt.Printf("edit: %s: Is a directory\n", filename)
		return
	}

	// Start editor mode
	t.runEditor(file)
}

// runEditor implements a simple line-based text editor
func (t *Terminal) runEditor(file *VirtualFile) {
	reader := bufio.NewReader(os.Stdin)

	// Convert current content to lines
	var lines []string
	if len(file.Content) > 0 {
		lines = strings.Split(string(file.Content), "\n")
	} else {
		lines = []string{""}
	}

	// Editor loop
	for {
		// Display file contents with line numbers
		fmt.Println("\n--- Editor: %s (Type :w to save, :q to quit, :wq to save and quit) ---", file.Name)
		for i, line := range lines {
			fmt.Printf("%3d | %s\n", i+1, line)
		}
		fmt.Println("---")

		// Prompt for command
		fmt.Print("> ")

		// Read input
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)

		// Process editor commands
		if strings.HasPrefix(input, ":") {
			cmd := input[1:]
			switch cmd {
			case "w":
				// Save file
				content := []byte(strings.Join(lines, "\n"))
				file.UpdateContent(content)
				fmt.Printf("File saved: %s\n", file.Name)
			case "q":
				// Quit without saving
				return
			case "wq":
				// Save and quit
				content := []byte(strings.Join(lines, "\n"))
				file.UpdateContent(content)
				fmt.Printf("File saved: %s\n", file.Name)
				return
			default:
				fmt.Printf("Unknown command: %s\n", cmd)
			}
		} else {
			// Parse line editing commands
			if strings.HasPrefix(input, "a ") {
				// Add line after specified line number
				parts := strings.SplitN(input, " ", 3)
				if len(parts) >= 3 {
					lineNum, err := strconv.Atoi(parts[1])
					if err != nil || lineNum < 1 || lineNum > len(lines) {
						fmt.Println("Invalid line number")
						continue
					}
					lines = append(lines[:lineNum], append([]string{parts[2]}, lines[lineNum:]...)...)
				}
			} else if strings.HasPrefix(input, "d ") {
				// Delete specified line number
				parts := strings.SplitN(input, " ", 2)
				if len(parts) >= 2 {
					lineNum, err := strconv.Atoi(parts[1])
					if err != nil || lineNum < 1 || lineNum > len(lines) {
						fmt.Println("Invalid line number")
						continue
					}
					lines = append(lines[:lineNum-1], lines[lineNum:]...)
					if len(lines) == 0 {
						lines = []string{""}
					}
				}
			} else if strings.HasPrefix(input, "e ") {
				// Edit specified line number
				parts := strings.SplitN(input, " ", 3)
				if len(parts) >= 3 {
					lineNum, err := strconv.Atoi(parts[1])
					if err != nil || lineNum < 1 || lineNum > len(lines) {
						fmt.Println("Invalid line number")
						continue
					}
					lines[lineNum-1] = parts[2]
				}
			} else if input == "i" {
				// Insert line at the beginning
				fmt.Print("Enter text to insert: ")
				newLine, _ := reader.ReadString('\n')
				newLine = strings.TrimSpace(newLine)
				lines = append([]string{newLine}, lines...)
			} else {
				fmt.Println("Unknown command. Available commands:")
				fmt.Println("  :w - Save file")
				fmt.Println("  :q - Quit without saving")
				fmt.Println("  :wq - Save and quit")
				fmt.Println("  i - Insert line at beginning")
				fmt.Println("  a <line> <text> - Add line after specified line")
				fmt.Println("  d <line> - Delete specified line")
				fmt.Println("  e <line> <text> - Edit specified line")
			}
		}
	}
}

// Clear clears the terminal screen
func (t *Terminal) Clear(args []string) {
	if len(args) > 0 {
		fmt.Println("clear: too many arguments")
		return
	}

	// Clear the screen by printing ANSI escape code
	fmt.Print("\033[2J\033[H")
}

// Exit exits the terminal emulator
func (t *Terminal) Exit(args []string) {
	if len(args) > 0 {
		fmt.Println("exit: too many arguments")
		return
	}

	t.Running = false
}

// Help displays available commands
func (t *Terminal) Help(args []string) {
	if len(args) > 0 {
		fmt.Println("help: too many arguments")
		return
	}

	fmt.Println("Available commands:")
	fmt.Println("  pwd              - Print working directory")
	fmt.Println("  cd [path]        - Change directory")
	fmt.Println("  touch [file]     - Create empty file")
	fmt.Println("  rm [-r] [file]   - Remove file or directory")
	fmt.Println("  cp [-r] [src] [dest] - Copy file or directory")
	fmt.Println("  mv [src] [dest]  - Move/rename file or directory")
	fmt.Println("  mkdir [-p] [dir] - Create directory")
	fmt.Println("  rmdir [dir]      - Remove empty directory")
	fmt.Println("  ls [-l] [-a] [path] - List directory contents")
	fmt.Println("  cat [file]       - Display file contents")
	fmt.Println("  echo [text] > [file] - Write text to file")
	fmt.Println("  echo [text] >> [file] - Append text to file")
	fmt.Println("  edit [file]      - Edit file with simple text editor")
	fmt.Println("  clear            - Clear terminal screen")
	fmt.Println("  exit/quit        - Exit terminal emulator")
	fmt.Println("  help             - Display this help message")
}
