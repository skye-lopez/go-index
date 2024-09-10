package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/skye-lopez/go-index/pg"
)

type API struct {
	Db *pg.PG
}

var _api *API

func Open() {
	r := gin.Default()

	db, err := pg.NewPg()
	if err != nil {
		log.Fatalf("Could not open db\n%e", err)
	}

	_api = &API{
		Db: db,
	}

	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "im alive :)",
		})
	})

	r.GET("/search/by-path", func(c *gin.Context) {
		search := c.DefaultQuery("search", "")
		page := c.DefaultQuery("page", "0")
		limit := c.DefaultQuery("limit", "20")
		suffix := c.DefaultQuery("suffix", "false")

		if suffix != "true" && suffix != "false" {
			c.JSON(400, gin.H{"message": "Provided suffix option was not valid. suffix= must be either suffix=true or suffix=false."})
		}

		pageInt, err := strconv.Atoi(page)
		if err != nil {
			errMessage := fmt.Sprintf("Error converting %s to an int. Please ensure the page= param is a valid int.", page)
			c.JSON(400, gin.H{"message": errMessage})
			return
		}

		limitInt, err := strconv.Atoi(limit)
		if err != nil {
			errMessage := fmt.Sprintf("Error converting %s to an int. Please ensure the limit= param is a valid int.", limit)
			c.JSON(400, gin.H{"message": errMessage})
			return
		}

		if limitInt > 2000 {
			c.JSON(400, gin.H{"message": "Provided limitInt was > 2000. Please provide a limit <= 2000"})
			return
		}

		packages, err := _api.InclusiveSearch(search, pageInt, limitInt, suffix)
		if err != nil {
			c.JSON(500, gin.H{"message": "internal error querying data."})
			return
		}

		c.JSON(200, gin.H{
			"packages": packages,
			"nextPage": pageInt + 1,
		})
	})

	r.GET("/search/by-owner", func(c *gin.Context) {
		owner := c.DefaultQuery("owner", "")
		page := c.DefaultQuery("page", "0")
		limit := c.DefaultQuery("limit", "100")

		pageInt, err := strconv.Atoi(page)
		if err != nil {
			errMessage := fmt.Sprintf("Error converting %s to an int. Please ensure the page= param is a valid int.", page)
			c.JSON(400, gin.H{"message": errMessage})
			return
		}

		limitInt, err := strconv.Atoi(limit)
		if err != nil {
			errMessage := fmt.Sprintf("Error converting %s to an int. Please ensure the limit= param is a valid int.", limit)
			c.JSON(400, gin.H{"message": errMessage})
			return
		}

		if owner == "" {
			c.JSON(400, gin.H{"message": "No owner= string provided. this field is required."})
			return
		}

		packages, err := _api.SearchByOwner(owner, pageInt, limitInt)
		if err != nil {
			c.JSON(500, gin.H{"message": "internal error querying data."})
			return
		}

		c.JSON(200, gin.H{
			"packages": packages,
			"nextPage": pageInt + 1,
		})
	})

	r.GET("/search/by-package", func(c *gin.Context) {
		pkg := c.DefaultQuery("package", "")

		if pkg == "" {
			c.JSON(400, gin.H{"message": "No package= string provided. This field is required."})
			return
		}

		p, err := _api.SearchByPackage(pkg)
		if err != nil {
			c.JSON(500, gin.H{"message": "internal error querying data."})
			return
		}

		c.JSON(200, gin.H{
			"url":      p.Owner,
			"versions": p.Versions,
		})
	})

	r.Run()
}

func (a *API) InclusiveSearch(search string, page int, limit int, suffix string) ([]string, error) {
	offset := page * limit
	query := `SELECT url FROM packages WHERE url LIKE $1 LIMIT $2 OFFSET $3`

	var s string
	if suffix == "true" {
		s = search + "%"
	} else {
		s = "%" + search + "%"
	}

	packages, err := a.Db.GQ.QueryString(query,
		s,
		limit,
		offset,
	)
	if err != nil {
		return []string{}, err
	}

	res := []string{}
	for _, r := range packages {
		res = append(res, r.([]interface{})[0].(string))
	}
	return res, nil
}

func (a *API) SearchByOwner(owner string, page int, limit int) ([]string, error) {
	query := `SELECT url FROM packages WHERE owner = $1 LIMIT $2 OFFSET $3`

	offset := page * limit
	packages, err := a.Db.GQ.QueryString(query,
		owner,
		limit,
		offset,
	)
	if err != nil {
		return []string{}, err
	}

	res := []string{}
	for _, r := range packages {
		res = append(res, r.([]interface{})[0].(string))
	}
	return res, nil
}

type Version struct {
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

type Package struct {
	Owner    string    `json:"owner"`
	Versions []Version `json:"versions"`
}

func (p *Package) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("Could not type assert query response to []byte")
	}

	return json.Unmarshal(b, &p)
}

func (a *API) SearchByPackage(pkg string) (*Package, error) {
	resp := &Package{}
	query := `SELECT JSONB_BUILD_OBJECT('owner', owner, 'versions', JSONB_AGG(JSONB_BUILD_OBJECT('version', version, 'timestamp', update_time))) FROM package_version WHERE owner = $1 GROUP BY owner`
	err := a.Db.Conn.QueryRow(query, pkg).Scan(&resp)
	if err != nil {
		return nil, err
	}
	fmt.Println(resp)
	return resp, nil
}
