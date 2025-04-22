package appenv

type VersionInfo struct {
	GitTag    string
	Commit    string
	BuildTime string
	GoVersion string
}

func (vi VersionInfo) BuildYear() string {
	if len(vi.BuildTime) > 4 {
		return vi.BuildTime[:4]
	}
	return "2025"
}

var Version VersionInfo

func SetVersionInfo(v VersionInfo) {
	Version = v
}

func GetVersionInfo() VersionInfo {
	return Version
}
