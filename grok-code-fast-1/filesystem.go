package main

import (
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

// NewFileSystem creates a new virtual file system with root directory
func NewFileSystem() *FileSystem {
	root := &VirtualFile{
		Name:        "/",
		Type:        Directory,
		Children:    make(map[string]*VirtualFile),
		Permissions: 0755,
		ModTime:     time.Now(),
		Size:        0,
	}
	root.Parent = root // Root's parent is itself

	// Create home directory
	home := &VirtualFile{
		Name:        "home",
		Type:        Directory,
		Children:    make(map[string]*VirtualFile),
		Parent:      root,
		Permissions: 0755,
		ModTime:     time.Now(),
		Size:        0,
	}
	root.Children["home"] = home

	// Create user directory
	user := &VirtualFile{
		Name:        "user",
		Type:        Directory,
		Children:    make(map[string]*VirtualFile),
		Parent:      home,
		Permissions: 0755,
		ModTime:     time.Now(),
		Size:        0,
	}
	home.Children["user"] = user

	fs := &FileSystem{
		Root:       root,
		CurrentDir: user, // Start in /home/user
		PrevDir:    nil,
	}

	return fs
}

// ResolvePath resolves a path to a VirtualFile
func (fs *FileSystem) ResolvePath(path string) (*VirtualFile, error) {
	if path == "" {
		return fs.CurrentDir, nil
	}

	var start *VirtualFile
	var parts []string

	if strings.HasPrefix(path, "/") {
		// Absolute path
		start = fs.Root
		parts = strings.Split(strings.Trim(path, "/"), "/")
		if path == "/" {
			return fs.Root, nil
		}
	} else if path == "~" {
		// Home directory
		return fs.Root.Children["home"].Children["user"], nil
	} else {
		// Relative path
		start = fs.CurrentDir
		parts = strings.Split(path, "/")
	}

	current := start
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if current.Parent != nil && current.Parent != current {
				current = current.Parent
			}
			continue
		}

		if current.Type != Directory {
			return nil, &PathError{"not a directory", path}
		}

		child, exists := current.Children[part]
		if !exists {
			return nil, &PathError{"no such file or directory", path}
		}
		current = child
	}

	return current, nil
}

// GetPath returns the absolute path of a VirtualFile
func (fs *FileSystem) GetPath(file *VirtualFile) string {
	if file == fs.Root {
		return "/"
	}

	path := file.Name
	current := file.Parent
	for current != nil && current != fs.Root {
		path = current.Name + "/" + path
		current = current.Parent
	}
	return "/" + path
}

// PathError represents a path resolution error
type PathError struct {
	Msg  string
	Path string
}

func (e *PathError) Error() string {
	return e.Msg + ": " + e.Path
}
