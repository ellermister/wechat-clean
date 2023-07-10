package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"

	sqlite3 "github.com/mutecomm/go-sqlcipher"
)

func CheckDB(path string) bool {
	isEncrypted, err := sqlite3.IsEncrypted(path)
	if err != nil {
		log.Fatalf("Check err=%v\n", err)
	}

	log.Printf("[+] The file encrypted is %v \n", isEncrypted)

	return isEncrypted
}

func ConnectDB(path string, key string) *sql.DB {
	key = url.QueryEscape(key)
	dbname := fmt.Sprintf("%s?_pragma_key=%s&_pragma_cipher_page_size=1024&_pragma_cipher_use_hmac=OFF&_pragma_kdf_iter=4000", path, key)
	db, err := sql.Open("sqlite3", dbname)
	if err != nil {
		log.Fatalf("Open Error %v\n", err)
	}

	_, err = db.Exec(fmt.Sprintf("PRAGMA key = '%s';", key))
	if err != nil {
		log.Fatalf("Set PRAGMA key Error %v\n", err)
	}

	_, err = db.Exec(fmt.Sprintf("PRAGMA cipher_use_hmac = OFF;"))
	if err != nil {
		log.Fatalf("Set PRAGMA cipher_use_hmac Error %v\n", err)
	}
	_, err = db.Exec(fmt.Sprintf("PRAGMA cipher_page_size = 1024;"))
	if err != nil {
		log.Fatalf("Set PRAGMA cipher_page_size Error %v\n", err)
	}
	_, err = db.Exec(fmt.Sprintf("PRAGMA kdf_iter = 4000;"))
	if err != nil {
		log.Fatalf("Set PRAGMA kdf_iter Error %v\n", err)
	}
	var result string
	//db.Exec(fmt.Sprintf("PRAGMA auto_vacuum = 2"))
	err = db.QueryRow(fmt.Sprintf("PRAGMA auto_vacuum")).Scan(&result)
	if err != nil {
		log.Fatalf("Set PRAGMA auto_vacuum Error %v\n", err)
	}
	log.Printf("ddd auto_vacuum = %v", result)

	return db
}
