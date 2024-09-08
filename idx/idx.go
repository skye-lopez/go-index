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

// TODO: Make this non-sync again.
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
		// lastTime = "2019-04-10T19:08:52.997264Z"
		tempStep := time.Duration(120) * time.Hour
		tempLT := time.Now().Add(-tempStep)
		lastTime = tempLT.Format(time.RFC3339Nano)

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
	maxWorkers := 10
	sem := make(chan int, maxWorkers)

	maxEntries := len(urls) * 2000
	entries := make(chan string, maxEntries)

	fmt.Println("Starting url iter")
	for i, url := range urls {
		fmt.Printf("\r Parsing url: %d/%d", i, len(url))
		sem <- 1
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(url)
			if err != nil {
				<-sem
				panic(err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				<-sem
				panic(err)
			}

			lines := strings.Split(string(body), "\n")
			for _, e := range lines {
				ie := &IdxEntry{}
				json.Unmarshal([]byte(e), ie)
				entries <- ie.Path
			}
			<-sem
		}()
	}

	wg.Wait()
	close(sem)
	close(entries)

	var wg2 sync.WaitGroup
	sem2 := make(chan int, maxWorkers)

	cache := make(map[string]bool, 0)
	searchDB, err := leveldb.OpenFile(".go-index-search", nil)
	if err != nil {
		log.Fatalf("Error opening .go-index-search \n %e", err)
	}
	defer searchDB.Close()

	for e := range entries {
		fmt.Printf("\r Storing entry to DB, %d left", len(entries))
		sem2 <- 1
		wg2.Add(1)
		go func() {
			defer wg2.Done()

			// Skip entries we have already checked.
			// this is much faster than checking leveldb every time.
			_, cached := cache[e]
			if cached {
				<-sem
				return
			}

			// Update searchDB
			prefix := ""
			for _, c := range e {
				prefix += string(c)
				existing, err := searchDB.Get([]byte(prefix), nil)
				if errors.Is(err, leveldb.ErrNotFound) {
					_ = searchDB.Put([]byte(prefix), []byte(e), nil)
				} else {
					packet := []byte(string(existing) + "~" + e)
					_ = searchDB.Put([]byte(prefix), packet, nil)
				}
			}
			<-sem2
		}()
	}

	wg2.Wait()
	close(sem2)
}

func SaveIndexToDB() {
	fmt.Printf("\r Opening dbs...")
	db, err := leveldb.OpenFile(".go-index-search", nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db2, err := leveldb.OpenFile(".go-index-store", nil)
	if err != nil {
		panic(err)
	}
	defer db2.Close()

	db3, err := leveldb.OpenFile(".go-index-version", nil)
	if err != nil {
		panic(err)
	}
	defer db3.Close()

	// Check last write time
	var lastTime string
	val, err := db2.Get([]byte("writetime"), nil)
	if errors.Is(err, leveldb.ErrNotFound) {
		lastTime = "2019-04-10T19:08:52.997264Z"
	} else {
		lastTime = string(val)
	}

	baseUrl := "https://index.golang.org/index?since="
	startTime := time.Now()
	endTime, err := time.Parse(time.RFC3339Nano, lastTime)

	step := time.Duration(12) * time.Hour

	fmt.Printf("\r Getting urls...")
	urls := []string{}
	for startTime.Unix() > endTime.Unix() {
		urls = append(urls, baseUrl+startTime.Format(time.RFC3339Nano))
		startTime = startTime.Add(-step)
	}

	var wg sync.WaitGroup

	maxWorkers := 10
	sem := make(chan int, maxWorkers)

	maxEntries := len(urls) * 2000
	entries := make(chan string, maxEntries)

	for i, url := range urls {
		fmt.Printf("\r Getting entries: %d/%d                                                                                      ", i, len(urls))
		sem <- 1

		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(time.Millisecond * 25)
			resp, err := http.Get(url)
			if err != nil {
				// TODO: Something awesome here
				panic(err)
				<-sem
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				// TODO: Something awesome here
				panic(err)
				<-sem
				return
			}

			splitBody := strings.Split(string(body), "\n")
			for _, e := range splitBody {
				if len(e) < 5 {
					continue
				}
				entries <- e
			}

			<-sem
		}()
	}

	wg.Wait()
	close(sem)
	close(entries)

	sem = make(chan int, maxWorkers)
	for entry := range entries {
		fmt.Printf("\r Processing entries  %d Leftttt                                                                                     ", len(entries))
		sem <- 1
		wg.Add(1)
		go func() {
			ie := &IdxEntry{}
			json.Unmarshal([]byte(entry), ie)

			// Add each index to the global store
			_, err := db.Get([]byte(ie.Path), nil)
			if errors.Is(err, leveldb.ErrNotFound) {
				db.Put([]byte(ie.Path), []byte(ie.Path), nil)
			}

			// Add each indexs version to its version path
			existingVersions, err := db3.Get([]byte(ie.Path), nil)
			if errors.Is(err, leveldb.ErrNotFound) {
				db.Put([]byte(ie.Path), []byte(ie.Timestamp+"|"+ie.Version), nil)
			} else {
				versionTouples := strings.Split(string(existingVersions), "~")

				track := make(map[string]bool, 0)
				versionResult := ""
				for _, v := range versionTouples {
					splitTouple := strings.Split(v, "|")
					_, exists := track[splitTouple[1]]
					if !exists {
						track[splitTouple[1]] = true
						if len(versionResult) < 2 {
							versionResult += v
						} else {
							versionResult += "~" + v
						}
					}
				}
				fmt.Printf("\n\r test: %s", versionResult)
				db.Put([]byte(ie.Path), []byte(versionResult+"~"+ie.Timestamp+"|"+ie.Version), nil)
			}

			prefix := ""
			for _, char := range ie.Path {
				s := string(char)
				prefix += s
				prefixB := []byte(prefix)

				existing, err := db2.Get(prefixB, nil)
				if errors.Is(err, leveldb.ErrNotFound) {
					db.Put(prefixB, []byte(ie.Path), nil)
				} else {
					db.Put(prefixB, []byte(string(existing)+"~"+ie.Path), nil)
				}
			}
			<-sem
		}()
	}

	// Update writetime
	err = db.Put([]byte("writetime"), []byte(startTime.Format(time.RFC3339Nano)), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println("Updated last write time")
	wg.Wait()
	close(sem)
}
