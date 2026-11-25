package runtime

import (
	"fmt"

	"github.com/gabrielcampellounb/gopherflow/internal/token"
)

// CallDepthError already describes the innermost call that exceeded the limit.
type CallDepthError struct {
	Position token.Position
	Limit    int
}

func (e *CallDepthError) Error() string {
	return fmt.Sprintf("runtime error at %d:%d: maximum function call depth (%d) exceeded",
		e.Position.Line, e.Position.Column, e.Limit)
}
