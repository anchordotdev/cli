package clitest

import (
	"io/fs"
	"os"
	"testing/fstest"
)

type TestFS fstest.MapFS

func (t TestFS) Open(name string) (fs.File, error)          { return (fstest.MapFS)(t).Open(name) }
func (t TestFS) ReadDir(name string) ([]fs.DirEntry, error) { return (fstest.MapFS)(t).ReadDir(name) }
func (t TestFS) Stat(name string) (fs.FileInfo, error)      { return (fstest.MapFS)(t).Stat(name) }

func (t TestFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	t[name] = &fstest.MapFile{
		Data: data,
		Mode: perm,
	}

	return nil
}
