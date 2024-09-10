package idx

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/skye-lopez/go-index/pg"
)

type IdxEntry struct {
	Path      string `json:"Path"`
	Version   string `json:"Version"`
	Timestamp string `json:"Timestamp"`
}

func FetchAndUpdate() {
	db, err := pg.NewPg()
	if err != nil {
		log.Fatalf("Error opening DB instance\n%e", err)
	}
	defer db.Conn.Close()

	r, err := db.GQ.QueryString("SELECT last_write FROM log WHERE id = $1", "log")
	if err != nil {
		log.Fatalf("Could not retrieve last_write from DB\n%e", err)
	}
	lastWriteStamp := r[0].([]interface{})[0].(string)

	endTime, err := time.Parse(time.RFC3339Nano, lastWriteStamp)
	if err != nil {
		log.Fatalf("Error parsing lastWriteStamp into RFC3339Nano\n%e", err)
	}

	startTime := time.Now()
	step := time.Duration(12) * time.Hour

	baseUrl := "https://index.golang.org/index?since="
	urls := []string{}
	for startTime.Unix() > endTime.Unix() {
		urls = append(urls, baseUrl+startTime.Format(time.RFC3339Nano))
		startTime = startTime.Add(-step)
	}

	var wg sync.WaitGroup
	maxWorkers := 20
	sem := make(chan int, maxWorkers)

	maxEntries := len(urls) * 2000
	entries := make(chan *IdxEntry, maxEntries)

	for i, url := range urls {
		fmt.Printf("\r Parsing url: %d/%d", i, len(urls))
		sem <- 1
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(url)
			if err != nil {
				<-sem
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				<-sem
				return
			}

			lines := strings.Split(string(body), "\n")
			for _, e := range lines {
				ie := &IdxEntry{}
				json.Unmarshal([]byte(e), ie)
				if len(ie.Path) < 5 {
					<-sem
					return
				}
				entries <- ie
			}
			<-sem
		}()
	}

	wg.Wait()
	close(entries)
	close(sem)

	var wg2 sync.WaitGroup
	sem2 := make(chan int, maxWorkers)

	localCache := make(map[string]bool, 0)

	for e := range entries {
		fmt.Printf("\r Storing entry to DB, %d left", len(entries))
		sem2 <- 1
		wg2.Add(1)
		go func() {
			defer wg2.Done()

			_, alreadyDone := localCache[e.Path]
			if alreadyDone {
				return
			}

			parsedUrl := ParseUrlInfo(e.Path)
			var owner any
			if len(parsedUrl.Owner) > 0 {
				owner = parsedUrl.Owner
			} else {
				owner = nil
			}

			// Update main package list
			db.Conn.Exec("INSERT INTO packages (url, host, path, owner) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
				e.Path,
				parsedUrl.Host,
				parsedUrl.Path,
				owner,
			)

			// Update that packages version
			r, err := db.GQ.QueryString("SELECT EXISTS(SELECT version FROM package_version WHERE owner = $1 and version = $2)",
				e.Path,
				e.Version,
			)
			if err != nil {
				log.Fatalf("Issue retrieving stored version for package\n%e", err)
			}

			versionExists := r[0].([]interface{})[0].(bool)

			if !versionExists {
				db.Conn.Exec("INSERT INTO package_version (owner, version, update_time) VALUES ($1, $2, $3)",
					e.Path,
					e.Version,
					e.Timestamp,
				)
			}
			<-sem2
		}()
	}

	wg2.Wait()
	close(sem2)

	logTime := time.Now().Format(time.RFC3339Nano)
	db.Conn.Exec("UPDATE log set last_write = $1 WHERE id = $2",
		logTime,
		"log",
	)
}

type UrlInfo struct {
	Host  string
	Path  string
	Owner string
}

func ParseUrlInfo(url string) *UrlInfo {
	urli := &UrlInfo{}

	// The host should always be the prepend of the 1st /
	split := strings.Split(url, "/")
	urli.Host = split[0]

	// The path is always anything to the right of the host
	urli.Path = strings.Join(split[1:], "/")

	// Owners are for github/gitlab (maybe more?)
	// TODO: There is probably a better way to do this
	if strings.Contains(url, "github.com/") || strings.Contains(url, "gitlab.com/") {
		urli.Owner = split[1]
	}

	return urli
}
