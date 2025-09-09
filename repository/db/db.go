package db

import (
	"context"
	"database/sql"
	"errors"
	"feature-flag-2/models"
	"fmt"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"gopkg.in/reform.v1"
	"reflect"
	"strings"
)

var (
	ErrDBAlreadyExists = errors.New("already exists")

	ErrDBNotFound = errors.New("not found")

	ErrDBCreatedByIsNotEqual = errors.New("created_by from service is not equal created_by from base")

	ErrDBFlagIsDeleted = errors.New("flag is deleted")

	ErrDBInternal = errors.New("internal error")

	ErrDBRUnexpectrdType = errors.New("unexpected type")

	ErrDBUnknownFlag = errors.New("unknown flag")
)

type RepoFlagDB struct {
	db    *reform.DB
	cache *expirable.LRU[string, models.Flag]
}

func NewRepoFlagDB(
	db *reform.DB,
	cache *expirable.LRU[string, models.Flag],
) *RepoFlagDB {
	return &RepoFlagDB{db: db, cache: cache}
}

// Create создает новый флаг
func (r *RepoFlagDB) CreateFlag(ctx context.Context, newFlag models.Flag) error {
	exec := func(tx *reform.TX) error {
		var oldFlag models.Flag
		if err := tx.WithContext(ctx).SelectOneTo(
			&oldFlag,
			`WHERE flag_name = $1 FOR UPDATE`,
			newFlag.FlagName,
		); err != nil {
			return err
		}
		if !oldFlag.IsDeleted {
			return ErrDBAlreadyExists
		}
		if err := tx.WithContext(ctx).Update(&newFlag); err != nil {
			return err
		}
		r.cache.Remove(newFlag.FlagName)

		return nil
	}
	if err := r.db.InTransactionContext(ctx, nil, exec); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return r.db.WithContext(ctx).Insert(&newFlag)
		}
		return err
	}

	return nil
}

// GetByFlagName возвращает флаг по имени
func (r *RepoFlagDB) GetFlagByName(ctx context.Context, flagName string) (models.Flag, error) {
	flag, ok := r.cache.Get(flagName)
	if ok {
		return flag, nil
	}
	flag.FlagName = flagName
	if err := r.db.WithContext(ctx).FindByPrimaryKeyTo(&flag, flagName); err != nil {
		return flag, err
	}
	r.cache.Add(flagName, flag)

	return flag, nil
}

// Update обновляет флаг
func (r *RepoFlagDB) UpdateFlag(ctx context.Context, newFlag models.Flag) error {
	exec := func(tx *reform.TX) error {
		var oldFlag models.Flag
		if err := tx.WithContext(ctx).SelectOneTo(
			&oldFlag,
			`WHERE is_deleted = false AND flag_name = $1 FOR UPDATE`,
			newFlag.FlagName,
		); err != nil {
			return err
		}
		return tx.WithContext(ctx).Update(&newFlag)
	}
	if err := r.db.InTransactionContext(ctx, nil, exec); err != nil {
		return err
	}
	r.cache.Remove(newFlag.FlagName)

	return nil
}

// Delete удаляет флаг
func (r *RepoFlagDB) DeleteFlag(ctx context.Context, flagName string) error {
	exec := func(tx *reform.TX) error {
		var flagFromDB models.Flag
		if err := tx.WithContext(ctx).SelectOneTo(
			&flagFromDB,
			`WHERE flag_name = $1 FOR UPDATE`,
			flagName,
		); err != nil {
			return err
		}
		if flagFromDB.IsDeleted {
			return ErrDBNotFound
		}
		flagFromDB.IsDeleted = true
		return tx.WithContext(ctx).Update(&flagFromDB)
	}
	if err := r.db.InTransactionContext(ctx, nil, exec); err != nil {
		return err
	}
	r.cache.Remove(flagName)

	return nil
}

// ListOfAllFkags возвращает список всех флагов
func (r *RepoFlagDB) ListOfAllFlags(ctx context.Context) ([]models.Flag, error) {
	flags, err := r.db.WithContext(ctx).SelectAllFrom(models.FlagTable, "")
	if err != nil {
		return nil, err
	}

	listOfFlags, err := convertReformStructToFlag[models.Flag](flags)
	if err != nil {
		return nil, err
	}

	for _, flag := range listOfFlags {
		r.cache.Add(flag.FlagName, flag)
	}

	return listOfFlags, nil
}

func (r *RepoFlagDB) ListOfFlagByNames(
	ctx context.Context,
	flagNames []string,
) ([]models.Flag, error) {
	listOfFlags := make([]models.Flag, 0, len(flagNames))
	findFlagsByNamesFromDB := make([]string, 0, len(flagNames))

	for _, nameOfFlag := range flagNames {
		if flag, ok := r.cache.Get(nameOfFlag); ok {
			listOfFlags = append(listOfFlags, flag)
			continue
		}
		findFlagsByNamesFromDB = append(findFlagsByNamesFromDB, nameOfFlag)
	}
	if len(findFlagsByNamesFromDB) > 0 {
		args := make([]any, 0, len(findFlagsByNamesFromDB))
		for _, name := range findFlagsByNamesFromDB {
			args = append(args, name)
		}
		flags, err := r.db.WithContext(ctx).FindAllFrom(models.FlagTable, "flag_name", args...)
		if err != nil {
			return nil, err
		}
		listOfFlagsFromDB, err := convertReformStructToFlag[models.Flag](flags)
		if err != nil {
			return nil, err
		}
		listOfFlags = append(listOfFlags, listOfFlagsFromDB...)
	}
	if len(listOfFlags) == 0 {
		return nil, ErrDBNotFound
	}
	if len(listOfFlags) != len(flagNames) {
		return nil, errorWithUnknownFlags(flagNames, listOfFlags)
	}
	for _, flag := range listOfFlags {
		r.cache.Add(flag.FlagName, flag)
	}

	return listOfFlags, nil
}

func errorWithUnknownFlags(
	uniqFlagsNames []string,
	listOfFlags []models.Flag,
) error {
	unknownFlags := []string{}
link:
	for _, flagName := range uniqFlagsNames {
		for _, flag := range listOfFlags {
			if flag.FlagName == flagName {
				continue link
			}
		}
		unknownFlags = append(unknownFlags, flagName)
	}
	return fmt.Errorf(
		"error - {%v}, flagNames - {%s}",
		ErrDBUnknownFlag,
		strings.Join(unknownFlags, ", "),
	)
}

// convertReformStructToFlag создаем массив флагов из []reform.Struct
func convertReformStructToFlag[T models.Models](dataFromDB []reform.Struct) ([]T, error) {
	var t T
	if reflect.ValueOf(t).Kind() == reflect.Ptr {
		return nil, models.ErrDBModelShouldNotBePointer
	}
	expectedPtrType := reflect.TypeOf((*T)(nil))
	listOfFlags := make([]T, 0, len(dataFromDB))
	for _, f := range dataFromDB {
		fVal := reflect.ValueOf(f)
		if fVal.Type() != expectedPtrType {
			return nil, ErrDBRUnexpectrdType
		}
		flag := fVal.Elem().Interface().(T)
		listOfFlags = append(listOfFlags, flag)
	}
	return listOfFlags, nil
}
