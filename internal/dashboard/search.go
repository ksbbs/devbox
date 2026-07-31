package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// searchClient bounds upstream registry API calls so a hung upstream cannot
// stall the request (or leak goroutines) forever.
var searchClient = &http.Client{Timeout: 10 * time.Second}

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

type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Page    int            `json:"page"`
	PerPage int            `json:"per_page"`
	HasMore bool           `json:"has_more"`
}

func (sh *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	registry := r.URL.Query().Get("registry")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 50 {
		perPage = 10
	}

	resp := SearchResponse{Page: page, PerPage: perPage}

	if query == "" {
		writeJSON(w, resp)
		return
	}

	var results []SearchResult

	if registry == "" || registry == "npm" {
		r, hasMore := searchNpm(query, page, perPage)
		results = append(results, r...)
		if hasMore {
			resp.HasMore = true
		}
	}
	if registry == "" || registry == "docker" {
		r, hasMore := searchDocker(query, page, perPage)
		results = append(results, r...)
		if hasMore {
			resp.HasMore = true
		}
	}
	if registry == "" || registry == "pypi" {
		r := searchPyPI(query)
		results = append(results, r...)
	}
	if registry == "" || registry == "conda" || registry == "conda-forge" {
		r := searchConda(query)
		results = append(results, r...)
	}
	if registry == "" || registry == "rubygems" {
		r, hasMore := searchRubyGems(query, page, perPage)
		results = append(results, r...)
		if hasMore {
			resp.HasMore = true
		}
	}
	if registry == "" || registry == "cargo" {
		r, hasMore := searchCargo(query, page, perPage)
		results = append(results, r...)
		if hasMore {
			resp.HasMore = true
		}
	}
	if registry == "" || registry == "nuget" {
		r, hasMore := searchNuGet(query, page, perPage)
		results = append(results, r...)
		if hasMore {
			resp.HasMore = true
		}
	}

	resp.Results = results
	writeJSON(w, resp)
}

func searchNpm(query string, page, perPage int) ([]SearchResult, bool) {
	resp, err := searchClient.Get(fmt.Sprintf("https://registry.npmjs.org/-/v1/search?text=%s&size=%d&from=%d", url.QueryEscape(query), perPage, (page-1)*perPage))
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	var data struct {
		Objects []struct {
			Package struct {
				Name string `json:"name"`
				Desc string `json:"description"`
			} `json:"package"`
		} `json:"objects"`
		Total int `json:"total"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, false
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
	return results, page*perPage < data.Total
}

func searchDocker(query string, page, perPage int) ([]SearchResult, bool) {
	resp, err := searchClient.Get(fmt.Sprintf("https://registry.hub.docker.com/v2/search/repositories/?query=%s&page_size=%d&page=%d", url.QueryEscape(query), perPage, page))
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	var data struct {
		Results []struct {
			RepoName  string `json:"repo_name"`
			ShortDesc string `json:"short_description"`
		} `json:"results"`
		Total int `json:"num_results"`
		Page  int `json:"page"`
		Pages int `json:"num_pages"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, false
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
	return results, page < data.Pages
}

func searchPyPI(query string) []SearchResult {
	resp, err := searchClient.Get(fmt.Sprintf("https://pypi.org/pypi/%s/json", url.QueryEscape(query)))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}

	var data struct {
		Info struct {
			Name    string `json:"name"`
			Summary string `json:"summary"`
		} `json:"info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil
	}
	return []SearchResult{{
		Registry: "pypi",
		Name:     data.Info.Name,
		Desc:     data.Info.Summary,
		URL:      "https://pypi.org/project/" + data.Info.Name,
	}}
}

func searchConda(query string) []SearchResult {
	resp, err := searchClient.Get(fmt.Sprintf("https://api.anaconda.org/package/%s", url.QueryEscape(query)))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}

	var data struct {
		Name        string `json:"name"`
		Summary     string `json:"summary"`
		PackageType string `json:"package_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil
	}
	channel := "main"
	if data.PackageType == "conda" {
		channel = "conda-forge"
	}
	return []SearchResult{{
		Registry: "conda",
		Name:     data.Name,
		Desc:     data.Summary,
		URL:      fmt.Sprintf("https://anaconda.org/%s/%s", channel, data.Name),
	}}
}

func searchRubyGems(query string, page, perPage int) ([]SearchResult, bool) {
	resp, err := searchClient.Get(fmt.Sprintf("https://rubygems.org/api/v1/search.json?query=%s&page=%d", url.QueryEscape(query), page))
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	var data []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Info        string `json:"info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, false
	}

	results := make([]SearchResult, 0, len(data))
	for _, r := range data {
		desc := r.Description
		if desc == "" {
			desc = r.Info
		}
		results = append(results, SearchResult{
			Registry: "rubygems",
			Name:     r.Name,
			Desc:     desc,
			URL:      "https://rubygems.org/gems/" + r.Name,
		})
	}
	return results, len(data) >= perPage
}

func searchCargo(query string, page, perPage int) ([]SearchResult, bool) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://crates.io/api/v1/crates?q=%s&page=%d&per_page=%d", url.QueryEscape(query), page, perPage), nil)
	req.Header.Set("User-Agent", "devbox/1.0")
	resp, err := searchClient.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	var data struct {
		Crates []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"crates"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, false
	}

	results := make([]SearchResult, 0, len(data.Crates))
	for _, c := range data.Crates {
		results = append(results, SearchResult{
			Registry: "cargo",
			Name:     c.Name,
			Desc:     c.Description,
			URL:      "https://crates.io/crates/" + c.Name,
		})
	}
	return results, page*perPage < data.Meta.Total
}

func searchNuGet(query string, page, perPage int) ([]SearchResult, bool) {
	skip := (page - 1) * perPage
	resp, err := searchClient.Get(fmt.Sprintf("https://azuresearch-usnc.nuget.org/query?q=%s&skip=%d&take=%d", url.QueryEscape(query), skip, perPage))
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	var data struct {
		Data []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
		} `json:"data"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, false
	}

	results := make([]SearchResult, 0, len(data.Data))
	for _, d := range data.Data {
		results = append(results, SearchResult{
			Registry: "nuget",
			Name:     d.ID,
			Desc:     d.Description,
			URL:      "https://www.nuget.org/packages/" + d.ID,
		})
	}
	return results, page*perPage < data.Total
}
