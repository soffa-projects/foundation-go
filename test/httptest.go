package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-resty/resty/v2"
	matcher "github.com/panta/go-json-matcher"
	"github.com/soffa-projects/foundation-go/h"
)

type RestClient struct {
	client *resty.Client
	assert Assertions
	bearer string
}

type HttpRes struct {
	resp   *resty.Response
	err    error
	assert Assertions
}

type HttpReq struct {
	Body     any
	Headers  map[string]string
	Files    map[string]string
	Form     map[string]string
	Bearer   string
	TenantId string
	Result   any
	Host     string
}

type ApiDef struct {
	Method string
	Path   string
	Body   any
	Bearer string
	Result any
}

func ProjectRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func NewRestClient(t *testing.T, baseUrl string) *RestClient {
	r := resty.New()
	r.SetRedirectPolicy(resty.NoRedirectPolicy())
	r.SetBaseURL(baseUrl)
	return &RestClient{client: r, assert: NewAssertions(t)}
}

func (c *RestClient) Get(path string, opts ...HttpReq) HttpRes {
	return c.invoke("GET", path, opts...)
}

func (c *RestClient) Post(path string, opts ...HttpReq) HttpRes {
	return c.invoke("POST", path, opts...)
}

func (c *RestClient) SetBearerAuth(token string) *RestClient {
	c.bearer = token
	return c
}

func (c *RestClient) invoke(method string, path string, opts ...HttpReq) HttpRes {
	q := c.client.R()
	var result map[string]any
	bearerAuth := ""
	tenantId := ""
	for _, opt := range opts {
		if opt.Body != nil {
			q = q.SetBody(opt.Body)
		}
		if opt.Bearer != "" {
			bearerAuth = opt.Bearer
		}
		if opt.Form != nil {
			q = q.SetFormData(opt.Form)
		}
		if opt.Result != nil {
			q = q.SetResult(opt.Result)
		} else {
			q = q.SetResult(&result)
		}
		if opt.Headers != nil {
			for key, value := range opt.Headers {
				q = q.SetHeader(key, value)
			}
		}
		if opt.Files != nil {
			q = q.SetFiles(opt.Files)
		}
		if opt.TenantId != "" {
			tenantId = opt.TenantId
		}
		if opt.Host != "" {
			q = q.SetHeader("Host", opt.Host)
			q = q.SetHeader("X-Forwarded-Host", opt.Host)
		}
	}
	if bearerAuth == "" && c.bearer != "" {
		bearerAuth = c.bearer
	}
	if bearerAuth != "" {
		q = q.SetHeader("Authorization", fmt.Sprintf("Bearer %s", bearerAuth))
	}
	if tenantId != "" {
		q = q.SetHeader("X-TenantId", tenantId)
	}
	resp, err := q.Execute(method, path)
	return HttpRes{
		resp:   resp,
		err:    err,
		assert: c.assert,
	}
}

func (c *RestClient) Invoke(up ApiDef) HttpRes {
	return c.invoke(up.Method, up.Path, HttpReq{Body: up.Body, Bearer: up.Bearer, Result: up.Result})
}

func (r HttpRes) IsOk() HttpRes {
	r.assert.Equals(r.resp.StatusCode(), http.StatusOK)
	return r
}

func (r HttpRes) IsRedirect() HttpRes {
	r.assert.Equals(r.resp.StatusCode(), http.StatusFound)
	return r
}

func (r HttpRes) IsCreated() HttpRes {
	r.assert.Equals(r.resp.StatusCode(), http.StatusCreated)
	return r
}
func (r HttpRes) Result() []byte {
	return r.resp.Body()
}

func (r HttpRes) NoContent() HttpRes {
	r.assert.Equals(r.resp.StatusCode(), http.StatusNoContent)
	return r
}
func (r HttpRes) Is(status int) HttpRes {
	r.assert.Equals(r.resp.StatusCode(), status)
	return r
}

func (r HttpRes) GetLocation() string {
	return r.resp.Header().Get("Location")
}

func (r HttpRes) IsConflict() HttpRes {
	r.assert.Equals(r.resp.StatusCode(), http.StatusConflict)
	return r
}

func (r HttpRes) IsBadRequest() HttpRes {
	r.assert.Equals(r.resp.StatusCode(), http.StatusBadRequest)
	return r
}

func (r HttpRes) IsForbidden() HttpRes {
	r.assert.Equals(r.resp.StatusCode(), http.StatusForbidden)
	return r
}

func (r HttpRes) IsUnauthorized() HttpRes {
	r.assert.Equals(r.resp.StatusCode(), http.StatusUnauthorized)
	return r
}

func (r HttpRes) JSONValue() h.JsonValue {
	return h.NewJsonValue(string(r.Result()))
}

func (r HttpRes) JSON() *JsonMatcher {
	result := r.Result()
	r.assert.NotNil(result)
	var data map[string]any
	err := json.Unmarshal(result, &data)
	r.assert.Nil(err, "failed to unmarshal json")
	return &JsonMatcher{assert: r.assert, value: string(result)}
}

type JsonMatcher struct {
	assert Assertions
	value  string
}

func (j JsonMatcher) Match(pattern string) JsonMatcher {
	j.assert.MatchJson(j.value, pattern)
	return j
}

func (j JsonMatcher) MatchShape(pattern string) JsonMatcher {
	match, err := matcher.JSONStringMatches(j.value, pattern)
	j.assert.Nil(err)
	j.assert.True(match)
	return j
}

func (j JsonMatcher) Value() h.JsonValue {
	return h.NewJsonValue(j.value)
}
