package fs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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
	Content     []byte                  // For files
	Children    map[string]*VirtualFile // For directories
	Parent      *VirtualFile
	Permissions uint32
	ModTime     time.Time
	Size        int64
}

type FileSystem struct {
	Root       *VirtualFile
	CurrentDir *VirtualFile
	PrevDir    *VirtualFile // For cd -
}

type Terminal struct {
	FS      *FileSystem
	History []string
	Running bool
}

func NewDirectory(name string, parent *VirtualFile) *VirtualFile {
	return &VirtualFile{
		Name:        name,
		Type:        Directory,
		Children:    make(map[string]*VirtualFile),
		Parent:      parent,
		Permissions: 0755,
		ModTime:     time.Now(),
		Size:        0,
	}
}

func NewFile(name string, parent *VirtualFile, content []byte) *VirtualFile {
	size := int64(len(content))
	return &VirtualFile{
		Name:        name,
		Type:        RegularFile,
		Content:     content,
		Parent:      parent,
		Permissions: 0644,
		ModTime:     time.Now(),
		Size:        size,
	}
}

func NewFileSystem() *FileSystem {
	root := NewDirectory("", nil)
	root.Name = "/"

	home := NewDirectory("home", root)
	user := NewDirectory("user", home)

	root.Children["home"] = home
	home.Children["user"] = user
	home.Parent = root
	user.Parent = home

	return &FileSystem{
		Root:       root,
		CurrentDir: user,
		PrevDir:    root,
	}
}

func NewTerminal() *Terminal {
	fs := NewFileSystem()
	return &Terminal{
		FS:      fs,
		History: []string{},
		Running: true,
	}
}

// ResolvePath resolves a path to a VirtualFile, handling absolute/relative paths, ., .., and ~
func (fs *FileSystem) ResolvePath(path string) (*VirtualFile, error) {
	if path == "" {
		return fs.CurrentDir, nil
	}

	// Handle ~ as home directory
	if path == "~" {
		homePath := "/home/user"
		return fs.ResolvePath(homePath)
	}

	// Split into components
	components := strings.Split(path, "/")
	if len(components) == 1 && components[0] == "" {
		components = []string{}
	}

	var current *VirtualFile
	if components[0] == "" { // Absolute path
		current = fs.Root
		components = components[1:]
	} else { // Relative path
		current = fs.CurrentDir
	}

	for _, comp := range components {
		if comp == "" {
			continue
		}
		if comp == "." {
			continue
		}
		if comp == ".." {
			if current.Parent != nil {
				current = current.Parent
			}
			continue
		}

		if current.Type != Directory {
			return nil, fmt.Errorf("not a directory: %s", current.Name)
		}

		child, exists := current.Children[comp]
		if !exists {
			return nil, fmt.Errorf("no such file or directory: %s", comp)
		}
		current = child
	}

	return current, nil
}

// GetPath returns the full path of a VirtualFile relative to root
func (fs *FileSystem) GetPath(file *VirtualFile) string {
	if file == fs.Root {
		return "/"
	}

	pathParts := []string{}
	current := file
	for current != nil && current != fs.Root {
		pathParts = append([]string{current.Name}, pathParts...)
		current = current.Parent
	}
	return "/" + strings.Join(pathParts, "/")
}

// IsAbsolute returns true if path is absolute
func IsAbsolute(path string) bool {
	return strings.HasPrefix(path, "/")
}

// IsDirectory checks if the resolved path is a directory
func (fs *FileSystem) IsDirectory(path string) (bool, error) {
	file, err := fs.ResolvePath(path)
	if err != nil {
		return false, err
	}
	return file.Type == Directory, nil
}

// Exists checks if the path exists
func (fs *FileSystem) Exists(path string) (bool, error) {
	_, err := fs.ResolvePath(path)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// ParseCommand parses the input string into command and arguments, handling basic quoted strings
func ParseCommand(input string) (cmd string, args []string, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", nil, nil
	}

	var tokens []string
	var current strings.Builder
	var inQuote bool
	var quoteChar rune

	for i, r := range input {
		if inQuote {
			current.WriteRune(r)
			if r == quoteChar {
				inQuote = false
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		switch {
		case r == ' ' || r == '\t':
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		case r == '"' || r == '\'':
			inQuote = true
			quoteChar = r
			current.Reset()
		default:
			current.WriteRune(r)
		}

		if i == len(input)-1 && current.Len() > 0 {
			tokens = append(tokens, current.String())
		}
	}

	if len(tokens) == 0 {
		return "", nil, nil
	}

	cmd = tokens[0]
	args = tokens[1:]
	return cmd, args, nil
}

// Pwd returns the current working directory path
func (fs *FileSystem) Pwd() string {
	return fs.GetPath(fs.CurrentDir)
}

// Cd changes the current directory
func (fs *FileSystem) Cd(path string) error {
	if path == "" {
		path = "~"
	}

	if path == "-" {
		if fs.PrevDir == nil {
			return fmt.Errorf("no previous directory")
		}
		oldDir := fs.CurrentDir
		fs.CurrentDir = fs.PrevDir
		fs.PrevDir = oldDir
		return nil
	}

	newDir, err := fs.ResolvePath(path)
	if err != nil {
		return err
	}
	if newDir.Type != Directory {
		return fmt.Errorf("%s is not a directory", path)
	}

	fs.PrevDir = fs.CurrentDir
	fs.CurrentDir = newDir
	return nil
}

// Mkdir creates a new directory at the given path. If parents is true, creates parent directories as needed.
func (fs *FileSystem) Mkdir(path string, parents bool) error {
	if path == "" {
		return fmt.Errorf("mkdir: missing operand")
	}

	// Normalize path to absolute
	var absPath string
	if !IsAbsolute(path) {
		currentPath := fs.GetPath(fs.CurrentDir)
		if !strings.HasSuffix(currentPath, "/") {
			currentPath += "/"
		}
		absPath = currentPath + path
	} else {
		absPath = path
	}

	// Clean the path
	absPath = filepath.Clean(absPath)
	if absPath == "/" {
		return fmt.Errorf("cannot create directory at root")
	}

	components := strings.Split(strings.Trim(absPath, "/"), "/")
	if len(components) == 0 {
		return fmt.Errorf("invalid path")
	}

	current := fs.Root
	for i, comp := range components {
		if comp == "" || comp == "." {
			continue
		}
		if comp == ".." {
			if current.Parent != nil {
				current = current.Parent
			}
			continue
		}

		// If not the last component, ensure it's a directory
		isLast := i == len(components)-1

		// If not parents and not exists, error if trying to create parent
		if !parents && !isLast {
			child, exists := current.Children[comp]
			if !exists {
				return fmt.Errorf("no such file or directory")
			}
			if child.Type != Directory {
				return fmt.Errorf("%s is not a directory", comp)
			}
			current = child
			continue
		}

		// Create if not exists
		if _, exists := current.Children[comp]; !exists {
			if isLast {
				// Create the directory
				newDir := NewDirectory(comp, current)
				current.Children[comp] = newDir
			} else {
				// Create intermediate directory
				newDir := NewDirectory(comp, current)
				current.Children[comp] = newDir
			}
		} else {
			child := current.Children[comp]
			if child.Type != Directory {
				return fmt.Errorf("cannot create directory '%s': not a directory", comp)
			}
		}
		current = current.Children[comp]
	}

	return nil
}

// Touch creates a new empty file or updates the modification time of an existing file
func (fs *FileSystem) Touch(path string) error {
	if path == "" {
		return fmt.Errorf("touch: missing operand")
	}

	// Resolve the parent directory
	dirPath, fileName := filepath.Split(path)
	dir, err := fs.ResolvePath(dirPath)
	if err != nil {
		return fmt.Errorf("touch: %s: %v", path, err)
	}
	if dir.Type != Directory {
		return fmt.Errorf("touch: %s: not a directory", dirPath)
	}

	if _, exists := dir.Children[fileName]; exists {
		// Update timestamp
		file := dir.Children[fileName]
		file.ModTime = time.Now()
		file.Size = int64(len(file.Content))
	} else {
		// Create new empty file
		newFile := NewFile(fileName, dir, []byte{})
		dir.Children[fileName] = newFile
	}

	return nil
}

// getPermString returns the permission string for a file
func getPermString(perm uint32, isDir bool) string {
	var sb strings.Builder
	if isDir {
		sb.WriteRune('d')
	} else {
		sb.WriteRune('-')
	}
	// Owner
	if perm&(1<<8) != 0 { // owner read
		sb.WriteRune('r')
	} else {
		sb.WriteRune('-')
	}
	if perm&(1<<7) != 0 { // owner write
		sb.WriteRune('w')
	} else {
		sb.WriteRune('-')
	}
	if perm&(1<<6) != 0 { // owner exec
		sb.WriteRune('x')
	} else {
		sb.WriteRune('-')
	}
	// Group
	if perm&(1<<5) != 0 {
		sb.WriteRune('r')
	} else {
		sb.WriteRune('-')
	}
	if perm&(1<<4) != 0 {
		sb.WriteRune('w')
	} else {
		sb.WriteRune('-')
	}
	if perm&(1<<3) != 0 {
		sb.WriteRune('x')
	} else {
		sb.WriteRune('-')
	}
	// Other
	if perm&(1<<2) != 0 {
		sb.WriteRune('r')
	} else {
		sb.WriteRune('-')
	}
	if perm&(1<<1) != 0 {
		sb.WriteRune('w')
	} else {
		sb.WriteRune('-')
	}
	if perm&1 != 0 {
		sb.WriteRune('x')
	} else {
		sb.WriteRune('-')
	}
	return sb.String()
}

// Ls lists the contents of the directory at path
func (fs *FileSystem) Ls(path string, long, all bool) (string, error) {
	if path == "" {
		path = "."
	}

	dir, err := fs.ResolvePath(path)
	if err != nil {
		return "", fmt.Errorf("ls: %v", err)
	}
	if dir.Type != Directory {
		return "", fmt.Errorf("ls: %s: not a directory", path)
	}

	var lines []string
	if long {
		// Long format
		for name, child := range dir.Children {
			if !all && strings.HasPrefix(name, ".") && name != "." && name != ".." {
				continue
			}
			permStr := getPermString(child.Permissions, child.Type == Directory)
			timeStr := child.ModTime.Format("Jan 02 15:04")
			line := fmt.Sprintf("%s 1 user user %d %s %s", permStr, child.Size, timeStr, name)
			lines = append(lines, line)
		}
	} else {
		// Short format
		var names []string
		for name := range dir.Children {
			if !all && strings.HasPrefix(name, ".") && name != "." && name != ".." {
				continue
			}
			names = append(names, name)
		}
		lines = append(lines, strings.Join(names, " "))
	}

	return strings.Join(lines, "\n"), nil
}

// Rm removes the file or directory at the given path. If recursive is true, removes directories recursively.
func (fs *FileSystem) Rm(path string, recursive bool) error {
	if path == "" {
		return fmt.Errorf("rm: missing operand")
	}

	target, err := fs.ResolvePath(path)
	if err != nil {
		return fmt.Errorf("rm: %s: %v", path, err)
	}

	if target == fs.Root {
		return fmt.Errorf("rm: cannot remove root directory")
	}

	parent := target.Parent
	if parent == nil {
		return fmt.Errorf("rm: cannot remove root")
	}

	name := target.Name

	// If it's a directory and not recursive, error
	if target.Type == Directory {
		if !recursive {
			return fmt.Errorf("rm: %s: is a directory", path)
		}
		// Recursive delete
		err = fs.deleteRecursive(target)
		if err != nil {
			return err
		}
	} else {
		// File, just delete
		delete(parent.Children, name)
	}

	return nil
}

// deleteRecursive deletes a directory and all its contents
func (fs *FileSystem) deleteRecursive(dir *VirtualFile) error {
	for _, child := range dir.Children {
		if child.Type == Directory {
			err := fs.deleteRecursive(child)
			if err != nil {
				return err
			}
		}
		// Delete the child
		delete(dir.Parent.Children, child.Name)
	}
	// Now delete the dir itself
	delete(dir.Parent.Children, dir.Name)
	return nil
}

// Rmdir removes an empty directory at the given path
func (fs *FileSystem) Rmdir(path string) error {
	if path == "" {
		return fmt.Errorf("rmdir: missing operand")
	}

	target, err := fs.ResolvePath(path)
	if err != nil {
		return fmt.Errorf("rmdir: %s: %v", path, err)
	}
	if target.Type != Directory {
		return fmt.Errorf("rmdir: %s: not a directory", path)
	}
	if len(target.Children) > 0 {
		return fmt.Errorf("rmdir: %s: directory not empty", path)
	}

	parent := target.Parent
	if parent == nil {
		return fmt.Errorf("rmdir: cannot remove root")
	}

	delete(parent.Children, target.Name)
	return nil
}

// Cp copies the source to the destination. If recursive is true, copies directories recursively.
func (fs *FileSystem) Cp(source string, dest string, recursive bool) error {
	if source == "" || dest == "" {
		return fmt.Errorf("cp: missing file operand")
	}

	srcFile, err := fs.ResolvePath(source)
	if err != nil {
		return fmt.Errorf("cp: %s: %v", source, err)
	}

	// Determine destination parent and name
	var destParent *VirtualFile
	var destName string

	destExists, err := fs.Exists(dest)
	if err != nil {
		return err
	}

	if destExists {
		destFile, err := fs.ResolvePath(dest)
		if err != nil {
			return err
		}
		if destFile.Type == Directory {
			// Copy into directory with source name
			destParent = destFile
			destName = srcFile.Name
		} else {
			// Overwrite file
			destParent = destFile.Parent
			destName = destFile.Name
		}
	} else {
		// Create in parent dir
		destParentPath := filepath.Dir(dest)
		destParent, err = fs.ResolvePath(destParentPath)
		if err != nil {
			return fmt.Errorf("cp: %s: %v", destParentPath, err)
		}
		if destParent.Type != Directory {
			return fmt.Errorf("cp: %s: not a directory", destParentPath)
		}
		destName = filepath.Base(dest)
	}

	if srcFile.Type == RegularFile {
		// Copy file
		newContent := make([]byte, len(srcFile.Content))
		copy(newContent, srcFile.Content)
		newFile := NewFile(destName, destParent, newContent)
		destParent.Children[destName] = newFile
	} else if srcFile.Type == Directory {
		if !recursive {
			return fmt.Errorf("cp: omitting directory %s", source)
		}
		// Recursive copy
		err = fs.copyRecursive(srcFile, destParent, destName)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("cp: %s: not a file or directory", source)
	}

	return nil
}

// copyRecursive copies a directory and its contents recursively
func (fs *FileSystem) copyRecursive(srcDir *VirtualFile, destParent *VirtualFile, destName string) error {
	destDir := NewDirectory(destName, destParent)
	destParent.Children[destName] = destDir

	for name, child := range srcDir.Children {
		if child.Type == Directory {
			err := fs.copyRecursive(child, destDir, name)
			if err != nil {
				return err
			}
		} else {
			newContent := make([]byte, len(child.Content))
			copy(newContent, child.Content)
			newFile := NewFile(name, destDir, newContent)
			destDir.Children[name] = newFile
		}
	}

	return nil
}

// Mv moves or renames the source to the destination
func (fs *FileSystem) Mv(source string, dest string) error {
	if source == "" || dest == "" {
		return fmt.Errorf("mv: missing file operand")
	}

	srcFile, err := fs.ResolvePath(source)
	if err != nil {
		return fmt.Errorf("mv: %s: %v", source, err)
	}

	// Determine destination parent and name
	var destParent *VirtualFile
	var destName string

	destExists, err := fs.Exists(dest)
	if err != nil {
		return err
	}

	if destExists {
		destFile, err := fs.ResolvePath(dest)
		if err != nil {
			return err
		}
		if destFile.Type == Directory {
			// Move into directory with source name
			destParent = destFile
			destName = srcFile.Name
		} else {
			// Overwrite file
			destParent = destFile.Parent
			destName = destFile.Name
			// Remove existing dest if it's a file
			delete(destParent.Children, destName)
		}
	} else {
		// Create in parent dir
		destParentPath := filepath.Dir(dest)
		destParent, err = fs.ResolvePath(destParentPath)
		if err != nil {
			return fmt.Errorf("mv: %s: %v", destParentPath, err)
		}
		if destParent.Type != Directory {
			return fmt.Errorf("mv: %s: not a directory", destParentPath)
		}
		destName = filepath.Base(dest)
	}

	// Remove from source parent
	srcParent := srcFile.Parent
	if srcParent == nil {
		return fmt.Errorf("mv: cannot move root")
	}
	delete(srcParent.Children, srcFile.Name)

	// Update parent and name
	srcFile.Parent = destParent
	srcFile.Name = destName
	destParent.Children[destName] = srcFile

	// If directory, update all children parents recursively
	if srcFile.Type == Directory {
		err = fs.updateParentsRecursive(srcFile, destParent)
		if err != nil {
			return err
		}
	}

	return nil
}

// updateParentsRecursive updates the parent for all descendants of a directory
func (fs *FileSystem) updateParentsRecursive(dir *VirtualFile, newParent *VirtualFile) error {
	for _, child := range dir.Children {
		child.Parent = dir
		if child.Type == Directory {
			err := fs.updateParentsRecursive(child, newParent)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Cat displays the contents of the file at the given path
func (fs *FileSystem) Cat(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("cat: missing operand")
	}

	file, err := fs.ResolvePath(path)
	if err != nil {
		return "", fmt.Errorf("cat: %s: %v", path, err)
	}
	if file.Type != RegularFile {
		return "", fmt.Errorf("cat: %s: not a file", path)
	}

	return string(file.Content), nil
}

// EchoWrite writes or appends text to the file at the given path
func (fs *FileSystem) EchoWrite(text string, path string, appendMode bool) error {
	if path == "" {
		return fmt.Errorf("echo: missing filename")
	}

	// Resolve the parent directory
	dirPath, fileName := filepath.Split(path)
	dir, err := fs.ResolvePath(dirPath)
	if err != nil {
		return fmt.Errorf("echo: %s: %v", path, err)
	}
	if dir.Type != Directory {
		return fmt.Errorf("echo: %s: not a directory", dirPath)
	}

	var content []byte
	if appendMode {
		// Append mode
		if file, exists := dir.Children[fileName]; exists {
			if file.Type == RegularFile {
				content = append(file.Content, []byte(text+"\n")...)
				file.Content = content
				file.ModTime = time.Now()
				file.Size = int64(len(content))
				return nil
			} else {
				return fmt.Errorf("echo: %s: not a file", path)
			}
		} else {
			// Create new file with append (same as write for new)
			content = []byte(text + "\n")
		}
	} else {
		// Overwrite mode
		content = []byte(text + "\n")
	}

	// Create or update file
	newFile := NewFile(fileName, dir, content)
	dir.Children[fileName] = newFile

	return nil
}

// Edit opens a simple line-based editor for the given filename
func (t *Terminal) Edit(filename string) error {
	file, err := t.FS.ResolvePath(filename)
	if err != nil {
		// Create new file if not exists
		dirPath, fileName := filepath.Split(filename)
		dir, err := t.FS.ResolvePath(dirPath)
		if err != nil {
			return fmt.Errorf("edit: cannot create %s: %v", filename, err)
		}
		if dir.Type != Directory {
			return fmt.Errorf("edit: %s: not a directory", dirPath)
		}
		file = NewFile(fileName, dir, []byte{})
		dir.Children[fileName] = file
	}

	// Load content into lines
	content := string(file.Content)
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		// Display current buffer with line numbers
		fmt.Println("--- Editor ---")
		for i, line := range lines {
			fmt.Printf("%d: %s\n", i+1, line)
		}
		fmt.Print("> ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		input = strings.TrimSpace(input)

		if strings.HasPrefix(input, ":") {
			cmd := strings.TrimPrefix(input, ":")
			cmd = strings.TrimSpace(cmd)
			switch cmd {
			case "w":
				newContent := strings.Join(lines, "\n") + "\n"
				file.Content = []byte(newContent)
				file.ModTime = time.Now()
				file.Size = int64(len(newContent))
				fmt.Println("Saved")
			case "q":
				return nil
			case "wq":
				newContent := strings.Join(lines, "\n") + "\n"
				file.Content = []byte(newContent)
				file.ModTime = time.Now()
				file.Size = int64(len(newContent))
				fmt.Println("Saved and quit")
				return nil
			default:
				fmt.Printf("Unknown command: %s\n", cmd)
			}
		} else if input == "" {
			continue
		} else {
			// Insert/append the line
			lines = append(lines, input)
		}
	}
}

// Clear clears the terminal screen
func (t *Terminal) Clear() {
	fmt.Print("\033[2J\033[H")
}

// Exit sets the terminal running state to false
func (t *Terminal) Exit() {
	t.Running = false
}

// Help returns a string with available commands
func (t *Terminal) Help() string {
	helpText := `Available commands:
	pwd - Print working directory
	cd [path] - Change directory
	mkdir [dirname] [-p] - Create directory
	touch [filename] - Create empty file
	ls [path] [-l] [-a] - List directory contents
	rm [filename] [-r] - Delete file or directory
	rmdir [dirname] - Remove empty directory
	cp [source] [dest] [-r] - Copy file or directory
	mv [source] [dest] - Move/rename file or directory
	cat [filename] - Display file contents
	echo [text] > [filename] - Write to file
	echo [text] >> [filename] - Append to file
	edit [filename] - Edit file
	clear - Clear screen
	exit - Exit emulator
	quit - Exit emulator
	help - Show this help
	`
	return helpText
}
