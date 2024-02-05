package database

import (
	"database/sql"

	"github.com/spf13/viper"
)

type Database struct {
	DB *sql.DB
}

func Initialize() (*Database, error) {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	connect_db := "host=" + viper.GetString("db.host") + " " + "user=" + viper.GetString("db.username") + " " + "port=" + viper.GetString("db.port") + " " + "password=" + viper.GetString("db.password") + " " + "dbname=" + viper.GetString("db.dbname") + " " + "sslmode=" + viper.GetString("db.sslmode")
	db, err := sql.Open("postgres", connect_db)
	if err != nil {
		panic(err)
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &Database{DB: db}, nil
}

func (db *Database) Close() error {
	return db.DB.Close()
}
