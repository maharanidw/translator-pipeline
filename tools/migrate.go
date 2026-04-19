package main

import (
    "github.com/maharanidw/translator-pipeline/internal/config"
    "github.com/joho/godotenv"
    "log"
)

func main() {
    godotenv.Load()
    config.InitDB()
    if err := config.DB.Exec("ALTER TABLE chapters ALTER COLUMN language_source TYPE character varying(50)").Error; err != nil {
        log.Fatal(err)
    }
    log.Println("Altered language_source length!")
}
