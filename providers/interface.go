package providers

import "io"

type Options struct {
	Token string
	URL   string
}

type Provider interface {
	ListAllProjects() <-chan Project
	GetProject(int) (Project, error)
	Username() string
}

type Project interface {
	Files() ([]File, error)
	Name() string
	URL() string
	ID() int
	LastCommit() (string, error)
}

type File interface {
	Path() string
	Size() (int, error)
	Contents() (io.Reader, error)
}

var Providers map[string]func(*Options) (Provider, error) = map[string]func(*Options) (Provider, error){}
