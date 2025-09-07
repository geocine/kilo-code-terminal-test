package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"terminal-emulator/fs"
)

type Terminal struct {
	FS      *fs.FileSystem
	History []string
	Running bool
}

func NewTerminal() *Terminal {
	fsInstance := fs.NewFileSystem()
	// Create home directory /home/user
	root := fsInstance.Root
	homeDir := &fs.VirtualFile{
		Name:     "home",
		Type:     fs.Directory,
		Children: make(map[string]*fs.VirtualFile),
		Parent:   root,
		ModTime:  time.Now(),
		Size:     0,
	}
	userDir := &fs.VirtualFile{
		Name:     "user",
		Type:     fs.Directory,
		Children: make(map[string]*fs.VirtualFile),
		Parent:   homeDir,
		ModTime:  time.Now(),
		Size:     0,
	}
	homeDir.Children["user"] = userDir
	root.Children["home"] = homeDir
	fsInstance.CurrentDir = userDir
	fsInstance.PrevDir = root

	return &Terminal{
		FS:      fsInstance,
		History: []string{},
		Running: true,
	}
}

func (t *Terminal) Run() {
	scanner := bufio.NewScanner(os.Stdin)
	prompt := t.prompt()
	fmt.Print(prompt)

	for scanner.Scan() {
		if !t.Running {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Print(prompt)
			continue
		}
		t.History = append(t.History, line)
		output, err := executeCommand(t.FS, line)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else if output != "" {
			fmt.Print(output)
		}
		if line == "exit" || line == "quit" {
			t.Running = false
			break
		}
		prompt = t.prompt()
		fmt.Print(prompt)
	}
}

func (t *Terminal) prompt() string {
	path := t.FS.CurrentPath()
	return fmt.Sprintf("%s$ ", path)
}

func executeCommand(fs *fs.FileSystem, cmd string) (string, error) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", nil
	}
	command := parts[0]
	args := parts[1:]

	switch command {
	case "pwd":
		return fs.CurrentPath() + "\n", nil
	case "cd":
		if len(args) == 0 {
			return "", fmt.Errorf("cd: missing path")
		}
		return "", fs.ChangeDir(args[0])
	case "ls":
		return lsCommand(fs, args)
	case "touch":
		if len(args) == 0 {
			return "", fmt.Errorf("touch: missing file name")
		}
		return "", fs.Touch(args[0])
	case "mkdir":
		if len(args) == 0 {
			return "", fmt.Errorf("mkdir: missing directory name")
		}
		parents := false
		path := args[0]
		if len(args) > 1 && args[0] == "-p" {
			parents = true
			path = args[1]
		}
		return "", fs.MkDir(path, parents)
	case "cat":
		if len(args) == 0 {
			return "", fmt.Errorf("cat: missing file name")
		}
		content, err := fs.Cat(args[0])
		if err != nil {
			return "", err
		}
		return string(content) + "\n", nil
	case "echo":
		if len(args) < 2 {
			return "", fmt.Errorf("echo: invalid syntax")
		}
		// Simple echo with redirection
		text := strings.Join(args[:len(args)-1], " ")
		filename := args[len(args)-1]
		if strings.Contains(text, ">") {
			parts := strings.SplitN(text, ">", 2)
			if len(parts) == 2 {
				text = strings.TrimSpace(parts[0])
				filename = strings.TrimSpace(parts[1])
				return "", fs.Echo(text, filename, false)
			}
		} else if strings.Contains(text, ">>") {
			parts := strings.SplitN(text, ">>", 2)
			if len(parts) == 2 {
				text = strings.TrimSpace(parts[0])
				filename = strings.TrimSpace(parts[1])
				return "", fs.Echo(text, filename, true)
			}
		}
		return text + "\n", nil
	case "clear":
		return "\033[2J\033[H", nil
	case "exit", "quit":
		return "", nil
	case "rm":
		if len(args) == 0 {
			return "", fmt.Errorf("rm: missing operand")
		}
		recursive := false
		path := args[0]
		if len(args) > 1 && args[0] == "-r" {
			recursive = true
			path = args[1]
		}
		return "", fs.Rm(path, recursive)
	case "rmdir":
		if len(args) == 0 {
			return "", fmt.Errorf("rmdir: missing operand")
		}
		return "", fs.RmDir(args[0])
	case "cp":
		if len(args) < 2 {
			return "", fmt.Errorf("cp: missing destination")
		}
		recursive := false
		src := args[0]
		dest := args[1]
		if len(args) > 2 && args[0] == "-r" {
			recursive = true
			src = args[1]
			dest = args[2]
		}
		return "", fs.Copy(src, dest, recursive)
	case "mv":
		if len(args) < 2 {
			return "", fmt.Errorf("mv: missing destination")
		}
		return "", fs.Move(args[0], args[1])
	case "edit":
		if len(args) == 0 {
			return "", fmt.Errorf("edit: missing filename")
		}
		return editor(fs, args[0])
	case "help":
		helpText := `Available commands:
- pwd: Print working directory
- cd [path]: Change directory (supports .., ~, -)
- ls [-l] [-a] [path]: List directory contents
- touch [filename]: Create empty file
- mkdir [-p] [dirname]: Create directory
- rmdir [dirname]: Remove empty directory
- rm [-r] [filename]: Remove file or directory
- cp [-r] [source] [dest]: Copy file or directory
- mv [source] [dest]: Move/rename file or directory
- cat [filename]: Display file contents
- echo [text] > [filename]: Write to file
- echo [text] >> [filename]: Append to file
- edit [filename]: Edit file
- clear: Clear screen
- exit/quit: Exit emulator
- help: Show this help`
		return helpText + "\n", nil
	default:
		return "", fmt.Errorf("unknown command: %s", command)
	}
}

func lsCommand(fs *fs.FileSystem, args []string) (string, error) {
	path := "."
	flags := map[string]bool{}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			for _, f := range arg[1:] {
				flags[string(f)] = true
			}
		} else {
			path = arg
		}
	}
	return fs.Ls(path, flags)
}

func editor(fs *fs.FileSystem, filename string) (string, error) {
	content, err := fs.Cat(filename)
	if err != nil {
		fmt.Printf("Cannot open file: %v\n", err)
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	buffer := make([]string, len(lines))
	copy(buffer, lines)

	fmt.Println("--- Editor Mode ---")
	fmt.Println("Commands: :w (save), :q (quit), :wq (save and quit)")
	fmt.Println("Type lines to edit, empty line to continue")

	scanner := bufio.NewScanner(os.Stdin)
	lineNum := 0
	for {
		if lineNum < len(buffer) {
			fmt.Printf("%d: %s\n", lineNum, buffer[lineNum])
		} else {
			fmt.Printf("%d: \n", lineNum)
		}

		if !scanner.Scan() {
			break
		}
		line := scanner.Text()

		if strings.HasPrefix(line, ":") {
			// Command
			cmd := strings.TrimSpace(strings.TrimPrefix(line, ":"))
			switch cmd {
			case "q":
				return "", nil
			case "w":
				newContent := strings.Join(buffer, "\n") + "\n"
				err := fs.Echo(newContent, filename, false)
				if err != nil {
					fmt.Printf("Save failed: %v\n", err)
				} else {
					fmt.Println("Saved")
				}
				continue
			case "wq":
				newContent := strings.Join(buffer, "\n") + "\n"
				err := fs.Echo(newContent, filename, false)
				if err != nil {
					fmt.Printf("Save failed: %v\n", err)
				} else {
					fmt.Println("Saved and quit")
					return "", nil
				}
			default:
				fmt.Printf("Unknown command: %s\n", cmd)
				continue
			}
		}

		if line == "" {
			// Empty line, continue editing
			lineNum++
			continue
		}

		if lineNum < len(buffer) {
			buffer[lineNum] = line
		} else {
			buffer = append(buffer, line)
		}
		lineNum++
	}

	// Save on exit if modified
	newContent := strings.Join(buffer, "\n") + "\n"
	err = fs.Echo(newContent, filename, false)
	if err != nil {
		fmt.Printf("Save failed: %v\n", err)
	}
	return "", nil
}

func main() {
	t := NewTerminal()
	t.Run()
}
