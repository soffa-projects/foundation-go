package h

import (
	"context"

	"github.com/a-h/templ"
)

func RenderTempl(ctx context.Context, t templ.Component) (string, error) {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	if err := t.Render(ctx, buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}
