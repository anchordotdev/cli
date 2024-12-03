package component

import (
	"context"
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/anchordotdev/cli/component/models"
	"github.com/anchordotdev/cli/ui"
)

type Selector[T Choosable] struct {
	Prompt string
	Flag   string

	Creatable bool

	Choices []T
	Fetcher *Fetcher[T]
}

func (s *Selector[T]) Choice(ctx context.Context, drv *ui.Driver) (*T, error) {
	if s.Fetcher != nil {
		var err error
		if s.Choices, err = s.Fetcher.Choices(ctx, drv, s.Flag, s.Creatable); err != nil {
			return nil, err
		}

		if len(s.Choices) == 1 && !s.Creatable {
			return &s.Choices[0], nil
		}
		if len(s.Choices) == 0 {
			if s.Creatable {
				return nil, nil
			}

			var t T
			return nil, ui.Error{
				Model: ui.Section{
					Name: "FetcherNoneFound",
					Model: ui.MessageLines{
						ui.Header(fmt.Sprintf("%s %s",
							ui.Danger("Error!"),
							fmt.Sprintf("Cannot proceed without %s.", t.Singular()),
						)),
					},
				},
			}
		}
	}

	var choices []ui.ListItem[T]
	for _, item := range s.Choices {
		choice := ui.ListItem[T]{
			Key: item.Key(),
			String: fmt.Sprintf("%s %s",
				item.String(),
				ui.Whisper(fmt.Sprintf("(%s)", item.Key())),
			),
			Value: item,
		}
		choices = append(choices, choice)
	}
	if s.Creatable {
		var t T
		choices = append(choices, ui.ListItem[T]{
			String: fmt.Sprintf("Create New %s", cases.Title(language.English).String(t.Singular())),
		})
	}

	choicec := make(chan T)
	mdl := &models.Selector[T]{
		Prompt: s.Prompt,
		Flag:   s.Flag,

		Choices:  choices,
		ChoiceCh: choicec,
	}
	drv.Activate(ctx, mdl)

	select {
	case choice := <-choicec:
		return &choice, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
