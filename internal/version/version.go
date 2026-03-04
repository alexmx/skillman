package version

// Set via ldflags at build time:
//   go build -ldflags "-X github.com/alexmx/skillman/internal/version.Version=v1.0.0"
var Version = "dev"
