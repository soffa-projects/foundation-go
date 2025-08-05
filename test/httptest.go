package test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/onsi/gomega"
	matcher "github.com/panta/go-json-matcher"
)

type RestClient struct {
	client *resty.Client
	az     *gomega.WithT
	bearer string
}

type HttpRes struct {
	resp *resty.Response
	err  error
	az   *gomega.WithT
}

type HttpReq struct {
	Body   any
	Bearer string
	Result any
}

type ApiDef struct {
	Method string
	Path   string
	Body   any
	Bearer string
	Result any
}

func NewRestClient(t *testing.T, baseUrl string) *RestClient {
	r := resty.New()
	r.SetBaseURL(baseUrl)
	return &RestClient{client: r, az: gomega.NewWithT(t)}
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
	hasAuth := false
	for _, opt := range opts {
		if opt.Body != nil {
			q = q.SetBody(opt.Body)
		}
		if opt.Bearer != "" {
			q = q.SetHeader("Authorization", "Bearer "+opt.Bearer)
			hasAuth = true
		}
		if opt.Result != nil {
			q = q.SetResult(opt.Result)
		} else {
			q = q.SetResult(&result)
		}
	}
	if !hasAuth && c.bearer != "" {
		q = q.SetHeader("Authorization", "Bearer "+c.bearer)
	}
	resp, err := q.Execute(method, path)
	return HttpRes{
		resp: resp,
		err:  err,
		az:   c.az,
	}
}

func (c *RestClient) Invoke(up ApiDef) HttpRes {
	return c.invoke(up.Method, up.Path, HttpReq{Body: up.Body, Bearer: up.Bearer, Result: up.Result})
}

func (r HttpRes) IsOk() HttpRes {
	r.az.Expect(r.resp.StatusCode()).To(gomega.Equal(http.StatusOK))
	return r
}

func (r HttpRes) IsCreated() HttpRes {
	r.az.Expect(r.resp.StatusCode()).To(gomega.Equal(http.StatusCreated))
	return r
}
func (r HttpRes) Result() []byte {
	return r.resp.Body()
}

func (r HttpRes) NoContent() HttpRes {
	r.az.Expect(r.resp.StatusCode()).To(gomega.Equal(http.StatusNoContent))
	return r
}
func (r HttpRes) Is(status int) HttpRes {
	r.az.Expect(r.resp.StatusCode()).To(gomega.Equal(status))
	return r
}

func (r HttpRes) IsConflict() {
	r.az.Expect(r.resp.StatusCode()).To(gomega.Equal(http.StatusConflict))
}

func (r HttpRes) JSON() *JsonMatcher {
	result := r.Result()
	if result == nil {
		r.az.Fail("result is nil")
		return nil
	}
	var data map[string]any
	err := json.Unmarshal(result, &data)
	if err != nil {
		r.az.Fail(err.Error())
		return nil
	}
	return &JsonMatcher{az: r.az, value: string(result)}
}

type JsonMatcher struct {
	az    *gomega.WithT
	value string
}

func (j JsonMatcher) Match(pattern string) JsonMatcher {
	j.az.Expect(j.value).To(gomega.MatchJSON(pattern))
	return j
}

func (j JsonMatcher) MatchShape(pattern string) JsonMatcher {
	match, err := matcher.JSONStringMatches(j.value, pattern)
	j.az.Expect(err).To(gomega.BeNil(), "failed to match json", err)
	j.az.Expect(match).To(gomega.BeTrue(), "json does not match - %v", j.value)
	return j
}
