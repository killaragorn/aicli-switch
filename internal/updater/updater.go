package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	repoOwner  = "killaragorn"
	repoName   = "aicli-switch"
	releaseURL = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// CheckForUpdate checks GitHub for a newer version. Non-blocking: prints a
// notice if a newer version exists, silently returns on any error.
func CheckForUpdate(currentVersion string) {
	latest, url, err := fetchLatestVersion()
	if err != nil || latest == "" {
		return
	}

	latest = strings.TrimPrefix(latest, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	if latest != current && isNewer(latest, current) {
		fmt.Fprintf(os.Stderr,
			"\033[33mUpdate available: %s → %s\033[0m\n"+
				"  Run: npm update -g @kio_ai/aicli-switch\n"+
				"  Or:  %s\n\n",
			current, latest, url)
	}
}

func fetchLatestVersion() (tag, url string, err error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", releaseURL, nil)
	req.Header.Set("User-Agent", "aicli-switch")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var rel githubRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return "", "", err
	}

	return rel.TagName, rel.HTMLURL, nil
}

// isNewer returns true if a > b using simple semver comparison.
func isNewer(a, b string) bool {
	ap := splitVersion(a)
	bp := splitVersion(b)
	for i := 0; i < 3; i++ {
		if ap[i] > bp[i] {
			return true
		}
		if ap[i] < bp[i] {
			return false
		}
	}
	return false
}

func splitVersion(v string) [3]int {
	var parts [3]int
	fmt.Sscanf(v, "%d.%d.%d", &parts[0], &parts[1], &parts[2])
	return parts
}
