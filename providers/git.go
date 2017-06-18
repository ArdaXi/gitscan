package providers

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/src-d/go-billy.v2"
	"gopkg.in/src-d/go-billy.v2/memfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func init() {
	Providers["gitlab-local"] = newGitLabLocal
}

type gitlabLocalProvider struct {
	gitlab Provider
	path   string
}

func newGitLabLocal(opts *Options) (Provider, error) {
	if opts.Path == "" {
		return nil, errors.New("Path not provided.")
	}
	if _, err := os.Stat(opts.Path); os.IsNotExist(err) {
		return nil, err
	}

	provider, err := newGitlab(opts)
	if err != nil {
		return nil, err
	}

	return &gitlabLocalProvider{
		path:   opts.Path,
		gitlab: provider,
	}, nil
}

func (g *gitlabLocalProvider) ListAllProjects() <-chan Project {
	c := make(chan Project)
	go func(c chan<- Project) {
		for project := range g.gitlab.ListAllProjects() {
			p, err := g.newGitlabLocalProject(project)
			if err != nil {
				continue
			}

			c <- p
		}
		close(c)
	}(c)
	return c
}

func (g *gitlabLocalProvider) GetProject(id int) (Project, error) {
	project, err := g.gitlab.GetProject(id)
	if err != nil {
		return nil, err
	}

	return g.newGitlabLocalProject(project)
}

func (g *gitlabLocalProvider) Username() string {
	return g.gitlab.Username()
}

type gitlabLocalProject struct {
	project Project
	fs      billy.Filesystem
	repo    *git.Repository
}

func (g *gitlabLocalProvider) newGitlabLocalProject(project Project) (Project, error) {
	path := project.Path()
	fs := memfs.New()
	storer := memory.NewStorage()
	repo, err := git.Clone(storer, fs, &git.CloneOptions{
		URL:           fmt.Sprintf("file://%s.git", filepath.Join(g.path, path)),
		ReferenceName: plumbing.Master,
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		return nil, err
	}

	return &gitlabLocalProject{
		project: project,
		fs:      fs,
		repo:    repo,
	}, nil
}

func (p *gitlabLocalProject) ID() int {
	return p.project.ID()
}

func (p *gitlabLocalProject) Name() string {
	return p.project.Name()
}

func (p *gitlabLocalProject) URL() string {
	return p.project.URL()
}

func (p *gitlabLocalProject) Path() string {
	return p.project.Path()
}

func (p *gitlabLocalProject) LastCommit() (string, error) {
	ref, err := p.repo.Head()
	if err != nil {
		return "", err
	}

	return ref.Hash().String(), nil
}

func (p *gitlabLocalProject) Files() ([]File, error) {
	ref, err := p.repo.Head()
	if err != nil {
		return nil, err
	}

	commit, err := p.repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	var files []File

	for {
		file, err := tree.Files().Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		files = append(files, &gitlabLocalFile{file: file})
	}

	return files, nil
}

type gitlabLocalFile struct {
	file *object.File
}

func (f *gitlabLocalFile) Path() string {
	return f.file.Name
}

func (f *gitlabLocalFile) Size() (int, error) {
	return int(f.file.Size), nil
}

func (f *gitlabLocalFile) Contents() (io.Reader, error) {
	return f.file.Reader()
}
