package adapters

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/invopop/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	f "github.com/soffa-projects/foundation-go/core"
)

type mcpServerImpl struct {
	f.MCPServer
	internal *mcp.Server
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
	// Create MCP server with implementation details
	// Note: Official SDK auto-detects capabilities from registered tools/prompts
	s.internal = mcp.NewServer(
		&mcp.Implementation{
			Name:    appID.Name,
			Version: appID.Version,
		},
		nil, // ServerOptions - use defaults
	)
	return nil
}

func (s *mcpServerImpl) HttpHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(
		func(req *http.Request) *mcp.Server {
			return s.internal
		},
		&mcp.StreamableHTTPOptions{
			JSONResponse: true, // Return JSON instead of SSE for easier testing
		},
	)
}

func (s *mcpServerImpl) IsEmpty() bool {
	return s.tools == 0
}

func (s *mcpServerImpl) Add(op f.MCP) {
	// Create tool definition
	tool := &mcp.Tool{
		Name:        op.Name,
		Description: op.Desc,
	}

	// Add input schema if provided
	if op.InputSchema != nil {
		schema := reflector.Reflect(op.InputSchema)
		jsonSchema, _ := json.MarshalIndent(schema, "", "  ")
		var schemaMap map[string]any
		json.Unmarshal(jsonSchema, &schemaMap)
		tool.InputSchema = schemaMap
	}

	// Register tool with type-safe handler
	// Using map[string]any for dynamic input since we don't know the exact type at compile time
	mcp.AddTool(s.internal, tool,
		func(ctx context.Context, req *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, any, error) {
			// Create MCP context for the handler
			c := &mcpOperationContextImpl{ctx: ctx}

			// Call the user's handler
			res, err := op.Handle(c)
			if err != nil {
				// Return error as tool result
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{
							Text: "Error: " + err.Error(),
						},
					},
					IsError: true,
				}, nil, nil
			}

			// Handle different result types
			switch v := res.(type) {
			case *mcp.CallToolResult:
				return v, nil, nil
			case string:
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{
							Text: v,
						},
					},
				}, nil, nil
			default:
				// For structured data, serialize to JSON
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{
							Text: "Result: " + string(mustJSON(v)),
						},
					},
				}, v, nil
			}
		},
	)
	s.tools++
}

type mcpOperationContextImpl struct {
	f.Context
	ctx context.Context
}

func (r *mcpOperationContextImpl) Structured(data any) any {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(mustJSON(data)),
			},
		},
	}
}

func (r *mcpOperationContextImpl) Text(data string) any {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: data,
			},
		},
	}
}

// Helper function to marshal JSON
func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
