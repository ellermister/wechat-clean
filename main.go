package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/mutecomm/go-sqlcipher"
)

const WeshitPath = "/data/user/[UID]/com.tencent.mm"

var UserSdcardPrefix = "/storage/emulated/[UID]"

var WeshitUserPath = "/data/user/[UID]/com.tencent.mm/MicroMsg/[32ID]"
var WeshitUserPathSdcard = UserSdcardPrefix + "/Android/data/com.tencent.mm/MicroMsg/[32ID]"

var PathEnMicroMsgDB = "EnMicroMsg.db"
var PathWxFileIndexDB = "WxFileIndex.db"

const version = "v20230716"

func main() {
	fmt.Printf("\n\nWechat-Clean %s\n\n", version)

	var aUserId int
	var user32HexId string
	var KeyDb string
	var commandType string
	var fromType string
	var VacuumDb string
	flag.IntVar(&aUserId, "user", 0, "Android user id")
	flag.StringVar(&KeyDb, "key", "", "db key")
	flag.StringVar(&user32HexId, "id", "", "user 32 length hex id")
	flag.StringVar(&commandType, "cmd", "scan", "scan/clean")
	flag.StringVar(&fromType, "from", "", "groups/friends/all")
	flag.StringVar(&VacuumDb, "vd", "", "Vacuum db full path")
	flag.Parse()

	if VacuumDb != "" {
		if Exists(VacuumDb + "-wal") {
			os.Remove(VacuumDb + "-wal")
		}
		CommandVacuum(VacuumDb, KeyDb)
		return
	}

	if fromType != "groups" && fromType != "friends" && fromType != "all" {
		log.Fatalf("Invalid from type of input")
	}

	if len(user32HexId) != 32 {
		log.Fatalf("Invalid user 32 length hex id of input")
	}

	if fromType == "all" && commandType == "clean" {
		log.Fatalf("防止误操作, 不支持删除全部数据, 删除全部数据建议通过直接删除目录或者应用的形式进行！")
	}

	UserSdcardPrefix = fmt.Sprintf("/storage/emulated/%d", aUserId)

	WeshitUserPath = fmt.Sprintf("/data/user/%d/com.tencent.mm/MicroMsg/%s", aUserId, user32HexId)

	PathEnMicroMsgDB = fmt.Sprintf("%s/EnMicroMsg.db", WeshitUserPath)
	PathWxFileIndexDB = fmt.Sprintf("%s/WxFileIndex.db", WeshitUserPath)

	log.Printf("Loaded database EnMicroMsg = %s", PathEnMicroMsgDB)
	log.Printf("Loaded database WxFileIndex = %s", PathWxFileIndexDB)

	if CheckDB(PathEnMicroMsgDB) == false {
		log.Fatal("This database file is not encrypted, will be skipped.")
	}

	db := ConnectDB(PathEnMicroMsgDB, KeyDb)
	defer db.Close()

	wxFileDb := ConnectDB(PathWxFileIndexDB, KeyDb)
	defer wxFileDb.Close()

	WeshitUserPathSdcard = DetectSdcardDirPath(db)

	if WeshitUserPathSdcard == "" {
		log.Fatalf("Not found User SD card directory of wechat")
	}
	log.Printf("found User SD card directory of wechat: %s", WeshitUserPathSdcard)

	if commandType == "scan" {
		log.Printf(">> This operation is a test, is safed! << ")
	} else {
		// 提前删除预写缓存文件
		if Exists(PathEnMicroMsgDB + "-wal") {
			os.Remove(PathEnMicroMsgDB + "-wal")
		}
		if Exists(PathWxFileIndexDB + "-wal") {
			os.Remove(PathWxFileIndexDB + "-wal")
		}
	}
	log.Printf(">> The scope of this scan is %s << ", fromType)

	messageCount := getMessagesCount(db)
	log.Printf("Get %d message records\n", messageCount)

	var scanResult scannedFile
	if fromType == "all" {
		scanResult = ScanMessages(db, FromTypeALL)
	} else if fromType == "friends" {
		scanResult = ScanMessages(db, FromTypeFriends)
	} else if fromType == "groups" {
		scanResult = ScanMessages(db, FromTypeGroups)
	}

	if commandType == "clean" {
		CleanWeshitUserFiles(db, wxFileDb, &scanResult)
	}

}
