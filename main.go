package main

import (
	"context"
	"database/sql"
	"errors"
	"feature-flag-2/adapter/humafiberv3"
	"feature-flag-2/config"
	"feature-flag-2/entity"
	_ "feature-flag-2/migrations"
	"feature-flag-2/models"
	mydb "feature-flag-2/repository/db"
	"feature-flag-2/service"
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
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	ErrMainFlagNamesNotEqual        = errors.New("flag name from param not equal falg name from body")
	ErrMainInvalidActionOfMigration = errors.New("invalid action of migration")
)

func main() {
	cfg, err := config.NewConfig(`./.env`)
	if err != nil {
		log.Fatal("Failed config", err)
	}
	//dsn := "postgres://ekvo:qwert12345@localhost/ekvodb?sslmode=disable"
	db, err := sql.Open("pgx", cfg.DB.URL)
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		log.Printf("main: db.Ping error - {%v}", err)
		return
	}

	ctx := context.Background()

	if err := doMigrations(ctx, &cfg.Migrations, db); err != nil {
		log.Printf("main: doMigrations error - {%v}", err)
		return
	}
	lru := expirable.NewLRU[string, models.Flag](cfg.Cache.SizeLRU, nil, cfg.Cache.TTLLRU)
	reformDB := reform.NewDB(db, postgresql.Dialect, reform.NewPrintfLogger(log.Printf))
	repoDB := mydb.NewRepoFlagDB(reformDB, lru)
	serviceFlag := service.NewServiceFlag(repoDB)

	// Create a new Fiber app
	app := fiber.New()
	fcache := cache.New(cache.Config{
		Expiration:   cfg.Cache.TTLMiddlewareFiber,
		CacheControl: true,
		Methods:      []string{"GET"},
	})
	app.Group("/flags").Use(fcache)
	api := humafiberv3.New(app, huma.DefaultConfig("feature Flags API", "1.0.0"))

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
		Summary:     "get list of flags by names",
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
		OperationID: "put-flag-by-name",
		Method:      "PUT",
		Path:        "/flag/{name}",
		Summary:     "get flag name from param and return flag after update",
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
		Summary:     "get flag name from param and delete",
	}, func(ctx context.Context, input *struct {
		Name string `path:"name"`
	}) (*struct{}, error) {
		flagName := input.Name
		if err := serviceFlag.DeleteFlag(ctx, flagName); err != nil {
			return nil, huma.Error404NotFound(fmt.Sprintf("flag by name {%s} - not found", flagName), err)
		}
		return nil, nil
	})

	go func() {
		addr := net.JoinHostPort(cfg.Server.Host, cfg.Server.Port)
		if err := app.Listen(addr); !errors.Is(err, http.ErrServerClosed) {
			db.Close()
			log.Fatalf("failed to listen: %v", err)
		}
	}()

	chStop := make(chan os.Signal, 3)
	signal.Notify(chStop,
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
	)

	<-chStop

	log.Println("Получен сигнал завершения, останавливаем сервер...")

	ctx, cancel := context.WithTimeout(ctx, cfg.Server.ShutDown)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("main: app.Shutdown error - {%v}", err)
		return
	}

	log.Println("Сервер остановлен корректно.")
}

func doMigrations(ctx context.Context, cfg *config.MigrationConfig, db *sql.DB) error {
	action := cfg.Action
	dirOfMigrations := cfg.PathToMigrations
	version := cfg.Version

	if version == 0 && (action == "up-to" || action == "down-to") {
		return ErrMainInvalidActionOfMigration
	}

	switch action {
	case "up":
		if err := goose.UpContext(ctx, db, dirOfMigrations); err != nil {
			return err
		}
	case "down":
		if err := goose.DownContext(ctx, db, dirOfMigrations); err != nil {
			return err
		}
	case "up-to":
		if err := goose.UpToContext(ctx, db, dirOfMigrations, version); err != nil {
			return err
		}
	case "down-to":
		if err := goose.DownToContext(ctx, db, dirOfMigrations, version); err != nil {
			return err
		}
	default:
		return ErrMainInvalidActionOfMigration
	}
	return nil
}
