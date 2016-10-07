package main

import (
	"log"
	"os"

	"github.com/abourget/teamcity"
	"github.com/kisielk/gotool"
)

const teamcityAPIUserEnv = "TC_API_USER"
const teamcityAPIPasswordEnv = "TC_API_PASSWORD"

func main() {
	username, ok := os.LookupEnv(teamcityAPIUserEnv)
	if !ok {
		log.Fatalf("teamcity API username environment variable %s is not set", teamcityAPIUserEnv)
	}
	password, ok := os.LookupEnv(teamcityAPIPasswordEnv)
	if !ok {
		log.Fatalf("teamcity API password environment variable %s is not set", teamcityAPIPasswordEnv)
	}
	importPaths := gotool.ImportPaths([]string{"github.com/cockroachdb/cockroach/..."})

	client := teamcity.New("teamcity.cockroachdb.com", username, password)
	for _, importPath := range importPaths {
		build, err := client.QueueBuild("Cockroach_Nightlies_Stress", "master", map[string]string{
			"env.PKG": importPath,
		})
		if err != nil {
			log.Fatalf("failed to create teamcity build for %s: %s", importPath, err)
		}
		log.Printf("created teamcity build for %s: %s", importPath, build)
	}
}
