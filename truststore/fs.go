package truststore

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

type CmdFS interface {
	fs.StatFS

	Command(name string, arg ...string) *exec.Cmd
	Exec(cmd *exec.Cmd) ([]byte, error)
	SudoExec(cmd *exec.Cmd) ([]byte, error)
	LookPath(cmd string) (string, error)
}

type DataFS interface {
	fs.StatFS
	fs.ReadFileFS

	AppendToFile(name string, p []byte) error
	Rename(oldpath, newpath string) error
	Remove(name string) error
}

type FS interface {
	CmdFS
	DataFS
}

func RootFS() FS {
	return &rootFS{
		StatFS:   os.DirFS("/").(fs.StatFS),
		rootPath: "/",
	}
}

type rootFS struct {
	fs.StatFS

	rootPath string

	sudoWarningOnce sync.Once
}

func (r *rootFS) Command(name string, arg ...string) *exec.Cmd {
	path, _ := r.LookPath(name)
	return exec.Command(path, arg...)
}

func (r *rootFS) Exec(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}

func (r *rootFS) SudoExec(cmd *exec.Cmd) (out []byte, err error) {
	if u, err := user.Current(); err == nil && u.Uid == "0" {
		return r.Exec(cmd)
	}
	if _, serr := r.LookPath("sudo"); serr != nil {
		defer func() {
			r.sudoWarningOnce.Do(func() {
				err = Error{
					Op: OpSudo,

					Fatal:   err,
					Warning: ErrNoSudo,
				}
			})
		}()

		return r.Exec(cmd)
	}

	sudo := r.Command("sudo", append([]string{"--prompt=Sudo password:", "--"}, cmd.Args...)...)
	sudo.Env = cmd.Env
	sudo.Dir = cmd.Dir
	sudo.Stdin = cmd.Stdin

	return r.Exec(sudo)
}

func (r *rootFS) LookPath(cmd string) (string, error) {
	return exec.LookPath(cmd)
}

func (r *rootFS) AppendToFile(name string, p []byte) error {
	f, err := os.OpenFile(filepath.Join(r.rootPath, name), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		if f, err = os.Create(filepath.Join(r.rootPath, name)); err != nil {
			return err
		}
	}
	defer f.Close()

	if _, err := f.Write(p); err != nil {
		return err
	}
	return f.Close()
}

func (r *rootFS) Rename(oldpath, newpath string) error {
	src, dst := filepath.Join(r.rootPath, oldpath), filepath.Join(r.rootPath, newpath)
	return os.Rename(src, dst)
}

func (r *rootFS) Remove(name string) error {
	return os.Remove(filepath.Join(r.rootPath, name))
}

func (r *rootFS) ReadFile(name string) ([]byte, error) {
	if err := r.checkFile(name); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(filepath.Join(r.rootPath, name), os.O_RDONLY, 0444)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}

func (r *rootFS) checkFile(name string) error {
	infoFS, err := r.StatFS.Stat(strings.Trim(name, string(os.PathSeparator)))
	if err != nil {
		return err
	}

	infoOS, err := os.Stat(filepath.Join(r.rootPath, name))
	if err != nil {
		return err
	}

	if infoFS.Name() == infoOS.Name() && infoFS.Size() == infoOS.Size() &&
		infoFS.Mode() == infoOS.Mode() && infoFS.ModTime().Equal(infoOS.ModTime()) &&
		reflect.DeepEqual(infoFS.Sys(), infoOS.Sys()) {
		return nil
	}

	return errors.New("file system setup is misconfigured or corrupt")
}
