package auth

import (
	"fmt"
	"sort"

	"PsiHero/database"
	"PsiHero/logging"
	"PsiHero/utils"
)

type CharacterDeletionHandler struct {
	index int
	name  string
}

var (
	CHARACTER_DELETED = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x01, 0xB2, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA}
)

func (cdh *CharacterDeletionHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	cdh.index = int(data[6])
	length := int(data[7])
	cdh.name = string(data[8 : length+8])
	return cdh.deleteCharacter(s)
}

func (cdh *CharacterDeletionHandler) deleteCharacter(s *database.Socket) ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("no socket founded")
	}
	if s.User == nil {
		return nil, fmt.Errorf("no socket user founded")
	}
	chars, err := database.FindCharactersByUserID(s.User.ID)
	if err != nil {
		return nil, err
	}

	sort.Slice(chars, func(i, j int) bool {
		return chars[i].ID < chars[j].ID
	})

	character := chars[cdh.index]
	for i := 0; i < len(chars); i++ {
		character = chars[i]
		if character.Name == cdh.name {
			break
		}
	}

	resp := CHARACTER_DELETED
	if character.Name == cdh.name {

		if err = character.Delete(); err != nil {
			return nil, err
		}

		if character.GuildID > 0 {
			guild, err := database.FindGuildByID(character.GuildID)
			if err != nil {
				return nil, err
			}

			if guild != nil {
				if guild.LeaderID == character.ID {
					guild.Delete()
				} else {
					guild.RemoveMember(character.ID)
					go guild.Update()
				}
			}
		}

		consItems, err := database.FindConsignmentItemsBySellerID(character.ID)
		if err != nil {
			return nil, err
		}

		for _, item := range consItems {
			err = item.Delete()
			if err != nil {
				return nil, err
			}
		}

		stat, err := database.FindStatByID(character.ID)
		if err != nil {
			return nil, err
		}
		stat.Delete()
		database.DeleteAllFriendsByCharID(character.ID)
		skills, err := database.FindSkillsByID(character.ID)
		if err != nil {
			return nil, err
		}
		skills.Delete()
		teleports, err := database.FindTeleportsByID(character.ID)
		if err != nil {
			return nil, err
		}
		teleports.Delete()
		length := int16(len(cdh.name)) + 6
		resp.SetLength(length)

		resp[8] = byte(cdh.index)         // character index
		resp[9] = byte(len(cdh.name))     // character name length
		resp.Insert([]byte(cdh.name), 10) // character name

		lch := &ListCharactersHandler{}
		data, err := lch.showCharacterMenu(s)
		if err != nil {
			return nil, err
		}

		logger.Log(logging.ACTION_DELETE_CHARACTER, character.ID, "Character deleted", s.User.ID)
		resp.Concat(data)
	}

	return resp, nil
}
