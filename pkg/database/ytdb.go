package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"

	config "github.com/JustHumanz/Go-Simp/pkg/config"
	log "github.com/sirupsen/logrus"
)

//Get Youtube data from status
func YtGetStatus(Group, Member int64, Status, Region, Uniq string) ([]LiveStream, error) {
	var (
		Data  []LiveStream
		list  LiveStream
		limit int
		Key   = strconv.Itoa(int(Group)) + strconv.Itoa(int(Member)) + Status + Region + Uniq
		rows  *sql.Rows
		err   error
	)
	val := LiveCache.LRange(context.Background(), Key, 0, -1).Val()
	if len(val) == 0 {
		if (Group != 0 && Status != "live") || (Member != 0 && Status == "past") {
			limit = 3
		} else {
			limit = 2525
		}

		if Region != "" {
			rows, err = DB.Query(`SELECT Youtube.* FROM Vtuber.Youtube Inner join Vtuber.VtuberMember on VtuberMember.id=VtuberMember_id Inner join Vtuber.VtuberGroup on VtuberGroup.id = VtuberGroup_id Where VtuberGroup.id=? AND Status=? AND Region=? Order by ScheduledStart DESC Limit ?`, Group, Status, Region, limit)
			if err != nil {
				return nil, err
			} else if err == sql.ErrNoRows {
				return nil, errors.New("not found any schdule")
			}
			defer rows.Close()

		} else if Status == config.PastStatus {
			rows, err = DB.Query(`SELECT Youtube.* FROM Vtuber.Youtube Inner join Vtuber.VtuberMember on VtuberMember.id=VtuberMember_id Inner join Vtuber.VtuberGroup on VtuberGroup.id = VtuberGroup_id Where (VtuberGroup.id=? or VtuberMember.id=?) AND Status=? Order by EndStream DESC Limit ?`, Group, Member, Status, limit)
			if err != nil {
				return nil, err
			} else if err == sql.ErrNoRows {
				return nil, errors.New("not found any schdule")
			}
			defer rows.Close()
		} else {
			rows, err = DB.Query(`SELECT Youtube.* FROM Vtuber.Youtube Inner join Vtuber.VtuberMember on VtuberMember.id=VtuberMember_id Inner join Vtuber.VtuberGroup on VtuberGroup.id = VtuberGroup_id Where (VtuberGroup.id=? or VtuberMember.id=?) AND Status=? Order by ScheduledStart DESC Limit ?`, Group, Member, Status, limit)
			if err != nil {
				return nil, err
			} else if err == sql.ErrNoRows {
				return nil, errors.New("not found any schdule")
			}
			defer rows.Close()
		}

		for rows.Next() {
			err = rows.Scan(&list.ID, &list.VideoID, &list.Type, &list.Status, &list.Title, &list.Thumb, &list.Desc, &list.Published, &list.Schedul, &list.End, &list.Viewers, &list.Length, &list.Member.ID)
			if err != nil {
				return nil, err
			}
			list.Status = Status
			Data = append(Data, list)
			err = LiveCache.LPush(context.Background(), Key, list).Err()
			if err != nil {
				return nil, err
			}
		}
		err = LiveCache.Expire(context.Background(), Key, config.YtGetStatusTTL).Err()
		if err != nil {
			return nil, err
		}
	} else {
		for _, result := range val {
			err := json.Unmarshal([]byte(result), &list)
			if err != nil {
				return nil, err
			}
			Data = append(Data, list)
		}
	}

	return Data, nil

}

func RemoveYtCache(Key string, ctx context.Context) error {
	log.WithFields(log.Fields{
		"Key": Key,
	}).Info("Drop cache")

	err := LiveCache.Del(ctx, Key).Err()
	if err != nil {
		return err
	}
	return nil
}

//Input youtube new video
func (Data *LiveStream) InputYt() (int64, error) {
	if !Data.Member.IsMemberNill() {
		stmt, err := DB.Prepare(`INSERT INTO Youtube (VideoID,Type,Status,Title,Thumbnails,Description,PublishedAt,ScheduledStart,EndStream,Viewers,Length,VtuberMember_id) values(?,?,?,?,?,?,?,?,?,?,?,?)`)
		if err != nil {
			return 0, err
		}
		defer stmt.Close()

		res, err := stmt.Exec(Data.VideoID, Data.Type, Data.Status, Data.Title, Data.Thumb, Data.Desc, Data.Published, Data.Schedul, Data.End, Data.Viewers, Data.Length, Data.Member.ID)
		if err != nil {
			return 0, err
		}

		id, err := res.LastInsertId()
		if err != nil {
			return 0, err
		}

		return id, nil
	} else {
		stmt, err := DB.Prepare(`INSERT INTO GroupVideos (VideoID,Type,Status,Title,Thumbnails,Description,PublishedAt,ScheduledStart,EndStream,Viewers,Length,LiveBili,VtuberGroup_id) values(?,?,?,?,?,?,?,?,?,?,?,?,?)`)
		if err != nil {
			return 0, err
		}
		defer stmt.Close()

		res, err := stmt.Exec(Data.VideoID, Data.Type, Data.Status, Data.Title, Data.Thumb, Data.Desc, Data.Published, Data.Schedul, Data.End, Data.Viewers, Data.Length, Data.IsBiliLive, Data.Group.ID)
		if err != nil {
			return 0, err
		}

		id, err := res.LastInsertId()
		if err != nil {
			return 0, err
		}

		return id, nil
	}
}

func (Data LiveStream) YtIsEmpty() bool {
	if Data.VideoID != "" {
		return false
	} else {
		return true
	}
}

//Check new video or not
func (Member Member) CheckYoutubeVideo(VideoID string) (*LiveStream, error) {
	var Data LiveStream
	rows, err := DB.Query(`SELECT * FROM Vtuber.Youtube Where VideoID=? AND VtuberMember_id=?`, VideoID, Member.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&Data.ID, &Data.VideoID, &Data.Type, &Data.Status, &Data.Title, &Data.Thumb, &Data.Desc, &Data.Published, &Data.Schedul, &Data.End, &Data.Viewers, &Data.Length, &Data.Member.ID)
		if err != nil {
			return nil, err
		}
	}
	if Data.ID == 0 {
		return nil, errors.New("VideoID not found in database")
	} else {
		Data.AddMember(Member)
		return &Data, nil
	}
}

//Check new video or not
func (Group GroupYtChannel) CheckYoutubeVideo(VideoID string) (*LiveStream, error) {
	var Data LiveStream
	rows, err := DB.Query(`SELECT * FROM Vtuber.GroupVideos Where VideoID=? AND VtuberGroup_id=?`, VideoID, Group.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&Data.ID, &Data.VideoID, &Data.Type, &Data.Status, &Data.Title, &Data.Thumb, &Data.Desc, &Data.Published, &Data.Schedul, &Data.End, &Data.Viewers, &Data.Length, &Data.IsBiliLive, &Data.Group.ID)
		if err != nil {
			return nil, err
		}
	}
	if Data.ID == 0 {
		return nil, errors.New("VideoID not found in database")
	} else {
		return &Data, nil
	}
}

//Update youtube data
func (Data *LiveStream) UpdateYt(Status string) {
	_, err := DB.Exec(`Update Youtube set Type=?,Status=?,Title=?,Thumbnails=?,Description=?,PublishedAt=?,ScheduledStart=?,EndStream=?,Viewers=?,Length=? where id=?`, Data.Type, Status, Data.Title, Data.Thumb, Data.Desc, Data.Published, Data.Schedul, Data.End, Data.Viewers, Data.Length, Data.ID)
	if err != nil {
		log.Error(err)
	}
}

func CheckReminder(Num int) bool {
	var count int
	err := DB.QueryRow(`SELECT id FROM User where Reminder=?`, Num).Scan(&count)
	if err != nil {
		log.Error(err)
	} else if err == sql.ErrNoRows {
		return false
	}
	return true
}
