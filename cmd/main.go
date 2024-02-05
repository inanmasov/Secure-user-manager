package main

import (
	"log"
	"net/http"

	reg "example.com/Go/internal/transport/router"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

func main() {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	reg.RegisterRoutes()

	log.Println("server start listening on port", viper.GetString("port"))
	err := http.ListenAndServe(":"+viper.GetString("port"), nil)
	if err != nil {
		log.Fatal(err)
	}
}
