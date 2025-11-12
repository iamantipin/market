package data

import (
	"antipinegor/cyclingmarket/internal/validator"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type Ad struct {
	ID          int64     `json:"id"`
	CreatedAt   time.Time `json:"-"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Categories  []string  `json:"categories"`
	Price       Price     `json:"price"`
	Version     int32     `json:"version"`
}

func ValidateAd(v *validator.Validator, ad *Ad) {
	v.Check(ad.Title != "", "title", "must be provided")
	v.Check(len(ad.Title) <= 80, "title", "must not be more than 80 bytes long")

	v.Check(ad.Description != "", "description", "must be provided")
	v.Check(len(ad.Description) <= 500, "description", "must not be more than 500 bytes long")

	v.Check(ad.Price != 0, "price", "must be provided")
	v.Check(ad.Price > 0, "price", "must be positive integer")

	v.Check(ad.Categories != nil, "categories", "must be provided")
	v.Check(len(ad.Categories) >= 1, "categories", "must contain at least 1 categories")
	v.Check(len(ad.Categories) <= 5, "categories", "must not contain more than 5 categories")
	v.Check(validator.Unique(ad.Categories), "categories", "must not contain duplicate values")
}

type AdModel struct {
	DB *sql.DB
}

func (ad AdModel) Insert(adToInsert *Ad) error {
	query := `
		insert 
		into ads (title, description, price, categories)
		values ($1, $2, $3, $4)
		returning id, created_at, version
	`
	args := []any{adToInsert.Title, adToInsert.Description, adToInsert.Price, pq.Array(adToInsert.Categories)}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return ad.DB.QueryRowContext(ctx, query, args...).Scan(&adToInsert.ID, &adToInsert.CreatedAt, &adToInsert.Version)
}

func (ad AdModel) GetById(id int64) (*Ad, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		select 
			id, created_at, title, description, price, categories, version
		from 
			ads
		where
			id = $1
	`
	var adResponse Ad

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := ad.DB.QueryRowContext(ctx, query, id).Scan(
		&adResponse.ID,
		&adResponse.CreatedAt,
		&adResponse.Title,
		&adResponse.Description,
		&adResponse.Price,
		pq.Array(&adResponse.Categories),
		&adResponse.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &adResponse, nil
}

func (ad AdModel) GetAll(title string, categories []string, filters Filters) ([]*Ad, Metadata, error) {
	query := fmt.Sprintf(`
		select
			count(*) over(), id, created_at, title, description, price, categories, version
		from
			ads
		where
			(to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '') 
			and
			(categories @> $2 or $2 = '{}')
		order by
			%s %s, id ASC
		limit $3 offset $4
	`, filters.sortColumn(), filters.sortDirection())

	args := []any{title, pq.Array(categories), filters.limit(), filters.offset()}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := ad.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	ads := []*Ad{}

	for rows.Next() {
		var adResponse Ad
		err := rows.Scan(
			&totalRecords,
			&adResponse.ID,
			&adResponse.CreatedAt,
			&adResponse.Title,
			&adResponse.Description,
			&adResponse.Price,
			pq.Array(&adResponse.Categories),
			&adResponse.Version,
		)

		if err != nil {
			return nil, Metadata{}, err
		}
		ads = append(ads, &adResponse)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return ads, metadata, nil
}

func (ad AdModel) Update(adToUpdate *Ad) error {
	query := `
		update 
			ads
		set 
			title = $1, description = $2, price = $3, categories = $4, version = version + 1
		where 
			id = $5 and version = $6
		returning version
	`
	args := []any{
		adToUpdate.Title,
		adToUpdate.Description,
		adToUpdate.Price,
		pq.Array(adToUpdate.Categories),
		adToUpdate.ID,
		adToUpdate.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := ad.DB.QueryRowContext(ctx, query, args...).Scan(&adToUpdate.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (ad AdModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
		delete 
			from ads
		where 
			id = $1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := ad.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}
