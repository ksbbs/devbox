package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type SearchHandler struct{}

func NewSearchHandler() *SearchHandler {
	return &SearchHandler{}
}

type SearchResult struct {
	Registry string `json:"registry"`
	Name     string `json:"name"`
	Desc     string `json:"desc"`
	URL      string `json:"url"`
}

func (sh *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	registry := r.URL.Query().Get("registry")
	if query == "" {
		writeJSON(w, []SearchResult{})
		return
	}

	var results []SearchResult

	// Search npm registry
	if registry == "" || registry == "npm" {
		npmResults, err := searchNpm(query)
		if err == nil {
			results = append(results, npmResults...)
		}
	}

	// Search Docker Hub
	if registry == "" || registry == "docker" {
		dockerResults, err := searchDocker(query)
		if err == nil {
			results = append(results, dockerResults...)
		}
	}

	// Search PyPI
	if registry == "" || registry == "pypi" {
		pypiResults, err := searchPyPI(query)
		if err == nil {
			results = append(results, pypiResults...)
		}
	}

	writeJSON(w, results)
}

func searchNpm(query string) ([]SearchResult, error) {
	resp, err := http.Get(fmt.Sprintf("https://registry.npmjs.org/-/v1/search?text=%s&size=10", query))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Objects []struct {
			Package struct {
				Name string `json:"name"`
				Desc string `json:"description"`
				Link string `json:"links"`
			} `json:"package"`
		} `json:"objects"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(data.Objects))
	for _, obj := range data.Objects {
		results = append(results, SearchResult{
			Registry: "npm",
			Name:     obj.Package.Name,
			Desc:     obj.Package.Desc,
			URL:      "https://www.npmjs.com/package/" + obj.Package.Name,
		})
	}
	return results, nil
}

func searchDocker(query string) ([]SearchResult, error) {
	resp, err := http.Get(fmt.Sprintf("https://registry.hub.docker.com/v2/search/repositories/?query=%s&page_size=10", query))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Results []struct {
			RepoName  string `json:"repo_name"`
			ShortDesc string `json:"short_description"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(data.Results))
	for _, r := range data.Results {
		results = append(results, SearchResult{
			Registry: "docker",
			Name:     r.RepoName,
			Desc:     r.ShortDesc,
			URL:      "https://hub.docker.com/r/" + r.RepoName,
		})
	}
	return results, nil
}

func searchPyPI(query string) ([]SearchResult, error) {
	// PyPI JSON API for package search
	resp, err := http.Get(fmt.Sprintf("https://pypi.org/search/?q=%s&format=json", query))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// PyPI search API may not return JSON reliably, try simple package name lookup
	// Use the PyPI simple index approach as fallback
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("pypi search returned %d", resp.StatusCode)
	}

	// Try parsing as JSON
	var packages []struct {
		Name string `json:"name"`
		Desc string `json:"summary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&packages); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(packages))
	for _, p := range packages {
		results = append(results, SearchResult{
			Registry: "pypi",
			Name:     p.Name,
			Desc:     p.Desc,
			URL:      "https://pypi.org/project/" + p.Name,
		})
	}
	return results, nil
}