package main

import (
	"archive/zip"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"raidhub/shared/bungie"
	"raidhub/shared/postgres"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

var (
	out   = flag.String("dir", "./", "where to store the sqlite")
	force = flag.Bool("f", false, "force the defs to be updated")
)

func main() {
	flag.Parse()
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	manifest, err := bungie.GetDestinyManifest()
	if err != nil {
		log.Fatal("get manifest: ", err)
	}

	dbURL := fmt.Sprintf("https://www.bungie.net%s", manifest.MobileWorldContentPaths["en"])
	dbFileName := filepath.Join(*out, filepath.Base(dbURL))
	sqlitePath := dbFileName + ".sqlite3" // name for the cached file

	if _, err := os.Stat(sqlitePath); os.IsNotExist(err) {
		log.Printf("Loading new manifest definitions: %s", manifest.Version)
	} else if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("No new manifest definitions")
		if !*force {
			return
		}
	}

	// Download the ZIP file
	zipFileName := dbFileName + ".zip"
	resp, err := http.Get(dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Create the file to save the ZIP
	zipFile, err := os.Create(zipFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer zipFile.Close()

	// Write the downloaded content to the ZIP file
	_, err = io.Copy(zipFile, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Extract the ZIP file
	zipReader, err := zip.OpenReader(zipFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer zipReader.Close()

	// Extract each file from the ZIP archive
	for _, file := range zipReader.File {
		filePath := filepath.Join(*out, file.Name)
		if file.FileInfo().IsDir() {
			// Create directories
			err = os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}
			continue
		}

		// Create the file
		extractedFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			log.Fatal(err)
		}
		defer extractedFile.Close()

		// Extract the file
		zipFile, err := file.Open()
		if err != nil {
			log.Fatal(err)
		}
		defer zipFile.Close()

		_, err = io.Copy(extractedFile, zipFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("Downloaded sqlite3 successfully")

	// Rename the SQLite database file to have a recognizable extension
	err = os.Rename(dbFileName, sqlitePath)
	if err != nil {
		log.Fatal(err)
	}

	err = os.Remove(zipFileName)
	if err != nil {
		log.Fatal(err)
	}

	definitions, err := sql.Open("sqlite3", sqlitePath)
	if err != nil {
		log.Fatal(err)
	}
	defer definitions.Close()

	db, err := postgres.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := definitions.Query("SELECT json_extract(json, '$.hash'), json_extract(json, '$.displayProperties.name'), json_extract(json, '$.displayProperties.icon') FROM DestinyInventoryItemDefinition WHERE json_extract(json, '$.itemType') = 3")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO weapon_definition (hash, name, icon_path) VALUES ($1::bigint, $2, $3)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	log.Println("Statement prepared")

	_, err = tx.Exec("DELETE FROM weapon_definition")
	if err != nil {
		log.Fatal(err)
	}

	// Iterate over the rows and process the data
	for rows.Next() {
		var hash uint32
		var name string
		var icon string
		if err := rows.Scan(&hash, &name, &icon); err != nil {
			log.Fatal(err)
		}

		_, err := stmt.Exec(hash, name, icon)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Inserted %d: %s", hash, name)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Done")
}
