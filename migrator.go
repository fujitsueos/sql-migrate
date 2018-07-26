package migrate

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/rubenv/sql-migrate/sqlparse"
	log "github.com/sirupsen/logrus"
	"gopkg.in/gorp.v2"
)

type MigrationDirection int

const (
	Up MigrationDirection = iota
	Down
)

var tableName = "migrations"

type Migrator struct {
	tx         *gorp.Transaction
	dbMap      *gorp.DbMap
	migrations []*Migration
}

func NewMigrator(connStr, dir string) (migrator *Migrator, err error) {
	var db *sql.DB

	if db, err = sql.Open("postgres", connStr); err != nil {
		return
	}

	dbMap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}
	dbMap.AddTableWithName(MigrationRecord{}, tableName).SetKeys(false, "Id")

	if err = dbMap.CreateTablesIfNotExists(); err != nil {
		return
	}

	migrator = &Migrator{dbMap: dbMap}
	err = migrator.findMigrations(dir)

	return
}

func (m *Migrator) begin() (err error) {
	if m.tx, err = m.dbMap.Begin(); err != nil {
		return
	}

	_, err = m.tx.Exec(fmt.Sprintf("LOCK TABLE %s IN ACCESS EXCLUSIVE MODE", tableName))
	return
}

func (m *Migrator) findMigrations(dir string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	m.migrations = make([]*Migration, 0, len(files))
	for _, info := range files {
		if strings.HasSuffix(info.Name(), ".sql") {

			file, err := os.Open(filepath.Join(dir, info.Name()))
			if err != nil {
				return fmt.Errorf("Error while opening %s: %s", info.Name(), err)
			}

			migration := &Migration{FileName: info.Name()}
			if migration.Id, err = ParseVersion(info.Name()); err != nil {
				return err
			}

			parsed, err := sqlparse.ParseMigration(file)
			if err != nil {
				return fmt.Errorf("Error parsing migration (%d): %s", migration.Id, err)
			}
			migration.Up = parsed.UpStatements
			migration.Down = parsed.DownStatements

			if err != nil {
				return fmt.Errorf("Error while parsing %s: %s", info.Name(), err)
			}

			m.migrations = append(m.migrations, migration)
		}
	}

	// Make sure migrations are sorted
	sort.Sort(byId(m.migrations))

	return nil
}

// Execute a set of migrations
//
// Returns the number of applied migrations.
func (m *Migrator) Exec(dir MigrationDirection) (applied int, err error) {
	if err = m.begin(); err != nil {
		return
	}

	var migrations []*PlannedMigration
	if migrations, err = m.planMigration(dir); err != nil {
		if rollbackErr := m.tx.Rollback(); rollbackErr != nil {
			log.WithField("error", err).Error("Failed to roll back")
		}
		return
	}

	// Apply migrations
	for _, migration := range migrations {
		log.Infof("Applying %s", migration.FileName)
		for _, stmt := range migration.Queries {
			if _, err := m.tx.Exec(stmt); err != nil {
				if rollbackErr := m.tx.Rollback(); rollbackErr != nil {
					log.WithField("error", err).Error("Failed to roll back")
				}
				return applied, newTxError(migration, err)
			}
		}

		if dir == Up {
			if err = m.tx.Insert(&MigrationRecord{
				Id:        migration.Id,
				FileName:  migration.FileName,
				AppliedAt: time.Now(),
			}); err != nil {
				if rollbackErr := m.tx.Rollback(); rollbackErr != nil {
					log.WithField("error", err).Error("Failed to roll back")
				}
				return applied, newTxError(migration, err)
			}
		} else if dir == Down {
			if _, err = m.tx.Delete(&MigrationRecord{
				Id: migration.Id,
			}); err != nil {
				if rollbackErr := m.tx.Rollback(); rollbackErr != nil {
					log.WithField("error", err).Error("Failed to roll back")
				}
				return applied, newTxError(migration, err)
			}
		}

		applied++
	}

	err = m.tx.Commit()
	return
}

// Plan a migration.
func (m *Migrator) planMigration(dir MigrationDirection) ([]*PlannedMigration, error) {
	var migrationRecords []MigrationRecord
	if _, err := m.tx.Select(&migrationRecords, "SELECT * FROM $1", tableName); err != nil {
		return nil, err
	}

	// Sort migrations that have been run by Id.
	var existingMigrations []*Migration
	for _, migrationRecord := range migrationRecords {
		existingMigrations = append(existingMigrations, &Migration{
			Id: migrationRecord.Id,
		})
	}
	sort.Sort(byId(existingMigrations))

	// Get last migration that was run
	record := &Migration{}
	if len(existingMigrations) > 0 {
		record = existingMigrations[len(existingMigrations)-1]
	}

	result := make([]*PlannedMigration, 0)

	// Figure out which migrations to apply
	apply := m.filter(record.Id, dir)
	for _, v := range apply {

		if dir == Up {
			result = append(result, &PlannedMigration{
				Migration: v,
				Queries:   v.Up,
			})
		} else if dir == Down {
			result = append(result, &PlannedMigration{
				Migration: v,
				Queries:   v.Down,
			})
		}
	}

	return result, nil
}

// Filter a slice of migrations into ones that should be applied.
func (m *Migrator) filter(current int64, direction MigrationDirection) []*Migration {
	var index = -1
	if current > 0 {
		for index < len(m.migrations)-1 {
			index++
			if m.migrations[index].Id == current {
				break
			}
		}
	}

	if direction == Up {
		return m.migrations[index+1:]
	}

	if index == -1 {
		return []*Migration{}
	}

	// Add in reverse order
	toApply := make([]*Migration, index+1)
	for i := 0; i < index+1; i++ {
		toApply[index-i] = m.migrations[i]
	}
	return toApply
}
