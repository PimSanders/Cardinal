package misc

import (
	"encoding/json"
	"time"

	log "unknwon.dev/clog/v2"

	"Cardinal/internal/dbold"
	"Cardinal/internal/dynamic_config"
	"Cardinal/internal/locales"

	"github.com/gin-gonic/gin"
	"github.com/parnurzeal/gorequest"

	"Cardinal/internal/utils"
)

const GITHUB_RELEASE_API = "https://api.github.com/repos/vidar-team/Cardinal/releases/latest"

func CheckVersion() {
	// Check Cardinal version.
	resp, body, _ := gorequest.New().Get(GITHUB_RELEASE_API).Timeout(5 * time.Second).End()
	if resp != nil && resp.StatusCode == 200 {
		type releaseApiJson struct {
			Name        string `json:"name"`
			NodeID      string `json:"node_id"`
			PublishedAt string `json:"published_at"`
			TagName     string `json:"tag_name"`
		}

		var releaseData releaseApiJson
		err := json.Unmarshal([]byte(body), &releaseData)
		if err == nil {
			// Compare version.
			if !utils.CompareVersion(utils.VERSION, releaseData.TagName) {
				log.Info(locales.T("misc.version_out_of_date", gin.H{
					"currentVersion": utils.VERSION,
					"latestVersion":  releaseData.TagName,
				}))
			}
		}
	}
}

// CheckDatabaseVersion compares the database version in the dynamic_config with now version.
// It will show a alert if database need update.
func CheckDatabaseVersion() {
	databaseVersion := dynamic_config.Get(utils.DATBASE_VERSION)
	if databaseVersion != dbold.VERSION {
		log.Warn(locales.T("misc.database_version_out_of_date"))
	}
}
