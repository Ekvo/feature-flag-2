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
    is_deleted,
    is_enabled,
    active_from,
    data,
    default_data,
    created_by,
    created_at,
    updated_at
) VALUES (
    'new_feature_rollout',
    FALSE,
    TRUE,
    NOW(),
    '{"target_users": ["beta", "internal"], "percentage": 10}'::JSONB,
    '{"target_users": ["all"], "percentage": 0}'::JSONB,
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'::UUID,  -- пример UUID
    NOW(),
    NOW()
);`); err != nil {
		return err
	}
	return nil
}

func Down1(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM public.flags WHERE flag_name = 'new_feature_rollout';`); err != nil {
		return err
	}
	return nil
}
