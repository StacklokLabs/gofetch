package main

import (
	"runtime/debug"
	"testing"

	"github.com/stackloklabs/gofetch/pkg/config"
)

func TestResolveVersion(t *testing.T) {
	t.Parallel()

	const testVersion = "1.2.3"
	buildInfo := func(v string) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Version: v}}, true
		}
	}
	noBuildInfo := func() (*debug.BuildInfo, bool) { return nil, false }

	tests := []struct {
		name      string
		ldflags   string
		buildInfo func() (*debug.BuildInfo, bool)
		want      string
	}{
		{name: "ldflags wins", ldflags: testVersion, buildInfo: buildInfo("v9.9.9"), want: testVersion},
		{name: "go install module version", ldflags: config.DefaultVersion, buildInfo: buildInfo("v" + testVersion), want: testVersion},
		{name: "local build", ldflags: config.DefaultVersion, buildInfo: buildInfo("(devel)"), want: config.DefaultVersion},
		{name: "no build info", ldflags: config.DefaultVersion, buildInfo: noBuildInfo, want: config.DefaultVersion},
		{name: "empty ldflags", ldflags: "", buildInfo: buildInfo(""), want: config.DefaultVersion},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveVersion(tt.ldflags, tt.buildInfo); got != tt.want {
				t.Errorf("resolveVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
