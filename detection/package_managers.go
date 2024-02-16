package detection

type PackageManager string
type PackageManagerManifest string

var SupportedPackageManagers = []PackageManager{
	RubyGemsPkgManager,
	NPMPkgManager,
	YarnPkgManager,
}

const (
	RubyGemsPkgManager PackageManager = "rubygems"
	NPMPkgManager      PackageManager = "npm"
	YarnPkgManager     PackageManager = "yarn"
)

const (
	Gemfile         PackageManagerManifest = "Gemfile"
	GemfileLock     PackageManagerManifest = "Gemfile.lock"
	PackageJSON     PackageManagerManifest = "package.json"
	PackageLockJSON PackageManagerManifest = "package-lock.json"
	YarnLock        PackageManagerManifest = "yarn.lock"
)

func (pmm PackageManagerManifest) String() string {
	return string(pmm)
}

func (pm PackageManager) String() string {
	return string(pm)
}
