package main

import (
	"encoding/hex"
	. "github.com/pony-huang/dht-admin/db"
	"github.com/shiyanhui/dht"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	go func() {
		http.ListenAndServe(":6060", nil)
	}()

	w := dht.NewWire(65536, 1024, 256)

	// init db
	db, err := NewTorrentDB("./torrent.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	go func() {
		for resp := range w.Response() {
			metadata, err := dht.Decode(resp.MetadataInfo)
			if err != nil {
				continue
			}
			info := metadata.(map[string]interface{})

			if _, ok := info["name"]; !ok {
				continue
			}

			bt := BitTorrent{
				InfoHash: hex.EncodeToString(resp.InfoHash),
				Name:     info["name"].(string),
			}

			if v, ok := info["files"]; ok {
				files := v.([]interface{})
				bt.Files = make([]File, len(files))

				for i, item := range files {
					f := item.(map[string]interface{})
					bt.Files[i] = File{
						Path:   f["path"].([]interface{}),
						Length: f["length"].(int),
					}
				}
			} else if _, ok := info["length"]; ok {
				bt.Length = info["length"].(int)
			}

			err = db.InsertTorrent(bt)
			if err != nil {
				log.Fatal(err)
			}
		}
	}()
	go w.Run()

	config := dht.NewCrawlConfig()
	config.OnAnnouncePeer = func(infoHash, ip string, port int) {
		w.Request([]byte(infoHash), ip, port)
	}
	d := dht.New(config)

	d.Run()
}
