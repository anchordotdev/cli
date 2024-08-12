package truststore

import "slices"

var MockCAs []*CA

func ResetMockCAs() { MockCAs = []*CA{} }

type Mock struct{}

func (Mock) Check() (bool, error) { return true, nil }

func (Mock) CheckCA(ca *CA) (bool, error) {
	for _, ca2 := range MockCAs {
		if ca.UniqueName == ca2.UniqueName {
			return true, nil
		}
	}

	return false, nil
}

func (Mock) Description() string { return "Mock" }

func (Mock) InstallCA(ca *CA) (bool, error) {
	MockCAs = append(MockCAs, ca)
	return true, nil
}

func (Mock) ListCAs() ([]*CA, error) {
	return MockCAs, nil
}

func (Mock) UninstallCA(ca *CA) (bool, error) {
	MockCAs = slices.DeleteFunc(MockCAs, func(ca2 *CA) bool { return ca.Equal(ca2) })
	return true, nil
}
