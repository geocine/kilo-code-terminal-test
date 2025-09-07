package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"terminal-emulator/fs"
)

func main() {
	t := fs.NewTerminal()

	for t.Running {
		prompt := t.FS.GetPath(t.FS.CurrentDir) + "$ "
		fmt.Print(prompt)

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			fmt.Println("Error reading input:", err)
			continue
		}
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		cmd, args, err := fs.ParseCommand(input)
		if err != nil {
			fmt.Println("Error parsing command:", err)
			continue
		}

		output, err := executeCommand(t, cmd, args)
		if output != "" {
			fmt.Println(output)
		}
		if err != nil {
			fmt.Println("Error:", err.Error())
		}
	}
}

func executeCommand(t *fs.Terminal, cmd string, args []string) (string, error) {
	switch cmd {
	case "pwd":
		return t.FS.Pwd(), nil
	case "cd":
		if len(args) == 0 {
			return "", fmt.Errorf("cd: missing path")
		}
		return "", t.FS.Cd(args[0])
	case "mkdir":
		if len(args) == 0 {
			return "", fmt.Errorf("mkdir: missing operand")
		}
		parents := false
		path := args[0]
		if path == "-p" && len(args) > 1 {
			parents = true
			path = args[1]
		}
		return "", t.FS.Mkdir(path, parents)
	case "touch":
		if len(args) == 0 {
			return "", fmt.Errorf("touch: missing operand")
		}
		return "", t.FS.Touch(args[0])
	case "ls":
		path := "."
		long := false
		all := false
		if len(args) > 0 {
			path = args[0]
		}
		// Simple flag parsing, assume flags are separate args
		for _, arg := range args {
			if arg == "-l" {
				long = true
			}
			if arg == "-a" {
				all = true
			}
		}
		return t.FS.Ls(path, long, all)
	case "rm":
		if len(args) == 0 {
			return "", fmt.Errorf("rm: missing operand")
		}
		recursive := false
		path := args[0]
		if path == "-r" && len(args) > 1 {
			recursive = true
			path = args[1]
		}
		return "", t.FS.Rm(path, recursive)
	case "rmdir":
		if len(args) == 0 {
			return "", fmt.Errorf("rmdir: missing operand")
		}
		return "", t.FS.Rmdir(args[0])
	case "cp":
		if len(args) < 2 {
			return "", fmt.Errorf("cp: missing file operand")
		}
		recursive := false
		src := args[0]
		dest := args[1]
		if src == "-r" && len(args) > 2 {
			recursive = true
			src = args[1]
			dest = args[2]
		}
		return "", t.FS.Cp(src, dest, recursive)
	case "mv":
		if len(args) < 2 {
			return "", fmt.Errorf("mv: missing file operand")
		}
		return "", t.FS.Mv(args[0], args[1])
	case "cat":
		if len(args) == 0 {
			return "", fmt.Errorf("cat: missing operand")
		}
		return t.FS.Cat(args[0])
	case "echo":
		if len(args) == 0 {
			fmt.Println("")
			return "", nil
		}
		// Handle redirection
		if len(args) == 1 {
			fmt.Println(args[0])
			return "", nil
		}
		// Assume last arg is filename with possible redirection
		filename := args[len(args)-1]
		text := strings.Join(args[:len(args)-1], " ")
		appendMode := false
		if strings.HasSuffix(filename, ">>") {
			filename = strings.TrimSuffix(filename, ">>")
			appendMode = true
		} else if strings.HasSuffix(filename, ">") {
			filename = strings.TrimSuffix(filename, ">")
			appendMode = false
		} else {
			// No redirection, print
			fmt.Println(text)
			return "", nil
		}
		return "", t.FS.EchoWrite(text, filename, appendMode)
	case "edit":
		if len(args) == 0 {
			return "", fmt.Errorf("edit: missing operand")
		}
		return "", t.Edit(args[0])
	case "clear":
		t.Clear()
		return "", nil
	case "exit", "quit":
		t.Exit()
		return "", nil
	case "help":
		return t.Help(), nil
	default:
		return "", fmt.Errorf("command not found: %s", cmd)
	}
}
