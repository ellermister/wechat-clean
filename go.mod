module ellermister/wechat-clean

go 1.18

require (
	github.com/dustin/go-humanize v1.0.1
	github.com/gorilla/websocket v1.5.0
	github.com/mattn/go-sqlite3 v1.14.17
//github.com/mutecomm/go-sqlcipher v0.0.0-20190227152316-55dbde17881f
)

replace github.com/mattn/go-sqlite3 => github.com/jgiannuzzi/go-sqlite3 v1.14.17-0.20230719111531-6e53453ccbd3
