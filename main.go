package main

import (
	"context"
	"database/sql"
	"errors"
	"feature-flag-2/adapter/humafiberv3"
	"feature-flag-2/entity"
	_ "feature-flag-2/migrations"
	"feature-flag-2/models"
	mycache "feature-flag-2/repository/cache"
	mydb "feature-flag-2/repository/db"
	"feature-flag-2/service"
	"flag"
	"fmt"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cache"
	"github.com/hashicorp/golang-lru/v2/expirable"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/pressly/goose/v3"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
	"log"
	"net/http"
	"strings"
	"time"
)

// Флаги командной строки
var (
	action  = flag.String("action", "up", "Действие: up, down, up-to, down-to, status")
	version = flag.Int64("version", 0, "Версия миграции (для up-to, down-to)")
)

var ErrMainFlagNamesNotEqual = errors.New("flag name from param not equal falg name from body")

func main() {
	ctx := context.Background()
	dsn := "postgres://ekvo:qwert12345@localhost/ekvodb?sslmode=disable"
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		log.Printf("main: db.Ping error - {%v}", err)
		return
	}

	flag.Parse()
	// Логируем выбранный action
	log.Printf("▶️  Выбрано действие: %s", *action)

	switch *action {
	case "up":
		if err := goose.UpContext(ctx, db, "migrations"); err != nil {
			log.Printf("main: goose.Up err - {%v}", err)
			return
		}
	case "down":
		if err := goose.DownContext(ctx, db, "migrations"); err != nil {
			log.Printf("main: goose.Down err - {%v}", err)
			return
		}
	case "up-to":
		if *version == 0 {
			log.Printf("main: goose.UpTo err - {%v}", err)
			return
		}
		if err := goose.UpToContext(ctx, db, "migrations", *version); err != nil {
			log.Printf("main: goose.UpTo err - {%v}", err)
			return
		}
	case "down-to":
		if *version == 0 {
			log.Printf("main: goose.DownTo err - {%v}", err)
			return
		}
		if err := goose.DownToContext(ctx, db, "migrations", *version); err != nil {
			log.Printf("main: goose.DownTo err - {%v}", err)
			return
		}
	}
	lru := expirable.NewLRU[string, models.Flag](1000, nil, 5*time.Minute)
	reformDB := reform.NewDB(db, postgresql.Dialect, reform.NewPrintfLogger(log.Printf))
	repoDB := mydb.NewRepoFlagDB(reformDB, lru)

	repoCache := mycache.NewRepoCacheFlag(lru)
	serviceFlag := service.NewServiceFlag(repoDB, repoCache)

	// Create a new Fiber app
	app := fiber.New()
	fcache := cache.New(cache.Config{
		Expiration:   5 * time.Minute,
		CacheControl: true,
		Methods:      []string{"GET"},
	})
	app.Group("/flags").Use(fcache)

	api := humafiberv3.New(app, huma.DefaultConfig("feature Flags API", "1.0.0"))

	huma.Register(api, huma.Operation{
		OperationID: "get-list-of-flags",
		Method:      "PUT",
		Path:        "/flags/migrations",
		Summary:     "get list of flags and cached",
	}, func(ctx context.Context, input *struct{}) (*entity.ListOfFlagResponse, error) {
		flags, err := serviceFlag.RetrieveListOfAllFlags(ctx)
		if err != nil {
			return nil, huma.Error404NotFound("empty list of flags", err)
		}

		return flags, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-list-of-flags",
		Method:      "GET",
		Path:        "/flags",
		Summary:     "get list of flags and cached",
	}, func(ctx context.Context, input *struct{}) (*entity.ListOfFlagResponse, error) {
		flags, err := serviceFlag.RetrieveListOfAllFlags(ctx)
		if err != nil {
			return nil, huma.Error404NotFound("empty list of flags", err)
		}

		return flags, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "post-list-of-flags",
		Method:      "POST",
		Path:        "/flags",
		Summary:     "get list of flags by names and cached",
	}, func(ctx context.Context, input *struct {
		Body entity.FlagNamesDecode `json:"body"`
	}) (*entity.ListOfFlagResponse, error) {
		flagNames := input.Body
		flagsByNames, err := serviceFlag.RetrieveListOfFlagsByNames(ctx, flagNames.FlagNames)
		if err != nil {
			return nil, huma.Error404NotFound("empty list of flags1", err)
		}
		return flagsByNames, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "post-new-flag",
		Method:        "POST",
		DefaultStatus: 201,
		Path:          "/flag",
		Summary:       "create a new flag",
	}, func(ctx context.Context, input *struct {
		Body models.Flag `json:"body"`
	}) (*entity.FlagResponse, error) {
		newFlag := input.Body
		respFlag, err := serviceFlag.CreateNewFlag(ctx, newFlag)
		if err != nil {
			return nil, huma.NewError(http.StatusConflict, "flag was not created", err)
		}
		return respFlag, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-flag-by-name",
		Method:      "GET",
		Path:        "/flag/{name}",
		Summary:     "get flag name from param and return flag",
	}, func(ctx context.Context, input *struct {
		Name string `path:"name" maxLength:"30" example:"world"`
	}) (*entity.FlagResponse, error) {
		flagName := input.Name
		respFlag, err := serviceFlag.GetFlagByName(ctx, flagName)
		if err != nil {
			return nil, huma.Error404NotFound(
				fmt.Sprintf("flag by name {%s} - not found", flagName),
				err,
			)
		}
		return respFlag, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "put-update-flag-by-name",
		Method:      "PUT",
		Path:        "/flag/{name}",
		Summary:     "get flag name from param and return flag",
	}, func(ctx context.Context, input *struct {
		Name string      `path:"name"`
		Body models.Flag `json:"body"`
	}) (*entity.FlagResponse, error) {
		flagName := input.Name
		flagDecode := input.Body
		if strings.TrimSpace(flagName) != strings.TrimSpace(flagDecode.FlagName) {
			return nil, huma.Error400BadRequest("flag name is invalid", ErrMainFlagNamesNotEqual)
		}
		flagName = flagDecode.FlagName
		respFlag, err := serviceFlag.UpdateFlag(ctx, flagDecode)
		if err != nil {
			return nil, huma.Error404NotFound(fmt.Sprintf("flag by name {%s} - not found", flagName), err)
		}
		return respFlag, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-flag-by-name",
		Method:      "DELETE",
		Path:        "/flag/{name}",
		Summary:     "get flag name from param and return flag",
	}, func(ctx context.Context, input *struct {
		Name string `path:"name"`
	}) (*struct{}, error) {
		flagName := input.Name
		if err := serviceFlag.DeleteFlag(ctx, flagName); err != nil {
			return nil, huma.Error404NotFound(fmt.Sprintf("flag by name {%s} - not found", flagName), err)
		}
		return nil, nil
	})

	app.Listen(":8000")
}
