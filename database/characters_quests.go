package database

import (
	"database/sql"
	"fmt"

	"gopkg.in/gorp.v1"
)

type Quest struct {
	ID          int `db:"id" json:"id"`
	CharacterID int `db:"character_id" json:"character_id"`
	QuestState  int `db:"quest_state" json:"quest_state"`
}

func (b *Quest) Create() error {
	return db.Insert(b)
}

func (b *Quest) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(b)
}

func (b *Quest) Delete() error {
	_, err := db.Delete(b)
	return err
}

func (b *Quest) Update() error {
	_, err := db.Update(b)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error: %s", err.Error()))
	}
	return err
}

func FindQuestsByCharacterID(characterID int) ([]*Quest, error) {

	var quests []*Quest
	query := `select * from hops.characters_quests where character_id = $1`

	if _, err := db.Select(&quests, query, characterID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindQuestsByCharacterID: %s", err.Error())
	}

	return quests, nil
}

func FindQuestsAcceptedByID(characterID int) ([]int, error) {

	var quests []int
	QuestState := 2
	aqueststate := 3
	query := `select id from hops.characters_quests where character_id = $1 OR (quest_state <> $2 AND quest_state <> $3)`

	if _, err := db.Select(&quests, query, characterID, QuestState, aqueststate); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindQuestsByCharacterID: %s", err.Error())
	}

	return quests, nil
}

func FindPlayerQuestByID(questID, characterID int) (*Quest, error) {

	var q *Quest
	query := `select * from hops.characters_quests where id = $1 and character_id = $2`

	if err := db.SelectOne(&q, query, questID, characterID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindBuffByID: %s", err.Error())
	}

	return q, nil
}
