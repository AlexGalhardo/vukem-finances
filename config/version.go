package config

// Version is the release the binary was built from. The image build sets it
// with -ldflags "-X github.com/AlexGalhardo/vukem-finances/config.Version=...".
// It is shown by the about command and never by the public /up route, so a
// scanner cannot read the exact version.
var Version = "0.0.0-dev"
