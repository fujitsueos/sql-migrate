package migrate

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var numberPrefixRegex = regexp.MustCompile(`^(\d+).*$`)

type Migration struct {
	Id       int64
	FileName string
	Up       []string
	Down     []string
}

func ParseVersion(name string) (int64, error) {
	matches := numberPrefixRegex.FindStringSubmatch(name)
	if len(matches) < 2 {
		return 0, fmt.Errorf("No version number in %s", name)
	}
	return strconv.ParseInt(matches[1], 10, 64)
}

type PlannedMigration struct {
	*Migration
	Queries []string
}

type byId []*Migration

func (b byId) Len() int           { return len(b) }
func (b byId) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byId) Less(i, j int) bool { return b[i].Id < b[j].Id }

type MigrationRecord struct {
	Id        int64     `db:"id"`
	FileName  string    `db:"file_name"`
	AppliedAt time.Time `db:"applied_at"`
}
