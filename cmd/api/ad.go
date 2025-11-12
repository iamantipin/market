package main

import (
	"errors"
	"fmt"
	"net/http"

	"antipinegor/cyclingmarket/internal/data"
	"antipinegor/cyclingmarket/internal/validator"
)

func (app *application) postAdHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string     `json:"title"`
		Description string     `json:"description"`
		Categories  []string   `json:"categories"`
		Price       data.Price `json:"price"`
	}

	err := app.readJSON(w, r, &req)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	ad := &data.Ad{
		Title:       req.Title,
		Description: req.Description,
		Categories:  req.Categories,
		Price:       req.Price,
	}
	data.ValidateAd(v, ad)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Ads.Insert(ad)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/ad/%d", ad.ID))
	err = app.writeJSON(w, http.StatusCreated, envelope{"ad": ad}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

func (app *application) showAdHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	ad, err := app.models.Ads.GetById(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"ad": ad}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) showAdsHandler(w http.ResponseWriter, r *http.Request) {
	var response struct {
		Title      string
		Categories []string
		data.Filters
	}

	v := validator.New()

	queryString := r.URL.Query()
	response.Title = app.readString(queryString, "title", "")
	response.Categories = app.readCSV(queryString, "categories", []string{})
	response.Filters.Page = app.readInt(queryString, "page", 1, v)
	response.Filters.PageSize = app.readInt(queryString, "page_size", 20, v)
	response.Sort = app.readString(queryString, "sort", "id")
	response.Filters.SortSafelist = []string{"id", "title", "price", "-id", "-title", "-price"}

	if data.ValidateFilters(v, response.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	ads, metadata, err := app.models.Ads.GetAll(response.Title, response.Categories, response.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.writeJSON(w, http.StatusOK, envelope{"ads": ads, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) updateAdHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	ad, err := app.models.Ads.GetById(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	var dataToUpdate struct {
		Title       *string     `json:"title"`
		Description *string     `json:"description"`
		Categories  []string    `json:"categories"`
		Price       *data.Price `json:"price"`
	}
	err = app.readJSON(w, r, &dataToUpdate)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if dataToUpdate.Title != nil {
		ad.Title = *dataToUpdate.Title
	}
	if dataToUpdate.Description != nil {
		ad.Description = *dataToUpdate.Description
	}
	if dataToUpdate.Categories != nil {
		ad.Categories = dataToUpdate.Categories
	}
	if dataToUpdate.Price != nil {
		ad.Price = *dataToUpdate.Price
	}

	v := validator.New()
	if data.ValidateAd(v, ad); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Ads.Update(ad)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"ad": ad}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteAdHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Ads.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)

		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "Ad successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}
