# sql-migrate

Migration tool based on rubenv's [sql-migrate](https://github.com/rubenv/sql-migrate).

## Usage

Import sql-migrate into your application:

```go
package main

import (
    log "github.com/sirupsen/logrus"
    migrate "github.com/fujitsueos/sql-migrate"
)

func main() {
    m, err := migrate.NewMigrator("host=localhost port=5432 user=tag_service", "migrations")
    if err != nil {
        log.Fatal(err)
    }

    n, err := m.Exec(migrate.Up)
    if err != nil {
        log.Fatal(err)
    }

    log.Infof("Applied %d migrations\n", n)
}

```

## Writing migrations
Migrations are defined in SQL files, which contain a set of SQL statements in [goose](https://bitbucket.org/liamstask/goose) format.

```sql
-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE people (id int);


-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE people;
```

You can put multiple statements in each block, as long as you end them with a semicolon (`;`).

If you have complex statements which contain semicolons, use `StatementBegin` and `StatementEnd` to indicate boundaries:

```sql
-- +migrate Up
CREATE TABLE people (id int);

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION do_something()
returns void AS $$
DECLARE
  create_query text;
BEGIN
  -- Do something here
END;
$$
language plpgsql;
-- +migrate StatementEnd

-- +migrate Down
DROP FUNCTION do_something();
DROP TABLE people;
```

The order in which migrations are applied is defined through the filename: sql-migrate will sort migrations based on their name. Use an increasing version number as the first part of the filename.
