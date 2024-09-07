package idx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

func SaveIndexToDB() {
	fmt.Printf("\r Opening dbs...")
	db, err := leveldb.OpenFile(".go-index/search", nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db2, err := leveldb.OpenFile(".go-index/store", nil)
	if err != nil {
		panic(err)
	}
	defer db2.Close()

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
				<-sem
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				// TODO: Something awesome here
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
		fmt.Printf("\r Processing entries: %d left                                                                                      ", len(entries))
		sem <- 1
		wg.Add(1)
		go func() {
			ie := &IdxEntry{}
			json.Unmarshal([]byte(entry), ie)

			_, err := db.Get([]byte(ie.Path), nil)
			if errors.Is(err, leveldb.ErrNotFound) {
				db.Put([]byte(ie.Path), []byte(ie.Path), nil)
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

	fmt.Println("Done")

	// Update writetime
	err = db.Put([]byte("writetime"), []byte(startTime.Format(time.RFC3339Nano)), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println("Updated last write time")
	wg.Wait()
	close(sem)
}
