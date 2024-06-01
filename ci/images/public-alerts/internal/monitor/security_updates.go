package monitor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"public-alerts/internal/config"
	"public-alerts/internal/notify"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

////////////////////////////////////////////////////////////////////////////////
// SeecurityUpdateMonitor
////////////////////////////////////////////////////////////////////////////////

// SecurityUpdateMonitor struct to track last known states
type SecurityUpdatesMonitor struct {
	lastCommit   map[string]string
	lastBranches map[string]map[string]struct{}
	lastPRs      map[string]map[int]struct{}
	mu           sync.Mutex
}

// NewSecurityUpdateMonitor initializes the SecurityUpdateMonitor
func NewSecurityUpdatesMonitor() *SecurityUpdatesMonitor {
	return &SecurityUpdatesMonitor{
		lastCommit:   make(map[string]string),
		lastBranches: make(map[string]map[string]struct{}),
		lastPRs:      make(map[string]map[int]struct{}),
	}
}

// Name returns the name of the monitor
func (sum *SecurityUpdatesMonitor) Name() string {
	return "SecurityUpdatesMonitor"
}

////////////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////////////

// fetchJSON is a helper function to fetch JSON data from a URL
func fetchJSON(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch data, status code: %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

// FetchFunc defines a function type for fetching JSON data
type FetchFunc func(url string, target interface{}) error

// //////////////////////////////////////////////////////////////////////////////
// checkSecurityUpdates
// //////////////////////////////////////////////////////////////////////////////

// checkSecurityUpdates checks for new commits, branches, and PRs, and logs notifications
func checkSecurityUpdates(fetch FetchFunc, lastCommit map[string]string, lastBranches map[string]map[string]struct{}, lastPRs map[string]map[int]struct{}) ([]notify.Alert, error) {
	var alerts []notify.Alert
	// TODO add to config
	githubRepos := config.Get().SecurityUpdatesMonitor.Repos

	for _, repo := range githubRepos {
		// Check for new commits on master branch
		log.Info().Msgf("checking %s for commits...", repo)
		commitURL := fmt.Sprintf("https://api.github.com/repos/%s/branches/master", repo)
		var commitData struct {
			Commit struct {
				SHA     string `json:"sha"`
				Message string `json:"message"`
			} `json:"commit"`
		}
		if err := fetch(commitURL, &commitData); err != nil {
			return nil, err
		}
		commit := commitData.Commit.SHA
		if lastCommit[repo] != "" && lastCommit[repo] != commit {
			alertMessage := fmt.Sprintf("### New Commit Detected\n> **Repo:** https://github.com/%s\n> **Message:** `%s`", repo, strings.Split(commitData.Commit.Message, "\n")[0])
			alerts = append(alerts, notify.Alert{Webhooks: config.Get().Webhooks.Security, Message: alertMessage})
		}
		lastCommit[repo] = commit

		// Check for new branches
		log.Info().Msgf("checking %s for new branches...", repo)
		branchesURL := fmt.Sprintf("https://api.github.com/repos/%s/branches", repo)
		var branchesData []struct {
			Name string `json:"name"`
		}
		if err := fetch(branchesURL, &branchesData); err != nil {
			return nil, err
		}
		branches := make(map[string]struct{})
		for _, branch := range branchesData {
			branches[branch.Name] = struct{}{}
		}
		if lastBranches[repo] != nil {
			var newBranches []string
			for branch := range branches {
				if _, exists := lastBranches[repo][branch]; !exists {
					newBranches = append(newBranches, branch)
				}
			}
			if len(newBranches) > 0 {
				alertMessage := fmt.Sprintf("### New Branch Detected\n> **Repo:** https://github.com/%s", repo)
				for _, branch := range newBranches {
					alertMessage += fmt.Sprintf("\n> **Branch:** `%s`", branch)
				}
				alerts = append(alerts, notify.Alert{Webhooks: config.Get().Webhooks.Security, Message: alertMessage})
			}
		}
		lastBranches[repo] = branches

		// Check for new PRs
		prsURL := fmt.Sprintf("https://api.github.com/repos/%s/pulls", repo)
		var prsData []struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
			URL    string `json:"html_url"`
		}
		if err := fetch(prsURL, &prsData); err != nil {
			return nil, err
		}
		prs := make(map[int]struct{})
		for _, pr := range prsData {
			prs[pr.Number] = struct{}{}
		}
		if lastPRs[repo] != nil {
			var newPRs []struct {
				Number int
				Title  string
				URL    string
			}
			for _, pr := range prsData {
				if _, exists := lastPRs[repo][pr.Number]; !exists {
					newPRs = append(newPRs, struct {
						Number int
						Title  string
						URL    string
					}{Number: pr.Number, Title: pr.Title, URL: pr.URL})
				}
			}
			if len(newPRs) > 0 {
				alertMessage := fmt.Sprintf("### New PR Detected\n> **Repo:** https://github.com/%s", repo)
				for _, pr := range newPRs {
					alertMessage += fmt.Sprintf("\n> **PR:** [%s](%s)", pr.Title, pr.URL)
				}
				alerts = append(alerts, notify.Alert{Webhooks: config.Get().Webhooks.Security, Message: alertMessage})
			}
		}
		lastPRs[repo] = prs
	}

	return alerts, nil
}

// //////////////////////////////////////////////////////////////////////////////
// Check
// //////////////////////////////////////////////////////////////////////////////

func (sum *SecurityUpdatesMonitor) Check() ([]notify.Alert, error) {
	sum.mu.Lock()
	defer sum.mu.Unlock()
	log.Info().Msg("Checking for security updates (TSS Repo)...")
	return checkSecurityUpdates(fetchJSON, sum.lastCommit, sum.lastBranches, sum.lastPRs)
}
