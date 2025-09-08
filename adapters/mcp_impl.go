package adapters

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/invopop/jsonschema"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	f "github.com/soffa-projects/foundation-go/core"
)

type mcpServerImpl struct {
	f.MCPServer
	internal *server.MCPServer
	cfg      f.MCPServerConfig
}

var reflector = jsonschema.Reflector{}

func NewMCPServer(cfg f.MCPServerConfig) f.MCPServer {
	return &mcpServerImpl{cfg: cfg}
}

func (s *mcpServerImpl) Init(appID f.AppID) error {
	s.internal = server.NewMCPServer(
		appID.Name,
		appID.Version,
		server.WithToolCapabilities(s.cfg.ToolsCapabilities),
		server.WithPromptCapabilities(s.cfg.PromptsCapabilities),
	)
	return nil
}

func (s *mcpServerImpl) HttpHandler() http.Handler {
	return server.NewStreamableHTTPServer(s.internal)
}

type mcpOperationContextImpl struct {
	f.Context
}

func (s *mcpServerImpl) AddOperation(operation f.Operation) {
	var t mcp.Tool
	if operation.Schemas.Input != nil {
		schema := reflector.Reflect(operation.Schemas.Input)
		jsonSchema, _ := json.MarshalIndent(schema, "", "  ")
		t = mcp.NewToolWithRawSchema(operation.Name, operation.Description, jsonSchema)
	} else {
		t = mcp.NewTool(operation.Name, mcp.WithDescription(operation.Description))
	}
	s.internal.AddTool(t, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c := &mcpOperationContextImpl{}
		res, err := operation.Handle(c)
		if err != nil {
			return nil, err
		}
		return formaMcpResponse(res), nil
	})
}

func formaMcpResponse(res any) *mcp.CallToolResult {
	var wrapped f.Response
	if _, ok := res.(f.Response); ok {
		wrapped = res.(f.Response)
	} else {
		wrapped = f.Response{
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
