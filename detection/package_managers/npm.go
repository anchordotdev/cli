package package_managers

import "encoding/json"

type PackageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func (p PackageJSON) HasDependency(name string) bool {
	if _, ok := p.Dependencies[name]; ok {
		return true
	} else if _, ok := p.DevDependencies[name]; ok {
		return true
	}

	return false
}

func ParsePackageJSON(contents []byte) (PackageJSON, error) {
	var packageJSON PackageJSON
	err := json.Unmarshal(contents, &packageJSON)
	return packageJSON, err
}
