package app

import (
	"io"
	"log"
)

type Options struct {
	Logger *log.Logger
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}
