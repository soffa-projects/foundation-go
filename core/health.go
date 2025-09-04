package f

type HealthCheckComponent struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

type HealthCheckBuilder struct {
	Status     string                          `json:"status,omitempty"`
	Components map[string]HealthCheckComponent `json:"components,omitempty"`
}

func NewHealthCheckBuilder() HealthCheckBuilder {
	return HealthCheckBuilder{
		Status:     "UP",
		Components: make(map[string]HealthCheckComponent),
	}
}

func (b *HealthCheckBuilder) Add(name string, tester func() error) {
	var message string
	status := "UP"
	if err := tester(); err != nil {
		status = "DOWN"
		message = err.Error()
	}
	b.Components[name] = HealthCheckComponent{
		Message: message,
		Status:  status,
	}
}
