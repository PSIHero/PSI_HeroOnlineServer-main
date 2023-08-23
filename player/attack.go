package player

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"time"

	"PsiHero/database"
	"PsiHero/messaging"
	"PsiHero/nats"
	"PsiHero/server"
	"PsiHero/utils"

	"github.com/thoas/go-funk"
)

type (
	AttackHandler        struct{}
	InstantAttackHandler struct{}
	DealDamageHandler    struct{}
	CastSkillHandler     struct{}
	CastMonkSkillHandler struct{}
	RemoveBuffHandler    struct{}
)

var (
	ATTACKED      = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x41, 0x01, 0x0D, 0x02, 0x01, 0x00, 0x00, 0x00, 0x55, 0xAA}
	INST_ATTACKED = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x41, 0x01, 0x0D, 0x02, 0x01, 0x00, 0x00, 0x00, 0x55, 0xAA}
)

func (h *AttackHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	st := s.Stats
	if st == nil {
		return nil, nil
	}

	aiID := uint16(utils.BytesToInt(data[7:9], true))
	ai, ok := database.GetFromRegister(s.User.ConnectedServer, s.Character.Map, aiID).(*database.AI)
	if ok {
		if ai == nil || ai.HP <= 0 {
			return nil, nil
		}

		npcPos := database.NPCPos[ai.PosID]
		if npcPos == nil {
			return nil, nil
		}

		npc := database.NPCs[npcPos.NPCID]
		if npc == nil {
			return nil, nil
		}

		if npcPos.Attackable {
			ai.MovementToken = 0
			ai.IsMoving = false
			ai.TargetPlayerID = c.ID

			dmg, err := c.CalculateDamage(ai, false)
			if err != nil {
				return nil, err
			}

			if diff := int(npc.Level) - c.Level; diff > 0 {
				reqAcc := utils.SigmaFunc(float64(diff))
				if float64(st.Accuracy) < reqAcc {
					probability := float64(st.Accuracy) * 1000 / reqAcc
					if utils.RandInt(0, 1000) > int64(probability) {
						dmg = 0
					}
				}
			}
			if dmg > 0 {
				if st.PoisonATK > 0 {
					buff, err := database.FindBuffByAIID(257, int(ai.PseudoID))
					if buff == nil && err == nil {
						probability := st.PoisonATK
						if utils.RandInt(0, 1000) < int64(probability) {
							now := time.Now()
							secs := now.Unix()
							infection := database.BuffInfections[257]
							ai, _ := database.GetFromRegister(s.User.ConnectedServer, s.Character.Map, uint16(s.Character.Selection)).(*database.AI)
							buff := &database.AiBuff{ID: 257, AiID: int(ai.PseudoID), Name: infection.Name, HPRecoveryRate: st.PoisonATK, StartedAt: secs, CharacterID: s.Character.ID, Duration: int64(s.Character.Socket.Stats.PoisonTime)}
							err = buff.Create()
							if err != nil {
								fmt.Println(fmt.Sprintf("Error: %s", err.Error()))
								return nil, err
							}
							s.Character.DealPoisonDamageToAI(ai)
						}
					}
				}
			}
			c.Targets = append(c.Targets, &database.Target{Damage: dmg, AI: ai, Skill: false})
		}

	} else if enemy := server.FindCharacter(s.User.ConnectedServer, aiID); enemy != nil {
		enemy := server.FindCharacter(s.User.ConnectedServer, aiID)
		if enemy == nil || !enemy.IsActive {
			return nil, nil
		}

		dmg, err := c.CalculateDamageToPlayer(enemy, false)
		if err != nil {
			return nil, err
		}

		c.PlayerTargets = append(c.PlayerTargets, &database.PlayerTarget{Damage: dmg, Enemy: enemy, Skill: false})
	}

	resp := ATTACKED
	resp[4] = data[4]
	resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 6) // character pseudo id
	resp.Insert(utils.IntToBytes(uint64(aiID), 2, true), 9)       // ai id

	p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: resp, Type: nats.MOB_ATTACK}
	if err := p.Cast(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *InstantAttackHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	st := s.Stats
	if st == nil {
		return nil, nil
	}

	aiID := uint16(utils.BytesToInt(data[7:9], true))
	ai, ok := database.GetFromRegister(s.User.ConnectedServer, s.Character.Map, aiID).(*database.AI)
	if ok {
		if ai == nil || ai.HP <= 0 {
			return nil, nil
		}

		npcPos := database.NPCPos[ai.PosID]
		if npcPos == nil {
			return nil, nil
		}

		npc := database.NPCs[npcPos.NPCID]
		if npc == nil {
			return nil, nil
		}

		if npcPos.Attackable {
			ai.MovementToken = 0
			ai.IsMoving = false
			ai.TargetPlayerID = c.ID

			dmg := int(utils.RandInt(int64(st.MinATK), int64(st.MaxATK))) - npc.DEF
			if dmg < 0 {
				dmg = 0
			} else if dmg > ai.HP {
				dmg = ai.HP
			}

			if diff := int(npc.Level) - c.Level; diff > 0 {
				reqAcc := utils.SigmaFunc(float64(diff))
				if float64(st.Accuracy) < reqAcc {
					probability := float64(st.Accuracy) * 1000 / reqAcc
					if utils.RandInt(0, 1000) > int64(probability) {
						dmg = 0
					}
				}
			}
			critical := false
			seed := utils.RandInt(0, 10000)
			characterCrit := int64(5 + float64(((c.Socket.Stats.DEX+c.Socket.Stats.DEXBuff)/200))*100)
			//fmt.Println("Seed:", seed, " CharCrit:", characterCrit)
			if characterCrit > seed {
				critical = true
			}
			//time.AfterFunc(time.Second/2, func() { // Quoted out to test if delay is neccessary
			go c.DealDamage(ai, dmg, false, critical)
			//})
		}

	} else if enemy := server.FindCharacter(s.User.ConnectedServer, aiID); enemy != nil {

		if enemy == nil || !enemy.IsActive {
			return nil, nil
		}

		dmg, err := c.CalculateDamageToPlayer(enemy, false)
		if err != nil {
			return nil, err
		}
		critical := false
		seed := utils.RandInt(0, 10000)
		characterCrit := int64(5 + float64(((c.Socket.Stats.DEX+c.Socket.Stats.DEXBuff)/200))*100)
		//fmt.Println("Seed:", seed, " CharCrit:", characterCrit)
		if characterCrit > seed {
			critical = true
		}
		//time.AfterFunc(time.Second/2, func() { // Quoted out to test if delay is neccessary
		if c.CanAttack(enemy) {
			go DealDamageToPlayer(s, enemy, dmg, false, critical)
		}
		//})
	}

	resp := INST_ATTACKED
	resp[4] = data[4]
	resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 6) // character pseudo id
	resp.Insert(utils.IntToBytes(uint64(aiID), 2, true), 9)       // ai id

	p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: resp, Type: nats.MOB_ATTACK}
	if err := p.Cast(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *DealDamageHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}
	resp := utils.Packet{}
	if c.TamingAI != nil {
		ai := c.TamingAI
		pos := database.NPCPos[ai.PosID]
		npc := database.NPCs[pos.NPCID]
		petInfo := database.Pets[int64(npc.ID)]

		seed := utils.RandInt(0, 1000)
		//proportion := float64(ai.HP) / float64(npc.MaxHp) proportion < 0.1 &&
		if seed < 460 && petInfo != nil {
			go c.DealDamage(ai, ai.HP, true, false)
			item := &database.InventorySlot{ItemID: int64(npc.ID), Quantity: 1}
			expInfo := database.PetExps[petInfo.Level-1]
			item.Pet = &database.PetSlot{
				Fullness: 100, Loyalty: 100,
				Exp:   uint64(expInfo.ReqExpEvo1),
				HP:    petInfo.BaseHP,
				Level: byte(petInfo.Level),
				Name:  petInfo.Name,
				CHI:   petInfo.BaseChi,
			}

			r, _, err := s.Character.AddItem(item, -1, true)
			if err != nil {
				return nil, err
			}

			resp.Concat(*r)
		}

		c.TamingAI = nil
		return resp, nil
	}

	targets := c.Targets
	dealt := make(map[int]struct{})
	for _, target := range targets {
		if target == nil {
			continue
		}

		ai := target.AI
		if _, ok := dealt[ai.ID]; ok {
			continue
		}
		dmg := target.Damage
		seed := utils.RandInt(0, 10000)
		characterCrit := int64(5 + float64(((c.Socket.Stats.DEX+c.Socket.Stats.DEXBuff)/200))*100)
		if characterCrit > seed {
			target.Critical = true
		}
		go c.DealDamage(ai, dmg, target.Skill, target.Critical)
		dealt[ai.ID] = struct{}{}
	}

	pTargets := c.PlayerTargets
	dealt = make(map[int]struct{})
	for _, target := range pTargets {
		if target == nil {
			continue
		}

		enemy := target.Enemy
		if _, ok := dealt[enemy.ID]; ok {
			continue
		}

		if c.CanAttack(enemy) {
			dmg := target.Damage
			seed := utils.RandInt(0, 10000)
			characterCrit := int64(5 + float64(((c.Socket.Stats.DEX+c.Socket.Stats.DEXBuff)/200))*100)
			if characterCrit > seed {
				target.Critical = true
			}
			go DealDamageToPlayer(s, enemy, dmg, target.Skill, target.Critical)
		}

		dealt[enemy.ID] = struct{}{}
	}

	c.Targets = []*database.Target{}
	c.PlayerTargets = []*database.PlayerTarget{}
	return nil, nil
}

func DealDamageToPlayer(s *database.Socket, enemy *database.Character, dmg int, skill bool, critical bool) {
	c := s.Character
	enemySt := enemy.Socket.Stats

	if c == nil {
		log.Println("character is nil")
		return
	} else if enemySt.HP <= 0 {
		return
	}

	//Quoted Out for now because we dont want Invisiblity to away when attacking
	/*
		if s.Character.Invisible {
			buff, _ := database.FindBuffByID(241, s.Character.ID)
			if buff != nil {
				buff.Duration = 0
				go buff.Update()
			}

			buff, _ = database.FindBuffByID(244, s.Character.ID)
			if buff != nil {
				buff.Duration = 0
				go buff.Update()
			}

			buff, _ = database.FindBuffByID(50, s.Character.ID)
			if buff != nil {
				buff.Duration = 0
				go buff.Update()
			}

			buff, _ = database.FindBuffByID(139, s.Character.ID)
			if buff != nil {
				buff.Duration = 0
				go buff.Update()
			}
		}
	*/

	if critical {
		dmg = int(math.Round(float64(dmg) * 1.3))
	}
	enemySt.HP -= dmg
	if enemySt.HP < 0 {
		enemySt.HP = 0
	}
	buffs, err := database.FindBuffsByCharacterID(int(c.PseudoID))
	r := database.DEAL_DAMAGE
	index := 5
	r.Insert(utils.IntToBytes(uint64(enemy.PseudoID), 2, true), index) // character pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // mob pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(enemySt.HP), 4, true), index) // character hp
	index += 4
	r.Insert(utils.IntToBytes(uint64(enemySt.CHI), 4, true), index) // character chi
	index += 4
	//r.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 21) // INJURY
	if skill {
		r.Overwrite([]byte{0xFF, 0xFF, 0x00, 0x00}, index)
	}
	if critical {
		index += 4
		r.Overwrite([]byte{0x02}, index)
		index += 1
	}

	if err == nil {
		r.Overwrite(utils.IntToBytes(uint64(len(buffs)), 1, true), index) //BUFF ID
		index++
		count := 0
		for _, buff := range buffs {
			r.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), index) //BUFF ID
			index += 4
			if count < len(buffs)-1 {
				r.Insert(utils.IntToBytes(uint64(0), 2, true), index) //BUFF ID
				index += 2
			}
			count++
		}
		index += 4
	} else {
		fmt.Println("Valami error: %s", err)
	}
	//index += 3
	r.Insert([]byte{0x00, 0x00, 0x00}, index) // INJURY
	index += 2
	r.SetLength(int16(binary.Size(r) - 6))
	r.Concat(enemy.GetHPandChi())
	p := &nats.CastPacket{CastNear: true, CharacterID: enemy.ID, Data: r, Type: nats.PLAYER_ATTACK}
	if err := p.Cast(); err != nil {
		log.Println("deal damage broadcast error:", err)
		return
	}

	if enemySt.HP <= 0 {
		enemySt.HP = 0
		enemy.KilledByCharacter = c
		enemy.Socket.Write(enemy.GetHPandChi())
		info := fmt.Sprintf("[%s] has defeated [%s]", c.Name, enemy.Name)
		r := messaging.InfoMessage(info)
		if database.WarStarted && c.IsinWar && enemy.IsinWar {
			c.WarKillCount++
			if c.Faction == 1 {
				database.ShaoPoints -= 5
			} else {
				database.OrderPoints -= 5
			}
		}
		if !funk.Contains(database.GMRanks, int8(enemy.Socket.User.UserType)) && !funk.Contains(database.GMRanks, int8(c.Socket.User.UserType)) && funk.Contains(database.LoseEXPServers, int16(c.Socket.User.ConnectedServer)) && funk.Contains(database.LoseEXPServers, int16(enemy.Socket.User.ConnectedServer)) && !c.IsinWar && !enemy.IsinWar {
			randInt := utils.RandInt(1, 3)
			exp, _ := enemy.LosePlayerExp(int(randInt))
			different := int(enemy.Level + 20)
			if c.Level <= different {
				resp, levelUp := c.AddPlayerExp(exp)
				if levelUp {
					statData, err := c.GetStats()
					if err == nil {
						c.Socket.Write(statData)
					}
				}
				c.Socket.Write(resp)
			}
		}
		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: r, Type: nats.PVP_FINISHED}
		p.Cast()
	}
}

func (h *CastSkillHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if len(data) < 24 {
		return nil, nil
	}

	attackCounter := int(data[6])
	skillID := int(utils.BytesToInt(data[7:11], true))
	cX := utils.BytesToFloat(data[11:15], true)
	cY := utils.BytesToFloat(data[15:19], true)
	cZ := utils.BytesToFloat(data[19:23], true)
	targetID := int(utils.BytesToInt(data[23:25], true))
	return s.Character.CastSkill(attackCounter, skillID, targetID, cX, cY, cZ)
}

func (h *CastMonkSkillHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if len(data) < 12 {
		return nil, nil
	}

	attackCounter := 0x1B
	skillID := int(utils.BytesToInt(data[6:10], true))
	cX := utils.BytesToFloat(data[10:14], true)
	cY := utils.BytesToFloat(data[14:18], true)
	cZ := utils.BytesToFloat(data[18:22], true)
	targetID := int(utils.BytesToInt(data[22:24], true))

	resp := utils.Packet{0xAA, 0x55, 0x16, 0x00, 0x49, 0x10, 0x55, 0xAA}
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6) // character pseudo id
	resp.Insert(utils.FloatToBytes(cX, 4, true), 8)                         // coordinate-x
	resp.Insert(utils.FloatToBytes(cY, 4, true), 12)                        // coordinate-y
	resp.Insert(utils.FloatToBytes(cZ, 4, true), 16)                        // coordinate-z
	resp.Insert(utils.IntToBytes(uint64(targetID), 2, true), 20)            // target pseudo id
	resp.Insert(utils.IntToBytes(uint64(skillID), 4, true), 22)             // skill id

	skill := database.SkillInfos[skillID]
	token := s.Character.MovementToken

	time.AfterFunc(time.Duration(skill.CastTime*1000)*time.Millisecond, func() {
		if token == s.Character.MovementToken {
			data, _ := s.Character.CastSkill(attackCounter, skillID, targetID, cX, cY, cZ)
			s.Conn.Write(data)
		}
	})

	return resp, nil
}

func (h *RemoveBuffHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	infectionID := int(utils.BytesToInt(data[6:10], true))
	buff, err := database.FindBuffByCharacter(infectionID, s.Character.ID)
	if err != nil {
		return nil, err
	} else if buff == nil {
		return nil, nil
	}

	buff.Duration = 0
	go buff.Update()
	return nil, nil
}
