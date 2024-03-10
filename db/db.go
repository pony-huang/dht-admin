package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

type File struct {
	Path   []interface{} `json:"path"`
	Length int           `json:"length"`
}

type BitTorrent struct {
	InfoHash string `json:"infohash"`
	Name     string `json:"name"`
	Files    []File `json:"files,omitempty"`
	Length   int    `json:"length,omitempty"`
}

// TorrentDB represents a SQLite database for storing BitTorrent data.
type TorrentDB struct {
	db *sql.DB
}

// NewTorrentDB creates a new instance of TorrentDB with the given SQLite database file.
func NewTorrentDB(dbFile string) (*TorrentDB, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}
	// Check if table exists
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='torrents'")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	// If the table doesn't exist, create it
	if !rows.Next() {
		_, err = db.Exec(`CREATE TABLE torrents (torrent jsonb)`)
		if err != nil {
			log.Fatal(err)
		}
	}
	return &TorrentDB{db: db}, nil
}

// Close closes the database connection.
func (t *TorrentDB) Close() error {
	return t.db.Close()
}

// QueryByName searches for a torrent with the specified name in the database and returns it.
func (t *TorrentDB) QueryByName(name string) (BitTorrent, error) {
	var torrent BitTorrent
	row := t.db.QueryRow("SELECT torrent FROM torrents WHERE torrent->>'name' = ?", name)
	err := row.Scan(&torrent)
	if err != nil {
		return torrent, err
	}
	return torrent, nil
}

// InsertTorrent inserts a new torrent into the database.
func (t *TorrentDB) InsertTorrent(torrent BitTorrent) error {
	torrentJSON, err := json.Marshal(torrent)
	if err != nil {
		return err
	}
	log.Printf("%s\n\n", torrentJSON)
	_, err = t.db.Exec("INSERT INTO torrents(torrent) VALUES (?)", string(torrentJSON))
	if err != nil {
		return err
	}
	return nil
}

// ExampleUsage demonstrates how to use the TorrentDB.
func ExampleUsage() {
	db, err := NewTorrentDB("./torrent.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	torrent := BitTorrent{
		InfoHash: "0123456789abcdef",
		Name:     "example.torrent",
		Files: []File{
			{Path: []interface{}{"folder1", "file1.txt"}, Length: 1024},
			{Path: []interface{}{"folder2", "file2.txt"}, Length: 2048},
		},
	}

	err = db.InsertTorrent(torrent)
	if err != nil {
		log.Fatal(err)
	}

	result, err := db.QueryByName("example.torrent")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Found Torrent:", result)
}
