The authentication failure is caused by the OAuth redirect validation introduced in the recent security fix. `OAuthCallback` validates `claims.Redirect` with `url.ParseRequestURI`, but OAuth state commonly contains an absolute URL. `ParseRequestURI` rejects some valid absolute redirect URLs, causing the callback to fail before redirecting to the auth endpoint. Replace it with `url.Parse` and validate the parsed URL before continuing.

// In OAuthCallback, replace:
redirectURL, err := url.ParseRequestURI(claims.Redirect)

// with:
redirectURL, err := url.Parse(claims.Redirect)
if err != nil || redirectURL.Scheme == "" || redirectURL.Host == "" {
	return c.Forbidden()
}
