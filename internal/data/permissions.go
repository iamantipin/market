package data

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/lib/pq"
)

type Permissions []string

type PermissionModel struct {
	DB *sql.DB
}

func (permModel PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	query := `
		select p.code
		from permissions p
		inner join users_permissions up on up.permission_id = p.id
		inner join users u on u.id = up.user_id
		where u.id = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := permModel.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var permissions Permissions
	for rows.Next() {
		var permission string
		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

func (p Permissions) Include(code string) bool {
	return slices.Contains(p, code)
}

func (permModel PermissionModel) AddForUser(userID int64, codes ...string) error {
	query := `
		insert into users_permissions
		select $1, permissions.id from permissions where permissions.code = ANY($2)
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := permModel.DB.ExecContext(ctx, query, userID, pq.Array(codes))
	return err
}
