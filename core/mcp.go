package f

import (
	"net/http"
)

type MCPServerConfig struct {
	ToolsCapabilities   bool
	PromptsCapabilities bool
}

type MCPServer interface {
	McpRouter
	Init(appID AppInfo) error
	HttpHandler() http.Handler
}

type MCPRequest interface{}

type MCPResult interface{}

type MCPResultText struct {
	MCPResult
	Text string
}

type MCPResultImage struct {
	MCPResult
	Text      string
	ImageData string
	MimeType  string
}
