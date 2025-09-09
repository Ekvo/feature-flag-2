package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up, Down)
}

func Up(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS public.flags (
	flag_name      TEXT                        NOT NULL,
	is_deleted     BOOLEAN                     NOT NULL,
	is_enabled     BOOLEAN                     NOT NULL,
	active_from    TIMESTAMP WITH TIME ZONE    NOT NULL,
	data           JSONB                       NOT NULL,
	default_data   JSONB                       NOT NULL,
	created_by     UUID                        NOT NULL,
	created_at     TIMESTAMP WITH TIME ZONE    NOT NULL,
	updated_at     TIMESTAMP WITH TIME ZONE    NOT NULL,	    
	CONSTRAINT pk_flags PRIMARY KEY (flag_name)
);`)
	if err != nil {
		return err
	}
	return nil
}

func Down(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS public.flags;")
	if err != nil {
		return err
	}
	return nil
}
