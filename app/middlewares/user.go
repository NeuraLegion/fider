package middlewares

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"

	"github.com/getfider/fider/app/pkg/validate"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/web"
	webutil "github.com/getfider/fider/app/pkg/web/util"
)

// User gets JWT Auth token from cookie and insert into context
func User() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			var (
				token            string
				user             *entity.User
				fromSignUpCookie bool
			)

			cookie, err := c.Request.Cookie(web.CookieAuthName)
			if err == nil {
				token = cookie.Value
			} else {
				token = webutil.GetSignUpAuthCookie(c)
				fromSignUpCookie = token != ""
			}

			if token != "" {
				claims, err := jwt.DecodeFiderClaims(token)
				if err != nil {
					c.RemoveCookie(web.CookieAuthName)
					return next(c)
				}

				tenantID := 0
				if c.Tenant() != nil {
					tenantID = c.Tenant().ID
				}
				userByClaimsID := &query.GetUserByID{UserID: claims.UserID, TenantID: tenantID}
				err = bus.Dispatch(c, userByClaimsID)
				user = userByClaimsID.Result
				if err != nil {
					if errors.Cause(err) == app.ErrNotFound {
						c.RemoveCookie(web.CookieAuthName)
						return next(c)
					}
					return err
				}

				// Security stamps are present only on tokens issued after the stamp fix.
				// Legacy tokens remain valid; new tokens are invalidated when the stamp changes.
				if claims.SecurityStamp != "" && user != nil && claims.SecurityStamp != user.SecurityStamp {
					c.RemoveCookie(web.CookieAuthName)
					if c.IsAjax() {
						return c.JSON(401, web.Map{})
					}
					redirectTarget := c.Request.URL.RequestURI()
					if redirectTarget != "" && redirectTarget != "/" &&
						!strings.HasPrefix(redirectTarget, "/signin") &&
						!strings.HasPrefix(redirectTarget, "/signout") {
						return c.Redirect("/signin?redirect=" + url.QueryEscape(redirectTarget))
					}
					return c.Redirect("/signin")
				}
			} else if c.Request.IsAPI() {
				authHeader := c.Request.GetHeader("Authorization")
				parts := strings.SplitN(authHeader, "Bearer", 2)
				if len(parts) == 2 {
					apiKey := strings.TrimSpace(parts[1])
					getUserByAPIKey := &query.GetUserByAPIKey{APIKey: apiKey}
					err = bus.Dispatch(c, getUserByAPIKey)
					if err != nil {
						if errors.Cause(err) == app.ErrNotFound {
							return c.HandleValidation(validate.Failed("API Key is invalid"))
						}
						return err
					}
					user = getUserByAPIKey.Result
					if !user.IsCollaborator() {
						return c.HandleValidation(validate.Failed("API Key is invalid"))
					}

					if impersonateUserIDStr := c.Request.GetHeader("X-Fider-UserID"); impersonateUserIDStr != "" {
						if !user.IsAdministrator() {
							return c.HandleValidation(validate.Failed("Only Administrators are allowed to impersonate another user"))
						}
						impersonateUserID, err := strconv.Atoi(impersonateUserIDStr)
						if err != nil {
							return c.HandleValidation(validate.Failed(fmt.Sprintf("User not found for given impersonate UserID '%s'", impersonateUserIDStr)))
						}
						userByImpersonateID := &query.GetUserByID{UserID: impersonateUserID, TenantID: user.Tenant.ID}
						err = bus.Dispatch(c, userByImpersonateID)
						user = userByImpersonateID.Result
						if err != nil {
							if errors.Cause(err) == app.ErrNotFound {
								return c.HandleValidation(validate.Failed(fmt.Sprintf("User not found for given impersonate UserID '%s'", impersonateUserIDStr)))
							}
							return err
						}
					}
				}
			}

			if user != nil && c.Tenant() != nil && user.Tenant.ID == c.Tenant().ID {
				if user.Status == enum.UserBlocked {
					c.RemoveCookie(web.CookieAuthName)
					return c.Unauthorized()
				}
				if fromSignUpCookie {
					webutil.AddAuthTokenCookie(c, token)
				}
				c.SetUser(user)
			}

			return next(c)
		}
	}
}
