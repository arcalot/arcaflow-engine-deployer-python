package pythondeployer

import (
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
)

type CliPlugin struct {
	w cliwrapper.CliWrapperPlugin
}

func (p *CliPlugin) Write(b []byte) (n int, err error) {
	return p.w.Write(b)
}

func (p *CliPlugin) Read(b []byte) (n int, err error) {
	return p.w.Read(b)
}

func (p *CliPlugin) Close() error {
	return p.w.Close()
}

func (p *CliPlugin) ID() string {
	return p.w.ID()
}
