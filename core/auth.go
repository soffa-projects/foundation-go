package f

import "context"

type AuthProvider interface {
	Authenticate(ctx context.Context, authToken string) (*Authentication, error)
}
