package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"time"

	toml "github.com/pelletier/go-toml"
	"blog-log-agent/client"
)

type Event struct {
	Time int64
	Msg  string
}

func postEvents(client *http.Client, url string, events []Event) error {
	data, err := json.Marshal(events)
	if err != nil {
		panic(err)
	}
	fmt.Println("Sending events", string(data))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Unable to POST: %s - %v", url, err)
	}
	defer res.Body.Close()
	if res.StatusCode != 201 && res.StatusCode != 200 {
		msg, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("Unable to update: %s - HTTP_%d: %s", url, res.StatusCode, string(msg))
	}
	return nil
}

func main() {
	interval := flag.Int("interval", 10, "Time in seconds to sleep between loops")
	eventsDir := flag.String("eventsdir", "/var/run/events", "Directory to find events in")
	sotaToml := flag.String("sotatoml", "/var/sota/sota.toml", "aktualizr-lite sota.toml")
	url := flag.String("eventsurl", "", "Events url like https://foo.com/events")

	flag.Parse()
	duration := time.Duration(*interval) * time.Second
	if len(*url) == 0 {
		fmt.Println("ERROR - eventsurl is required")
		os.Exit(1)
	}

	err := os.Chdir(*eventsDir)
	if err != nil {
		fmt.Println("ERROR - unable to chdir into", *eventsDir, ":", err)
		os.Exit(1)
	}

	sota, err := toml.LoadFile(*sotaToml)
	if err != nil {
		fmt.Println("ERROR - unable to decode sota.toml:", err)
		os.Exit(1)
	}
	client := client.New(sota)

	for {
		var events []Event
		time.Sleep(duration)
		files, err := ioutil.ReadDir("./")
		if err != nil {
			fmt.Println("Unable to find events:", err)
		}
		for _, file := range files {
			if !file.IsDir() {
				bytes, err := ioutil.ReadFile(file.Name())
				if err != nil {
					fmt.Println("Unable to read", file.Name(), ":", err)
				} else {
					events = append(events, Event{file.ModTime().Unix(), string(bytes)})
				}
			}
		}
		if len(events) > 0 {
			fmt.Println("Found event(s)")
			sort.Slice(events, func(i, j int) bool {
				return events[i].Time < events[j].Time
			})
			if err := postEvents(client, *url, events); err != nil {
				fmt.Println(err)
			} else {
				for _, file := range files {
					if err := os.Remove(file.Name()); err != nil {
						fmt.Println("ERROR - deleting", file.Name(), ":", err)
					}
				}
			}
			fmt.Println("Event(s) uploaded")
		}
	}
}
