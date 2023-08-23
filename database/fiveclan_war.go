package database

import (
	"database/sql"
	"fmt"
	"time"

	null "gopkg.in/guregu/null.v3"
)

var (
	FiveClans     = make(map[int]*FiveClan)
	FiveclanMobs  = []int{423308, 423310, 423312, 423314, 423316}
	FiveclanBuffs = []int{60001, 60006, 60011, 60016, 60021}
)

type FiveClan struct {
	AreaID     int       `db:"id"`
	ClanID     int       `db:"clanid"`
	ExpiresAt  null.Time `db:"expires_at" json:"expires_at"`
	TempleName string    `db:"name" json:"name"`
}

func (b *FiveClan) Update() error {
	_, err := db.Update(b)
	return err
}

func getFiveAreas() error {
	var areas []*FiveClan
	query := `select * from data.fiveclan_war`

	if _, err := db.Select(&areas, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getFiveAreas: %s", err.Error())
	}

	for _, cr := range areas {
		FiveClans[cr.AreaID] = cr
	}

	return nil
}

func CaptureFive(number int, char *Character) {
	buffID := FiveclanBuffs[number-1]
	guild, err := FindGuildByID(char.GuildID)
	if err != nil {
		return
	}
	allmembers, _ := guild.GetMembers()
	for _, m := range allmembers {
		c, err := FindCharacterByID(m.ID)
		if err != nil || c == nil {
			continue
		}
		infection := BuffInfections[buffID]
		MakeAnnouncement("[" + guild.Name + "] captured [" + FiveClans[number].TempleName + "]")
		buff := &Buff{ID: infection.ID, CharacterID: c.ID, Name: infection.Name, ExpRate: 200, StartedAt: c.Epoch, Duration: 7200, CanExpire: true}
		err = buff.Create()
		if err != nil {
			continue
		}
	}
}

func LogoutFiveBuffDelete(char *Character) {
	for _, fivebuff := range FiveclanBuffs {
		buff, err := FindBuffByCharacter(fivebuff, char.ID)
		if err != nil {
			continue
		}
		if buff == nil {
			continue
		}
		buff.Delete()
	}
}

func AddFiveBuffWhenLogin(char *Character) error {
	if char.GuildID > 0 {
		guild, err := FindGuildByID(char.GuildID)
		if err != nil {
			return err
		}
		for _, clans := range FiveClans {
			if clans.ClanID == guild.ID {
				buffID := FiveclanBuffs[clans.AreaID-1]
				currentTime := time.Now()
				diff := clans.ExpiresAt.Time.Sub(currentTime)
				if diff < 0 {
					clans.ClanID = 0
					clans.Update()
					continue
				}
				infection := BuffInfections[buffID]
				buff := &Buff{ID: infection.ID, CharacterID: char.ID, Name: infection.Name, ExpRate: 200, StartedAt: char.Epoch, Duration: int64(diff.Seconds()), CanExpire: true}
				err = buff.Create()
				if err != nil {
					continue
				}
			}
		}
	}
	return nil
}
