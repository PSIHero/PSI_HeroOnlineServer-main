package database

import (
	"database/sql"
	"encoding/json"
	"fmt"

	gorp "gopkg.in/gorp.v1"
)

var (
	QuestsList = make(map[int]*QuestList)
)

type QuestList struct {
	ID               int              `db:"id"`
	Group            string           `db:"groupid"`
	MenuID           int              `db:"menuid"`
	Scenario         int              `db:"scenario"`
	MapID            int              `db:"mapid"`
	NPCID            int64            `db:"npcid"`
	RequestItems     []byte           `db:"request_items"`
	DropFromMobs     bool             `db:"dropfrommobs"`
	TitleID          int              `db:"titleid"`
	RewardItems      []byte           `db:"reward_items"`
	RewardExp        int              `db:"reward_exp"`
	RewardGold       int              `db:"reward_gold"`
	FinishNPC        int              `db:"finishnpcid"`
	LevelRequirement int              `db:"levelrequirement"`
	NextMissionID    int              `db:"nextmissionid"`
	PrevMissionID    int              `db:"prevmissionid"`
	request_items    []*QuestReqItems `db:"-"`
	reward_items     []*QuestRewItems `db:"-"`
}

type QuestReqItems struct {
	ItemID    int64 `json:"item_id"`
	DropMobID int   `json:"dropmobid"`
	ItemCount int   `json:"count"`
}
type QuestRewItems struct {
	ItemID    int64 `json:"item_id"`
	ItemCount int   `json:"count"`
}

func (e *QuestList) Create() error {
	return db.Insert(e)
}

func (e *QuestList) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *QuestList) Delete() error {
	_, err := db.Delete(e)
	return err
}

func (e *QuestList) Update() error {
	_, err := db.Update(e)
	return err
}
func getAllQuests() error {
	var questlist []*QuestList
	query := `select * from data.quests`

	if _, err := db.Select(&questlist, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getAllQuest: %s", err.Error())
	}

	for _, q := range questlist {
		QuestsList[q.ID] = q
	}

	return nil
}

func RefreshAllQuests() error {
	var questlist []*QuestList
	query := `select * from data.quests`

	if _, err := db.Select(&questlist, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getAllQuest: %s", err.Error())
	}

	for _, q := range questlist {
		QuestsList[q.ID] = q
	}

	return nil
}

func (e *QuestList) GetQuestReqItems() ([]*QuestReqItems, error) {
	if len(e.request_items) > 0 {
		return e.request_items, nil
	}

	err := json.Unmarshal(e.RequestItems, &e.request_items)
	if err != nil {
		return nil, err
	}

	return e.request_items, nil
}

func (e *QuestList) GetQuestRewItems() ([]*QuestRewItems, error) {
	if len(e.reward_items) > 0 {
		return e.reward_items, nil
	}

	err := json.Unmarshal(e.RewardItems, &e.reward_items)
	if err != nil {
		return nil, err
	}

	return e.reward_items, nil
}

func FindQuestByMenuID(menuID int64, npcID int) (*QuestList, error) {

	var quest *QuestList
	query := `select * from data.quests where menuid = $1 and (npcid = $2 or finishnpcid = $3)`
	fmt.Println(fmt.Sprintf("MENU: %d NPC: %d", menuID, npcID))
	if err := db.SelectOne(&quest, query, menuID, npcID, npcID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindBuffByID: %s", err.Error())
	}

	return quest, nil
}

func FindQuestByID(ID int64) (*QuestList, error) {

	var quest *QuestList
	query := `select * from data.quests where id = $1`

	if err := db.SelectOne(&quest, query, ID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindBuffByID: %s", err.Error())
	}

	return quest, nil
}

func FindQuestByMapID(mapID int) ([]*QuestList, error) {

	//var quests []QuestList
	query := `select * from data.quests where mapid = $1`
	quests := []*QuestList{}
	if _, err := db.Select(&quests, query, mapID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getAllQuest: %s", err.Error())
	}

	return quests, nil
}

func FindQuestByNpcID(npcID int) ([]*QuestList, error) {

	//var quests []QuestList
	query := `select * from data.quests where npcid = $1 or finishnpcid = $2`
	quests := []*QuestList{}
	if _, err := db.Select(&quests, query, npcID, npcID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getAllQuest: %s", err.Error())
	}

	return quests, nil
}

func GetQuestForPlayer(c *Character) ([]*QuestList, error) {

	query := `select * from data.quests where levelrequirement <= $1`
	quests := []*QuestList{}
	if _, err := db.Select(&quests, query, c.Level); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getAllQuest: %s", err.Error())
	}
	return quests, nil
}
