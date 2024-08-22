package main

import (
	"archive/zip"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"raidhub/shared/bungie"
	"raidhub/shared/postgres"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

var (
	out      = flag.String("dir", "./", "where to store the sqlite")
	force    = flag.Bool("f", false, "force the defs to be updated")
	verbose  = flag.Bool("verbose", false, "log more")
	fromDisk = flag.Bool("disk", false, "read from disk, not bnet")
)

func main() {
	flag.Parse()
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	var sqlitePath string
	if !*fromDisk {
		manifest, err := bungie.GetDestinyManifest()
		if err != nil {
			log.Fatal("get manifest: ", err)
		}

		dbURL := fmt.Sprintf("https://www.bungie.net%s", manifest.MobileWorldContentPaths["en"])
		dbFileName := filepath.Join(*out, filepath.Base(dbURL))
		sqlitePath = dbFileName + ".sqlite3" // name for the cached file

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
		resp, err := http.Get(fmt.Sprintf("%s?c=%d", dbURL, rand.Int()))
		if resp.StatusCode != 200 {
			log.Fatal(fmt.Errorf("invalid status code: %d", resp.StatusCode))
		}
		if err != nil {
			log.Fatal(err)
		}
		if *verbose {
			log.Println("Downloaded files")
		}
		defer resp.Body.Close()

		// Create the file to save the ZIP
		zipFile, err := os.Create(zipFileName)
		if err != nil {
			log.Fatal(err)
		}
		if *verbose {
			log.Println("Created zip file")
		}
		defer func() {
			zipFile.Close()
			err = os.Remove(zipFileName)
			if err != nil {
				log.Fatal(err)
			}
			if *verbose {
				log.Println("Removed zip file")
			}
		}()

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
		if *verbose {
			log.Println("Opened zip file")
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
		if *verbose {
			log.Println("Extracted zip file")
		}

		log.Println("Downloaded sqlite3 successfully")

		// Rename the SQLite database file to have a recognizable extension
		err = os.Rename(dbFileName, sqlitePath)
		if err != nil {
			log.Fatal(err)
		}
		if *verbose {
			log.Println("Remame sqlite3 file")
		}
	} else {
		var newestModTime time.Time

		err := filepath.Walk(*out, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil // skip directories
			}

			if info.ModTime().After(newestModTime) {
				newestModTime = info.ModTime()
				sqlitePath = path
			}

			return nil
		})

		if err != nil {
			log.Fatal(err)
		}

		if sqlitePath == "" {
			log.Fatalf("directory %s is empty", *out)
		}
	}

	definitions, err := sql.Open("sqlite3", sqlitePath)
	if err != nil {
		log.Fatal(err)
	}
	if *verbose {
		log.Println("Connected to sqlite3")
	}
	defer definitions.Close()

	db, err := postgres.Connect()
	if err != nil {
		log.Fatal(err)
	}
	if *verbose {
		log.Println("Connected to postgres")
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rows, err := definitions.QueryContext(ctx, `SELECT 
			json_extract(json, '$.hash'), 
			json_extract(json, '$.displayProperties.name'), 
			json_extract(json, '$.displayProperties.icon'), 
			json_extract(json, '$.defaultDamageType'), 
			json_extract(json, '$.equippingBlock.ammoType'), 
			json_extract(json, '$.equippingBlock.equipmentSlotTypeHash'), 
			json_extract(json, '$.inventory.tierTypeName') 
		FROM DestinyInventoryItemDefinition 
		WHERE json_extract(json, '$.itemType') = 3`)
	if err != nil {
		log.Fatal(err)
	}
	if *verbose {
		log.Println("Scanning definitions")
	}
	defer rows.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO weapon_definition 
		(hash, name, icon_path, element, ammo_type, slot, rarity) 
		VALUES ($1::bigint, $2, $3, get_element($4), get_ammo_type($5), get_slot($6), $7)`)
	if err != nil {
		log.Fatal(err)
	}
	if *verbose {
		log.Println("Prepared postgres statement")
	}
	defer stmt.Close()

	log.Println("Statement prepared")

	_, err = tx.ExecContext(ctx, "TRUNCATE TABLE weapon_definition")
	if err != nil {
		log.Fatal(err)
	}
	if *verbose {
		log.Println("Truncated weapons table")
	}

	// Iterate over the rows and process the data
	for rows.Next() {
		var hash uint32
		var name string
		var icon string
		var element uint8
		var ammoType uint8
		var slot uint32
		var rarity string
		if err := rows.Scan(&hash, &name, &icon, &element, &ammoType, &slot, &rarity); err != nil {
			log.Fatal(err)
		}

		_, err := stmt.ExecContext(ctx, hash, name, icon, element, ammoType, slot, rarity)
		if err != nil {
			log.Fatal(err)
		}

		if *verbose {
			log.Printf("Inserted %d: %s", hash, name)
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Done")
}
