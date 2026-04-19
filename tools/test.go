//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/maharanidw/translator-pipeline/internal/config"
	"github.com/maharanidw/translator-pipeline/internal/models"
)

func main() {
	godotenv.Load()
	config.InitDB()
	var novel models.Novel
	err := config.DB.Where("title = ?", "Test").FirstOrCreate(&novel, models.Novel{
		Title:     "Test",
		SourceURL: "https://example.com/ch-1",
	}).Error
	fmt.Printf("Novel ID: %v, Error: %v\n", novel.ID, err)
}
