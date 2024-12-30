package ghrepos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"sort"
	"strconv"
)

const GH_REPOS_PAGE_SIZE = 120

type GHAPIFacade struct {
	BaseURL    string
	GHAPIKey   string
	GHOwner    string
	HttpClient *http.Client
}

type Option func(*GHAPIFacade)

func WithBaseURL(baseUrl string) Option {
	return func(g *GHAPIFacade) {
		g.BaseURL = baseUrl
	}
}

func WithGHAPIKey(ghApiKey string) Option {
	return func(g *GHAPIFacade) {
		g.GHAPIKey = ghApiKey
	}
}

func WithGHOwner(owner string) Option {
	return func(g *GHAPIFacade) {
		g.GHOwner = owner
	}
}

func NewGHAPIFacade(opts ...Option) *GHAPIFacade {
	client := &GHAPIFacade{
		BaseURL:    "https://api.github.com/user/repos",
		HttpClient: &http.Client{},
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

type Repo struct {
	Visibility  string `json:"visibility"`
	Fork        bool   `json:"fork"`
	Name        string `json:"name"`
	Archived    bool   `json:"archived"`
	CloneUrl    string `json:"clone_url"`
	ghApiFacade *GHAPIFacade
}

func (repo *Repo) Delete() error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", repo.ghApiFacade.GHOwner, repo.Name)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header = http.Header{
		"Authorization":        {fmt.Sprintf("Bearer %v", repo.ghApiFacade.GHAPIKey)},
		"Accept":               {"application/vnd.github+json"},
		"X-GitHub-Api-Version": {"2022-11-28"},
	}

	dumpReq, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return fmt.Errorf("failed to dump request: %w", err)
	}
	slog.Debug(fmt.Sprintf("request:\n%s\n", string(dumpReq)))

	resp, err := repo.ghApiFacade.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (repo *Repo) Open() error {
	cmd := exec.Command("open", repo.CloneUrl)
	if err := cmd.Run(); err != nil {
		slog.Debug("Error", slog.Any("Couldn't open", err))
	}
	return nil
}

func (repo *Repo) Archive() error {
	updateFields := map[string]interface{}{
		"archived": true,
	}

	err := repo.update(updateFields)
	if err != nil {
		return fmt.Errorf("Failed to update repo: %v", err)
	}
	return nil
}

func (repo *Repo) Unarchive() error {
	updateFields := map[string]interface{}{
		"archived": false,
	}

	err := repo.update(updateFields)
	if err != nil {
		return fmt.Errorf("Failed to update repo: %v", err)
	}
	return nil
}

func (repo *Repo) MakePublic() error {
	updateFields := map[string]interface{}{
		"visibility": "public",
	}

	err := repo.update(updateFields)
	if err != nil {
		return fmt.Errorf("Failed to update repo: %v", err)
	}
	return nil
}

func (repo *Repo) MakePrivate() error {
	updateFields := map[string]interface{}{
		"visibility": "private",
	}

	err := repo.update(updateFields)
	if err != nil {
		return fmt.Errorf("Failed to update repo: %v", err)
	}
	return nil
}

func (repo *Repo) update(updateFields map[string]interface{}) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", repo.ghApiFacade.GHOwner, repo.Name)
	payload, err := json.Marshal(updateFields)
	if err != nil {
		return fmt.Errorf("failed to marshal update fields: %w", err)
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header = http.Header{
		"Authorization":        {fmt.Sprintf("Bearer %v", repo.ghApiFacade.GHAPIKey)},
		"Accept":               {"application/vnd.github+json"},
		"Content-Type":         {"application/json"},
		"X-GitHub-Api-Version": {"2022-11-28"},
	}

	dumpReq, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return fmt.Errorf("failed to dump request: %w", err)
	}
	slog.Debug(fmt.Sprintf("request:\n%s\n", string(dumpReq)))

	resp, err := repo.ghApiFacade.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(body, &repo)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func (facade *GHAPIFacade) GetRepos() ([]*Repo, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/repos", nil)
	if err != nil {
		log.Fatal(err)
	}
	q := req.URL.Query()
	q.Add("per_page", strconv.Itoa(GH_REPOS_PAGE_SIZE))
	req.URL.RawQuery = q.Encode()
	req.Header = http.Header{
		"Authorization":        {fmt.Sprintf("Bearer %v", facade.GHAPIKey)},
		"Accept":               {"application/vnd.github+json"},
		"X-GitHub-Api-Version": {"2022-11-28"},
	}

	dumpReq, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Fatal(err)
	}
	slog.Debug(fmt.Sprintf("request:\n%s\n", string(dumpReq)))

	resp, err := facade.HttpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("error: received status code %d\n", resp.StatusCode)
		os.Exit(1)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var repos []*Repo
	err = json.Unmarshal(body, &repos)
	if err != nil {
		log.Fatal(err)
	}

	for _, repo := range repos {
		repo.ghApiFacade = facade
		_ = repo.ghApiFacade
	}

	slog.Debug("number of repos fetched", "NrRepos", len(repos))

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	return repos, nil
}
