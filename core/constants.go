package adapters

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

type TenantProviderPort struct{}
type LocalizerPort struct{}
type DataSourcePort struct{}
type EmailSenderPort struct{}
type PubSubPort struct{}
type CacheProviderPort struct{}
type ErrorReporterPort struct{}
type TokenProviderPort struct{}
type CsrfTokenProviderPort struct{}

func (TenantProviderPort) String() string {
	return "TenantProviderPort"
}

func (LocalizerPort) String() string {
	return "LocalizerPort"
}

func (DataSourcePort) String() string {
	return "DataSourcePort"
}

func (EmailSenderPort) String() string {
	return "EmailSenderPort"
}

func (PubSubPort) String() string {
	return "PubSubPort"
}

func (CacheProviderPort) String() string {
	return "CacheProviderPort"
}

func (ErrorReporterPort) String() string {
	return "ErrorReporterPort"
}

func (TokenProviderPort) String() string {
	return "TokenProviderPort"
}

func (CsrfTokenProviderPort) String() string {
	return "CsrfTokenProviderPort"
}
