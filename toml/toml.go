package toml

import (
	"io"
	"reflect"
	"slices"

	"github.com/fatih/structtag"
	"github.com/mohae/deepcopy"
	"github.com/pelletier/go-toml/v2"
)

type Decoder = toml.Decoder

func NewDecoder(r io.Reader) *Decoder { return toml.NewDecoder(r) }

type Encoder[T any] struct {
	enc *toml.Encoder
}

func NewEncoder[T any](w io.Writer) *Encoder[T] {
	return &Encoder[T]{
		enc: toml.NewEncoder(w),
	}
}

func (e *Encoder[T]) Encode(v T) error {
	vv := deepcopy.Copy(v).(T)

	val := reflect.ValueOf(vv)
	if val.Kind() != reflect.Pointer {
		val = reflect.ValueOf(&vv)
	}

	if err := applyReadonlyOption(val); err != nil {
		return err
	}
	return e.enc.Encode(vv)
}

func applyReadonlyOption(val reflect.Value) error {
	if val.Kind() != reflect.Pointer && val.Elem().Kind() != reflect.Struct {
		return nil
	}

	typ := val.Elem().Type()
	for i := 0; i < val.Elem().NumField(); i++ {
		if typ.Field(i).Type.Kind() == reflect.Struct {
			if err := applyReadonlyOption(val.Elem().Field(i).Addr()); err != nil {
				return err
			}
		}
		if typ.Field(i).Type.Kind() == reflect.Pointer && !val.Elem().Field(i).IsZero() {
			if err := applyReadonlyOption(val.Elem().Field(i)); err != nil {
				return err
			}
		}

		tags, err := structtag.Parse(string(typ.Field(i).Tag))
		if err != nil {
			return err
		}

		tomlTag, err := tags.Get("toml")
		if tomlTag == nil {
			continue
		}
		if err != nil {
			return err
		}

		if slices.Contains(tomlTag.Options, "readonly") {
			val.Elem().Field(i).Set(reflect.Zero(typ.Field(i).Type))
		}
	}

	return nil
}
