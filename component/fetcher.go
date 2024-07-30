package component

import (
	"context"

	"github.com/anchordotdev/cli/component/models"
	"github.com/anchordotdev/cli/ui"
)

type Choosable = models.Choosable

type Fetcher[T Choosable] struct {
	FetchFn func() ([]T, error)
}

func (f *Fetcher[T]) Choices(ctx context.Context, drv *ui.Driver, flag string, creatable bool) ([]T, error) {
	var t T
	drv.Activate(ctx, &models.Fetcher[T]{
		Flag:      flag,
		Plural:    t.Plural(),
		Singular:  t.Singular(),
		Creatable: creatable,
	})

	items, err := f.FetchFn()
	if err != nil {
		return nil, err
	}

	drv.Send(items)
	return items, nil
}
