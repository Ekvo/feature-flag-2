package db

import (
	"context"
	"database/sql"
	"errors"
	"feature-flag-2/models"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"gopkg.in/reform.v1"
)

var (
	ErrDBAlreadyExists = errors.New("already exists")

	ErrDBNotFound = errors.New("not found")

	ErrDBIsDeleted = errors.New("is deleted")
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
func (r *RepoFlagDB) UpdateFlag(
	ctx context.Context,
	newFlag models.Flag,
) (models.Flag, error) {
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
		return newFlag, err
	}
	r.cache.Remove(newFlag.FlagName)
	return newFlag, nil
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
			return ErrDBIsDeleted
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
	listOfFlags, err := models.ConvertReformStructToModel[models.Flag](flags)
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
		listOfFlagsFromDB, err := models.ConvertReformStructToModel[models.Flag](flags)
		if err != nil {
			return nil, err
		}
		listOfFlags = append(listOfFlags, listOfFlagsFromDB...)
	}
	if len(listOfFlags) == 0 {
		return nil, ErrDBNotFound
	}
	if len(listOfFlags) != len(flagNames) {
		return nil, models.ErrorWithUnknownModelNames[models.Flag](flagNames, listOfFlags)
	}
	for _, flag := range listOfFlags {
		r.cache.Add(flag.FlagName, flag)
	}
	return listOfFlags, nil
}
