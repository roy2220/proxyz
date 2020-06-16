package foo

import (
	"net/http"
)

type File2 http.File
type ErrX error
type ErrY = error

type ErrZ interface {
	error
	TT
}

type TT interface {
	Test(http.File)
}
