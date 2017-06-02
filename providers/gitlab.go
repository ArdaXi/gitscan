package providers

import (
	"bytes"
	"encoding/base64"
	"io"
	"log"

	"github.com/ardaxi/go-gitlab"
)

func init() {
	Providers["gitlab"] = newGitlab
}

type gitlabProvider struct {
	client *gitlab.Client
	user   *gitlab.User
}

func newGitlab(opts *Options) (Provider, error) {
	client := gitlab.NewClient(nil, opts.Token)
	if opts.URL != "" {
		if err := client.SetBaseURL(opts.URL); err != nil {
			return nil, err
		}
	}

	user, _, err := client.Users.CurrentUser()
	if err != nil {
		return nil, err
	}

	log.Printf("Logged in to GitLab as: %s", user.Username)

	return &gitlabProvider{client: client, user: user}, nil
}

func (g *gitlabProvider) ListAllProjects() <-chan Project {
	c := make(chan Project, 20)
	opts := gitlab.ListProjectsOptions{}
	opts.PerPage = 20
	opts.Page = 1
	go func(c chan<- Project, opts gitlab.ListProjectsOptions) {
		for {
			projects, resp, err := g.client.Projects.ListVisibleProjects(&opts)
			if err != nil {
				continue
			}

			for _, p := range projects {
				c <- &gitlabProject{
					provider: g,
					id:       p.ID,
					webURL:   p.WebURL,
					name:     p.Name,
					branch:   p.DefaultBranch,
				}
			}
			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	}(c, opts)
	return c
}

type gitlabProject struct {
	provider *gitlabProvider
	id       int
	webURL   string
	name     string
	branch   string
}

func (p *gitlabProject) ID() int {
	return p.id
}

func (p *gitlabProject) Name() string {
	return p.name
}

func (p *gitlabProject) URL() string {
	return p.webURL
}

func (p *gitlabProject) LastCommit() (string, error) {
	branch, _, err := p.provider.client.Branches.GetBranch(p.id, p.branch)
	if err != nil {
		return "", err
	}

	return branch.Commit.ID, nil
}

func (p *gitlabProject) Files() ([]File, error) {
	files, _, err := p.provider.client.Repositories.ListTree(p.id, &gitlab.ListTreeOptions{Recursive: gitlab.Bool(true)})
	if err != nil {
		return nil, err
	}

	var result []File
	for _, f := range files {
		result = append(result, &gitlabFile{
			project:  p,
			id:       f.ID,
			nodeType: f.Type,
			path:     f.Path,
		})
	}
	return result, nil
}

type gitlabFile struct {
	project  *gitlabProject
	id       string
	nodeType string
	path     string
	data     *gitlabFileData
}

func (f *gitlabFile) populate() error {
	opts := &gitlab.GetFileOptions{
		FilePath: &f.path,
		Ref:      gitlab.String("master"),
	}
	file, _, err := f.project.provider.client.RepositoryFiles.GetFile(f.project.id, opts)
	if err != nil {
		return err
	}

	content, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return err
	}

	f.data = &gitlabFileData{
		size:    file.Size,
		content: content,
	}

	return nil
}

func (f *gitlabFile) Path() string {
	return f.path
}

func (f *gitlabFile) Size() (int, error) {
	if f.data == nil {
		if err := f.populate(); err != nil {
			return 0, err
		}
	}

	return f.data.size, nil
}

func (f *gitlabFile) Contents() (io.Reader, error) {
	if f.data == nil {
		if err := f.populate(); err != nil {
			return nil, err
		}
	}

	return bytes.NewReader(f.data.content), nil
}

type gitlabFileData struct {
	size    int
	content []byte
}
