package main

import (
	"database/sql"
	"log"
)

type GroupStruct struct {
	Username     string  `json:"username"`
	Nickname     *string `json:"nickname"`
	ConRemark    *string `json:"con_remark"`
	Avatar       *string `json:"avatar"`
	ContactType  int64   `json:"type"`
	MessageCount int64   `json:"msg_count"`
}

func GetGroups(db *sql.DB) []GroupStruct {
	var sql = "SELECT r.username, r.nickname,i.reserved2 FROM \"rcontact\"  as r left join img_flag as i on i.username = r.username where r.username like '%@chatroom';"
	records, err := db.Query(sql)
	defer records.Close()
	if err != nil {
		log.Printf("Get Groups Failed, err:%e\n", err)
		return []GroupStruct{}
	}

	var groups []GroupStruct

	for records.Next() {
		var group GroupStruct
		err2 := records.Scan(&group.Username, &group.Nickname, &group.Avatar)
		if err2 != nil {
			log.Printf("Each GroupsStruct err: %e", err2)
		} else {
			groups = append(groups, group)
		}
	}
	return groups
}

func GetChatroom(db *sql.DB) []GroupStruct {
	var sql = `SELECT r.username,r.type, r.nickname, r.conRemark,i.reserved2,count(m.talker) as mcount FROM "rcontact"  as r left join img_flag as i on i.username = r.username left join message as m on m.talker = r.username where r.type in (2,3) group by r.username order by mcount desc;`
	records, err := db.Query(sql)
	defer records.Close()
	if err != nil {
		log.Printf("Get Groups Failed, err:%e\n", err)
		return []GroupStruct{}
	}

	var groups []GroupStruct

	for records.Next() {
		var group GroupStruct
		err2 := records.Scan(&group.Username, &group.ContactType, &group.Nickname, &group.ConRemark, &group.Avatar, &group.MessageCount)
		if err2 != nil {
			log.Printf("Each GroupsStruct err: %e", err2)
		} else {
			groups = append(groups, group)
		}
	}
	return groups
}

func SubmitCleanTask(db *sql.DB, wxFileDb *sql.DB, usernames []string) {
	sqlText := BuildQuerySqlByUserNames(usernames)
	log.Printf("sql: %v", sqlText)
	var scanResult = ScanMessages(db, sqlText)
	log.Printf("scanResult, msgIds count:%d, svrId: %d, file:%d", len(scanResult.msgIds), len(scanResult.msgSvrId), len(scanResult.foundFiles))
	CleanWeshitUserFiles(db, wxFileDb, &scanResult)
}
