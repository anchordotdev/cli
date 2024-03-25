package version

import (
	"fmt"
	"runtime"
)

var info = struct {
	version, commit, date string

	os, arch string
}{
	version: "dev",
	commit:  "none",
	date:    "unknown",
	os:      runtime.GOOS,
	arch:    runtime.GOARCH,
}

func Set(version, commit, date string) {
	info.version = version
	info.commit = commit
	info.date = date
}

func String() string {
	return fmt.Sprintf("%s (%s/%s) Commit: %s BuildDate: %s", info.version, info.os, info.arch, info.commit, info.date)
}

func UserAgent() string {
	return "Anchor CLI " + String()
}
