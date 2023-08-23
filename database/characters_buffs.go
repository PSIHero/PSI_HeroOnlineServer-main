package database

import (
	"database/sql"
	"fmt"
	"sort"

	gorp "gopkg.in/gorp.v1"
)

type Buff struct {
	ID              int     `db:"id" json:"id"`
	CharacterID     int     `db:"character_id" json:"character_id"`
	UserID          string  `db:"user_id" json:"user_id"`
	Name            string  `db:"name" json:"name"`
	ATK             int     `db:"atk" json:"atk"`
	ATKRate         int     `db:"atk_rate" json:"atk_rate"`
	ArtsATK         int     `db:"arts_atk" json:"arts_atk"`
	ArtsATKRate     int     `db:"arts_atk_rate" json:"arts_atk_rate"`
	PoisonDEF       int     `db:"poison_def" json:"poison_def"`
	ParalysisDEF    int     `db:"paralysis_def" json:"paralysis_def"`
	ConfusionDEF    int     `db:"confusion_def" json:"confusion_def"`
	DEF             int     `db:"def" json:"def"`
	DEFRate         int     `db:"def_rate" json:"def_rate"`
	ArtsDEF         int     `db:"arts_def" json:"arts_def"`
	ArtsDEFRate     int     `db:"arts_def_rate" json:"arts_def_rate"`
	Accuracy        int     `db:"accuracy" json:"accuracy"`
	Dodge           int     `db:"dodge" json:"dodge"`
	MaxHP           int     `db:"max_hp" json:"max_hp"`
	HPRecoveryRate  int     `db:"hp_recovery_rate" json:"hp_recovery_rate"`
	MaxCHI          int     `db:"max_chi" json:"max_chi"`
	CHIRecoveryRate int     `db:"chi_recovery_rate" json:"chi_recovery_rate"`
	STR             int     `db:"str" json:"str"`
	DEX             int     `db:"dex" json:"dex"`
	INT             int     `db:"int" json:"int"`
	ExpRate         int     `db:"exp_rate" json:"exp_rate"`
	DropRate        int     `db:"drop_rate" json:"drop_rate"`
	GoldRate        int     `db:"gold_rate" json:"gold_rate"`
	RunningSpeed    float64 `db:"running_speed" json:"running_speed"`
	StartedAt       int64   `db:"started_at" json:"started_at"`
	Duration        int64   `db:"duration" json:"duration"`
	BagExpansion    bool    `db:"bag_expansion" json:"bag_expansion"`
	SkillPlus       int     `db:"skill_plus" json:"skill_plus"`
	CanExpire       bool    `db:"canexpire" json:"canexpire"`
	Wind            int     `db:"wind" json:"wind"`
	Water           int     `db:"water" json:"water"`
	Fire            int     `db:"fire" json:"fire"`
	Reflect         int     `db:"reflect" json:"reflect"`
	Critical_Strike int     `db:"critical_strike" json:"critical_strike"`
	IsPercent       bool    `db:"ispercent" json:"ispercent"`
	IsServerEpoch   bool    `db:"server_epoch" json:"server_epoch"`
	Plus            int     `db:"plus" json:"plus"`
}

func (b *Buff) Create() error {
	return db.Insert(b)
}

func (b *Buff) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(b)
}

func (b *Buff) Delete() error {
	_, err := db.Delete(b)
	return err
}

func (b *Buff) Update() error {
	_, err := db.Update(b)
	return err
}

func FindBuffsByCharacterID(characterID int) ([]*Buff, error) {

	var buffs []*Buff
	query := `select * from hops.characters_buffs where character_id = $1`

	if _, err := db.Select(&buffs, query, characterID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindBuffsByCharacterID: %s", err.Error())
	}

	sort.Slice(buffs, func(i, j int) bool {
		return buffs[i].StartedAt+buffs[i].Duration <= buffs[j].StartedAt+buffs[j].Duration
	})

	return buffs, nil
}

func FindBuffsByUserID(userID string) ([]*Buff, error) {

	var buffs []*Buff
	query := `select * from hops.characters_buffs where user_id = $1`

	if _, err := db.Select(&buffs, query, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindBuffsByUserID: %s", err.Error())
	}

	sort.Slice(buffs, func(i, j int) bool {
		return buffs[i].StartedAt+buffs[i].Duration <= buffs[j].StartedAt+buffs[j].Duration
	})

	return buffs, nil
}

func FindBuffByCharacter(buffID, characterID int) (*Buff, error) {

	var buff *Buff
	query := `select * from hops.characters_buffs where id = $1 and character_id = $2`

	if err := db.SelectOne(&buff, query, buffID, characterID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindBuffByCharacter: %s", err.Error())
	}

	return buff, nil
}

func FindBuffByUser(buffID int, userID string) (*Buff, error) {

	var buff *Buff
	query := `select * from hops.characters_buffs where id = $1 and user_id = $2`

	if err := db.SelectOne(&buff, query, buffID, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindBuffByCharacter: %s", err.Error())
	}

	return buff, nil
}

func DeleteBuffByCharacterID(buffID, characterID int) ([]*Buff, error) {
	query := `DELETE from hops.characters_buffs where id = $1 and character_id = $2`

	_, err := db.Exec(query, buffID, characterID)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func DeleteBuffByUserID(buffID int, userID string) ([]*Buff, error) {
	query := `DELETE from hops.characters_buffs where id = $1 and user_id = $2`

	_, err := db.Exec(query, buffID, userID)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (c *Character) FindAllRelevantBuffs() ([]*Buff, error) {

	characterBuffs, err := FindBuffsByCharacterID(c.ID)
	if err != nil {
		return nil, err
	}
	userBuffs, err := FindBuffsByUserID(c.UserID)
	if err != nil {
		return nil, err
	}
	eventBuffs, err := FindBuffsByUserID("event")
	if err != nil {
		return nil, err
	}

	if len(characterBuffs) == 0 && len(userBuffs) == 0 {
		return nil, nil
	}

	buffMap := make(map[int]bool)
	allBuffs := []*Buff{}

	for _, buff := range characterBuffs {
		if _, ok := buffMap[buff.ID]; !ok {
			allBuffs = append(allBuffs, buff)
			buffMap[buff.ID] = true
		}
	}

	for _, buff := range userBuffs {
		if _, ok := buffMap[buff.ID]; !ok {
			allBuffs = append(allBuffs, buff)
			buffMap[buff.ID] = true
		}
	}

	for _, buff := range eventBuffs {
		if _, ok := buffMap[buff.ID]; !ok {
			allBuffs = append(allBuffs, buff)
			buffMap[buff.ID] = true
		}
	}
	return allBuffs, nil
}
