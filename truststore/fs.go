package truststore

import (
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"sync"
)

type CmdFS interface {
	fs.StatFS

	Command(name string, arg ...string) *exec.Cmd
	Exec(cmd *exec.Cmd) ([]byte, error)
	SudoExec(cmd *exec.Cmd) ([]byte, error)
	LookPath(cmd string) (string, error)
}

func RootFS() CmdFS {
	return &rootFS{
		StatFS: os.DirFS("/").(fs.StatFS),
	}
}

type rootFS struct {
	fs.StatFS

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
