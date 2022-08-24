package postgres

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"time"

	"github.com/phil-inc/admiral/config"
)

type Postgres struct {
	host       string
	port       int
	user       string
	password   string
	dbname     string
	connection *sql.DB
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

	err := checkMissingVars(p)
	if err != nil {
		return err
	}

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable", host, port, user, dbname)
	if password != "" {
		psqlInfo = fmt.Sprintf("%s password=%s", psqlInfo, password)
	}

	connection, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return err
	}
	p.connection = connection
	// Skipping a "defer connection.close()" call since the connection should die if the pod dies

	err = p.connection.Ping()
	if err != nil {
		return err
	}

	err = p.createTableIfNonexistent()
	if err != nil {
		return err
	}

	return nil
}

// Stream sends the logs to STDOUT
func (p *Postgres) Stream(log string, logMetadata map[string]string) error {
	sqlStatement := `INSERT INTO logs (stored_at, message, namespace, app, pod)
	VALUES ($1, $2, $3, $4, $5)`
	_, err := p.connection.Exec(sqlStatement, time.Now().Format("2006-01-02 15:04:05.000000"), log, logMetadata["namespace"], logMetadata["app"], logMetadata["pod"])
	if err != nil {
		panic(err)
	}
	// can we get region in here?
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
	sqlStatement := `CREATE TABLE IF NOT EXISTS logs (
		stored_at timestamp, 
		message text, 
		namespace text, 
		app text, 
		pod text
	);`
	_, err := p.connection.Exec(sqlStatement)
	if err != nil {
		return err
	}
	return nil
}
