# Проектирование сервиса Feature Flags

## Этап разработки репозиториев, маршрутизатора, адаптера humafiber для fiber/v3.

### SQL БД работаем через ORM c cache
* Задача - хранение данных флагов

* PostgreSQL
* gopkg.in/reform.v1 -> ORM
* jackc/pgx v3.6.2   -> драйвер
* golang-lru/v2


**Модель флага для команды:** `go generate`
```go
//reform:public.flags
type Flag struct {
    FlagName    string    `json:"flag_name" reform:"flag_name,pk"`
    IsDeleted   bool      `json:"is_deleted" reform:"is_deleted"`
    IsEnabled   bool      `json:"is_enabled" reform:"is_enabled"`
    ActiveFrom  time.Time `json:"active_from" reform:"active_from"`
    Data        JSONmap   `json:"data" reform:"data"`
    DefaultData JSONmap   `json:"default_data" reform:"default_data"`
    CreatedBy   uuid.UUID `json:"created_by" reform:"created_by"`
    CreatedAt   time.Time `json:"created_at" reform:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" reform:"updated_at"`
}

// JSONmap работа SQL с map 
type JSONmap map[string]any

// map в []byte для sql
func (jm JSONmap) Value() (driver.Value, error)

// из sql данных []byte в map 
func (jm *JSONmap) Scan(value any) error

// Models сущности для дженериков, при приобразовании данных из БД в определнный тип
// []reform.Struct -> models.models
type models interface {
    GetModelName() string
}

// ConvertReformStructToModel дженерик
// перобразуем []reform.Struct в []models.Models(./models/models.go)
// models.Models - может быть только объектом, не указателем! 
// проверяем какой к нам пришел T если is_ptr -> return err 
// получаем тип указателя(reflect.TypeOf((*T)(nil))) для проверки объекта из ответа БД
// в цикле 
/*
   for _, f := range dataFromDB {
       fVal := reflect.ValueOf(f)
       if (Type of 'f' != expected ptr type ) {
           return nil, err
       }
       flag := fVal.Elem().Interface().(T)
       пишем флаг в массив
   }
*/
// все ok -> пишем флаги в кеш
// return массив флагов
func ConvertReformStructToModel[T Models](dataFromDB []reform.Struct) ([]T,error)

// ErrorWithUnknownModelNames - дженерик
// создаем массив для неизвестных имен
// for uniqModelNames { for listOfModels { пишем в массив имена uniqModelNames } }
// делаем fmt.Errorf("error - {%v}, flagNames - {%s}") - в шаблон добавляем: костомную ошибку с описанием и строку "alien1, alien2"
func ErrorWithUnknownModelNames[T models](
    uniqModelNames []string,
    listOfModels []T,
) error
```

**Репозитори БД и методы**
```go
type RepoFlagDB struct {
    db    *reform.DB
    cache *expirable.LRU[string, models.Flag]
}

// NewRepoFlagDB - конструктор
func NewRepoFlagDB(
    db *reform.DB,
    cache *expirable.LRU[string, models.Flag],
) *RepoFlagDB

// CreateFlag создает новый флаг
// подводные камни:
// мы должны создавать флаг даже если он существует в базе -> со статусом (is_deleted = true)
// если флаг есть, надо залочить строку (FOR UPDATE)
// создаем функцию exec для транзакции (reform.DB.InTransactionContext)
/*
exec := func(tx *reform.TX) error{
    var oldFlag models.Flag
    идем в базу tx.WithContext(ctx).SelectOneTo  с tail = WHERE flag_name = $1 FOR UPDATE
    если ошибка -> возвращаем ошибку
    или мы НАШЛИ флаг 
    if !oldFlag.IsDeleted -> return Err Already Exists    
    производим полное обновление флага -> tx.WithContext(ctx).Update(&newFlag)
    удаляем из кеш (на всякий)
    return nil  
   }
 */
// вызываем InTransactionContext(ctx, nil, exec)
// если ошибка -> if errors.Is(err, sql.ErrNoRows) -> делаем запись в базу WithContext(ctx).Insert(&newFlag)
// все ok -> return nil
func (rb *RepoFlag) CreateFlag(ctx context.Context, flag models.Flag) error

// GetByFlagName возвращает флаг по имени
// идем в кеш, нашли -> return флаг
// сетаем имя флага в пустой флаг из кеша, идем reform.Querier.FindByPrimaryKeyTo
// если нет ошибок -> пишем в кеш
// все ok -> return flag, nil
func (rb *RepoFlag) GetFlagByName(ctx context.Context, flagName string) (models.Flag, error)

// UpdateFlag обновляет флаг
// подводные камни:
// если флаг есть, надо залочить строку (FOR UPDATE)
// создаем функцию exec для транзакции (reform.DB.InTransactionContext)
/*
   exec := func(tx *reform.TX) error{
       var oldFlag models.Flag
       идем в базу tx.WithContext(ctx).SelectOneTo  с tail = WHERE is_deleted = false AND flag_name = $1 FOR UPDATE
       если ошибка -> возвращаем ошибку
       или мы НАШЛИ флаг
       Здесь в будуюшем нужно будет провалидировать данные (например: сравнить время создания, uuid createdBy)
       производим полное обновление флага ->
       return tx.WithContext(ctx).Update(&newFlag)
    }
*/
// вызываем InTransactionContext(ctx, nil, exec)
// если ошибка -> возвращаем ошибку
// все ok -> удаляем флаг из кеш
// return flag, nil
func (r *RepoFlagDB) UpdateFlag(
    ctx context.Context,
    newFlag models.Flag,
) (models.Flag, error)

// DeleteFlag обновляет флаг
// подводные камни:
// 1. мы должны поменять статус в таблице is_deleted на true, при условии что флаг есть и is_deleted = false
// 2. если флаг есть, надо залочить строку (FOR UPDATE)
// создаем функцию exec для транзакции (reform.DB.InTransactionContext)
/*
   exec := func(tx *reform.TX) error{
       var flagFromDB models.Flag
       идем в базу tx.WithContext(ctx).SelectOneTo  с tail = WHERE flag_name = $1 FOR UPDATE
       если ошибка -> возвращаем ошибку
       или мы НАШЛИ флаг
       if flagFromDB.IsDeleted  -> return Err Not Found
       flagFromDB.IsDeleted = true и делаем обновление    
       return tx.WithContext(ctx).Update(&flagFromDB)
    }
*/
// вызываем InTransactionContext(ctx, nil, exec)
// если ошибка -> возвращаем ошибку
// все ok -> удаляем флаг из кеш
// return nil
func (rb *RepoFlag) DeleteFlag(ctx context.Context, flagName string) error

// ListOfAllFkags возвращает список всех флагов
// вызываем reform.Querier.SelectAllFrom(models.FlagTable,"")
// вызываем 'models.convertReformStructToFlag[models.Flag]'
// если ошибка -> возвращаем ошибку
// все ok -> пишем флаги в кеш
// return массив флагов
func (rb *RepoFlag) ListOfAllFlags(ctx context.Context) ([]models.Flag, error)

// ListOfFkagByNames возвращает список флагов по списку имен
// делаем массивы для ответа(listOfFlags), и массив с именами для запроса в БД(findFlagsByNamesFromDB)
// идем в кеш и в цикле по имена флагов -> если есть пишем в listOfFlags, если нет пишем имя флага в findFlagsByNamesFromDB
// длина findFlagsByNamesFromDB > 0 -> идем в БД  -> если ошибка return err
// преобразовываем flagNames []string в []any 
// вызываем reform.Querier.Delete.FindAllFrom9(models.FlagTable, "flag_name", args...)
// если ошибка return err
// вызываем 'models.ConvertReformStructToModels[models.Flag]'
// если ошибка return err
// объединяем данные из кеш и базы
// длина полученного списка(listOfFlags) == 0, делаем ошибку (Not found)
// if len(listOfFlags) != len(flagNames), т.е. у нас есть unknown flags -> делаем ошибку в 'errorWithUnknownFlags'
// все ok -> пишем флаги в кеш
// return массив флагов
func (rb *RepoFlag) ListOfFkagByNames(ctx context.Context, flagNames []string) ([]models.Flag, error)
```

### fiber, middleware/cache
* Задача - реализовать кеширование для маршрутизатора

* fiber/v3
* fiber/v3/middleware/cache"

```go
// выбрал так как он прост и понятен,
// пришел к этому во время создания репозитория(получилась - ехидна) 7 строчек превратились в пакет, это тратило бы время других людей, чтобы разобраться

// в дальнейшем буду сетать поля для middlewareCache через конфиг
// вызываем в main
app := fiber.New()
middlewareCache := cache.New(cache.Config{
    Expiration:   5 * time.Minute, 
    CacheControl: true,
    Methods:      []string{"GET"},// кешировать весь список флагов  
})
// кешируем в middlewareCache только по пути /flags - пишем в кеш только весь список флагов
app.Group("/flags").Use(middlewareCache)
```

### Сущности для Request & Response
* Задача - реализовать кеширование для маршрутизатора

* fiber/v3
* fiber/v3/middleware/cache"

```go
// FlagResponse - формат для ответа флага
type FlagResponse struct {
	Body struct {
		Flag models.Flag `json:"flag"`
	}
}
// NewFlagResponse - конструктор
func NewFlagResponse(flag models.Flag) *FlagResponse

// ListOfFlagResponse - формат для ответа списка флагов
type ListOfFlagResponse struct {
	Body struct {
		Flags []models.Flag `json:"flags"`
	}
}

// NewListOfFlagResponse - конструктор
func NewListOfFlagResponse(flags []models.Flag) *ListOfFlagResponse

// FlagNamesDecode для получения списков имен флагов
type FlagNamesDecode struct {
    FlagNames []string `json:"flag_names"`
}
```

### Service - ServiceFlag
* Задача - логика обработки объекта флаг

```go
// бизнес логика - обработки флага
// в начале ищем в кеш, потом идем в базу
// при запросах на получение models.Flag пишем в кеш
type ServiceFlag struct {
	repoDB    *db.RepoFlagDB
}

// NewServiceFlag - конструктор
func NewServiceFlag(db *db.RepoFlagDB) *ServiceFlag

// CreateNewFlag - создания флага
// если нет -> идем в репозиторий flag.CreateFlag передаем newFlag  -> если ошибка, return err
// если все ok -> делаем и возвращаем ответ с данными из flag репозитория 
func (sf *ServiceFlag) CreateNewFlag(
    ctx context.Context,
    newFlag models.Flag,
) (*entity.FlagResponse, error) {

// GetFlagByName - получение флага
// идем в flag.GetFlagByName репозиторий c flagName -> если ошибка, return err
// делаем и возвращаем объект для ответ с данными флага из репозитория flag
func (sf *ServiceFlag) GetFlagByName(
    ctx context.Context,
    flagName string,
) (*entity.FlagResponse, error)

// UpdateFlag - обновление флага
// идем в flag.UpdateFlag репозиторий c newFlag -> если ошибка, return err
// делаем и возвращаем объект для ответ с данными флага из репозитория flag
func (sf *ServiceFlag) UpdateFlag(
    ctx context.Context,
    newFlag models.Flag,
) (*entity.FlagResponse, error)

// DeleteFlag - удаление флага
// идем в flag.DeleteFlag репозиторий с именем флага -> если ошибка, return err
// return nil
func (sf *ServiceFlag) DeleteFlag(
    ctx context.Context,
    flagName string,
) (error)

// RetrieveListOfAllFlags - получение всех флагов
// идем в flag.ListOfAllFlags репозиторий -> если ошибка, return err
// делаем и возвращаем объект для ответ с данными флагов
func (sf *ServiceFlag) RetrieveListOfAllFlags(ctx context.Context) (*entity.ListOfFlagResponse, error)

// RetrieveListOfAllFlags - получение флагов по списку имен
// делаем массив с уникальными именами utils.UniqWords (./utils/utils.go)
// идем в flag.ListOfFlagByNames репозиторий -> если ошибка, return err
// делаем и возвращаем объект для ответа с данными флагов
func (sf *ServiceFlag) RetrieveListOfFlagsByNames(
    ctx context.Context,
    flagNames []string,
) (*entity.ListOfFlagResponse, error)
```

### huma 
* Задача - логика контрактов, генерации OpenAPI

* huma/v2

```go
// работаем с флагами через service.ServiceFlag
api := humafiberv3.New(app, huma.DefaultConfig("Feature Flags API", "1.0.0"))

// Контроллер для получения полного списка флагов

// получаем список флагов через service.RetrieveListOfAllFlags 
// если ошибка делаем huma.Error404NotFound
// все хорошо отдаем ответ полученный из сервисного слоя
huma.Register(api, huma.Operation{
    OperationID: "get-list-of-flags",
    Method:      "GET",
    Path:        "/flags",
    Summary:     "get list of flags and cached",
}, func(ctx context.Context, input *struct{}) (*entity.ListOfFlagResponse, error)

// Контроллер получения списка флагов по именам

// получаем имена флагов через input.Body
// идем в service.RetrieveListOfFlagsByNames со списком имен 
// если ошибка делаем huma.Error404NotFound
// все хорошо отдаем ответ полученный из сервисного слоя
huma.Register(api, huma.Operation{
    OperationID: "post-list-of-flags",
    Method:      "POST",
    Path:        "/flags",
    Summary:     "get list of flags by names",
}, func(ctx context.Context, input *struct {
    Body entity.FlagNamesDecode `json:"body"`
}) (*entity.ListOfFlagResponse, error) {

// Контроллер создания флага

// получаем флаг через input.Body
// идем в service.CreateNewFlag с полученным флагом
// если ошибка делаем huma.NewError
// все хорошо отдаем ответ полученный из сервисного слоя (новый флаг)
huma.Register(api, huma.Operation{
    OperationID:   "post-new-flag",
    Method:        "POST",
    DefaultStatus: 201,
    Path:          "/flag",
    Summary:       "create a new flag",
}, func(ctx context.Context, input *struct {
    Body models.Flag `json:"body"`
}) (*entity.FlagResponse, error)

// Контроллер получения флага по имени 

// получаем имя флага из параметра
// идем в service.GetFlagByName с полученным именем флага
// если ошибка делаем huma.Error404NotFound
// все хорошо отдаем ответ полученный из сервисного слоя
huma.Register(api, huma.Operation{
    OperationID: "get-flag-by-name",
    Method:      "GET",
    Path:        "/flag/{name}",
    Summary:     "get flag name from param and return flag",
}, func(ctx context.Context, input *struct {
    Name string `path:"name" maxLength:"30" example:"world"`
}) (*entity.FlagResponse, error)

// Контроллер для изменения флага по имени 

// получаем имя флага из параметра и сам флаг 
// сравниваем имя из параметра и имя из объекта флага
// если не равно -> делаем  huma.Error400BadRequest
// идем в service.UpdateFlag с полученным флагов и пытаемся обновить данные
// если ошибка делаем huma.Error404NotFound
// все хорошо отдаем ответ полученный из сервисного слоя (обвноленный флаг)
huma.Register(api, huma.Operation{
    OperationID: "put-flag-by-name",
    Method:      "PUT",
    Path:        "/flag/{name}",
    Summary:     "get flag name from param and return flag after update",
}, func(ctx context.Context, input *struct {
    Name string      `path:"name"`
    Body models.Flag `json:"body"`
}) (*entity.FlagResponse, error)

// Контроллер для удаления флага по имени 

// получаем имя флага из параметра
// идем в service.DeleteFlag с полученным флагом и пытаемся удалить
// если ошибка делаем huma.Error404NotFound
// все хорошо отдаем nil
huma.Register(api, huma.Operation{
    OperationID: "delete-flag-by-name",
    Method:      "DELETE",
    Path:        "/flag/{name}",
    Summary:     "get flag name from param and delete",
}, func(ctx context.Context, input *struct {
    Name string `path:"name"`
}) (*struct{}, error) 
```

### humafiberv3 
* Задача - реализация адаптера из humafiber для fiber/v3

* huma/v2/adapters/humafiber

** **
```go
package humafiberv3

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
)

// Unwrap extracts the underlying Fiber context from a Huma context. If passed a
// context from a different adapter it will panic. Keep in mind the limitations
// of the underlying Fiber/fasthttp libraries and how that impacts
// memory-safety: https://docs.gofiber.io/#zero-allocation. Do not keep
// references to the underlying context or its values!
func Unwrap(ctx huma.Context) fiber.Ctx {
	for {
		if c, ok := ctx.(interface{ Unwrap() huma.Context }); ok {
			ctx = c.Unwrap()
			continue
		}
		break
	}
	if c, ok := ctx.(*fiberWrapper); ok {
		return c.Unwrap()
	}
	panic("not a humafiber context")
}

type fiberAdapter struct {
	tester requestTester
	router router
}

type fiberWrapper struct {
	op     *huma.Operation
	status int
	orig   fiber.Ctx // с указателя ни интерфейс
	ctx    context.Context
}

// check that fiberCtx implements huma.Context
var _ huma.Context = &fiberWrapper{}

func (c *fiberWrapper) Unwrap() fiber.Ctx {
	return c.orig
}

func (c *fiberWrapper) Operation() *huma.Operation {
	return c.op
}

func (c *fiberWrapper) Matched() string {
	return c.orig.Path()
}

func (c *fiberWrapper) Context() context.Context {
	return c.ctx
}

func (c *fiberWrapper) Method() string {
	return c.orig.Method()
}

func (c *fiberWrapper) Host() string {
	return c.orig.Hostname()
}

func (c *fiberWrapper) RemoteAddr() string {
	return c.orig.RequestCtx().RemoteAddr().String() // Context().RemoteAddr().String()
}

func (c *fiberWrapper) URL() url.URL {
	u, _ := url.Parse(string(c.orig.Request().RequestURI()))
	return *u
}

func (c *fiberWrapper) Param(name string) string {
	return c.orig.Params(name)
}

func (c *fiberWrapper) Query(name string) string {
	return c.orig.Query(name)
}

func (c *fiberWrapper) Header(name string) string {
	return c.orig.Get(name)
}

func (c *fiberWrapper) EachHeader(cb func(name, value string)) {
	c.orig.Request().Header.VisitAll(func(k, v []byte) {
		cb(string(k), string(v))
	})
}

func (c *fiberWrapper) BodyReader() io.Reader {
	var orig = c.orig
	if orig.App().Server().StreamRequestBody {
		// Streaming is enabled, so send the reader.
		return orig.Request().BodyStream()
	}
	return bytes.NewReader(orig.BodyRaw())
}

func (c *fiberWrapper) GetMultipartForm() (*multipart.Form, error) {
	return c.orig.MultipartForm()
}

func (c *fiberWrapper) SetReadDeadline(deadline time.Time) error {
	// Note: for this to work properly you need to do two things:
	// 1. Set the Fiber app's `StreamRequestBody` to `true`
	// 2. Set the Fiber app's `BodyLimit` to some small value like `1`
	// Fiber will only call the request handler for streaming once the limit is
	// reached. This is annoying but currently how things work.
	return c.orig.RequestCtx().Conn().SetReadDeadline(deadline) // Context().Conn().SetReadDeadline(deadline)
}

func (c *fiberWrapper) SetStatus(code int) {
	var orig = c.orig
	c.status = code
	orig.Status(code)
}

func (c *fiberWrapper) Status() int {
	return c.status
}
func (c *fiberWrapper) AppendHeader(name string, value string) {
	c.orig.Append(name, value)
}

func (c *fiberWrapper) SetHeader(name string, value string) {
	c.orig.Set(name, value)
}

func (c *fiberWrapper) BodyWriter() io.Writer {
	return c.orig
}

func (c *fiberWrapper) TLS() *tls.ConnectionState {
	return c.orig.RequestCtx().TLSConnectionState() // Context().TLSConnectionState()
}

func (c *fiberWrapper) Version() huma.ProtoVersion {
	return huma.ProtoVersion{
		Proto: c.orig.Protocol(),
	}
}

type router interface {
	Add(methods []string, path string, handler fiber.Handler, middleware ...fiber.Handler) fiber.Router
}

type requestTester interface {
	Test(req *http.Request, config ...fiber.TestConfig) (*http.Response, error)
}

type contextWrapperValue struct {
	Key   any
	Value any
}

type contextWrapper struct {
	values []*contextWrapperValue
	context.Context
}

var (
	_ context.Context = &contextWrapper{}
)

func (c *contextWrapper) Value(key any) any {
	var raw = c.Context.Value(key)
	if raw != nil {
		return raw
	}
	for _, pair := range c.values {
		if pair.Key == key {
			return pair.Value
		}
	}
	return nil
}

func (a *fiberAdapter) Handle(op *huma.Operation, handler func(huma.Context)) {
	// Convert {param} to :param
	path := op.Path
	path = strings.ReplaceAll(path, "{", ":")
	path = strings.ReplaceAll(path, "}", "")
	a.router.Add([]string{op.Method}, path, func(c fiber.Ctx) error {
		var values []*contextWrapperValue
		c.RequestCtx().VisitUserValuesAll(func(key, value any) { //Context().VisitUserValuesAll(func(key, value any) { // ошибка
			values = append(values, &contextWrapperValue{
				Key:   key,
				Value: value,
			})
		})
		handler(&fiberWrapper{
			op:   op,
			orig: c,
			ctx: &contextWrapper{
				values:  values,
				Context: userContext(c), //.UserContext(), // ошибка
			},
		})
		return nil
	})
}

// костыль 
func userContext(c fiber.Ctx) context.Context {
	ctx, ok := c.RequestCtx().UserValue(0).(context.Context) // fasthttp.UserValue(userContextKey).(context.Context)
	if !ok {
		ctx = context.Background()
		c.RequestCtx().SetUserValue(0, ctx) //SetUserContext(ctx)
	}

	return ctx
}

func (a *fiberAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// b, _ := httputil.DumpRequest(r, true)
	// fmt.Println(string(b))
	resp, err := a.tester.Test(r)
	if resp != nil && resp.Body != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		panic(err)
	}
	h := w.Header()
	for k, v := range resp.Header {
		for item := range v {
			h.Add(k, v[item])
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func New(r *fiber.App, config huma.Config) huma.API {
	return huma.NewAPI(config, &fiberAdapter{tester: r, router: r})
}

func NewWithGroup(r *fiber.App, g fiber.Router, config huma.Config) huma.API {
	return huma.NewAPI(config, &fiberAdapter{tester: r, router: g})
}
```

Следующай этап разработки:   
**включает**: миграции, .env, создание config. 








