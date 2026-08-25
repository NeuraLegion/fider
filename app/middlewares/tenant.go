package middlewares

import (
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/web"
)

// Tenant adds either SingleTenant or MultiTenant to the pipeline
func Tenant() web.MiddlewareFunc {
	if env.IsSingleHostMode() {
		return SingleTenant()
	}
	return MultiTenant()
}

// SingleTenant injects the default tenant into the current context.
func SingleTenant() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			firstTenant := &query.GetFirstTenant{}
			err := bus.Dispatch(c, firstTenant)
			if err != nil && errors.Cause(err) != app.ErrNotFound {
				return c.Failure(err)
			}

			if firstTenant.Result != nil && !firstTenant.Result.IsDisabled() {
				c.SetTenant(firstTenant.Result)
			}

			return next(c)
		}
	}
}

// MultiTenant extracts tenant information from the hostname and injects it into the context.
func MultiTenant() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			hostname := c.Request.URL.Hostname()
			byDomain := &query.GetTenantByDomain{Domain: hostname}
			err := bus.Dispatch(c, byDomain)
			if err != nil && errors.Cause(err) != app.ErrNotFound {
				return c.Failure(err)
			}

			if byDomain.Result != nil && !byDomain.Result.IsDisabled() {
				c.SetTenant(byDomain.Result)

				if byDomain.Result.CNAME != "" && !c.IsAjax() {
					baseURL := web.TenantBaseURL(c, byDomain.Result)
					if baseURL != c.BaseURL() {
						c.SetCanonicalURL(baseURL + c.Request.URL.RequestURI())
					}
				}
			}

			return next(c)
		}
	}
}

// RequireTenant returns 404 if no tenant is available.
func RequireTenant() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			if c.Tenant() == nil {
				if env.IsSingleHostMode() {
					return c.Redirect("/signup")
				}
				return c.NotFound()
			}
			return next(c)
		}
	}
}

// BlockPendingTenants blocks requests for pending tenants.
func BlockPendingTenants() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			if c.Tenant().Status == enum.TenantPending {
				return c.Page(http.StatusOK, web.Props{
					Page:        "SignUp/PendingActivation.page",
					Title:       "Pending Activation",
					Description: "We sent you a confirmation email with a link to activate your site. Please check your inbox to activate it.",
				})
			}
			return next(c)
		}
	}
}

// CheckTenantPrivacy blocks unauthenticated users for private tenants.
func CheckTenantPrivacy() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			// Authentication endpoints must remain reachable so users can log in.
			// This middleware is applied globally after the sign-in routes are
			// registered, but route middleware is evaluated before the handler.
			if c.IsAuthenticated() || isAuthenticationPath(c.Request.URL.Path) || !c.Tenant().IsPrivate {
				return next(c)
			}

			parsedURL, err := url.Parse(c.Request.URL.String())
			if err != nil {
				return c.Failure(err)
			}

			cleanPath := path.Clean(parsedURL.RequestURI())
			redirectTarget := ""
			if strings.HasPrefix(cleanPath, "/") &&
				cleanPath != "/" &&
				!strings.Contains(cleanPath, "..") {
				redirectTarget = cleanPath
			}

			if redirectTarget != "" {
				return c.Redirect("/signin?redirect=" + url.QueryEscape(redirectTarget))
			}
			return c.Redirect("/signin")
		}
	}
}

func isAuthenticationPath(p string) bool {
	switch p {
	case "/signin", "/signin/complete", "/loginemailsent", "/not-invited",
		"/access-denied", "/signin/verify", "/invite/verify",
		"/_api/signin", "/_api/signin/newuser", "/_api/signin/complete",
		"/_api/signin/verify", "/_api/signin/resend":
		return true
	default:
		return false
	}
}

// RequirePro blocks requests from non-pro tenants.
func RequirePro() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			if !c.Tenant().IsPro {
				return c.Forbidden()
			}
			return next(c)
		}
	}
}

// BlockLockedTenants blocks requests on locked tenants as they are read-only.
func BlockLockedTenants() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			if c.Tenant().Status == enum.TenantLocked {
				return c.JSON(http.StatusPaymentRequired, web.Map{})
			}
			return next(c)
		}
	}
}
