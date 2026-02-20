package github

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
)

type NewTemplate struct {
	Token    string
	Username string
	RepoName string
	Config   TemplateConfig
}

// CreateTemplate creates a public GitHub repo with template.yaml and skeleton/README.md.
func CreateTemplate(t NewTemplate) (string, error) {
	if err := createTemplateRepo(t.Token, t.RepoName); err != nil {
		return "", err
	}

	yamlBytes, err := yaml.Marshal(t.Config)
	if err != nil {
		return "", err
	}

	readmeVar := "project_name"
	if len(t.Config.Variables) > 0 {
		readmeVar = t.Config.Variables[0].Name
	}
	readmeContent := "# {{" + readmeVar + "}}\n\n> Generated from the " + t.Config.Name + " kickstart template.\n"

	files := map[string]string{
		"template.yaml":      string(yamlBytes),
		"skeleton/README.md": readmeContent,
	}

	if err := pushTemplateFiles(t.Token, t.Username, t.RepoName, files); err != nil {
		return "", err
	}

	return "https://github.com/" + t.Username + "/" + t.RepoName, nil
}

func createTemplateRepo(token, repoName string) error {
	bodyStr := fmt.Sprintf(`{"name":"%s","private":false,"auto_init":true}`, repoName)
	req, err := http.NewRequest("POST", "https://api.github.com/user/repos", strings.NewReader(bodyStr))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create repo: %s", string(respBody))
	}
	return nil
}

func pushTemplateFiles(token, username, repoName string, files map[string]string) error {
	var treeEntries []map[string]string
	for path, content := range files {
		encoded := base64.StdEncoding.EncodeToString([]byte(content))
		sha, err := createTemplateBlob(token, username, repoName, encoded)
		if err != nil {
			return fmt.Errorf("blob for %s: %w", path, err)
		}
		treeEntries = append(treeEntries, map[string]string{
			"path": path,
			"mode": "100644",
			"type": "blob",
			"sha":  sha,
		})
	}

	parentSHA, err := getTemplateHeadSHA(token, username, repoName)
	if err != nil {
		return err
	}

	treeSHA, err := createTemplateTree(token, username, repoName, treeEntries)
	if err != nil {
		return err
	}

	commitSHA, err := createTemplateCommit(token, username, repoName, treeSHA, parentSHA)
	if err != nil {
		return err
	}

	return updateTemplateRef(token, username, repoName, commitSHA)
}

func createTemplateBlob(token, username, repoName, content string) (string, error) {
	bodyStr := fmt.Sprintf(`{"content":"%s","encoding":"base64"}`, content)
	req, _ := http.NewRequest("POST",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/git/blobs", username, repoName),
		strings.NewReader(bodyStr),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("blob failed: %s", string(respBody))
	}

	var result struct {
		SHA string `json:"sha"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.SHA, nil
}

func createTemplateTree(token, username, repoName string, entries []map[string]string) (string, error) {
	entriesJSON, _ := json.Marshal(entries)
	bodyStr := fmt.Sprintf(`{"tree":%s}`, string(entriesJSON))
	req, _ := http.NewRequest("POST",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees", username, repoName),
		strings.NewReader(bodyStr),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("tree failed: %s", string(respBody))
	}

	var result struct {
		SHA string `json:"sha"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.SHA, nil
}

func getTemplateHeadSHA(token, username, repoName string) (string, error) {
	req, _ := http.NewRequest("GET",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/git/ref/heads/main", username, repoName),
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("get HEAD failed: %s", string(respBody))
	}

	var result struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Object.SHA, nil
}

func createTemplateCommit(token, username, repoName, treeSHA, parentSHA string) (string, error) {
	bodyStr := fmt.Sprintf(`{"message":"Add template.yaml and skeleton","tree":"%s","parents":["%s"]}`, treeSHA, parentSHA)
	req, _ := http.NewRequest("POST",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/git/commits", username, repoName),
		strings.NewReader(bodyStr),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("commit failed: %s", string(respBody))
	}

	var result struct {
		SHA string `json:"sha"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.SHA, nil
}

func updateTemplateRef(token, username, repoName, commitSHA string) error {
	bodyStr := fmt.Sprintf(`{"sha":"%s"}`, commitSHA)
	req, _ := http.NewRequest("PATCH",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs/heads/main", username, repoName),
		strings.NewReader(bodyStr),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update ref failed: %s", string(respBody))
	}
	return nil
}
