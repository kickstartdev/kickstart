package scaffold

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Step struct {
	Name string
	Fn   func() error
}

type Scaffolder struct {
	Token      string
	Owner      string
	Repo       string
	Branch     string
	ProjectName string
	Variables  map[string]string
	OutputDir  string
}

func New(token, owner, repo, branch, projectName string, variables map[string]string) *Scaffolder {
	if branch == "" {
		branch = "main"
	}
	return &Scaffolder{
		Token:       token,
		Owner:       owner,
		Repo:        repo,
		Branch:      branch,
		ProjectName: projectName,
		Variables:   variables,
		OutputDir:   filepath.Join(".", projectName),
	}
}

func (s *Scaffolder) Steps() []Step {
	return []Step{
		{Name: "Downloading skeleton", Fn: s.downloadSkeleton},
		{Name: "Replacing variables", Fn: s.replaceVariables},
		{Name: "Creating GitHub repository", Fn: s.createRepo},
		{Name: "Pushing files", Fn: s.pushFiles},
		{Name: "Cloning locally", Fn: s.cloneRepo},
	}
}

// Step 1: Download skeleton/ folder from the template repo
func (s *Scaffolder) downloadSkeleton() error {
	return s.downloadDir("skeleton", s.OutputDir)
}

func (s *Scaffolder) downloadDir(remotePath string, localPath string) error {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", s.Owner, s.Repo, remotePath, s.Branch),
		nil,
	)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to list %s: status %d", remotePath, resp.StatusCode)
	}

	var contents []struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		Type        string `json:"type"`
		DownloadURL string `json:"download_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return err
	}

	if err := os.MkdirAll(localPath, 0755); err != nil {
		return err
	}

	for _, item := range contents {
		localItemPath := filepath.Join(localPath, item.Name)

		if item.Type == "dir" {
			if err := s.downloadDir(item.Path, localItemPath); err != nil {
				return err
			}
		} else {
			if err := s.downloadFile(item.DownloadURL, localItemPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Scaffolder) downloadFile(url string, dest string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return os.WriteFile(dest, data, 0644)
}

// Step 2: Walk through all files and replace {{variable}} placeholders
func (s *Scaffolder) replaceVariables() error {
	return filepath.Walk(s.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		content := string(data)
		for key, value := range s.Variables {
			content = strings.ReplaceAll(content, "{{"+key+"}}", value)
		}

		return os.WriteFile(path, []byte(content), 0644)
	})
}

// Step 3: Create a new GitHub repo
func (s *Scaffolder) createRepo() error {
	body := fmt.Sprintf(`{"name":"%s","private":true,"auto_init":true}`, s.ProjectName)

	req, err := http.NewRequest("POST", "https://api.github.com/user/repos", strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create repo: %d %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Step 4: Push files to the new repo using GitHub's Git API
func (s *Scaffolder) pushFiles() error {
	// get the username from the token
	username, err := s.getUsername()
	if err != nil {
		return err
	}

	// collect all files
	var files []fileEntry
	err = filepath.Walk(s.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(s.OutputDir, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		files = append(files, fileEntry{
			Path:    relPath,
			Content: base64.StdEncoding.EncodeToString(data),
		})
		return nil
	})
	if err != nil {
		return err
	}

	// create blobs
	var treeEntries []map[string]string
	for _, f := range files {
		sha, err := s.createBlob(username, f.Content)
		if err != nil {
			return fmt.Errorf("blob for %s: %w", f.Path, err)
		}
		treeEntries = append(treeEntries, map[string]string{
			"path": f.Path,
			"mode": "100644",
			"type": "blob",
			"sha":  sha,
		})
	}

	// create tree
	treeSHA, err := s.createTree(username, treeEntries)
	if err != nil {
		return err
	}

	// get current commit SHA of main (from auto_init)
	parentSHA, err := s.getRefSHA(username)
	if err != nil {
		return err
	}

	// create commit with parent
	commitSHA, err := s.createCommit(username, treeSHA, "Initial scaffold from "+s.Repo, parentSHA)
	if err != nil {
		return err
	}

	// update main branch ref to point to new commit
	return s.updateRef(username, commitSHA)
}

type fileEntry struct {
	Path    string
	Content string // base64
}

func (s *Scaffolder) getUsername() (string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var user struct {
		Login string `json:"login"`
	}
	json.NewDecoder(resp.Body).Decode(&user)
	return user.Login, nil
}

func (s *Scaffolder) createBlob(username string, content string) (string, error) {
	body := fmt.Sprintf(`{"content":"%s","encoding":"base64"}`, content)
	req, _ := http.NewRequest("POST",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/git/blobs", username, s.ProjectName),
		strings.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create blob: %d %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		SHA string `json:"sha"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.SHA, nil
}

func (s *Scaffolder) createTree(username string, entries []map[string]string) (string, error) {
	entriesJSON, _ := json.Marshal(entries)
	body := fmt.Sprintf(`{"tree":%s}`, string(entriesJSON))

	req, _ := http.NewRequest("POST",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees", username, s.ProjectName),
		strings.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create tree: %d %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		SHA string `json:"sha"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.SHA, nil
}

func (s *Scaffolder) createCommit(username string, treeSHA string, message string, parentSHA string) (string, error) {
	body := fmt.Sprintf(`{"message":"%s","tree":"%s","parents":["%s"]}`, message, treeSHA, parentSHA)

	req, _ := http.NewRequest("POST",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/git/commits", username, s.ProjectName),
		strings.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create commit: %d %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		SHA string `json:"sha"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.SHA, nil
}

func (s *Scaffolder) getRefSHA(username string) (string, error) {
	req, _ := http.NewRequest("GET",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/git/ref/heads/main", username, s.ProjectName),
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get ref: %d %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Object.SHA, nil
}

func (s *Scaffolder) updateRef(username string, commitSHA string) error {
	body := fmt.Sprintf(`{"sha":"%s"}`, commitSHA)

	req, _ := http.NewRequest("PATCH",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs/heads/main", username, s.ProjectName),
		strings.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update ref: %d %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Step 5: Clone the repo locally (replace the downloaded skeleton)
func (s *Scaffolder) cloneRepo() error {
	// remove the skeleton we downloaded
	os.RemoveAll(s.OutputDir)

	username, err := s.getUsername()
	if err != nil {
		return err
	}

	// use git clone with the token embedded
	cloneURL := fmt.Sprintf("https://%s@github.com/%s/%s.git", s.Token, username, s.ProjectName)

	cmd := execCommand("git", "clone", cloneURL, s.OutputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s", string(output))
	}

	return nil
}