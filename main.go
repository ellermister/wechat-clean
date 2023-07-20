package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
)

var WeshitPath = "/data/user/[UID]/com.tencent.mm"
var MicroMsgPath = "/data/user/[UID]/com.tencent.mm/MicroMsg"

var UserSdcardPrefix = "/storage/emulated/[UID]"

var WeshitUserPath = "/data/user/[UID]/com.tencent.mm/MicroMsg/[32ID]"
var WeshitUserPathSdcard = UserSdcardPrefix + "/Android/data/com.tencent.mm/MicroMsg/[32ID]"

var PathEnMicroMsgDB = "EnMicroMsg.db"
var PathWxFileIndexDB = "WxFileIndex.db"

const version = "v20230720"

type FlagArgs struct {
	aUserId     int
	user32HexId string
	KeyDb       string
	commandType string
	fromType    string
	vacuumDB    string
	onlyMedia   bool // scan media records of db
}

var flags FlagArgs

type HubConfig struct {
	enMicroMsgConn *sql.DB // EnMicroMsg connect
	wxFileConn     *sql.DB // WxFileIndex connect
}

var appHub HubConfig

func main() {
	fmt.Printf("\n\nWechat-Clean %s\n\n", version)
	flag.IntVar(&flags.aUserId, "user", 0, "Android user id")
	flag.StringVar(&flags.KeyDb, "key", "", "db key")
	flag.StringVar(&flags.user32HexId, "id", "", "user 32 length hex id")
	flag.StringVar(&flags.commandType, "cmd", "scan", "scan/clean/server")
	flag.StringVar(&flags.fromType, "from", "", "groups/friends/all")
	flag.StringVar(&flags.vacuumDB, "vd", "", "Vacuum db full path")
	flag.BoolVar(&flags.onlyMedia, "only-media", false, "Scan media messages only")
	flag.Parse()

	if flags.vacuumDB != "" {
		if Exists(flags.vacuumDB + "-wal") {
			os.Remove(flags.vacuumDB + "-wal")
		}
		CommandVacuum(flags.vacuumDB, flags.KeyDb)
		return
	}
	if len(flags.user32HexId) != 32 {
		log.Fatalf("Invalid user 32 length hex id of input")
	}

	if flags.commandType != "server" {
		if flags.fromType != "groups" && flags.fromType != "friends" && flags.fromType != "all" {
			log.Fatalf("Invalid from type of input")
		}

		if flags.fromType == "all" && flags.commandType == "clean" {
			log.Fatalf("防止误操作, 不支持删除全部数据, 删除全部数据建议通过直接删除目录或者应用的形式进行！")
		}
	}

	WeshitPath = fmt.Sprintf("/data/user/%d/com.tencent.mm", flags.aUserId)
	MicroMsgPath = fmt.Sprintf("/data/user/%d/com.tencent.mm/MicroMsg", flags.aUserId)
	UserSdcardPrefix = fmt.Sprintf("/storage/emulated/%d", flags.aUserId)

	WeshitUserPath = fmt.Sprintf("/data/user/%d/com.tencent.mm/MicroMsg/%s", flags.aUserId, flags.user32HexId)

	PathEnMicroMsgDB = fmt.Sprintf("%s/EnMicroMsg.db", WeshitUserPath)
	PathWxFileIndexDB = fmt.Sprintf("%s/WxFileIndex.db", WeshitUserPath)

	log.Printf("Loaded database EnMicroMsg = %s", PathEnMicroMsgDB)
	log.Printf("Loaded database WxFileIndex = %s", PathWxFileIndexDB)

	if os.Getenv("DEBUG") != "" {
		PathEnMicroMsgDB = "EnMicroMsg.db"
		PathWxFileIndexDB = "WxFileIndex.db"
	}

	// 在连接前清理工作区
	if flags.commandType == "clean" || flags.commandType == "server" {
		// 提前删除预写缓存文件
		// 这一步存疑， 有时候会毁坏数据库文件（实际上没毁坏只是微信读到是损坏的，建议备份损坏前后的文件，把损坏DB里你需要的数据写到没损坏的db里，附加数据库之后 insert from select table就行）
		// 如果不清理的话可能也会有问题，正常情况下SQLITE协议关闭连接时会自动删除这些文件的。微信大多数情况处于意外关闭的。
		// 先暂时这样，最好在执行前手动结束微信进程，让系统给与微信进程信号能够安全退出，然后自己再执行 pm disable com.tencent.mm 来冻结微信确保不会自启
		if Exists(PathEnMicroMsgDB + "-wal") {
			os.Remove(PathEnMicroMsgDB + "-wal")
		}
		if Exists(PathWxFileIndexDB + "-wal") {
			os.Remove(PathWxFileIndexDB + "-wal")
		}

		DisableAPP("com.tencent.mm")
		defer EnableApp("com.tencent.mm")
	}

	appHub.enMicroMsgConn = ConnectDB(PathEnMicroMsgDB, flags.KeyDb)
	defer appHub.enMicroMsgConn.Close()

	appHub.wxFileConn = ConnectDB(PathWxFileIndexDB, flags.KeyDb)
	defer appHub.wxFileConn.Close()

	WeshitUserPathSdcard = DetectSdcardDirPath(appHub.enMicroMsgConn)

	if WeshitUserPathSdcard == "" {
		log.Fatalf("Not found User SD card directory of wechat")
	}
	log.Printf("found User SD card directory of wechat: %s", WeshitUserPathSdcard)

	if flags.onlyMedia {
		log.Printf("The current program only scans media records of database!")
	}

	SetupCloseHandler()

	switch flags.commandType {
	case "scan":
		fallthrough
	case "clean":
		ExecuteScanClean()
	case "server":
		ExecuteServer()
	}

	log.Printf("Done")
}

func ExecuteServer() {
	if !Exists(PathEnMicroMsgDB) {
		PathEnMicroMsgDB = "EnMicroMsg.db" // test
	}
	StartServer(appHub.enMicroMsgConn, ":9999")
}

func ExecuteScanClean() {
	if flags.commandType == "scan" {
		log.Printf(">> This operation is a test, is safed! << ")
	}
	log.Printf(">> The scope of this scan is %s << ", flags.fromType)

	getTablesRowsTotal(appHub.enMicroMsgConn, 10000)

	var scanResult scannedFile
	if flags.fromType == "all" {
		sqlText := BuildQuerySql(FromTypeALL)
		scanResult = ScanMessages(appHub.enMicroMsgConn, sqlText)
	} else if flags.fromType == "friends" {
		sqlText := BuildQuerySql(FromTypeFriends)
		scanResult = ScanMessages(appHub.enMicroMsgConn, sqlText)
	} else if flags.fromType == "groups" {
		sqlText := BuildQuerySql(FromTypeGroups)
		scanResult = ScanMessages(appHub.enMicroMsgConn, sqlText)
	}

	if flags.commandType == "clean" {
		CleanWeshitUserFiles(appHub.enMicroMsgConn, appHub.wxFileConn, &scanResult)
	}
}

func SetupCloseHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		appHub.enMicroMsgConn.Close()
		appHub.wxFileConn.Close()
		EnableApp("com.tencent.mm")
		os.Exit(0)
	}()
}
