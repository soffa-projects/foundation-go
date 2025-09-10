package f

type HealthCheckResponse struct {
	Whoami     string                          `json:"whoami"`
	Status     string                          `json:"status"`
	Components map[string]HealthCheckComponent `json:"components"`
}

type HealthCheckComponent struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

type HealthCheck struct {
	service    string
	status     string
	components map[string]HealthCheckComponent
}

func NewHealthCheck(service string) HealthCheck {
	return HealthCheck{
		service:    service,
		status:     "UP",
		components: make(map[string]HealthCheckComponent),
	}
}

func (b *HealthCheck) Add(name string, tester func() error) {
	var message string
	status := "UP"
	if err := tester(); err != nil {
		status = "DOWN"
		message = err.Error()
		b.status = "DOWN"
	}
	b.components[name] = HealthCheckComponent{
		Message: message,
		Status:  status,
	}
}

func (b *HealthCheck) Build() HealthCheckResponse {
	return HealthCheckResponse{
		Whoami:     b.service,
		Status:     b.status,
		Components: b.components,
	}
}
