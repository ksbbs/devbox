package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

	if registry == "" || registry == "npm" {
		npmResults, err := searchNpm(query)
		if err == nil {
			results = append(results, npmResults...)
		}
	}

	if registry == "" || registry == "docker" {
		dockerResults, err := searchDocker(query)
		if err == nil {
			results = append(results, dockerResults...)
		}
	}

	if registry == "" || registry == "pypi" {
		pypiResults, err := searchPyPI(query)
		if err == nil {
			results = append(results, pypiResults...)
		}
	}

	writeJSON(w, results)
}

func searchNpm(query string) ([]SearchResult, error) {
	resp, err := http.Get(fmt.Sprintf("https://registry.npmjs.org/-/v1/search?text=%s&size=10", url.QueryEscape(query)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Objects []struct {
			Package struct {
				Name string `json:"name"`
				Desc string `json:"description"`
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
	resp, err := http.Get(fmt.Sprintf("https://registry.hub.docker.com/v2/search/repositories/?query=%s&page_size=10", url.QueryEscape(query)))
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
	// PyPI search API is deprecated (returns Cloudflare HTML).
	// Use the JSON API for exact package name lookup instead.
	resp, err := http.Get(fmt.Sprintf("https://pypi.org/pypi/%s/json", url.QueryEscape(query)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Package not found, return empty results
		return nil, nil
	}

	var data struct {
		Info struct {
			Name    string `json:"name"`
			Summary string `json:"summary"`
		} `json:"info"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return []SearchResult{
		{
			Registry: "pypi",
			Name:     data.Info.Name,
			Desc:     data.Info.Summary,
			URL:      "https://pypi.org/project/" + data.Info.Name,
		},
	}, nil
}