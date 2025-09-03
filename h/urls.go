package h

import (
	"fmt"
	"html"
	"net/url"
	"strings"
)

type Url struct {
	Scheme   string
	Path     string
	Url      string
	Host     string
	User     string
	Password string
	query    map[string]any
}

func ParseUrl(input string) (Url, error) {
	queryParams := make(map[string]any)
	u, err := url.Parse(input)
	if err != nil {
		return Url{}, err
	}
	for key, values := range u.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0] // Take first value if multiple
		}
	}
	password, ok := u.User.Password()
	if !ok {
		password = ""
	}
	return Url{
		Scheme:   u.Scheme,
		Path:     u.Path,
		Url:      input,
		Host:     u.Host,
		User:     u.User.Username(),
		Password: password,
		query:    queryParams,
	}, nil
}

func (u Url) HasQueryParam(key string) bool {
	_, ok := u.query[key]
	return ok
}

func (u Url) Query(key string) any {
	return u.query[key]
}

func RemoveParamFromUrl(input string, param string) (string, error) {
	u, err := url.Parse(input)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Del(param)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func AppendParamToUrl(url string, param string, value string) string {
	url, err := RemoveParamFromUrl(url, param)
	if err != nil {
		return url
	}
	if strings.Contains(url, "?") {
		return fmt.Sprintf("%s&%s=%s", url, param, value)
	} else {
		return fmt.Sprintf("%s?%s=%s", url, param, value)
	}
}

func AppendParamsToUrl(input string, params map[string]any) string {
	for key, value := range params {
		input = AppendParamToUrl(input, key, url.QueryEscape(fmt.Sprintf("%v", value)))
	}
	return input
}

func StripOriginFromUrl(input string) (string, error) {
	// In your text the query uses &amp; — decode that first
	raw := html.UnescapeString(input)

	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}

	out := u.EscapedPath()
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	return out, nil
}
