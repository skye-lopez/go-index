package idx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

type IdxEntry struct {
	Path      string `json:"Path"`
	Version   string `json:"Version"`
	Timestamp string `json:"Timestamp"`
}

// TODO: Error handling
func Fetch() {
	// Load log db
	db, err := leveldb.OpenFile(".go-index-log", nil)
	if err != nil {
		log.Fatalf("Error opening .go-index-log : \n %e", err)
	}
	defer db.Close()

	// Attain last write time or set it to default
	var lastTime string
	lastStoredTime, err := db.Get([]byte("write-time"), nil)
	if errors.Is(err, leveldb.ErrNotFound) {
		lt, err := time.Parse(time.RFC3339Nano, "2019-04-10T19:08:52.997264Z")
		if err != nil {
			panic(err)
		}
		lastTime = lt.Format(time.RFC3339Nano)
	} else if err == nil {
		lastTime = string(lastStoredTime)
	} else {
		log.Fatalf("Error preforming get write-time on .go-index-log : \n %e", err)
	}

	// Get all of the URLS back to lastTime to be queried and indexed.
	baseUrl := "https://index.golang.org/index?since="
	startTime := time.Now()
	endTime, err := time.Parse(time.RFC3339Nano, lastTime)
	step := time.Duration(12) * time.Hour

	urls := []string{}
	for startTime.Unix() > endTime.Unix() {
		urls = append(urls, baseUrl+startTime.Format(time.RFC3339Nano))
		startTime = startTime.Add(-step)
	}

	var wg sync.WaitGroup
	maxWorkers := 20
	sem := make(chan int, maxWorkers)

	maxEntries := len(urls) * 2000
	entries := make(chan string, maxEntries)

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
				entries <- ie.Path + "|" + ie.Version + "|" + ie.Timestamp
			}
			<-sem
		}()
	}

	wg.Wait()
	close(entries)
	close(sem)
	fmt.Println("test")

	var wg2 sync.WaitGroup
	sem2 := make(chan int, maxWorkers)

	cache := make(map[string]bool, 0)
	storeDB, err := leveldb.OpenFile(".go-index-store", nil)
	if err != nil {
		log.Fatalf("Error opening .go-index-store \n %e", err)
	}
	defer storeDB.Close()

	searchDB, err := leveldb.OpenFile(".go-index-search", nil)
	if err != nil {
		log.Fatalf("Error opening .go-index-search \n %e", err)
	}
	defer searchDB.Close()

	versionDB, err := leveldb.OpenFile(".go-index-version", nil)
	if err != nil {
		log.Fatalf("Error opening .go-index-version \n %e", err)
	}
	defer versionDB.Close()

	for e := range entries {
		fmt.Printf("\r Storing entry to DB, %d left", len(entries))
		splitE := strings.Split(e, "|")
		path := splitE[0]
		version := splitE[1]
		timestamp := splitE[2]

		sem2 <- 1
		wg2.Add(1)
		go func() {
			defer wg2.Done()

			// We update the version regardless of the cache state to ensure no misses.
			existing, err := versionDB.Get([]byte(path), nil)
			if errors.Is(err, leveldb.ErrNotFound) {
				err = versionDB.Put([]byte(path), []byte(version+"|"+timestamp), nil)
				if err != nil {
					panic(err)
				}
			} else if err == nil {
				versionTouples := strings.Split(string(existing), "~")
				updateMe := true
				for _, vt := range versionTouples {
					splitVT := strings.Split(vt, "|")
					if splitVT[0] == version {
						updateMe = false
						break
					}
				}

				if updateMe {
					packet := []byte(string(existing) + "~" + version + "|" + timestamp)
					err = versionDB.Put([]byte(path), packet, nil)
					if err != nil {
						panic(err)
					}
				}
			} else {
				panic(err)
			}

			// Skip entries we have already checked.
			// this is much faster than checking leveldb every time.
			_, cached := cache[path]
			if cached {
				fmt.Println("Cached already")
				<-sem2
				return
			}

			// Update searchDB
			prefix := ""
			for _, c := range path {
				prefix += string(c)
				existing, err := searchDB.Get([]byte(prefix), nil)
				if errors.Is(err, leveldb.ErrNotFound) {
					_ = searchDB.Put([]byte(prefix), []byte(path), nil)
				} else {
					packet := []byte(string(existing) + "~" + path)
					_ = searchDB.Put([]byte(prefix), packet, nil)
				}
			}
			// Update storeDB
			_, err = storeDB.Get([]byte(path), nil)
			if errors.Is(err, leveldb.ErrNotFound) {
				err = storeDB.Put([]byte(path), []byte(path), nil)
				if err != nil {
					panic(err)
				}
			}

			<-sem2
		}()
	}

	wg2.Wait()
	close(sem2)

	newWriteTime := time.Now().Format(time.RFC3339Nano)
	err = db.Put([]byte("write-time"), []byte(newWriteTime), nil)
	if err != nil {
		panic(err)
	}
}
