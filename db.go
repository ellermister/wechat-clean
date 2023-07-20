package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"

	_ "github.com/mattn/go-sqlite3"
)

func ConnectDB(path string, key string) *sql.DB {
	key = url.QueryEscape(key)
	dbname := fmt.Sprintf("%s?_cipher=sqlcipher&_legacy=3&_hmac_use=off&_kdf_iter=4000&_legacy_page_size=1024&_key=%s", path, key)
	db, err := sql.Open("sqlite3", dbname)
	if err != nil {
		log.Fatalf("Open Error %v\n", err)
	}
	return db
}

func deleteRowsByIds(db *sql.DB, tableName string, columnName string, columnValue []int64) int64 {
	if len(columnValue) == 0 {
		return 0
	}
	sql := fmt.Sprintf("DELETE FROM %s WHERE %s in(%s)", tableName, columnName, implodeI2S(columnValue, ","))
	result, err := db.Exec(sql)
	if err != nil {
		log.Fatalf("deleteRowsByIds for table %s failed", tableName)
	}

	rows, err2 := result.RowsAffected()
	if err2 != nil {
		log.Printf("get affected rows error %v", err2)
	}

	return rows
}

func getTablesRowsTotal(db *sql.DB, showLimit int64) {
	records, err := db.Query("SELECT name  FROM sqlite_master WHERE type ='table'")

	var tables []string
	if err != nil {
		log.Printf("Err in get tables rows total query, %e", err)
	} else {
		for records.Next() {
			var name string
			records.Scan(&name)
			tables = append(tables, name)
		}
	}

	records.Close()

	for _, tblName := range tables {
		var rowsTotal int64
		err2 := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) as count FROM %s", tblName)).Scan(&rowsTotal)
		if err2 != nil {
			log.Printf("Err in get count table, %e", err2)
		} else if rowsTotal > showLimit {
			log.Printf("Table [%s]\tTotal:%d", tblName, rowsTotal)
		}

	}
}

func CleanTable(db *sql.DB, tableName string) int64 {
	result, err := db.Exec(fmt.Sprintf("DELETE FROM %s", tableName))

	if err != nil {
		log.Printf("Clean Table failed, err: %e", err)
		return 0
	}

	rows, _ := result.RowsAffected()
	return rows
}
