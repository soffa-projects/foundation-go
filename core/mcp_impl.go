package f

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/invopop/jsonschema"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type mcpServerImpl struct {
	MCPServer
	internal *server.MCPServer
	cfg      MCPServerConfig
	env      ApplicationEnv
}

var reflector = jsonschema.Reflector{}

func newMCPServer(cfg MCPServerConfig) MCPServer {
	return &mcpServerImpl{cfg: cfg}
}

func (s *mcpServerImpl) Init(env ApplicationEnv) error {
	s.internal = server.NewMCPServer(
		env.AppName,
		env.AppVersion,
		server.WithToolCapabilities(s.cfg.ToolsCapabilities),
		server.WithPromptCapabilities(s.cfg.PromptsCapabilities),
	)
	s.env = env
	return nil
}

func (s *mcpServerImpl) HttpHandler() http.Handler {
	return server.NewStreamableHTTPServer(s.internal)
}

type mcpOperationContextImpl struct {
	Context
	env    ApplicationEnv
	result *mcp.CallToolResult
}

func (s *mcpServerImpl) AddOperation(operation Operation) {
	var t mcp.Tool
	if operation.InputSchema != nil {
		schema := reflector.Reflect(operation.InputSchema)
		jsonSchema, _ := json.MarshalIndent(schema, "", "  ")
		t = mcp.NewToolWithRawSchema(operation.Name, operation.Description, jsonSchema)
	} else {
		t = mcp.NewTool(operation.Name, mcp.WithDescription(operation.Description))
	}
	s.internal.AddTool(t, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c := &mcpOperationContextImpl{
			env: s.env,
		}
		err := operation.Handle(c)
		internalResult := c.result
		return internalResult, err
	})
}

func (r *mcpOperationContextImpl) Env() ApplicationEnv {
	return r.env
}

func (r *mcpOperationContextImpl) Send(value any, opt ...ResponseOpt) error {
	if _, ok := value.(string); ok {
		r.result = mcp.NewToolResultText(value.(string))
		return nil
	}
	r.result = mcp.NewToolResultStructured(value, "")
	return nil
}

func (r *mcpOperationContextImpl) Error(error string, opt ...ResponseOpt) error {
	r.result = mcp.NewToolResultError(error)
	return nil
}

func (r *mcpOperationContextImpl) TenantId() string {
	return ""
}

func (r *mcpOperationContextImpl) Param(value string) string {
	return ""
}

func (r *mcpOperationContextImpl) QueryParam(value string) string {
	return ""
}
