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
	tools    int
}

var reflector = jsonschema.Reflector{}

func NewMCPServer(cfg f.MCPServerConfig) f.MCPServer {
	return &mcpServerImpl{
		cfg:   cfg,
		tools: 0,
	}
}

func (s *mcpServerImpl) Init(appID f.AppInfo) error {
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

func (s *mcpServerImpl) IsEmpty() bool {
	return s.tools == 0
}

func (s *mcpServerImpl) Add(op f.MCP) {
	var t mcp.Tool
	if op.InputSchema != nil {
		schema := reflector.Reflect(op.InputSchema)
		jsonSchema, _ := json.MarshalIndent(schema, "", "  ")
		t = mcp.NewToolWithRawSchema(op.Name, op.Desc, jsonSchema)
	} else {
		t = mcp.NewTool(op.Name, mcp.WithDescription(op.Desc))
	}
	s.internal.AddTool(t, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c := &mcpOperationContextImpl{}
		res, err := op.Handle(c)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return res.(*mcp.CallToolResult), nil
	})
	s.tools++
}

type mcpOperationContextImpl struct {
	f.Context
}

func (r *mcpOperationContextImpl) Structured(data any) any {
	return mcp.NewToolResultStructured(data, "")
}

func (r *mcpOperationContextImpl) Text(data string) any {
	return mcp.NewToolResultText(data)
}
