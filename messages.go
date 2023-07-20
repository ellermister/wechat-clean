package main

import (
	"bufio"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

const (
	FromTypeALL     = 0
	FromTypeFriends = 1
	FromTypeGroups  = 2

	TypeStickers = 47
	TypeImage    = 3
	TypeGraphic  = 49 // 群聊转发消息或者xml块、新闻图文链接，笔记、小程序之类的大块XML，特别占空间。删除前先把 imgpath 删除了。
	TypeVideo    = 43
	TypeVoice    = 34
)

type statMessage struct {
	records            int
	invalidFileRecords int
	filesSize          int64
	filesCount         int64
}

type statFileType struct {
	img   int64
	thumb int64
	voice int64
	video int64
	other int64
}

type scannedFile struct {
	foundFiles []string // abs path
	//localFiles []string // abs path, exist locally but not in the database, 暂时不做
	msgIds   []int64
	msgSvrId []int64
}

func ScanMessages(db *sql.DB, sqlText string) scannedFile {
	records, err := db.Query(sqlText)
	if err != nil {
		log.Fatalf("Scan chatroom error %e", err)
	}
	defer records.Close()

	var allStat statMessage
	var friendsStat statMessage
	var groupsStat statMessage
	var allStatFileType statFileType

	EchoDirStat()
	localFilenames := getDirFiles(WeshitUserPath + "/image2")
	var foundFilesName []string
	var localExtraFilesName []string

	if len(localFilenames) > 0 {
		log.Printf("Found a totoal %d file in local image2, example: %s\n", len(localFilenames), localFilenames[len(localFilenames)-1])
	}

	var scanResult scannedFile
	for records.Next() {
		var fileType int
		var msgId int64
		var msgSvrId sql.NullInt64
		var imgPath sql.NullString
		var talker string
		var bigImgPath sql.NullString
		var midImgPath sql.NullString
		var hevcPath sql.NullString
		var bigImgPath2 sql.NullString
		var midImgPath2 sql.NullString
		var hevcPath2 sql.NullString
		err := records.Scan(&msgId, &msgSvrId, &fileType, &imgPath, &talker, &bigImgPath, &midImgPath, &hevcPath, &bigImgPath2, &midImgPath2, &hevcPath2)
		if err != nil {
			log.Fatalf("Get records error %e", err)
		}

		if imgPath.Valid {
			var fileTypeSupport = IsSupportType(fileType)
			paths := BuildFilePath(fileType, imgPath.String)

			paths2 := BuildBigImgPath(bigImgPath)
			paths3 := BuildBigImgPath(midImgPath)
			paths4 := BuildBigImgPath(bigImgPath2)
			paths5 := BuildBigImgPath(midImgPath2)

			paths6 := BuildBigImgPath(hevcPath)
			paths7 := BuildBigImgPath(hevcPath2)

			paths = append(paths, paths2...)
			paths = append(paths, paths3...)
			paths = append(paths, paths4...)
			paths = append(paths, paths5...)
			paths = append(paths, paths6...)
			paths = append(paths, paths7...)

			paths = removeDuplicate(paths)

			var tmpFilesSize int64 = 0
			for _, absPath := range paths {
				tmpFilesSize += StatFileSize(absPath)

				foundFilesName = append(foundFilesName, filepath.Base(absPath))
				scanResult.foundFiles = append(scanResult.foundFiles, absPath)

				collectFileStat(&allStatFileType, fileType, absPath)
			}

			allStat.filesSize += tmpFilesSize
			allStat.filesCount += int64(len(paths))
			allStat.records++
			if fileTypeSupport && len(paths) == 0 {
				allStat.invalidFileRecords++
				//write.WriteString(fmt.Sprintf("%s,%s,%s,%s\n", msgId, fileType, imgPath, talker))
			}

			if strings.Contains(talker, "@chatroom") {
				groupsStat.records++
				groupsStat.filesCount += int64(len(paths))
				groupsStat.filesSize += tmpFilesSize
				if fileTypeSupport && len(paths) == 0 {
					groupsStat.invalidFileRecords++
				}
			} else {
				friendsStat.records++
				friendsStat.filesCount += int64(len(paths))
				friendsStat.filesSize += tmpFilesSize
				if fileTypeSupport && len(paths) == 0 {
					friendsStat.invalidFileRecords++
				}
			}
		}

		scanResult.msgIds = append(scanResult.msgIds, msgId)
		if msgSvrId.Valid {
			scanResult.msgSvrId = append(scanResult.msgSvrId, msgSvrId.Int64)
		}
	}

	log.Printf("Found a totoal %d file in database", len(foundFilesName))
	localExtraFilesName = difference(localFilenames, foundFilesName)
	log.Printf("Found a totoal %d file in other differences(exists locally but the database does not exist)", len(localExtraFilesName))
	OutputOtherImageList(localExtraFilesName)
	allStatFileType.other = int64(len(localExtraFilesName))

	log.Printf("Scan messages completed")
	log.Printf("[all]     records: %s, files size: %s, files count: %s, invalid file records: %s\n", humanize.Comma(int64(allStat.records)), humanize.Bytes(uint64(allStat.filesSize)), humanize.Comma(allStat.filesCount), humanize.Comma(int64(allStat.invalidFileRecords)))
	log.Printf("[friends] records: %s, files size: %s, files count: %s, invalid file records: %s\n", humanize.Comma(int64(friendsStat.records)), humanize.Bytes(uint64(friendsStat.filesSize)), humanize.Comma(friendsStat.filesCount), humanize.Comma(int64(friendsStat.invalidFileRecords)))
	log.Printf("[groups]  records: %s, files size: %s, files count: %s, invalid file records: %s\n", humanize.Comma(int64(groupsStat.records)), humanize.Bytes(uint64(groupsStat.filesSize)), humanize.Comma(groupsStat.filesCount), humanize.Comma(int64(groupsStat.invalidFileRecords)))

	log.Printf("File Type Statistics")
	log.Printf("img total: %d, thumbnail total: %d, voice total: %d, video total: %d, other total: %d", allStatFileType.img, allStatFileType.thumb, allStatFileType.voice, allStatFileType.video, allStatFileType.other)

	return scanResult
}

func collectFileStat(fileStat *statFileType, fileType int, path string) {
	name := filepath.Base(path)
	matchImg := regexp.MustCompile("\\.([a-z]+)")
	switch fileType {
	case TypeVoice:
		fileStat.voice++
	case TypeVideo:
		fileStat.video++
		if strings.Contains(name, "th_") {
			fileStat.thumb++
		}
		if !strings.Contains(name, ".") {
			fileStat.thumb++
		}

	case TypeGraphic:
		fallthrough
	case TypeImage:
		// **.jpg , ***.jpg.hevc, **.temp.jpg......
		if matchImg.MatchString(name) || strings.Contains(name, "hevc") {
			fileStat.img++
			break
		}
		fileStat.thumb++
	default:
	}
}

func BuildQuerySql(fromType int) string {
	var sqlText string

	var addWhere string
	if flags.onlyMedia {
		addWhere = " and m.imgPath !='' "
	}

	switch fromType {
	case FromTypeALL:
		// 为什么不用 img.msglocalid 关联, 因为实际数据中有一部分没有 msglocalid, msgSvrId 更全面
		sqlText = fmt.Sprintf(`SELECT m.msgId, m.msgSvrId, m.type, m.imgPath, m.talker, img.bigImgPath, img.midImgPath, img.hevcPath, img2.bigImgPath, img2.midImgPath, img2.hevcPath FROM 'message' as m 
			left join ImgInfo2 as img on img.msglocalid=m.msgId 
			left join ImgInfo2 as img2 on img2.msgSvrId=m.msgSvrId where 1=1 %s order by m.msgId desc`, addWhere)
		break
	case FromTypeFriends:
		sqlText = fmt.Sprintf(`SELECT m.msgId, m.msgSvrId, m.type, m.imgPath, m.talker, img.bigImgPath, img.midImgPath, img.hevcPath, img2.bigImgPath, img2.midImgPath, img2.hevcPath FROM 'message' as m 
			left join ImgInfo2 as img on img.msglocalid=m.msgId 
			left join ImgInfo2 as img2 on img2.msgSvrId=m.msgSvrId where  m.talker not like '%%@chatroom' %s   order by m.msgId desc`, addWhere)
		break
	case FromTypeGroups:
		sqlText = fmt.Sprintf(`SELECT m.msgId, m.msgSvrId, m.type, m.imgPath, m.talker, img.bigImgPath, img.midImgPath, img.hevcPath, img2.bigImgPath, img2.midImgPath, img2.hevcPath FROM 'message' as m 
			left join ImgInfo2 as img on img.msglocalid=m.msgId 
			left join ImgInfo2 as img2 on img2.msgSvrId=m.msgSvrId where  m.talker like '%%@chatroom' %s   order by m.msgId desc`, addWhere)
		break
	default:
		log.Fatalln("Invalid fromType")
	}
	return sqlText
}

func BuildQuerySqlByUserNames(usernames []string) string {
	var addWhere string
	if flags.onlyMedia {
		addWhere = " and m.imgPath !='' "
	}

	var formatUserName []string
	for _, value := range usernames {
		formatUserName = append(formatUserName, fmt.Sprintf("\"%s\"", value))
	}

	var usernamesJoined = strings.Join(formatUserName, ",")

	return fmt.Sprintf(`SELECT m.msgId, m.msgSvrId, m.type, m.imgPath, m.talker, img.bigImgPath, img.midImgPath, img.hevcPath, img2.bigImgPath, img2.midImgPath, img2.hevcPath FROM 'message' as m 
			left join ImgInfo2 as img on img.msglocalid=m.msgId 
			left join ImgInfo2 as img2 on img2.msgSvrId=m.msgSvrId where  m.talker in(%s) %s   order by m.msgId desc`, usernamesJoined, addWhere)
}

func EchoDirStat() {
	var statDir = []string{WeshitUserPath + "/image2", WeshitUserPathSdcard + "/voice2", WeshitUserPathSdcard + "/video"}
	for _, value := range statDir {
		tmp := getDirSize(value)
		log.Printf("The directory: %s, total: %s, size: %s", value, humanize.Comma(tmp.nFiles), humanize.Bytes(uint64(tmp.nBytes)))
	}
}

func OutputOtherImageList(otherFiles []string) {
	invalidFile, err := os.OpenFile("unknown.wechat.txt", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("Openfile error:%v \n", err)
	}
	defer invalidFile.Close()

	write := bufio.NewWriter(invalidFile)
	for _, value := range otherFiles {
		write.WriteString(value + "\n")
	}
	write.Flush()
}

// BuildFilePath
// 返回绝对路径
func BuildFilePath(fileType int, imgPath string) []string {
	switch fileType {
	case TypeStickers:
		// 不处理了
		break
	case TypeGraphic:
		fallthrough
	case TypeImage:
		imgId := MatchIdByThumbnailDirPath(imgPath)
		files := FindImagesByID(WeshitUserPath, imgId, true)
		files2 := FindImagesByID(WeshitUserPathSdcard, imgId, true)
		//log.Printf("in buildfilepath, =TypeImage, Id=%s, path=%s, count=%d + %d\n", imgId, imgPath, len(files2), len(files))
		return append(files, files2...)
	case TypeVideo:
		imgId := imgPath
		files := FindVideosByID(WeshitUserPath, imgId, true)
		files2 := FindVideosByID(WeshitUserPathSdcard, imgId, true)
		//log.Printf("in buildfilepath, =TypeVideo, Id=%s, path=%s, count=%d + %d\n", imgId, imgPath, len(files2), len(files))
		return append(files, files2...)
	case TypeVoice:
		md5I := md5.New()
		md5I.Write([]byte(imgPath))
		hexPath := hex.EncodeToString(md5I.Sum(nil))
		seg1 := hexPath[:2]
		seg2 := hexPath[2:4]
		amrPath := fmt.Sprintf("%s/voice2/%s/%s/msg_%s.amr", WeshitUserPathSdcard, seg1, seg2, imgPath)
		if Exists(amrPath) {
			return []string{amrPath}
		}
		break
	}
	return []string{}
}

func BuildBigImgPath(sBigImgPath sql.NullString) []string {
	if !sBigImgPath.Valid {
		return []string{}
	}
	var bigImgPath = sBigImgPath.String
	if len(bigImgPath) < 4 || strings.Contains(bigImgPath, "SERVERID://") {
		return []string{}
	}

	seg1 := bigImgPath[:2]
	seg2 := bigImgPath[2:4]

	var image2Dir = "/image2" + "/" + seg1 + "/" + seg2 + "/" + bigImgPath
	var path1 = WeshitUserPath + image2Dir
	var path2 = WeshitUserPathSdcard + image2Dir

	var result = make([]string, 0)
	if Exists(path1) {
		result = append(result, path1)
	}
	if Exists(path2) {
		result = append(result, path2)
	}

	return result
}

func IsSupportType(fileType int) bool {
	if fileType == TypeImage || fileType == TypeVideo || fileType == TypeGraphic {
		return true
	}
	return false
}

func MatchIdByThumbnailDirPath(imgPath string) string {
	reg1 := regexp.MustCompile("THUMBNAIL_DIRPATH://th_([a-z0-9]+)")

	result1 := reg1.FindStringSubmatch(imgPath)
	if len(result1) == 2 {
		return result1[1]
	} else {
		return ""
	}
}

func DetectSdcardDirPath(db *sql.DB) string {
	var userSdcardPath string

	mappingPath := WeshitUserPath + "/account.mapping"
	if Exists(mappingPath) {
		mappingHash, err := os.ReadFile(mappingPath)
		if err == nil && len(mappingHash) == 32 {
			userSdcardPath = UserSdcardPrefix + "/Android/data/com.tencent.mm/MicroMsg/" + string(mappingHash)
			log.Printf("Found user sdcard path via mappingHash: %s", userSdcardPath)
			if Exists(userSdcardPath) {
				return userSdcardPath
			}
		}
	}

	var orgPath string
	pathRegex := regexp.MustCompile("(/Android/data/com.tencent.mm/MicroMsg/([a-z0-9]+))/")

	db.QueryRow("SELECT orgpath FROM \"VideoHash\" WHERE orgpath like '%/MicroMsg/%' limit 1;").Scan(&orgPath)
	subMatches := pathRegex.FindStringSubmatch(orgPath)
	userSdcardPath = UserSdcardPrefix + subMatches[1]

	if Exists(userSdcardPath) {
		log.Printf("Found user sdcard path via VideoHash: %s", userSdcardPath)
		return userSdcardPath
	}

	db.QueryRow("SELECT path FROM \"MediaDuplication\"  WHERE path like '%/MicroMsg/%'  limit 1;").Scan(&orgPath)
	subMatches = pathRegex.FindStringSubmatch(orgPath)
	userSdcardPath = UserSdcardPrefix + subMatches[1]
	if Exists(userSdcardPath) {
		log.Printf("Found user sdcard path via MediaDuplication: %s", userSdcardPath)
		return userSdcardPath
	}

	return userSdcardPath
}

func getMessagesCount(db *sql.DB) int {
	var messageCount int
	err := db.QueryRow("SELECT COUNT(*) as count FROM message").Scan(&messageCount)
	if err != nil {
		log.Fatalf("getMessagesCount Error %v\n", err)
	}
	return messageCount
}

func deleteUserMessage(db *sql.DB, msgIds []int64) int64 {
	return deleteRowsByIds(db, "message", "msgId", msgIds)
}

func deleteVoiceInfo(db *sql.DB, msgIds []int64) int64 {
	return deleteRowsByIds(db, "voiceinfo", "MsgLocalId", msgIds)
}

func deleteVideoInfo2(db *sql.DB, msgIds []int64) int64 {
	return deleteRowsByIds(db, "videoinfo2", "msglocalid", msgIds)
}

func deleteImgInfo2(db *sql.DB, msgIds []int64) int64 {
	return deleteRowsByIds(db, "ImgInfo2", "msglocalid", msgIds)
}

func deleteImgInfo2ViaMsgSvrId(db *sql.DB, msgSvrId []int64) int64 {
	return deleteRowsByIds(db, "ImgInfo2", "msgSvrId", msgSvrId)
}

func deleteAppMessage(db *sql.DB, msgId []int64) int64 {
	return deleteRowsByIds(db, "AppMessage", "msgId", msgId)
}
func deleteMsgQuoteByMsgID(db *sql.DB, msgId []int64) int64 {
	return deleteRowsByIds(db, "MsgQuote", "msgId", msgId)
}

func deleteMsgQuoteBySvrId(db *sql.DB, msgSvrId []int64) int64 {
	return deleteRowsByIds(db, "MsgQuote", "msgSvrId", msgSvrId)
}

func deleteWxFileIndex(wxFileDb *sql.DB, msgId []int64) int64 {
	return deleteRowsByIds(wxFileDb, "WxFileIndex3", "msgId", msgId)
}

func cleanMediaDuplication(db *sql.DB) int64 {
	rows, err := db.Query("SELECT md5, path FROM MediaDuplication")
	if err != nil {
		log.Fatalf("Data failed: %v", err)
	}

	var count int64

	var md5Collects []string
	for rows.Next() {
		var md5Text string
		var path sql.NullString
		err = rows.Scan(&md5Text, &path)
		if err != nil {
			log.Fatalf("cleanMediaDuplication Get records error %v", err)
		}

		if !path.Valid || Exists(path.String) {
			md5Collects = append(md5Collects, md5Text)
		}
	}

	rows.Close()

	for _, md5Text := range md5Collects {
		result, err2 := db.Exec("DELETE FROM MediaDuplication WHERE `md5` = '%s'", md5Text)
		if err2 != nil {
			log.Fatalf("cleanMediaDuplication Delete record error: %v, md5Text = %s", err2, md5Text)
		}

		affectedRows, err3 := result.RowsAffected()
		if err3 != nil {
			log.Fatalf("cleanMediaDuplication Delete record get rows affected err: %v", err2)
		}
		count += affectedRows
	}

	return count
}

func cleanChatRoomNoticeAttachIndex(db *sql.DB) int64 {
	rows, err := db.Query("SELECT msgId,dataPath, thumbPath FROM ChatroomNoticeAttachIndex")
	if err != nil {
		log.Fatalf("Data failed: %v", err)
	}

	defer rows.Close()

	var count int64
	var msgIds []int64
	for rows.Next() {
		var msgId int64
		var dataPath string
		var thumbPath string
		err = rows.Scan(&msgId, &dataPath, &thumbPath)
		if err != nil {
			log.Fatalf("cleanChatRoomNoticeAttachIndex Get records error %v", err)
		}

		if !Exists(dataPath) && !Exists(thumbPath) {
			msgIds = append(msgIds, msgId)

		}
	}

	for _, msgId := range msgIds {
		result, err2 := db.Exec("DELETE FROM ChatroomNoticeAttachIndex WHERE msgId = '%d'", msgId)
		if err2 != nil {
			log.Fatalf("cleanChatRoomNoticeAttachIndex Delete record error: %v", err2)
		}

		affectedRows, err3 := result.RowsAffected()
		if err3 != nil {
			log.Fatalf("cleanChatRoomNoticeAttachIndex Delete record get rows affected err: %v", err2)
		}
		count += affectedRows
	}

	return count
}

func cleanOtherDirectoryFiles() {
	// MicroMsgPath 辣鸡
	removeSubDirAndFiles(MicroMsgPath + "/webservice")
	removeSubDirAndFiles(MicroMsgPath + "/webcompt")
	removeSubDirAndFiles(MicroMsgPath + "/CheckResUpdate")
	removeSubDirAndFiles(MicroMsgPath + "/luckymoneynewyear")
	removeSubDirAndFiles(MicroMsgPath + "/wepkg")

	// 用户资料目录辣鸡
	removeSubDirAndFiles(WeshitUserPath + "/corrupted")
	removeSubDirAndFiles(WeshitUserPath + "/record")
	removeSubDirAndFiles(WeshitUserPath + "/avatar")
	removeSubDirAndFiles(WeshitUserPath + "/game")
	removeSubDirAndFiles(WeshitUserPath + "/appbrand")

	// 藏在图片下的笔记图片
	removeSubDirAndFiles(WeshitUserPath + "/image2/No")
	// 藏起来的大图
	removeSubDirAndFiles(WeshitUserPath + "/image2/.ref")

	removeFileByGlob(WeshitUserPath + "/FTS5IndexMicroMsg*")

	//sdcard
	removeSubDirAndFiles(UserSdcardPrefix + "/Android/data/com.tencent.mm/cache")

	// 微信webview
	removeFileByGlob(WeshitPath + "/app_xwalk*")
	removeFileByGlob(WeshitPath + "/app_webview*")
}

func VacuumDb(db *sql.DB) {
	runtime.LockOSThread()
	log.Printf("Start slimming down the database, please be patient.")
	db.SetConnMaxIdleTime(3600 * time.Second)
	db.SetConnMaxLifetime(3600 * time.Second)
	// EnMicroMsg.db-wal 最好提前删除
	_, err := db.Exec(`VACUUM "main"`) // unknown database message

	if err != nil {
		log.Printf("Thin database failed! %v", err)
	}
	runtime.UnlockOSThread()
}

func CleanWeshitUserFiles(db *sql.DB, wxFileDb *sql.DB, scanResult *scannedFile) {
	var err error
	// 删除结果集匹配的本地文件
	var deletedFilesTotal int64 = 0
	var notExistFileTotal int64 = 0
	var failedDeleteFileTotal int64 = 0
	var mainDirSizeBeforeDelete DirStatSize
	var mainDirSizeAfterDeleted DirStatSize

	var sdcardDirSizeBeforeDelete DirStatSize
	var sdcardDirSizeAfterDeleted DirStatSize

	mainDirSizeBeforeDelete = getDirSize(WeshitUserPath)
	sdcardDirSizeBeforeDelete = getDirSize(WeshitUserPathSdcard)

	log.Printf("Main dir, Before total files %s, total size: %s",
		humanize.Comma(mainDirSizeBeforeDelete.nFiles),
		humanize.Bytes(uint64(mainDirSizeBeforeDelete.nBytes)),
	)
	log.Printf("Sdcard dir, Before total files %s, total size: %s",
		humanize.Comma(sdcardDirSizeBeforeDelete.nFiles),
		humanize.Bytes(uint64(sdcardDirSizeBeforeDelete.nBytes)),
	)

	log.Printf("Start deleting scanned files")
	for _, filePath := range scanResult.foundFiles {
		if !Exists(filePath) {
			notExistFileTotal++
			continue
		}
		err = os.Remove(filePath)
		if err != nil {
			log.Printf("Failed to delete file: %s", filePath)
			failedDeleteFileTotal++
		} else {
			deletedFilesTotal++
		}
	}
	log.Printf("Delete scanned files completed")

	// 这里不能直接删除 localFiles, 因为数据是根据  (本地所有文件) - (fromType 来源的结果集) 得到的
	// 【仅在 fromType = all 时结果集才有效,其他时刻数据均不正确】
	// 而且目前 localFiles 只有文件名 无法删除, 暂时不做处理

	// 根据 msgId 删除数据
	var deletedTotal int64 = 0
	log.Printf("Start deleting database records")

	chunkIdsData := chunksIds(scanResult.msgIds, 500)
	for index, itemIds := range chunkIdsData {
		fmt.Printf("Deleting database record, msgIds len = %d, progress = %d/%d \r", len(itemIds), index, len(chunkIdsData))

		deletedTotal += deleteUserMessage(db, itemIds)

		deletedTotal += deleteVoiceInfo(db, itemIds)

		deletedTotal += deleteVideoInfo2(db, itemIds)

		deletedTotal += deleteImgInfo2(db, itemIds)

		deletedTotal += deleteAppMessage(db, itemIds)

		deletedTotal += deleteMsgQuoteByMsgID(db, itemIds)

		deletedTotal += deleteWxFileIndex(wxFileDb, itemIds)
	}

	chunkMsgSvrIdsData := chunksIds(scanResult.msgSvrId, 500)
	for index, itemMsgSvrIds := range chunkMsgSvrIdsData {
		fmt.Printf("Deleting database record, msgSvrId len: %d, progress = %d/%d\r", len(itemMsgSvrIds), index, len(chunkMsgSvrIdsData))
		deletedTotal += deleteImgInfo2ViaMsgSvrId(db, itemMsgSvrIds)
		deletedTotal += deleteMsgQuoteBySvrId(db, itemMsgSvrIds)
	}

	log.Printf("Delete the MediaDuplication table of main database")
	deletedTotal += cleanMediaDuplication(db)

	log.Printf("Delete the ChatRoomNoticeAttachIndex table of main database")
	deletedTotal += cleanChatRoomNoticeAttachIndex(db)

	rows := CleanTable(db, "CleanDeleteItem")
	log.Printf("Deleted table %s %d records from main db", "CleanDeleteItem", rows)
	deletedTotal += rows

	rows = CleanTable(db, "img_flag")
	log.Printf("Deleted table %s %d records from main db", "img_flag", rows)
	deletedTotal += rows

	rows = CleanTable(db, "BizTimeLineInfo")
	log.Printf("Deleted table %s %d records from main db", "BizTimeLineInfo", rows)
	deletedTotal += rows

	rows = CleanTable(db, "appattach")
	log.Printf("Deleted table %s %d records from main db", "appattach", rows)
	deletedTotal += rows

	// rconversation 不要清理，是最近会话列表
	//rows = CleanTable(db, "rconversation")
	//log.Printf("Deleted table %s %d records from main db", "rconversation", rows)
	//deletedTotal += rows

	// github.com/jgiannuzzi/go-sqlite3
	// 使用了 jgiannuzzi 的包之后似乎arm64上执行大体积DB也不会内存溢出了，这里删除掉1GB文件判断。
	VacuumDb(db)
	log.Printf("Streamlined database %s completed", "EnMicroMsg.db")

	VacuumDb(wxFileDb)
	log.Printf("Streamlined database %s completed", "WxFileIndex.db")

	cleanOtherDirectoryFiles()

	log.Printf("Delete database records completed")

	log.Printf("Expected to delete %d files, actually deleted %d files, not exist %d files, failed delete %d files", len(scanResult.foundFiles), deletedFilesTotal, notExistFileTotal, failedDeleteFileTotal)
	log.Printf("A total of %d database records were cleaned up", deletedTotal)

	mainDirSizeAfterDeleted = getDirSize(WeshitUserPath)
	sdcardDirSizeAfterDeleted = getDirSize(WeshitUserPathSdcard)
	log.Printf("The data before and after the deletion of all directory is: ")
	log.Printf("Main dir, Before total files %s, total size: %s",
		humanize.Comma(mainDirSizeBeforeDelete.nFiles),
		humanize.Bytes(uint64(mainDirSizeBeforeDelete.nBytes)),
	)
	log.Printf("Main dir, After total files %s, total size: %s, Cleaned up a total files %s, total size: %s",
		humanize.Comma(mainDirSizeAfterDeleted.nFiles),
		humanize.Bytes(uint64(mainDirSizeAfterDeleted.nBytes)),
		humanize.Comma(mainDirSizeBeforeDelete.nFiles-mainDirSizeAfterDeleted.nFiles),
		humanize.Bytes(uint64(mainDirSizeBeforeDelete.nBytes-mainDirSizeAfterDeleted.nBytes)),
	)
	log.Printf("Sdcard dir, Before total files %s, total size: %s",
		humanize.Comma(sdcardDirSizeBeforeDelete.nFiles),
		humanize.Bytes(uint64(sdcardDirSizeBeforeDelete.nBytes)),
	)
	log.Printf("Sdcard dir, After total files %s, total size: %s, Cleaned up a total files %s, total size: %s",
		humanize.Comma(sdcardDirSizeAfterDeleted.nFiles),
		humanize.Bytes(uint64(sdcardDirSizeAfterDeleted.nBytes)),
		humanize.Comma(sdcardDirSizeBeforeDelete.nFiles-sdcardDirSizeAfterDeleted.nFiles),
		humanize.Bytes(uint64(sdcardDirSizeBeforeDelete.nBytes-sdcardDirSizeAfterDeleted.nBytes)),
	)

}

func CommandVacuum(dbpath string, key string) {
	db := ConnectDB(dbpath, key)
	defer db.Close()

	var result int64
	db.QueryRow("SELECT COUNT(*) FROM message").Scan(&result)
	log.Printf("The message table have rows %d", result)

	db.QueryRow("SELECT COUNT(*) FROM message WHERE  talker like '%@chatroom' ").Scan(&result)
	log.Printf("The message table have rows %d for groups", result)

	//db.Exec("DELETE FROM message where talker like '%@chatroom'")

	//db.QueryRow("SELECT COUNT(*) FROM message WHERE  talker like '%@chatroom' ").Scan(&result)
	//log.Printf("The message table have rows %d for groups", result)
	VacuumDb(db)

	log.Printf("Thin database complete!")
}
