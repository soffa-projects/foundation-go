package utils

import (
	"fmt"
	"net/url"
	"strings"
)

func AppendParamToUrl(url string, param string, value string) string {
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
