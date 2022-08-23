package postgres

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"

	"github.com/phil-inc/admiral/config"
	"github.com/sirupsen/logrus"
)

type Postgres struct {
	host     string
	port     int
	user     string
	password string
	dbname   string
}

func (p *Postgres) Init(c *config.Config) error {
	host := c.Logstream.Logstore.Postgres.Host
	port := c.Logstream.Logstore.Postgres.Port
	user := c.Logstream.Logstore.Postgres.User
	password := c.Logstream.Logstore.Postgres.Password
	dbname := c.Logstream.Logstore.Postgres.DBName

	p.host = host
	p.port = port
	p.user = user
	p.password = password
	p.dbname = dbname

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable", host, port, user, dbname)
	if password != "" {
		psqlInfo = fmt.Sprintf("%s password=%s", psqlInfo, password)
	}

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	logrus.Printf("Successfully connected!")

	sqlStatement := `CREATE TABLE IF NOT EXISTS logs (i integer);`
	result, err := db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
	logrus.Printf("Result was: %s", result)

	// p.createTableIfNonexistent()

	return checkMissingVars(p)
}

// Stream sends the logs to STDOUT
func (p *Postgres) Stream(log string, logMetadata map[string]string) error {
	logrus.Printf(log)
	return nil
}

func checkMissingVars(p *Postgres) error {
	if p.host == "" {
		return fmt.Errorf("Postgres host not set")
	}

	if p.port == 0 {
		return fmt.Errorf("Postgres port not set")
	}

	if p.user == "" {
		return fmt.Errorf("Postgres user not set")
	}

	if p.dbname == "" {
		return fmt.Errorf("Postgress dbname not set")
	}

	return nil
}

func (p *Postgres) createTableIfNonexistent() error {
	return nil
}
