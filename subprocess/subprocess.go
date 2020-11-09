package subprocess

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
)

// Subprocess is used to manipulate undelying, running os process.
type Subprocess struct {
	name string

	cmd    *exec.Cmd
	stdout *bytes.Buffer
	stderr *bytes.Buffer

	errCh chan<- error

	closed uint32
}

// NewSubprocess returns a subprocess which will run program using `name`, with current environment,
// as well as `extEnv` variables added to it (if they're not empty), it will also use the provided
// `args` as program arguments.
func NewSubprocess(errCh chan<- error, executable string, extEnv []string, args ...string) *Subprocess {
	_, name := filepath.Split(executable)

	cmd := exec.Command(executable, args...)
	cmd.Env = append(os.Environ(), extEnv...)

	return &Subprocess{
		name:  name,
		cmd:   cmd,
		errCh: errCh,
	}
}

// Run starts the program.
func (sp *Subprocess) Run() error {
	if err := sp.cmd.Start(); err != nil {
		atomic.StoreUint32(&sp.closed, 1)
		return fmt.Errorf("start process failed, %v", err)
	}

	go func() {
		for {
			if err := sp.cmd.Wait(); err == nil {
				return
			}

			if atomic.LoadUint32(&sp.closed) > 0 {
				return
			}

			if err := sp.cmd.Start(); err != nil {
				sp.errCh <- fmt.Errorf("restart process failed, %v", err)
			}
		}
	}()

	return nil
}

// Stop stops the program.
func (sp *Subprocess) Stop() error {
	if atomic.LoadUint32(&sp.closed) > 0 {
		return nil
	}
	atomic.StoreUint32(&sp.closed, 1)
	return sp.cmd.Process.Kill()
}

// Signal relays provided signal to the underlying os process.
func (sp *Subprocess) Signal(sig os.Signal) error {
	return sp.cmd.Process.Signal(sig)
}

// ReadStdout reads data from stdout into provided `buf`.
func (sp *Subprocess) ReadStdout(buf []byte) (int, error) {
	pipe, err := sp.cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("cmd.StdoutPipe error, %v", err)
	}
	return pipe.Read(buf)
}

// ReadStderr reads data from stderr into provided `buf`.
func (sp *Subprocess) ReadStderr(buf []byte) (int, error) {
	pipe, err := sp.cmd.StderrPipe()
	if err != nil {
		return 0, fmt.Errorf("cmd.StderrPipe error, %v", err)
	}
	return pipe.Read(buf)
}
