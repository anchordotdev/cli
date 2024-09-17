package clipboard

import (
	"sync/atomic"

	"github.com/atotto/clipboard"
)

type Mock atomic.Pointer[string]

func (m *Mock) ReadAll() (string, error) {
	if ptr := ((*atomic.Pointer[string])(m)).Load(); ptr != nil {
		return *ptr, nil
	}
	return "", nil
}

func (m *Mock) WriteAll(text string) error {
	((*atomic.Pointer[string])(m)).Store(&text)
	return nil
}

var System = system{}

type system struct{}

func (system) ReadAll() (string, error) { return clipboard.ReadAll() }

func (system) WriteAll(text string) error { return clipboard.WriteAll(text) }
