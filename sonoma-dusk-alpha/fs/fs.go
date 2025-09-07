package fs

import (
	"fmt"
	"path/filepath"
	"sort"
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

func NewFileSystem() *FileSystem {
	root := &VirtualFile{
		Name:     "/",
		Type:     Directory,
		Children: make(map[string]*VirtualFile),
		Parent:   nil,
		ModTime:  time.Now(),
		Size:     0,
	}
	return &FileSystem{
		Root:       root,
		CurrentDir: root,
		PrevDir:    root,
	}
}

func (f *VirtualFile) IsDir() bool {
	return f.Type == Directory
}

func (fs *FileSystem) resolvePath(path string) (*VirtualFile, error) {
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}
	if path == "~" {
		path = "/home/user"
	}
	if path == "-" {
		return fs.PrevDir, nil
	}
	isAbsolute := strings.HasPrefix(path, "/")
	var current *VirtualFile
	if isAbsolute {
		current = fs.Root
		path = strings.TrimLeft(path, "/")
	} else {
		current = fs.CurrentDir
	}
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part == "" {
			continue
		}
		if part == "." {
			continue
		}
		if part == ".." {
			if current.Parent != nil {
				current = current.Parent
			}
			continue
		}
		if child, ok := current.Children[part]; ok {
			current = child
		} else {
			return nil, fmt.Errorf("no such file or directory: %s", part)
		}
	}
	return current, nil
}

func (fs *FileSystem) CurrentPath() string {
	if fs.CurrentDir == fs.Root {
		return "/"
	}
	path := ""
	current := fs.CurrentDir
	for current != fs.Root {
		path = "/" + current.Name + path
		current = current.Parent
	}
	return path
}

func (fs *FileSystem) ChangeDir(path string) error {
	dir, err := fs.resolvePath(path)
	if err != nil {
		return err
	}
	if !dir.IsDir() {
		return fmt.Errorf("%s: not a directory", path)
	}
	fs.PrevDir = fs.CurrentDir
	fs.CurrentDir = dir
	return nil
}

func (fs *FileSystem) Touch(path string) error {
	file, err := fs.resolvePath(path)
	if err != nil {
		// Create new file
		parent, err := fs.resolvePath(filepath.Dir(path))
		if err != nil {
			return err
		}
		if !parent.IsDir() {
			return fmt.Errorf("cannot create file in non-directory")
		}
		filename := filepath.Base(path)
		if _, exists := parent.Children[filename]; exists {
			return fmt.Errorf("file already exists")
		}
		newFile := &VirtualFile{
			Name:    filename,
			Type:    RegularFile,
			Content: []byte{},
			Parent:  parent,
			ModTime: time.Now(),
			Size:    0,
		}
		parent.Children[filename] = newFile
		return nil
	}
	if file.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	// Update timestamp for existing file
	file.ModTime = time.Now()
	return nil
}

func (fs *FileSystem) MkDir(path string, parents bool) error {
	if path == "" {
		return fmt.Errorf("mkdir: missing directory name")
	}

	// Handle ~ expansion
	if path == "~" {
		path = "/home/user"
	} else if strings.HasPrefix(path, "~/") {
		path = "/home/user" + path[1:]
	}

	// Handle relative paths
	if !strings.HasPrefix(path, "/") {
		currentPath := fs.CurrentPath()
		if currentPath != "/" {
			path = currentPath + "/" + path
		}
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return fmt.Errorf("invalid path")
	}

	var current *VirtualFile = fs.Root
	for i, part := range parts {
		if part == "" {
			continue
		}
		if _, exists := current.Children[part]; !exists {
			if i == len(parts)-1 || parents {
				// Create directory
				newDir := &VirtualFile{
					Name:     part,
					Type:     Directory,
					Children: make(map[string]*VirtualFile),
					Parent:   current,
					ModTime:  time.Now(),
					Size:     0,
				}
				current.Children[part] = newDir
				current = newDir
			} else {
				return fmt.Errorf("cannot create directory %s: No such file or directory", part)
			}
		} else {
			child := current.Children[part]
			if !child.IsDir() {
				return fmt.Errorf("%s: Not a directory", part)
			}
			current = child
		}
	}
	return nil
}

func (fs *FileSystem) Cat(path string) ([]byte, error) {
	file, err := fs.resolvePath(path)
	if err != nil {
		return nil, err
	}
	if file.IsDir() {
		return nil, fmt.Errorf("%s is a directory", path)
	}
	return file.Content, nil
}

func (fs *FileSystem) Echo(text, path string, appendMode bool) error {
	file, err := fs.resolvePath(path)
	if err != nil {
		// Create new file
		parent, err := fs.resolvePath(filepath.Dir(path))
		if err != nil {
			return err
		}
		if !parent.IsDir() {
			return fmt.Errorf("cannot write to non-directory")
		}
		filename := filepath.Base(path)
		if _, exists := parent.Children[filename]; exists && !appendMode {
			return fmt.Errorf("file already exists")
		}
		newFile := &VirtualFile{
			Name:    filename,
			Type:    RegularFile,
			Content: []byte{},
			Parent:  parent,
			ModTime: time.Now(),
			Size:    0,
		}
		parent.Children[filename] = newFile
		file = newFile
	}

	if appendMode {
		file.Content = append(file.Content, []byte(text+"\n")...)
	} else {
		file.Content = []byte(text + "\n")
	}
	file.ModTime = time.Now()
	file.Size = int64(len(file.Content))
	return nil
}

func (fs *FileSystem) Rm(path string, recursive bool) error {
	target, err := fs.resolvePath(path)
	if err != nil {
		return err
	}

	parent := target.Parent
	if parent == nil {
		return fmt.Errorf("cannot remove root directory")
	}

	filename := target.Name

	if target.IsDir() {
		if !recursive {
			if len(target.Children) > 0 {
				return fmt.Errorf("%s: is a directory and not empty", path)
			}
		}
		// Remove directory recursively
		for _, child := range target.Children {
			if child.IsDir() {
				fs.Rm(child.Name, true)
			} else {
				delete(target.Children, child.Name)
			}
		}
	}

	delete(parent.Children, filename)
	return nil
}

func (fs *FileSystem) RmDir(path string) error {
	return fs.Rm(path, false)
}

func (fs *FileSystem) Copy(src, dest string, recursive bool) error {
	srcFile, err := fs.resolvePath(src)
	if err != nil {
		return err
	}

	destParent, err := fs.resolvePath(filepath.Dir(dest))
	if err != nil {
		return err
	}
	if !destParent.IsDir() {
		return fmt.Errorf("cannot copy to non-directory")
	}

	destName := filepath.Base(dest)
	if _, exists := destParent.Children[destName]; exists {
		return fmt.Errorf("file already exists")
	}

	if srcFile.IsDir() {
		if !recursive {
			return fmt.Errorf("%s: is a directory", src)
		}
		// Copy directory
		newDir := &VirtualFile{
			Name:     destName,
			Type:     Directory,
			Children: make(map[string]*VirtualFile),
			Parent:   destParent,
			ModTime:  time.Now(),
			Size:     0,
		}
		destParent.Children[destName] = newDir

		for name, child := range srcFile.Children {
			childPath := srcFile.Name + "/" + name
			destChildPath := newDir.Name + "/" + name
			if child.IsDir() {
				fs.Copy(childPath, destChildPath, true)
			} else {
				newFile := &VirtualFile{
					Name:    name,
					Type:    RegularFile,
					Content: make([]byte, len(child.Content)),
					Parent:  newDir,
					ModTime: time.Now(),
					Size:    int64(len(child.Content)),
				}
				copy(newFile.Content, child.Content)
				newDir.Children[name] = newFile
			}
		}
	} else {
		// Copy file
		newFile := &VirtualFile{
			Name:    destName,
			Type:    RegularFile,
			Content: make([]byte, len(srcFile.Content)),
			Parent:  destParent,
			ModTime: time.Now(),
			Size:    int64(len(srcFile.Content)),
		}
		copy(newFile.Content, srcFile.Content)
		destParent.Children[destName] = newFile
	}

	return nil
}

func (fs *FileSystem) Move(src, dest string) error {
	srcFile, err := fs.resolvePath(src)
	if err != nil {
		return err
	}

	destParent, err := fs.resolvePath(filepath.Dir(dest))
	if err != nil {
		return err
	}
	if !destParent.IsDir() {
		return fmt.Errorf("cannot move to non-directory")
	}

	destName := filepath.Base(dest)
	if _, exists := destParent.Children[destName]; exists {
		return fmt.Errorf("file already exists")
	}

	// Remove from original parent
	srcParent := srcFile.Parent
	if srcParent != nil {
		delete(srcParent.Children, srcFile.Name)
	}

	// Add to new parent
	srcFile.Parent = destParent
	srcFile.Name = destName
	destParent.Children[destName] = srcFile

	return nil
}

func (fs *FileSystem) Ls(path string, flags map[string]bool) (string, error) {
	if path == "" {
		path = "."
	}

	dir, err := fs.resolvePath(path)
	if err != nil {
		return "", err
	}
	if !dir.IsDir() {
		return "", fmt.Errorf("%s is not a directory", path)
	}

	var output strings.Builder
	names := make([]string, 0, len(dir.Children))
	for name := range dir.Children {
		if !flags["a"] && strings.HasPrefix(name, ".") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	showLong := flags["l"]
	for _, name := range names {
		file := dir.Children[name]
		if showLong {
			// Simple long format
			perm := "-rw-r--r--"
			if file.IsDir() {
				perm = "drwxr-xr-x"
			}
			sizeStr := strconv.Itoa(int(file.Size))
			timeStr := file.ModTime.Format("Jan 02 15:04")
			output.WriteString(fmt.Sprintf("%s 1 user user %s %s %s\n",
				perm, sizeStr, timeStr, name))
		} else {
			output.WriteString(name + "\n")
		}
	}
	return output.String(), nil
}
