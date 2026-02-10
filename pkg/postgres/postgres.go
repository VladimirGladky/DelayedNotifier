package postgres

import (
	"fmt"

	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/dbpg"
)

func NewPostgres(config *config.Config) (*dbpg.DB, error) {
	opts := &dbpg.Options{MaxOpenConns: 10, MaxIdleConns: 5}
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.GetString("postgres_user"),
		config.GetString("postgres_password"),
		config.GetString("postgres_host"),
		config.GetInt("postgres_port"),
		config.GetString("postgres_dbname"),
	)
	var slaveDSNs []string
	db, err := dbpg.New(dsn, slaveDSNs, opts)
	if err != nil {
		return nil, err
	}
	return db, nil
}
