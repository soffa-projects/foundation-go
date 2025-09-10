package test

import (
	"context"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

type Helper struct {
	app        f.App
	InstanceId string
	Context    context.Context
	Server     *httptest.Server
	Http       *RestClient
	Config     any
	Assert     Assertions
	openFiles  []string
	rootDir    string
}

func New(app f.App, t *testing.T) *Helper {

	// Create a test server
	server := httptest.NewServer(app.Router().Handler())
	rootDir := ProjectRoot(t)
	return &Helper{
		app:        app,
		InstanceId: app.InstanceId(),
		Context:    context.TODO(),
		Server:     server,
		Http:       NewRestClient(t, server.URL),
		Assert:     NewAssertions(t),
		openFiles:  []string{},
		rootDir:    rootDir,
	}
}

func (t *Helper) FilePath(p string) string {
	return path.Join(t.rootDir, p)
}

func (t *Helper) TearDown() {
	t.app.Shutdown(context.Background())
	for _, file := range t.openFiles {
		log.Info("removing db file: %s", file)
		_ = os.Remove(file)
	}
}

func (t *Helper) RegisterFile(file string) {
	t.openFiles = append(t.openFiles, file)
}

func (t *Helper) NewJwt(secret string, tenantId string, userId string, permissions string) string {
	token, err := h.NewJwt(h.JwtConfig{
		Subject:   userId,
		Issuer:    "test",
		Audience:  []string{tenantId},
		Claims:    map[string]any{"tenantId": tenantId, "permissions": permissions},
		Ttl:       time.Hour * 24,
		SecretKey: secret,
	})
	t.Assert.Nil(err)
	return token
}
