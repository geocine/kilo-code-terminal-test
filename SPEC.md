## Terminal Emulator Specification

### Overview
A sandboxed terminal emulator that simulates a Unix-like file system environment with core file manipulation commands, implemented entirely in memory without affecting the host system.

### Core Architecture

#### 1. Virtual File System (VFS)
- **In-memory file system** with directory tree structure
- Support for files and directories
- Path resolution (absolute and relative paths)
- Current working directory tracking
- File metadata (name, size, permissions, timestamps)

#### 2. Command Parser
- Parse command strings into command + arguments
- Handle quoted strings and escape characters
- Support for basic wildcards (optional)

#### 3. Command Executor
- Route parsed commands to appropriate handlers
- Return success/error codes
- Generate appropriate output or error messages

### Core Commands to Implement

#### File System Navigation
- **pwd** - Print working directory
- **cd [path]** - Change directory
  - Support for `cd ..`, `cd ~`, `cd -` (previous directory)
  - Absolute and relative paths

#### File Operations
- **touch [filename]** - Create empty file
- **rm [filename]** - Delete file
  - Optional: `-r` flag for recursive directory deletion
- **cp [source] [dest]** - Copy file
  - Optional: `-r` flag for recursive copy
- **mv [source] [dest]** - Move/rename file or directory

#### Directory Operations
- **mkdir [dirname]** - Create directory
  - Optional: `-p` flag for creating parent directories
- **rmdir [dirname]** - Remove empty directory
- **ls [path]** - List directory contents
  - Optional flags: `-l` (long format), `-a` (show hidden files)

#### File Content Operations
- **cat [filename]** - Display file contents
- **echo [text] > [filename]** - Write text to file (overwrite)
- **echo [text] >> [filename]** - Append text to file
- **edit [filename]** - Simple text editor
  - Basic multi-line editing
  - Save and exit commands

#### System Commands
- **clear** - Clear terminal screen
- **exit/quit** - Exit emulator
- **help** - Display available commands

### Data Structures

```go
type FileType int
const (
    RegularFile FileType = iota
    Directory
)

type VirtualFile struct {
    Name        string
    Type        FileType
    Content     []byte      // For files
    Children    map[string]*VirtualFile // For directories
    Parent      *VirtualFile
    Permissions uint32
    ModTime     time.Time
    Size        int64
}

type FileSystem struct {
    Root        *VirtualFile
    CurrentDir  *VirtualFile
    PrevDir     *VirtualFile // For cd -
}

type Terminal struct {
    FS          *FileSystem
    History     []string
    Running     bool
}
```

### Key Implementation Details

#### Path Resolution
- Handle absolute paths (starting with /)
- Handle relative paths
- Resolve `.` (current dir) and `..` (parent dir)
- Handle `~` as home directory

#### Error Handling
- File/directory not found
- Permission denied (if implementing permissions)
- Invalid command syntax
- File already exists
- Directory not empty (for rmdir)

#### Editor Implementation
- Simple line-based editor
- Commands: `:w` (save), `:q` (quit), `:wq` (save and quit)
- Basic text manipulation (insert, delete lines)
- Display line numbers

### User Interface
- Command prompt showing current directory
- Colored output (optional)
- Tab completion for file/directory names (optional)
- Command history with up/down arrows (optional)

### Example Usage Flow
```
$ pwd
/home/user
$ mkdir documents
$ cd documents
$ touch README.txt
$ echo "Hello World" > README.txt
$ cat README.txt
Hello World
$ ls -l
-rw-r--r-- 1 user user 11 Dec 10 10:30 README.txt
$ mv README.txt readme.md
$ edit readme.md
[enters editor mode]
```

### Testing Considerations
- Unit tests for each command
- Integration tests for command sequences
- Edge case testing (empty paths, special characters)
- Performance testing with large directory structures

### Optional Enhancements
- Pipe support (`|`)
- Input/output redirection (`>`, `>>`, `<`)
- Environment variables
- Basic scripting support
- File permissions system
- User/group ownership simulation
- Symbolic links
- Search functionality (`find`, `grep`)

This specification provides a solid foundation for building a terminal emulator that's both educational and safe to use since it operates entirely in memory without affecting the actual file system.