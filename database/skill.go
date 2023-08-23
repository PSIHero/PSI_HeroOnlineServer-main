package database

import (
	"database/sql"
	"fmt"
	"sort"

	gorp "gopkg.in/gorp.v1"
)

var (
	SkillInfos       = make(map[int]*SkillInfo)
	SkillInfosByBook = make(map[int64][]*SkillInfo)
	SkillPoints      = make([]uint64, 12000)
)

type SkillInfo struct {
	ID                        int     `db:"id"`
	BookID                    int64   `db:"book_id"`
	Name                      string  `db:"name"`
	Target                    int8    `db:"target"`
	PassiveType               uint8   `db:"passive_type"`
	Type                      uint8   `db:"type"`
	MaxPlus                   int8    `db:"max_plus"`
	Slot                      int     `db:"slot"`
	BaseTime                  int     `db:"base_duration"`
	AdditionalTime            int     `db:"additional_duration"`
	CastTime                  float64 `db:"cast_time"`
	BaseChi                   int     `db:"base_chi"`
	AdditionalChi             int     `db:"additional_chi"`
	BaseMinMultiplier         int     `db:"base_min_multiplier"`
	AdditionalMinMultiplier   int     `db:"additional_min_multiplier"`
	BaseMaxMultiplier         int     `db:"base_max_multiplier"`
	AdditionalMaxMultiplier   int     `db:"additional_max_multiplier"`
	BaseRadius                float64 `db:"base_radius"`
	AdditionalRadius          float64 `db:"additional_radius"`
	Passive                   bool    `db:"passive"`
	BasePassive               int     `db:"base_passive"`
	AdditionalPassive         float64 `db:"additional_passive"`
	InfectionID               int     `db:"infection_id"`
	AreaCenter                int     `db:"area_center"`
	Cooldown                  float64 `db:"cooldown"`
	PoisonDamage              int     `db:"poison_damage"`
	AdditionalPoisonDamage    int     `db:"additional_poison_damage"`
	ConfusionDamage           int     `db:"confusion_damage"`
	AdditionalConfusionDamage int     `db:"additional_confusion_damage"`
	ParaDamage                int     `db:"para_damage"`
	AdditionalParaDamage      int     `db:"additional_para_damage"`
	BaseMinHP                 int     `db:"base_min_hp"`
	AdditionalMinHP           int     `db:"additional_min_hp"`
	BaseMaxHP                 int     `db:"base_max_hp"`
	AdditionalMaxHP           int     `db:"additional_max_hp"`
}

func (e *SkillInfo) Create() error {
	return db.Insert(e)
}

func (e *SkillInfo) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *SkillInfo) Delete() error {
	_, err := db.Delete(e)
	return err
}

func (e *SkillInfo) Update() error {
	_, err := db.Update(e)
	return err
}

func getSkillInfos() error {
	var skills []*SkillInfo
	query := `select * from data.skills`

	if _, err := db.Select(&skills, query); err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Error to load skills")
			return nil
		}
		fmt.Println("getSkillInfos: %s", err.Error())
		return fmt.Errorf("getSkillInfos: %s", err.Error())
	}

	for _, s := range skills {
		SkillInfos[s.ID] = s
		if s.Slot > 0 {
			SkillInfosByBook[s.BookID] = append(SkillInfosByBook[s.BookID], s)
		}
	}

	for _, b := range SkillInfosByBook {
		sort.Slice(b, func(i, j int) bool {
			return b[i].Slot < b[j].Slot
		})
	}

	for i := uint64(0); i < uint64(len(SkillPoints)); i++ {
		SkillPoints[i] = 2000 * i * i
	}

	return nil
}

func RefreshSkillInfos() error {
	SkillInfosByBook = nil
	var skills []*SkillInfo
	query := `select * from data.skills`

	if _, err := db.Select(&skills, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getSkillInfos: %s", err.Error())
	}

	for _, s := range skills {
		SkillInfos[s.ID] = s
		if s.Slot > 0 {
			SkillInfosByBook[s.BookID] = append(SkillInfosByBook[s.BookID], s)
		}
	}

	for _, b := range SkillInfosByBook {
		sort.Slice(b, func(i, j int) bool {
			return b[i].Slot < b[j].Slot
		})
	}

	for i := uint64(0); i < uint64(len(SkillPoints)); i++ {
		SkillPoints[i] = 2000 * i * i
	}

	return nil
}
