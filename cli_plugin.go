package pythondeployer

import (
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
)

type Plugin struct {
	w cliwrapper.CliWrapperPlugin
}

func (p *Plugin) Write(b []byte) (n int, err error) {
	return p.w.Write(b)
}

func (p *Plugin) Read(b []byte) (n int, err error) {
	return p.w.Read(b)
}

func (p *Plugin) Close() error {
	return p.w.Close()
}

func (p *Plugin) ID() string {
	return p.w.ID()
}
