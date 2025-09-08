package adapters

import (
	"net/http"
)

type MCPServerConfig struct {
	ToolsCapabilities   bool
	PromptsCapabilities bool
}

type MCPServer interface {
	Init(appID AppID) error
	HttpHandler() http.Handler
	AddOperation(operation Operation)
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
