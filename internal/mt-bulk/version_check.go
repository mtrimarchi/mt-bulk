package mtbulk

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

func checkVersion(currentVersion string) error {
	type release struct {
		Draft   bool
		URL     string `json:"html_url"`
		Version string `json:"name"`
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://api.github.com/repos/migotom/mt-bulk/releases/latest", nil)
	if err != nil {
		return fmt.Errorf("Can't create request to fetch latest release info: %s", err)
	}

	req.Header.Add("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Can't fetch latest release info: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("Can't fetch latest release info, status code: %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Can't read details of latest release: %s", err)
	}

	var currentRelease release
	if err := json.Unmarshal(body, &currentRelease); err != nil {
		return fmt.Errorf("Can't parse details of latest release: %s", err)
	}

	if currentRelease.Draft {
		return nil
	}

	currentVersionInt, _ := parseVersion(currentVersion)
	releasedVersionInt, err := parseVersion(currentRelease.Version)

	if err != nil {
		return fmt.Errorf("Invalid version number in latest release: %s", err)
	}

	if currentVersionInt < releasedVersionInt {
		return fmt.Errorf("New version of MT-bulk v%v available at %v", currentRelease.Version, currentRelease.URL)
	}

	return nil
}

func parseVersion(version string) (result int64, err error) {
	if matches := regexp.MustCompile(`(\d+)\.(\d+)(?:\.(\d+))?`).FindStringSubmatch(version); len(matches) > 1 {
		v := ""
		for i := 1; i <= 3; i++ {
			v = fmt.Sprintf("%s%04s", v, matches[i])
		}

		if result, err = strconv.ParseInt(v, 10, 64); err != nil {
			return 0, err
		}
	}
	return
}
