package connector

import (
	"go.arcalot.io/exex"
	"go.arcalot.io/log/v2"
	"io"
)

type CliPlugin struct {
	deployCommand  *exex.Cmd
	containerImage string
	logger         log.Logger
	stdin          io.WriteCloser
	stdout         io.ReadCloser
	stderr         io.ReadCloser
}

func (p *CliPlugin) Write(b []byte) (n int, err error) {
	return p.stdin.Write(b)
}

func (p *CliPlugin) Read(b []byte) (n int, err error) {
	return p.stdout.Read(b)
}

func (p *CliPlugin) Close() error {
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
	if err := p.stderr.Close(); err != nil {
		p.logger.Infof("failed to close stderr pipe")
	} else {
		p.logger.Infof("stderr pipe successfully closed")
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

	slurp, err := io.ReadAll(p.stderr)
	if err != nil {
		return nil
	}
	p.logger.Debugf("python plugin module stderr: %s", slurp)
	return nil
}
