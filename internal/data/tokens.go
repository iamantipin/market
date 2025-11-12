package data

import (
	"antipinegor/cyclingmarket/internal/validator"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"time"
)

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

type TokenModel struct {
	DB *sql.DB
}

func generateToken(userID int64, timeToLive time.Duration, scope string) *Token {
	token := &Token{
		Plaintext: rand.Text(),
		UserID:    userID,
		Expiry:    time.Now().Add(timeToLive),
		Scope:     scope,
	}

	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token
}

func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

func (tokenModel TokenModel) New(userID int64, timeToLive time.Duration, scope string) (*Token, error) {
	token := generateToken(userID, timeToLive, scope)
	err := tokenModel.Insert(token)
	return token, err
}

func (tokenModel TokenModel) Insert(token *Token) error {
	query := `
		insert into tokens (hash, user_id, expiry, scope)
		values ($1, $2, $3, $4)
	`

	args := []any{token.Hash, token.UserID, token.Expiry, token.Scope}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := tokenModel.DB.ExecContext(ctx, query, args...)

	return err
}

func (tokenModel TokenModel) DeleteAllForUser(scope string, userID int64) error {
	query := `
		delete from tokens
		where scope = $1 and user_id = $2
	`

	args := []any{scope, userID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := tokenModel.DB.ExecContext(ctx, query, args...)

	return err
}
