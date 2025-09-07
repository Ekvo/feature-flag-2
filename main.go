package main

import (
	"context"
	"database/sql"
	"feature-flag-2/adapter/humafiberv3"
	"feature-flag-2/entity"
	"feature-flag-2/models"
	mycache "feature-flag-2/repository/cache"
	mydb "feature-flag-2/repository/db"
	"feature-flag-2/service"
	"fmt"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cache"
	"github.com/hashicorp/golang-lru/v2/expirable"
	_ "github.com/jackc/pgx/stdlib"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
	"log"
	"net/http"
	"time"
)

// GreetingOutput represents the response structure
type GreetingOutput struct {
	Body struct {
		Message string `json:"message" example:"Hello, world!"`
	}
}

type MyData struct {
	Data        string `json:"data"`
	DefaultData string `json:"default_data"`
}

type MessageDefault struct {
	Body struct {
		Some struct {
			X int `json:"its_int"`
		} `json:"some"`

		Data string `json:"default_data"`
	}
}

type MessageStandart struct {
	Data string `json:"data"`
}

func main() {
	dsn := "postgres://manager:qwert12345@localhost/manager?sslmode=disable"
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		log.Printf("main: db.Ping error - {%v}", err)
		return
	}
	reformDB := reform.NewDB(db, postgresql.Dialect, reform.NewPrintfLogger(log.Printf))
	repoDB := mydb.NewRepoFlagDB(reformDB)
	lru := expirable.NewLRU[string, models.Flag](1000, nil, 5*time.Minute)
	repoCache := mycache.NewRepoCacheFlag(lru)
	serviceFlag := service.NewServiceFlag(repoDB, repoCache)

	// Create a new Fiber app
	app := fiber.New()
	fcache := cache.New(cache.Config{
		Expiration:   30 * time.Minute,
		CacheControl: true,
		Methods:      []string{"GET"},
	})
	app.Group("/greetin").Use(fcache)

	config := huma.DefaultConfig("My API", "1.0.0")
	api := humafiberv3.New(app, config)

	// Register a GET operation
	huma.Register(api, huma.Operation{
		OperationID: "get-list-of-flags",
		Method:      "GET",
		Path:        "/flags",
		Summary:     "get list of flags and cached",
	}, func(ctx context.Context, input *struct{}) (*entity.ListOfFlagResponse, error) {
		flags, err := repoDB.ListOfAllFkags(ctx)
		if err != nil {
			return nil, err
		}

		resp := &entity.ListOfFlagResponse{}
		resp.Body.Flags = flags

		return resp, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "post-new-flag",
		Method:        "POST",
		DefaultStatus: 201,
		Path:          "/flag",
		Summary:       "create a new flag",
	}, func(ctx context.Context, input *struct {
		Body models.Flag `json:"body"`
	}) (*struct{ Body struct{} }, error) {
		flag := input.Body

		if err := serviceFlag.CreateNewFlag(ctx, flag); err != nil {
			return nil, huma.NewError(http.StatusConflict, err.Error())
		}

		return &struct{ Body struct{} }{}, nil
	})

	// Register a GET operation
	huma.Register(api, huma.Operation{
		OperationID: "get-greeting",
		Method:      "GET",
		Path:        "/greet/{name}",
		Summary:     "Cached greeting",
	}, func(ctx context.Context, input *struct {
		Name string `path:"name" maxLength:"30" example:"world"`
	}) (*GreetingOutput, error) {
		time.Sleep(2 * time.Second)
		resp := &GreetingOutput{}
		resp.Body.Message = fmt.Sprintf("no Hello, %s!", input.Name)
		//resp.Body.Time = time.Now().Format("15:04:05") // чтобы видеть, кешируется ли
		return resp, nil
	})

	// Start the server
	app.Listen(":8000")
}
