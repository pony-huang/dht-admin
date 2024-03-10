package test

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"testing"
)

func TestSample(t *testing.T) {
	os.Remove("./foo.db")

	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sqlStmt := `
	create table foo (id integer not null primary key, name text);
	delete from foo;
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("insert into foo(id, name) values(?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	for i := 0; i < 100; i++ {
		_, err = stmt.Exec(i, fmt.Sprintf("こんにちは世界%03d", i))
		if err != nil {
			log.Fatal(err)
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query("select id, name from foo")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(id, name)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err = db.Prepare("select name from foo where id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var name string
	err = stmt.QueryRow("3").Scan(&name)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(name)

	_, err = db.Exec("delete from foo")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("insert into foo(id, name) values(1, 'foo'), (2, 'bar'), (3, 'baz')")
	if err != nil {
		log.Fatal(err)
	}

	rows, err = db.Query("select id, name from foo")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(id, name)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
}

type Tag struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}

func (t *Tag) Scan(value interface{}) error {
	return json.Unmarshal([]byte(value.(string)), t)
}

func (t *Tag) Value() (driver.Value, error) {
	b, err := json.Marshal(t)
	return string(b), err
}

func TestJson(t *testing.T) {
	os.Remove("./foo.db")

	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`create table foo (tag jsonb)`)
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := db.Prepare("insert into foo(tag) values(?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(`{"name": "mattn", "country": "japan"}`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(`{"name": "michael", "country": "usa"}`)
	if err != nil {
		log.Fatal(err)
	}

	var country string
	err = db.QueryRow("select tag->>'country' from foo where tag->>'name' = 'mattn'").Scan(&country)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(country)

	var tag Tag
	err = db.QueryRow("select tag from foo where tag->>'name' = 'mattn'").Scan(&tag)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(tag.Name)

	tag.Country = "日本"
	_, err = db.Exec(`update foo set tag = ? where tag->>'name' == 'mattn'`, &tag)
	if err != nil {
		log.Fatal(err)
	}

	err = db.QueryRow("select tag->>'country' from foo where tag->>'name' = 'mattn'").Scan(&country)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(country)
}

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

func (f *File) Scan(value interface{}) error {
	return json.Unmarshal([]byte(value.(string)), f)
}

func (f *File) Value() (driver.Value, error) {
	b, err := json.Marshal(f)
	return string(b), err
}

func (b *BitTorrent) Scan(value interface{}) error {
	return json.Unmarshal([]byte(value.(string)), b)
}

func (b *BitTorrent) Value() (driver.Value, error) {
	b.Files = nil // Ensure that the 'Files' field is omitted when serializing
	return json.Marshal(b)
}

func TestBitTorrent(t *testing.T) {
	os.Remove("./torrent.db")

	db, err := sql.Open("sqlite3", "./torrent.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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

	//_, err = db.Exec(`create table torrents (torrent jsonb)`)
	//if err != nil {
	//	log.Fatal(err)
	//}

	torrent := BitTorrent{
		InfoHash: "0123456789abcdef",
		Name:     "example.torrent",
		Files: []File{
			{Path: []interface{}{"folder1", "file1.txt"}, Length: 1024},
			{Path: []interface{}{"folder2", "file2.txt"}, Length: 2048},
		},
	}

	stmt, err := db.Prepare("insert into torrents(torrent) values(?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	torrentJSON, err := json.Marshal(torrent)
	if err != nil {
		log.Fatal(err)
	}

	_, err = stmt.Exec(string(torrentJSON))
	if err != nil {
		log.Fatal(err)
	}

	var storedTorrent BitTorrent
	err = db.QueryRow("select torrent from torrents").Scan(&storedTorrent)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Stored Torrent:", storedTorrent)
}
