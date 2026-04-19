//go:build ignore
// +build ignore

package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/maharanidw/translator-pipeline/internal/config"
)

func main() {
	godotenv.Load()
	config.InitDB()
	if err := config.DB.Exec("ALTER TABLE chapters ALTER COLUMN language_source TYPE character varying(50)").Error; err != nil {
		log.Fatal(err)
	}
	log.Println("Altered language_source length!")
}
