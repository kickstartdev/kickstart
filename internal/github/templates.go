package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kickstartdev/kickstart/internal/debug"
	"gopkg.in/yaml.v3"
)


type TemplateConfig struct {
	Name			string		`yaml:"name"`
	Description		string		`yaml:"description"`
	Branch			string		`yaml:"branch"`
	Variables		[]Variable	`yaml:"variables"`
}

type Variable struct {
	Name			string		`yaml:"name"`
	Description		string		`yaml:"description"`
	Default			string		`yaml:"default"`
	Required		bool		`yaml:"required"`
}

type Template struct {
	Config		TemplateConfig
	Owner		string
	Repo		string
}

func ListTemplates(token string, username string) ([]Template, error) {
	debug.Log("ListTemplates: listing repos for user=%s", username)

	repos, err := listUserRepos(token)
	if err != nil {
		return nil, err
	}
	debug.Log("ListTemplates: found %d repos", len(repos))

	var templates []Template
	for _, repo := range repos {
		debug.Log("ListTemplates: checking %s/%s for template.yaml", repo.Owner, repo.Name)
		cfg, err := GetTemplateConfig(token, repo.Owner, repo.Name)
		if err != nil {
			debug.Log("ListTemplates: no template in %s/%s: %v", repo.Owner, repo.Name, err)
			continue
		}

		debug.Log("ListTemplates: found template in %s/%s: %s", repo.Owner, repo.Name, cfg.Name)
		templates = append(templates, Template{
			Config: *cfg,
			Owner:  repo.Owner,
			Repo:   repo.Name,
		})
	}

	debug.Log("ListTemplates: returning %d templates", len(templates))
	return templates, nil
}

type repoInfo struct {
	Owner string
	Name  string
}

func listUserRepos(token string) ([]repoInfo, error) {
	var allRepos []repoInfo
	page := 1

	for {
		url := fmt.Sprintf("https://api.github.com/user/repos?per_page=100&affiliation=owner,collaborator,organization_member&page=%d", page)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			debug.Log("listUserRepos: API returned %d: %s", resp.StatusCode, string(body))
			return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
		}

		var repos []struct {
			Name    string `json:"name"`
			Private bool   `json:"private"`
			Owner   struct {
				Login string `json:"login"`
			} `json:"owner"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			return nil, err
		}

		if len(repos) == 0 {
			break
		}

		for _, r := range repos {
			debug.Log("listUserRepos: repo=%s/%s private=%v", r.Owner.Login, r.Name, r.Private)
			allRepos = append(allRepos, repoInfo{Owner: r.Owner.Login, Name: r.Name})
		}

		page++
	}

	return allRepos, nil
}


func GetTemplateConfig(token string, owner string, repo string) (*TemplateConfig, error) {
	req, err := http.NewRequest("GET",fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/template.yaml", owner, repo), nil )

	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("template.yaml not found %s/%s", owner, repo)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var cfg TemplateConfig
	if err := yaml.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("invalid template.yaml in %s/%s: %v", owner, repo, err)
	}

	return &cfg, nil
}

