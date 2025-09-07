package fs

import (
	"strings"
	"testing"
	"time"
)

func TestNewFileSystem(t *testing.T) {
	fs := NewFileSystem()
	if fs.Root == nil {
		t.Error("Root should not be nil")
	}
	if fs.CurrentDir.Name != "user" {
		t.Error("Current dir should be /home/user")
	}
}

func TestResolvePath(t *testing.T) {
	fs := NewFileSystem()
	home, err := fs.ResolvePath("~")
	if err != nil {
		t.Error(err)
	}
	if home.Name != "user" {
		t.Error("~ should resolve to user dir")
	}

	root, err := fs.ResolvePath("/")
	if err != nil {
		t.Error(err)
	}
	if root.Name != "/" {
		t.Error(" / should resolve to root")
	}

	_, err = fs.ResolvePath("/nonexistent")
	if err == nil {
		t.Error("Nonexistent path should error")
	}
}

func TestPwd(t *testing.T) {
	fs := NewFileSystem()
	pwd := fs.Pwd()
	if pwd != "/home/user" {
		t.Errorf("Expected /home/user, got %s", pwd)
	}
}

func TestCd(t *testing.T) {
	fs := NewFileSystem()
	err := fs.Cd("..")
	if err != nil {
		t.Error(err)
	}
	if fs.CurrentDir.Name != "home" {
		t.Error("cd .. should go to home")
	}

	err = fs.Cd("-")
	if err != nil {
		t.Error(err)
	}
	if fs.CurrentDir.Name != "user" {
		t.Error("cd - should go back to user")
	}
}

func TestMkdir(t *testing.T) {
	fs := NewFileSystem()
	err := fs.Mkdir("testdir", false)
	if err != nil {
		t.Error(err)
	}

	_, err = fs.ResolvePath("testdir")
	if err != nil {
		t.Error("mkdir should create dir")
	}

	err = fs.Mkdir("/parent/child", true)
	if err != nil {
		t.Error(err)
	}

	_, err = fs.ResolvePath("/parent/child")
	if err != nil {
		t.Error("mkdir -p should create parents")
	}
}

func TestTouch(t *testing.T) {
	fs := NewFileSystem()
	err := fs.Touch("test.txt")
	if err != nil {
		t.Error(err)
	}

	file, err := fs.ResolvePath("test.txt")
	if err != nil {
		t.Error(err)
	}
	if file.Type != RegularFile {
		t.Error("touch should create regular file")
	}

	// Touch existing
	oldTime := file.ModTime
	time.Sleep(1 * time.Second)
	err = fs.Touch("test.txt")
	if err != nil {
		t.Error(err)
	}
	if file.ModTime == oldTime {
		t.Error("touch should update mod time")
	}
}

func TestLs(t *testing.T) {
	fs := NewFileSystem()
	err := fs.Mkdir("testdir", false)
	if err != nil {
		t.Error(err)
	}
	err = fs.Touch("testfile.txt")
	if err != nil {
		t.Error(err)
	}

	output, err := fs.Ls(".", false, false)
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(output, "testdir") || !strings.Contains(output, "testfile.txt") {
		t.Error("ls should list files and dirs")
	}

	output, err = fs.Ls(".", true, true)
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(output, "drwxr-xr-x") {
		t.Error("ls -l -a should show long format")
	}
}

func TestCat(t *testing.T) {
	fs := NewFileSystem()
	err := fs.Touch("test.txt")
	if err != nil {
		t.Error(err)
	}
	err = fs.EchoWrite("Hello World", "test.txt", false)
	if err != nil {
		t.Error(err)
	}

	output, err := fs.Cat("test.txt")
	if err != nil {
		t.Error(err)
	}
	if output != "Hello World\n" {
		t.Error("cat should display content")
	}
}

func TestEcho(t *testing.T) {
	fs := NewFileSystem()
	err := fs.EchoWrite("Hello", "test.txt", false)
	if err != nil {
		t.Error(err)
	}

	err = fs.EchoWrite(" World", "test.txt", true)
	if err != nil {
		t.Error(err)
	}

	output, err := fs.Cat("test.txt")
	if err != nil {
		t.Error(err)
	}
	if output != "Hello\n World\n" {
		t.Error("echo >> should append")
	}
}
