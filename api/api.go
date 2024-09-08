package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/syndtr/goleveldb/leveldb"
)

type API struct {
	SearchDB *leveldb.DB
}

var _api *API

func Open() {
	r := gin.Default()

	searchDB, err := leveldb.OpenFile(".go-index-store", nil)
	if err != nil {
		panic(err)
	}
	defer searchDB.Close()

	_api = &API{
		SearchDB: searchDB,
	}

	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "im alive :)",
		})
	})

	r.GET("/search/:s", func(c *gin.Context) {
		searchString := c.Param("s")

		data, status, message := _api.Search(searchString)
		c.JSON(status, gin.H{
			"entries": data,
			"message": message,
		})
	})
	r.Run()
}

// Entries, StatusCode, Message?
func (a *API) Search(searchString string) ([]string, int, string) {
	fmt.Println(searchString)
	entries, err := a.SearchDB.Get([]byte(searchString), nil)
	if errors.Is(err, leveldb.ErrNotFound) {
		return []string{}, 404, "No search results"
	}
	if err != nil {
		return []string{}, 500, "TODO: This other error stuff"
	}
	fmt.Println(string(entries))
	return strings.Split(string(entries), "~"), 200, "OK"
}
