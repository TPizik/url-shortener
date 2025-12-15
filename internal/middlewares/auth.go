package middlewares

import (
	"net/http"

	"github.com/TPizik/url-shortener/internal/context"
	httputils "github.com/TPizik/url-shortener/internal/handlers/http/http_utils"
	"github.com/TPizik/url-shortener/internal/jwt"
)

type userService interface {
	GenerateUniqueID() string
}

const CookieName = "token"

func AuthMiddleware(handler http.Handler, userService userService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHandler(w, r, handler, userService)
	})
}

func authHandler(w http.ResponseWriter, r *http.Request, handler http.Handler, userService userService) {
	var tokenString string

	cookie, err := r.Cookie(CookieName)

	if err != nil {
		tokenString, err = jwt.GenerateToken(userService.GenerateUniqueID())

		if err != nil {
			httputils.SendStatusCode(w, http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:  CookieName,
			Value: tokenString,
		})
	} else {
		tokenString = cookie.Value
	}

	jwtClaim, err := jwt.ParseToken(tokenString)

	if err != nil {
		httputils.SendStatusCode(w, http.StatusUnauthorized)
		return
	}

	ctx := context.SetUserIDToContext(r.Context(), jwtClaim.UserID)

	handler.ServeHTTP(w, r.WithContext(ctx))
}
