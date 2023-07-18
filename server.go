package main

import (
	"database/sql"
	"embed"
	_ "embed"
	"log"
	"net/http"
	"runtime"
)

//go:embed home.html
var f embed.FS

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if runtime.GOOS == "android" {
		data, _ := f.ReadFile("home.html")
		w.Write(data)
	} else {
		http.ServeFile(w, r, "home.html")
	}
}

func StartServer(db *sql.DB, serverPort string) {
	hub := newHub()
	go hub.run()
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	err := http.ListenAndServe(serverPort, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
