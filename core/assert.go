package f

import (
	"errors"

	"github.com/ztrue/tracerr"
)

func Check(value any, err string) {
	if value == nil {
		panic(tracerr.Wrap(errors.New(err)))
	}
}
