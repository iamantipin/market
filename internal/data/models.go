package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Ads         AdModel
	Users       UserModel
	Permissions PermissionModel
	Tokens      TokenModel
}

func NewModels(db *sql.DB) Models {
	adModel := AdModel{
		DB: db,
	}
	userModel := UserModel{
		DB: db,
	}
	permModel := PermissionModel{
		DB: db,
	}
	tokenModel := TokenModel{
		DB: db,
	}
	return Models{
		Ads:         adModel,
		Users:       userModel,
		Permissions: permModel,
		Tokens:      tokenModel,
	}
}
