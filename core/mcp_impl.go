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
	env ApplicationEnv
}

func (s *mcpServerImpl) AddOperation(operation Operation) {
	var t mcp.Tool
	if operation.Schemas.Input != nil {
		schema := reflector.Reflect(operation.Schemas.Input)
		jsonSchema, _ := json.MarshalIndent(schema, "", "  ")
		t = mcp.NewToolWithRawSchema(operation.Name, operation.Description, jsonSchema)
	} else {
		t = mcp.NewTool(operation.Name, mcp.WithDescription(operation.Description))
	}
	s.internal.AddTool(t, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c := &mcpOperationContextImpl{
			env: s.env,
		}
		res, err := operation.Handle(c)
		if err != nil {
			return nil, err
		}
		return formatResponse(res), nil
	})
}

func (r *mcpOperationContextImpl) Env() ApplicationEnv {
	return r.env
}

func formatResponse(res any) *mcp.CallToolResult {
	var wrapped Response
	if _, ok := res.(Response); ok {
		wrapped = res.(Response)
	} else {
		wrapped = Response{
			Data: res,
		}
	}
	if wrapped.Error != nil {
		return mcp.NewToolResultError(wrapped.Error.Error())
	}
	if _, ok := wrapped.Data.(string); ok {
		return mcp.NewToolResultText(wrapped.Data.(string))
	}
	return mcp.NewToolResultStructured(wrapped.Data, "")
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
