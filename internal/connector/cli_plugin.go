package connector

import (
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
	"io"
	"os/exec"
)

type CliPlugin struct {
	wrapper        *cliwrapper.CliWrapper
	deployCommand  *exec.Cmd
	stdErrBuff     *cliwrapper.BufferThreadSafe
	containerImage string
	logger         log.Logger
	stdin          io.WriteCloser
	stdout         io.ReadCloser
}

func (p *CliPlugin) Write(b []byte) (n int, err error) {
	return p.stdin.Write(b)
}

func (p *CliPlugin) Read(b []byte) (n int, err error) {
	return p.stdout.Read(b)
}

func (p *CliPlugin) Close() error {
	//wrapper := *p.wrapper
	if err := p.KillAndClean(); err != nil {
		return err
	}

	if err := p.stdin.Close(); err != nil {
		p.logger.Errorf("failed to close stdin pipe")
	} else {
		p.logger.Infof("stdin pipe successfully closed")
	}
	if err := p.stdout.Close(); err != nil {
		p.logger.Infof("failed to close stdout pipe")
	} else {
		p.logger.Infof("stdout pipe successfully closed")
	}
	return nil
}

func (p *CliPlugin) ID() string {
	return p.containerImage
}

func (p *CliPlugin) KillAndClean() error {
	p.logger.Infof("killing config process with pid %d", p.deployCommand.Process.Pid)

	// even if this error was non-nil, we would not handle it differently
	_ = p.deployCommand.Process.Kill()

	_, err := p.deployCommand.Process.Wait()
	if err != nil {
		return err
	}
	if p.stdErrBuff.Len() > 0 {
		p.logger.Warningf("stderr present after plugin execution: '%s'", p.stdErrBuff.String())
	}
	return nil
}
