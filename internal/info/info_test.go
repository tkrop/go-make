package info_test

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tkrop/go-make/internal/info"
	"github.com/tkrop/go-testing/test"
)

const (
	// revisionHead contains an arbitrary head revision.
	revisionHead = "1b66f320c950b25fa63b81fd4e660c5d1f9d758e"
	// buildPath contains an arbitrary command build path.
	buildPath = "github.com/tkrop/go-make"
	// setupPath contains an arbitrary command setup path.
	setupPath = "github.com/tkrop/go-make/internal/info"
)

type InfoParams struct {
	info       *info.Info
	build      *debug.BuildInfo
	expectInfo *info.Info
}

var testInfoParams = map[string]InfoParams{
	"nil build info": {
		info:       info.NewInfo("", "", "", "", "", false),
		expectInfo: info.NewInfo("", "", "", "", "", false),
	},
	"no build info": {
		info:       info.NewInfo("", "", "", "", "", false),
		build:      &debug.BuildInfo{},
		expectInfo: info.NewInfo("", "", "", "", "", false),
	},

	// Setup build info path.
	"build info setup path": {
		info: info.NewInfo(setupPath, "", "", "", "", false),
		build: &debug.BuildInfo{
			Main: debug.Module{Path: buildPath},
		},
		expectInfo: info.NewInfo(setupPath, "", "", "", "", false),
	},
	"build info build path": {
		info: info.NewInfo("", "", "", "", "", false),
		build: &debug.BuildInfo{
			Main: debug.Module{Path: buildPath},
		},
		expectInfo: info.NewInfo(buildPath, "", "", "", "", false),
	},

	// Setup build info version.
	"build info setup version": {
		info: info.NewInfo("", "v2.3.4", "beta.1", "", "", false),
		build: &debug.BuildInfo{
			Main: debug.Module{Version: "v1.2.3-alpha.1"},
		},
		expectInfo: info.NewInfo("", "v1.2.3-alpha.1", "alpha.1", "", "", false),
	},
	"build info build version": {
		info: info.NewInfo("", "", "", "", "", false),
		build: &debug.BuildInfo{
			Main: debug.Module{Version: "v1.2.3-alpha.1"},
		},
		expectInfo: info.NewInfo("", "v1.2.3-alpha.1", "alpha.1", "", "", false),
	},

	// Setup build info settings.
	"build info revision": {
		info: info.NewInfo("", "", "", "", "", false),
		build: &debug.BuildInfo{
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "beta.2"},
			},
		},
		expectInfo: info.NewInfo("", "", "beta.2", "", "", false),
	},
	"build info time": {
		info: info.NewInfo("", "", "", "", "", false),
		build: &debug.BuildInfo{
			Settings: []debug.BuildSetting{
				{Key: "vcs.time", Value: "2023-12-10T18:30:00Z"},
			},
		},
		expectInfo: info.NewInfo("", "", "", "", "2023-12-10T18:30:00Z", false),
	},
	"build info time revision": {
		info: info.NewInfo("", "", "", "", "", false),
		build: &debug.BuildInfo{
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "beta.2"},
				{Key: "vcs.time", Value: "2023-12-10T18:30:00Z"},
			},
		},
		expectInfo: info.NewInfo("", "", "beta.2", "",
			"2023-12-10T18:30:00Z", false),
	},
	"build info time hash": {
		info: info.NewInfo("", "", "", "", "", false),
		build: &debug.BuildInfo{
			Settings: []debug.BuildSetting{{
				Key:   "vcs.revision",
				Value: revisionHead,
			}, {Key: "vcs.time", Value: "2023-12-10T18:30:00Z"}},
		},
		expectInfo: info.NewInfo("", "", revisionHead, "",
			"2023-12-10T18:30:00Z", false),
	},
	"build info time hash setup": {
		info: info.NewInfo("", "v1.2.3", "", "", "", false),
		build: &debug.BuildInfo{
			Settings: []debug.BuildSetting{{
				Key:   "vcs.revision",
				Value: revisionHead,
			}, {Key: "vcs.time", Value: "2023-12-10T18:30:00Z"}},
		},
		expectInfo: info.NewInfo("", "v1.2.3", revisionHead, "",
			"2023-12-10T18:30:00Z", false),
	},
	"build info dirty": {
		info: info.NewInfo("", "", "", "", "", false),
		build: &debug.BuildInfo{
			Settings: []debug.BuildSetting{
				{Key: "vcs.modified", Value: "true"},
			},
		},
		expectInfo: info.NewInfo("", "", "", "", "", true),
	},
}

func TestUseDebug(t *testing.T) {
	test.Map(t, testInfoParams).
		Run(func(t test.Test, param InfoParams) {
			// When
			info := param.info.UseDebug(param.build, true).
				AdjustVersion()

			// Then
			assert.Equal(t, param.expectInfo, info)
			assert.Equal(t, param.expectInfo.String(), info.String())
		})
}
