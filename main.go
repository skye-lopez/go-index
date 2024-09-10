package main

import (
	"github.com/joho/godotenv"
	"github.com/skye-lopez/go-index/api"
)

/*
func main() {
	cmd.Execute()
}
*/

func main() {
	godotenv.Load(".env")
	api.Open()
}
