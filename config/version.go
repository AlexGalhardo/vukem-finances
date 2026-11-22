package config

import "runtime/debug"

const devVersion = "0.0.0-dev"

// Version is the release the binary was built from. The image build sets it
// with -ldflags "-X github.com/AlexGalhardo/vukem-finances/config.Version=...".
// It is shown by the about command and never by the public /up route, so a
// scanner cannot read the exact version.
var Version = devVersion

// DisplayVersion is what the screens show. A build with no release version adds
// its commit, so two development builds can be told apart.
func DisplayVersion() string {
	if Version != devVersion {
		return Version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return Version
	}

	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 7 {
			return Version + "+" + s.Value[:7]
		}
	}

	return Version
}
