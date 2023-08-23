package database

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	gorp "gopkg.in/gorp.v1"
	null "gopkg.in/guregu/null.v3"
)

type DungeonEvent struct {
	CharacterID     int       `db:"character_id"`
	LastStartedTime null.Time `db:"last_started" json:"LastStartAt"`
}

var (
	DungeonEvents = make(map[int]*DungeonEvent) //LOAD BY CHARACTER ID
)

func GetAllDungeonCharacters() error {
	var dungeonCharacters []*DungeonEvent
	warquery := `select * from data.dungeon_table`
	if _, err := db.Select(&dungeonCharacters, warquery); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("GetDungeonCharacters: %s", err.Error())
	}

	for _, char := range dungeonCharacters {
		DungeonEvents[char.CharacterID] = char
	}
	return nil
}

func (u *DungeonEvent) Create() error {
	return db.Insert(u)
}

func (u *DungeonEvent) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(u)
}

func (u *DungeonEvent) Update() error {
	_, err := db.Update(u)
	return err
}

func (u *DungeonEvent) Delete() error {
	_, err := db.Delete(u)
	return err
}

type BossEvent struct {
	EventID      int       `db:"event_id"`
	Activated    bool      `db:"activated"`
	MobsIds      string    `db:"mobs_id"`
	RespawnTime  null.Time `db:"respawn_time"`
	LastKilledAt null.Time `db:"last_killed" json:"LastKilledAt"`
	MapID        int       `db:"map_id"`
	MinLocation  string    `db:"min_location"`
	MaxLocation  string    `db:"max_location"`
}

var (
	BossEvents    = make(map[int]*BossEvent)
	BossMobsID    = []int{}
	BossNcashDrop = 100
)

func GetAllEvents() error {
	var bossevents []*BossEvent
	bossquery := `select * from data.mobevent_table`

	//BOSS EVENT
	if _, err := db.Select(&bossevents, bossquery); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		if strings.Contains(err.Error(), "does not exist") {
			_, err := db.Exec(`CREATE TABLE "data".mobevent_table (
				event_id int4 NOT NULL,
				activated bool NOT NULL DEFAULT false,
				mobs_id jsonb NOT NULL,
				respawn_time timestamptz NULL,
				last_killed timestamptz NULL,
				map_id int4 NOT NULL,
				min_location point NOT NULL,
				max_location point NOT NULL,
				CONSTRAINT mobevent_table_pkey PRIMARY KEY (event_id)
			);`)
			db.Exec(`ALTER TABLE hops.ai ADD event_id int4 NOT NULL DEFAULT 0;`)
			return err
		}

		return fmt.Errorf("GetMobsEvents: %s", err.Error())
	}

	for _, bossevent := range bossevents {
		BossEvents[bossevent.EventID] = bossevent
	}
	return nil
}

// #region BossEvent
func (e *BossEvent) Create() error {
	return db.Insert(e)
}

func (e *BossEvent) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *BossEvent) Delete() error {
	_, err := db.Delete(e)
	return err
}
func (e *BossEvent) Update() error {
	_, err := db.Update(e)
	return err
}
func (e *BossEvent) GetBosses() []int {
	bosses := strings.Trim(e.MobsIds, "{}")
	sBosses := strings.Split(bosses, ",")

	var arr []int
	BossMobsID = []int{}
	for _, sBoss := range sBosses {
		d, _ := strconv.Atoi(sBoss)
		arr = append(arr, d)
		BossMobsID = append(BossMobsID, d)
	}
	return arr
}
func getMobEventByID(eventid int) []*BossEvent {
	var bossevents []*BossEvent
	query := `select * from data.mobevent_table where activated = 'true' and event_id = $1`

	if _, err := db.Select(&bossevents, query, eventid); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return nil
	}
	return bossevents
}

func isMobEventActiveYetByID(eventid int) (bool, error) {
	var bossevents []*BossEvent
	query := `select * from data.mobevent_table where activated = 'true' and event_id = $1`

	if _, err := db.Select(&bossevents, query, eventid); err != nil {
		if err == sql.ErrNoRows {
			return false, err
		}
		return false, err
	}
	return len(bossevents) > 0, nil
}

//#endregion BossEvent
