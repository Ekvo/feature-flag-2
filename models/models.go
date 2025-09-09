package models

import (
	"errors"
	"fmt"
	"gopkg.in/reform.v1"
	"reflect"
	"strings"
)

var (
	ErrModelsModelShouldNotBePointer = errors.New("model should not be a pointer")

	ErrModelsRUnexpectrdType = errors.New("unexpected model type")

	ErrModelsUnknownModel = errors.New("unknown model name")
)

type models interface {
	GetModelName() string
}

// ConvertReformStructToModel создаем массив флагов из []reform.Struct
func ConvertReformStructToModel[T models](dataFromDB []reform.Struct) ([]T, error) {
	var t T
	if reflect.ValueOf(t).Kind() == reflect.Ptr {
		return nil, ErrModelsModelShouldNotBePointer
	}
	expectedPtrType := reflect.TypeOf((*T)(nil))
	listOfModels := make([]T, 0, len(dataFromDB))
	for _, f := range dataFromDB {
		fVal := reflect.ValueOf(f)
		if fVal.Type() != expectedPtrType {
			return nil, ErrModelsRUnexpectrdType
		}
		flag := fVal.Elem().Interface().(T)
		listOfModels = append(listOfModels, flag)
	}
	return listOfModels, nil
}

// ErrorWithUnknownModelNames -ошибка во время поиска models с неизвестными именами
func ErrorWithUnknownModelNames[T models](
	uniqModelNames []string,
	listOfModels []T,
) error {
	unknownModelNames := []string{}
link:
	for _, modelName := range uniqModelNames {
		for _, model := range listOfModels {
			if model.GetModelName() == modelName {
				continue link
			}
		}
		unknownModelNames = append(unknownModelNames, modelName)
	}
	return fmt.Errorf(
		"error - {%v}, flagNames - {%s}",
		ErrModelsUnknownModel,
		strings.Join(unknownModelNames, ", "),
	)
}
