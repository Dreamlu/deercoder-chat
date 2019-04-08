package chat

import (
	"errors"
	"fmt"
	"github.com/dreamlu/deercoder-gin"
	"github.com/dreamlu/deercoder-gin/util/lib"
	"github.com/dreamlu/go.uuid"
	"strings"
)

// Define our message object,teacher message model
type Message struct {
	UUID        string             `json:"uuid"`         //群组消息id
	GroupId     string             `json:"group_id"`     //组id
	FromUid     int64              `json:"from_uid"`     //来自用户id
	Headimg     string             `json:"headimg"`      //头像
	Name        string             `json:"username"`     //用户名
	Content     string             `json:"content"`      //消息内容
	ContentType string             `json:"content_type"` //前台用
	CreateTime  deercoder.JsonTime `json:"create_time"`  //创建时间
}

/*群聊发送模型*/
type GroupMsg struct {
	ID         int64              `json:"id"`
	GroupID    string             `json:"group_id"`    //群聊id
	Content    int64              `json:"content"`     //消息内容
	FromUid    int64              `json:"from_uid"`    //由谁发送
	CreateTime deercoder.JsonTime `json:"create_time"` //创建时间
}

/*群聊最后记录*/
type GroupLastMsg struct {
	ID             int64  `json:"id"`
	GroupID        string `json:"group_id"` //群聊id
	Uid            int64  `json:"uid"`
	LastGroupMsgId int64  `json:"last_group_msg_id"`
}

/*群组id极其成员id*/
type GroupUsers struct {
	ID      int64  `json:"id"`
	GroupId string `json:"group_id"`
	Uid     int64  `json:"uid"`
}

////群组,删除
//func DeleteGroup(){
//
//}

//建立群组,未来扩展
//返回群组id
func DistributeGroup(uids string) (groupId string, err error) {

	if uids == "" {
		return "", nil
	}

	userids := strings.Split(uids, ",")
	//唯一群id
	groupId = uuid.NewV1().String()
	sql := "insert `group_users`(group_id,uid) value"
	for _, v := range userids {
		if v == "" {
			continue
		}
		sql += "('" + groupId + "'," + v + "),"
	}
	sql = string([]byte(sql)[:len(sql)-1])

	dba := deercoder.DB.Exec(sql)
	num := dba.RowsAffected

	if num == 0 {
		return "", dba.Error
	}

	return groupId, nil
}

//群聊消息,创建
func CreateGroupMsg(uuid, group_id string, from_uid int64, content, content_type string) (err error) {

	//需要id,用来每次聊天生成的id作为聊天记录id,以便群离线消息记录该id
	sql := "insert `group_msg`(uuid,group_id,content,from_uid, content_type) value(?,?,?,?,?)"
	dba := deercoder.DB.Exec(sql, uuid, group_id, content, from_uid, content_type)

	if dba.Error != nil {
		return dba.Error
	}
	return nil
}

//群离线消息记录
//记录用户离线时,最后显示的消息id
func CreateGroupLastMsg(group_id string, uid int64, last_group_msg_uuid string) (err error) {
	if group_id == "" {
		return errors.New("用户gid不存在")
	}
	sql := "insert `group_last_msg`(group_id, uid, last_group_msg_uuid) value(?,?,?)"
	dba := deercoder.DB.Exec(sql, group_id, uid, last_group_msg_uuid)

	if dba.Error != nil {
		return dba.Error
	}
	return nil
}

//拉取群聊消息(所有)
func GetAllGroupMsg(group_id int64) ([]Message, error) {

	//拉取该群聊的所有消息
	sql := `select *
	from group_msg
	where group_id=?`

	var msg []Message

	deercoder.DB.Raw(sql, group_id).Scan(&msg)

	if len(msg) == 0 {

		return msg, errors.New("暂无离线消息")
	}
	sql = "select name,headimg from `user` where id = ?"
	for k, v := range msg { //查询对应的头像,用户名等信息

		deercoder.DB.Raw(sql, v.FromUid).Scan(&msg[k])
	}

	return msg, nil
}

//拉取用户离线消息
func GetGroupLastMsg(group_id, uid int64) ([]Message, error) {

	//1.找出群聊group_id中对应的最小的未读记录id
	var value deercoder.Value
	sql2 := "select min(last_group_msg_uuid) as value from group_last_msg where is_read=0 and group_id=? and uid=?"
	deercoder.DB.Raw(sql2, group_id, uid).Scan(&value)

	if value.Value == "" {
		return nil, errors.New("暂无离线消息")
	}
	//2.拉取离线后的该群聊的所有消息
	sql := `select *
	from group_msg
	where group_id=? and id >= (select id from group_msg where uuid = ?)`

	var msg []Message

	deercoder.DB.Raw(sql, group_id, value.Value).Scan(&msg)

	if len(msg) == 0 {

		return msg, errors.New("暂无离线消息")
	}
	sql = "select name,headimg from `user` where id = ?"
	for k, v := range msg { //查询对应的头像,用户名等信息

		deercoder.DB.Raw(sql, v.FromUid).Scan(&msg[k])
	}

	return msg, nil
}

//已读消息
func ReadGroupLastMsg(group_id, uid int64) interface{} {

	var info interface{}
	sql2 := "update `group_last_msg` set is_read=1 where is_read=0 and group_id=? and uid=?"
	dba := deercoder.DB.Exec(sql2, group_id, uid)
	num := dba.RowsAffected
	if dba.Error != nil {
		info = lib.GetSqlError(dba.Error.Error())
	} else if num == 0 && dba.Error == nil {
		info = lib.MapExistOrNo
	} else {
		info = lib.MapUpdate
	}
	return info
}

// 群发消息,对方默认未读
// flag 0老师, 1学生
// group_ids,逗号分割,群聊id
// send_uids老师或学生id,和群聊id一一对应
func MassMessage(group_ids, send_uids, from_uid, content string) interface{} {

	if group_ids == "" {
		return lib.GetMapData(lib.CodeChat, "group_ids不能为空")
	}
	sql := "insert `group_msg`(uuid, group_id, content, from_uid) value"
	sql2 := "insert `group_last_msg`(group_id, uid, last_group_msg_uuid) value"
	uuidS := uuid.NewV1().String()
	gids := strings.Split(group_ids, ",")
	uids := strings.Split(send_uids, ",")
	for k, v := range gids {
		sql += fmt.Sprintf("(%s,'%s','%s','%s'),", uuidS, v, content, from_uid) //这里肯定是老师群发,flag直接为０
		sql2 += fmt.Sprintf("('%s','%s','%s'),", v, uids[k], uuidS)
		uuidS = uuid.NewV1().String()
	}

	sql = string([]byte(sql)[:len(sql)-1])    //去,
	sql2 = string([]byte(sql2)[:len(sql2)-1]) //去,
	deercoder.DB.Exec(sql)
	dba := deercoder.DB.Exec(sql2) //创建存储群聊消息

	if dba.Error != nil {
		return lib.GetSqlError(dba.Error.Error())
	}

	return lib.MapCreate
}

//查找群聊中所有用户
func GetChatUsers(group_id string) []GroupUsers {

	var gusers []GroupUsers
	deercoder.DB.Raw("select id,group_id,uid from `group_users` where group_id=?", group_id).Scan(&gusers)
	return gusers
}