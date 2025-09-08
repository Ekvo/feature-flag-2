package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up1, Down1)
}

func Up1(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	if _, err := tx.ExecContext(ctx, `INSERT INTO public.flags (
    flag_name,
    is_enable,
    active_from,
    data,
    default_data,
    created_user,
    created_at,
    updated_at
) VALUES (
    'example_flag',                          -- flag_name
    true,                                    -- is_enable
    NOW(),                                   -- active_from
    '{"key": "value"}'::JSONB,               -- data
    '{"default_key": "default_value"}'::JSONB, -- default_data
    '00000000-0000-0000-0000-000000000000',  -- created_user (замените на реальный UUID)
    NOW(),                                   -- created_at
    NOW()                                    -- updated_at
);`); err != nil {
		return err
	}
	return nil
}

func Down1(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM public.flags WHERE flag_name = 'example_flag';`); err != nil {
		return err
	}
	return nil
}
