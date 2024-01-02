package cliwrapper

import (
	"bytes"
	"go.arcalot.io/log/v2"
	"io"
	"os"
)

type CliWrapperPlugin struct {
	proc           *os.Process
	containerImage string
	logger         log.Logger
	stdin          io.WriteCloser
	stdout         io.ReadCloser
	stdErrBuff     bytes.Buffer
}

func (p *CliWrapperPlugin) Write(b []byte) (n int, err error) {
	return p.stdin.Write(b)
}

func (p *CliWrapperPlugin) Read(b []byte) (n int, err error) {
	return p.stdout.Read(b)
}

func (p *CliWrapperPlugin) KillAndClean() error {
	p.logger.Infof("killing config process with pid %d", p.proc.Pid)

	// even if this error was non-nil, we would not handle it differently
	_ = p.proc.Kill()

	if p.stdErrBuff.Len() > 0 {
		p.logger.Warningf("stderr present after plugin execution: '%s'", p.stdErrBuff.String())
	}
	return nil
}

func (p *CliWrapperPlugin) Close() error {
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

func (p *CliWrapperPlugin) ID() string {
	return p.containerImage
}
