package utils

import "errors"

// ErrIgnorable stops the remaining flows without failing the job
var ErrIgnorable = errors.New("do not fail")

// ErrNoPrint fails the job, the detail has already been printed
var ErrNoPrint = errors.New("do not print")
