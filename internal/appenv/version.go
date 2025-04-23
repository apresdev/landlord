package appenv

import "os"

type VersionInfo struct {
	Commit    string
	BuildTime string
	GoVersion string
}

func (vi VersionInfo) Version() string {
	return TagVersion
}

var terraformVersion = "unset"

func init() {
	v := os.Getenv("LANDLORD_TERRAFORM_MODULE_VERSION")
	if v != "" {
		terraformVersion = v
	}
}

func (vi VersionInfo) TerraformVersion() string {
	return terraformVersion
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
