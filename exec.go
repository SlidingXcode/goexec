package exec

import (
	"fmt"
	"os"
	"sync"
	"time"

	goexec "os/exec"
)

type Command struct {
	Command string
	Args    []string
	Chdir   *string
	Env     map[string]string
	Stdout  func([]byte)
	Stderr  func([]byte)
}

func NewCommand(command string, args []string) *Command {
	return &Command{
		Command: command,
		Args:    args,
		Stdout: func(d []byte) {
			fmt.Fprintf(os.Stdout, string(d))
		},
		Stderr: func(d []byte) {
			fmt.Fprintf(os.Stderr, string(d))
		},
		Env: make(map[string]string),
	}
}

func (c *Command) Execute() error {
	cmd := goexec.Command(c.Command, c.Args...)

	if c.Chdir != nil && len(*c.Chdir) > 0 {
		working, _ := os.Getwd()

		if err := os.Chdir(*c.Chdir); err != nil {
			return fmt.Errorf("os.Chdir error (%s): %s", *c.Chdir, err)
		}
		defer func() {
			if err := os.Chdir(working); err != nil {
				// print to stderr?
			}
		}()
	}
	if len(c.Env) > 0 {
		newenv := os.Environ()
		for k, v := range c.Env {
			newenv = append(newenv, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = newenv
	}

	wg := sync.WaitGroup{}

	if c.Stdout != nil {
		if stdout, err := cmd.StdoutPipe(); err == nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				buffer := make([]byte, 1024)
				for {
					read, err := stdout.Read(buffer)
					if err != nil {
						return
					}
					if read > 0 {
						c.Stdout(buffer[0:read])
					} else {
						select {
						case <-time.After(time.Millisecond * 50):
						}
					}
				}
			}()
		}
	}

	if c.Stderr != nil {
		if stderr, err := cmd.StderrPipe(); err == nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				buffer := make([]byte, 1024)
				for {
					read, err := stderr.Read(buffer)
					if err != nil {
						return
					}
					if read > 0 {
						c.Stderr(buffer[0:read])
					} else {
						select {
						case <-time.After(time.Millisecond * 50):
						}
					}
				}
			}()
		}
	}

	var err, outErr error

	if err = cmd.Start(); err != nil {
		outErr = fmt.Errorf("Command.Start error: %s", err)
	} else {
		if err = cmd.Wait(); err != nil {
			outErr = fmt.Errorf("Command.Wait error: %s", err)
		}
	}

	wg.Wait()

	return outErr
}
