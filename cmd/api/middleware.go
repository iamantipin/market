package main

import (
	"antipinegor/cyclingmarket/internal/data"
	"antipinegor/cyclingmarket/internal/validator"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tomasen/realip"
	"golang.org/x/time/rate"
)

func (app *application) recoverPanic(nextHandler http.Handler) http.Handler {
	wrappedFunction := func(w http.ResponseWriter, r *http.Request) {
		defer app.handlePanic(w, r)
		nextHandler.ServeHTTP(w, r)
	}

	return http.HandlerFunc(wrappedFunction)
}

func (app *application) handlePanic(w http.ResponseWriter, r *http.Request) {
	if panicError := recover(); panicError != nil {
		w.Header().Set("Connection", "close")
		app.serverErrorResponse(w, r, fmt.Errorf("panic: %v", panicError))
	}
}

func (app *application) rateLimit(nextHandler http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mutex   sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)
			mutex.Lock()

			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mutex.Unlock()
		}
	}()

	wrappedFunction := func(w http.ResponseWriter, r *http.Request) {
		ip := realip.FromRequest(r)
		mutex.Lock()

		if _, found := clients[ip]; !found {
			// allows of 2 requests per second with a maximum of 4 requests in a single 'burst'
			clients[ip] = &client{
				limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst),
			}
		}

		clients[ip].lastSeen = time.Now()

		if !clients[ip].limiter.Allow() {
			mutex.Unlock()
			app.rateLimitExceededResponse(w, r)
			return
		}

		mutex.Unlock()
		nextHandler.ServeHTTP(w, r)
	}

	return http.HandlerFunc(wrappedFunction)
}

func (app *application) authenticate(next http.Handler) http.Handler {
	wrappedFunction := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}
		token := headerParts[1]

		v := validator.New()
		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		r = app.contextSetUser(r, user)
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(wrappedFunction)
}

func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	wrappedFunction := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(wrappedFunction)
}

func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	wrappedFunction := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	}

	return app.requireAuthenticatedUser(http.HandlerFunc(wrappedFunction))
}

func (app *application) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	wrappedFunction := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		permissions, err := app.models.Permissions.GetAllForUser(user.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		if !permissions.Include(code) {
			app.notPermittedResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	}

	return app.requireActivatedUser(wrappedFunction)
}
