package main

import (
	"github.com/skye-lopez/go-index/cmd"
	"github.com/skye-lopez/go-index/idx"
)

func main() {
	cmd.Execute()
}

func test_main() {
	idx.SaveIndexToDB()
}
