# Проектирование сервиса Feature Flags

**Поставленная задача:**
Описать, выбрать **стек**, создать первичное **описание таблицы** для хранения данных в БД, определить **кеш**.

Задача сервиса **создавать**, **читать**, **обновлять**, **удалять** _feature flags_.

## Тип построения
* REST API  

## Tech stack:
* go 1.25.0                 (так как -> [рабочий отладчик](https://youtrack.jetbrains.com/issue/GO-19210/Debugger-is-not-working-after-update-to-go-version-1.25.0#focus=Change-27-12530951.0-0 "https://youtrack.jetbrains.com/issue/GO-19210/Debugger-is-not-working-after-update-to-go-version-1.25.0#focus=Change-27-12530951.0-0"))  

**БД**       
* postgres:latest                    -> делаем в docker
* gopkg.in/reform.v1                 -> ORM для запросов в БД
* jackc/pgx v3.6.2                   -> в документации к [reform.v1](https://github.com/go-reform/reform "https://github.com/go-reform/reform") -> Stable. Tested with all supported versions.  

**Кеш**      
* gofiber/fiber/v3/middleware/cache  
* golang-lru/v2  
Четкое разделение по слоям:
_репозиторий_ (*fiber.App, [fiber.Handler](https://github.com/gofiber/fiber/blob/main/middleware/cache/cache.go "https://github.com/gofiber/fiber/blob/main/middleware/cache/cache.go"))  
Работа в **middleware** слое все близко и понятно, вот маршрутизатор, вот мидлваря с настройками кеша;  
_репозиторий_ (*expirable.LRU[string, Falg]):
Работа в **Service** слое: почему? **нет работы в middleware** -> _все просто и ясно_  

**Маршрутизатор**  
* gofiber/fiber/v2 -> на момент написания документации, **потом переход на fiber/v3** (пока не нашел решение)  

**генерации OpenAPI клиентов**  
* danielgtaylor/huma/v2  

**миграции**
* pressly/goose/v3

**.env**    
* github.com/caarlos0/env/v11 - выбрал тк просто и все знают
* github.com/joho/godotenv

### sql Таблица Flags
```sql
-- Создаём таблицу flags с поддержкой временных зон
CREATE TABLE IF NOT EXISTS public.flags (
     flag_name      TEXT                        NOT NULL,
     is_enabled     BOOLEAN                     NOT NULL,
     active_from    TIMESTAMP WITH TIME ZONE    NOT NULL,
     data           JSONB                       NOT NULL,
     default_data   JSONB                       NOT NULL,
     created_by     UUID                        NOT NULL,
     created_at     TIMESTAMP WITH TIME ZONE    NOT NULL,
     updated_at     TIMESTAMP WITH TIME ZONE    NOT NULL,
     is_deleted     BOOLEAN                     NOT NULL,    
     CONSTRAINT pk_flags PRIMARY KEY (flag_name)
);
```

Следующай этап разработки:   
**включает**: _**репозиториев** к БД, Кеш, Маршрутизации, генерации OpenAPI клиентов и методов к ним_.  
**не включает**: миграции, .env, создание config.