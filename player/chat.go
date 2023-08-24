package player

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"PsiHero/database"
	"PsiHero/dungeon"
	"PsiHero/messaging"
	"PsiHero/nats"
	"PsiHero/npc"
	"PsiHero/server"
	"PsiHero/utils"

	_ "github.com/go-sql-driver/mysql"
	"github.com/thoas/go-funk"
	"gopkg.in/guregu/null.v3"
)

type ChatHandler struct {
	chatType  int64
	message   string
	receivers map[int]*database.Character
}

var (
	CHAT_MESSAGE  = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x00, 0x55, 0xAA}
	SHOUT_MESSAGE = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x0E, 0x00, 0x00, 0x55, 0xAA}
	ANNOUNCEMENT  = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x06, 0x00, 0x55, 0xAA}
)

func (h *ChatHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.Character == nil {
		return nil, nil
	}

	user, err := database.FindUserByID(s.Character.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	stat := s.Stats
	if stat == nil {
		return nil, nil
	}

	h.chatType = utils.BytesToInt(data[4:6], false)

	switch h.chatType {
	case 28929: // normal chat
		messageLen := utils.BytesToInt(data[6:8], true)
		h.message = string(data[8 : messageLen+8])

		return h.normalChat(s)
	case 28930: // private chat
		index := 6
		recNameLength := int(data[index])
		index++

		recName := string(data[index : index+recNameLength])
		index += recNameLength

		c, err := database.FindCharacterByName(recName)
		if err != nil {
			return nil, err
		} else if c == nil {
			return messaging.SystemMessage(messaging.WHISPER_FAILED), nil
		}

		h.receivers = map[int]*database.Character{c.ID: c}

		messageLen := int(utils.BytesToInt(data[index:index+2], true))
		index += 2

		h.message = string(data[index : index+messageLen])
		return h.chatWithReceivers(s, h.createChatMessage)

	case 28931: // party chat
		party := database.FindParty(s.Character)
		if party == nil {
			return nil, nil
		}

		messageLen := int(utils.BytesToInt(data[6:8], true))
		h.message = string(data[8 : messageLen+8])

		members := funk.Filter(party.GetMembers(), func(m *database.PartyMember) bool {
			return m.Accepted
		}).([]*database.PartyMember)
		members = append(members, &database.PartyMember{Character: party.Leader, Accepted: true})

		h.receivers = map[int]*database.Character{}
		for _, m := range members {
			if m.ID == s.Character.ID {
				continue
			}

			h.receivers[m.ID] = m.Character
		}

		return h.chatWithReceivers(s, h.createChatMessage)

	case 28932: // guild chat
		if s.Character.GuildID > 0 {
			guild, err := database.FindGuildByID(s.Character.GuildID)
			if err != nil {
				return nil, err
			}

			members, err := guild.GetMembers()
			if err != nil {
				return nil, err
			}

			messageLen := int(utils.BytesToInt(data[6:8], true))
			h.message = string(data[8 : messageLen+8])
			h.receivers = map[int]*database.Character{}

			for _, m := range members {
				c, err := database.FindCharacterByID(m.ID)
				if err != nil || c == nil || !c.IsOnline || c.ID == s.Character.ID {
					continue
				}

				h.receivers[m.ID] = c
			}

			return h.chatWithReceivers(s, h.createChatMessage)
		}

	case 28933, 28946: // roar chat
		if stat.CHI < 100 || time.Now().Sub(s.Character.LastRoar) < 10*time.Second {
			return nil, nil
		}

		s.Character.LastRoar = time.Now()

		characters := make(map[int]*database.Character)

		players, err := database.FindOnlineCharacters()
		if err != nil {
			log.Println(err)
			return nil, err
		}

		for id, player := range players {
			characters[id] = player
		}

		//delete(characters, s.Character.ID)
		h.receivers = characters

		stat.CHI -= 100

		index := 6
		messageLen := int(utils.BytesToInt(data[index:index+2], true))
		index += 2

		h.message = string(data[index : index+messageLen])

		resp := utils.Packet{}
		_, err = h.chatWithReceivers(s, h.createChatMessage)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		//resp.Concat(chat)
		resp.Concat(s.Character.GetHPandChi())
		return resp, nil

	case 28935: // commands
		index := 6
		messageLen := int(data[index])
		index++

		h.message = string(data[index : index+messageLen])
		return h.cmdMessage(s, data)

	case 28943: // shout
		return h.Shout(s, data)

	case 28945: // faction chat
		characters, err := database.FindCharactersInServer(user.ConnectedServer)
		if err != nil {
			return nil, err
		}

		//delete(characters, s.Character.ID)
		for _, c := range characters {
			if c.Faction != s.Character.Faction {
				delete(characters, c.ID)
			}
		}

		h.receivers = characters
		index := 6
		messageLen := int(utils.BytesToInt(data[index:index+2], true))
		index += 2

		h.message = string(data[index : index+messageLen])
		resp := utils.Packet{}
		_, err = h.chatWithReceivers(s, h.createChatMessage)
		if err != nil {
			return nil, err
		}

		//resp.Concat(chat)
		return resp, nil

	}

	return nil, nil
}

func (h *ChatHandler) Shout(s *database.Socket, data []byte) ([]byte, error) {
	if time.Now().Sub(s.Character.LastRoar) < 10*time.Second {
		return nil, nil
	}

	characters, err := database.FindOnlineCharacters()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	//delete(characters, s.Character.ID)

	slot, _, err := s.Character.FindItemInInventory(nil, 15900001, 17500181, 17502689, 13000131)
	if err != nil {
		log.Println(err)
		return nil, err
	} else if slot == -1 {
		return nil, nil
	}

	resp := s.Character.DecrementItem(slot, 1)

	index := 6
	messageLen := int(data[index])
	index++

	h.chatType = 28942
	h.receivers = characters
	h.message = string(data[index : index+messageLen])

	_, err = h.chatWithReceivers(s, h.createShoutMessage)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	//resp.Concat(chat)
	return *resp, nil
}

func (h *ChatHandler) createChatMessage(s *database.Socket) *utils.Packet {

	resp := CHAT_MESSAGE

	index := 4
	resp.Insert(utils.IntToBytes(uint64(h.chatType), 2, false), index) // chat type
	index += 2

	if h.chatType != 28946 {
		resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), index) // sender character pseudo id
		index += 2
	}

	resp[index] = byte(len(s.Character.Name)) // character name length
	index++

	resp.Insert([]byte(s.Character.Name), index) // character name
	index += len(s.Character.Name)

	resp.Insert(utils.IntToBytes(uint64(len(h.message)), 2, true), index) // message length
	index += 2

	resp.Insert([]byte(h.message), index) // message
	index += len(h.message)

	length := index - 4
	resp.SetLength(int16(length)) // packet length

	return &resp
}

func (h *ChatHandler) createShoutMessage(s *database.Socket) *utils.Packet {

	msgString := strings.TrimPrefix(h.message, "/shout ")

	resp := SHOUT_MESSAGE
	length := len(s.Character.Name) + len(msgString) + 6
	resp.SetLength(int16(length)) // packet length

	index := 4
	resp.Insert(utils.IntToBytes(uint64(h.chatType), 2, false), index) // chat type
	index += 2

	resp[index] = byte(len(s.Character.Name)) // character name length
	index++

	resp.Insert([]byte(s.Character.Name), index) // character name
	index += len(s.Character.Name)

	resp[index] = byte(len(msgString)) // message length
	index++

	resp.Insert([]byte(msgString), index) // message
	return &resp
}

func (h *ChatHandler) normalChat(s *database.Socket) ([]byte, error) {

	if _, ok := server.MutedPlayers.Get(s.User.ID); ok {
		msg := "Chatting with this account is prohibited. Please contact our customer support service for more information."
		return messaging.InfoMessage(msg), nil
	}

	resp := h.createChatMessage(s)
	p := &nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Data: *resp, Type: nats.CHAT_NORMAL}
	err := p.Cast()

	return nil, err
}

func (h *ChatHandler) chatWithReceivers(s *database.Socket, msgHandler func(*database.Socket) *utils.Packet) ([]byte, error) {

	if _, ok := server.MutedPlayers.Get(s.User.ID); ok {
		msg := "Chatting with this account is prohibited. Please contact our customer support service for more information."
		return messaging.InfoMessage(msg), nil
	}

	resp := msgHandler(s)

	for _, c := range h.receivers {
		if c == nil || !c.IsOnline {
			if h.chatType == 28930 { // PM
				return messaging.SystemMessage(messaging.WHISPER_FAILED), nil
			}
			continue
		}

		socket := database.GetSocket(c.UserID)
		if socket != nil {
			_, err := socket.Conn.Write(*resp)
			if err != nil {
				log.Println(err)
				return nil, err
			}
		}
	}

	return *resp, nil
}

func MakeAnnouncement(msg string) {
	length := int16(len(msg) + 3)

	resp := ANNOUNCEMENT
	resp.SetLength(length)
	resp[6] = byte(len(msg))
	resp.Insert([]byte(msg), 7)

	p := nats.CastPacket{CastNear: false, Data: resp}
	p.Cast()
}

func (h *ChatHandler) cmdMessage(s *database.Socket, data []byte) ([]byte, error) {

	var (
		err  error
		resp utils.Packet
	)

	text := fmt.Sprintf("Name: "+s.Character.Name+"("+s.Character.UserID+") used command: (%s)", h.message)
	utils.NewLog("logs/cmd_logs.txt", text)
	//ADMIN LOG
	/*f, err := os.OpenFile("admin_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	adminlogger := log.New(f, "", log.LstdFlags)*/
	if parts := strings.Split(h.message, " "); len(parts) > 0 {
		cmd := strings.ToLower(strings.TrimPrefix(parts[0], "/"))
		switch cmd {
		case "auth":
			if s.User.UserType <= 1 {
				return nil, nil
			}

			if s.Character.GMAuthenticated == database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}
			input := parts[1]

			if input != database.GMPassword {
				text := "Name: " + s.Character.Name + "(" + s.User.ID + ") tried to authenticate with wrong password:" + input
				utils.NewLog("logs/gm_auth.txt", text)
				s.Character.GMAuthenticated = ""
			} else if input == database.GMPassword {
				text := "Name: " + s.Character.Name + "(" + s.User.ID + ") Authenticated himself correctly with:" + input
				utils.NewLog("logs/gm_auth.txt", text)
				s.Character.GMAuthenticated = input
			}

		case "buffjog": // Its a trap
			s.Conn.Close()
		case "shout":
			return h.Shout(s, data)
		case "uptime":
			duration := time.Since(database.ServerStart).Truncate(time.Second)
			hours := int(duration.Hours())
			minutes := int(duration.Minutes()) % 60
			seconds := int(duration.Seconds()) % 60
			formattedTime := fmt.Sprintf("Server is up for %02dh:%02dm:%02ds.", hours, minutes, seconds)
			return messaging.InfoMessage(formattedTime), nil
		case "rank":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			rankID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			rankID2, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}
			s.Character.HonorRank = rankID
			s.Character.Update()
			resp := database.CHANGE_RANK
			resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6)
			resp.Insert(utils.IntToBytes(uint64(s.Character.HonorRank), 4, true), 8)
			resp[12] = byte(rankID2)
			statData, _ := s.Character.GetStats()
			resp.Concat(statData)
			s.Write(resp)
		case "addhonorpoints":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			ch := s.Character
			if len(parts) >= 3 {
				c, err := database.FindCharacterByName(parts[1])
				if err != nil {
					return nil, err
				}
				if c == nil {
					return nil, nil
				}
				ch = c
			}
			honorpoints, err := strconv.ParseInt(parts[2], 10, 32)
			if err != nil {
				return nil, err
			}
			ch.Socket.Stats.Honor = int(honorpoints)
			ch.Socket.Stats.Update()

			typeHonor, honorRank := ch.GiveRankByHonorPoints()
			honorresp := database.CHANGE_RANK
			honorresp.Insert(utils.IntToBytes(uint64(ch.PseudoID), 2, true), 6)
			honorresp.Insert(utils.IntToBytes(uint64(honorRank), 4, true), 8)
			honorresp[12] = byte(typeHonor)
			ch.Socket.Write(honorresp)
		case "announce":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			msg := strings.Join(parts[1:], " ")
			MakeAnnouncement(msg)
		case "deleteitemslot":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}
			slotID, err := strconv.ParseInt(parts[1], 10, 16)
			slotMax := int16(slotID)
			if err != nil {
				return nil, err
			}
			ch := s.Character
			if len(parts) >= 3 {
				chr, _ := database.FindCharacterByName(parts[2])
				ch = chr
			}
			r, err := ch.RemoveItem(slotMax)
			if err != nil {
				return nil, err
			}
			ch.Socket.Write(r)
		case "discitem":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			itemID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			quantity := int64(1)
			itemtype := int64(0)
			judgestat := int64(0)
			info := database.Items[itemID]
			if info.Timer > 0 {
				quantity = int64(info.Timer)
			}
			if len(parts) >= 3 {
				quantity, err = strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
			}
			if len(parts) >= 4 {
				itemtype, err = strconv.ParseInt(parts[3], 10, 64)
				if err != nil {
					return nil, err
				}
			}
			if len(parts) >= 5 {
				judgestat, err = strconv.ParseInt(parts[4], 10, 64)
				if err != nil {
					return nil, err
				}
			}
			ch := s.Character
			if s.User.UserType >= server.HGM_USER {
				if len(parts) >= 6 {
					chID, err := strconv.ParseInt(parts[5], 10, 64)
					if err == nil {
						chr, err := database.FindCharacterByID(int(chID))
						if err == nil {
							ch = chr
						}
					}
				}
			}

			item := &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity), ItemType: int16(itemtype), JudgementStat: int64(judgestat)}

			if info.GetType() == database.PET_TYPE {
				petInfo := database.Pets[itemID]
				expInfo := database.PetExps[petInfo.Level-1]
				targetExps := []int{expInfo.ReqExpEvo1, expInfo.ReqExpEvo2, expInfo.ReqExpEvo3, expInfo.ReqExpHt, expInfo.ReqExpDivEvo1, expInfo.ReqExpDivEvo2, expInfo.ReqExpDivEvo3}
				item.Pet = &database.PetSlot{
					Fullness: 100, Loyalty: 100,
					Exp:   uint64(targetExps[petInfo.Evolution-1]),
					HP:    petInfo.BaseHP,
					Level: byte(petInfo.Level),
					Name:  "",
					CHI:   petInfo.BaseChi}
			}

			r, _, err := ch.AddItem(item, -1, false)
			if err != nil {
				return nil, err
			}
			//sItemID := fmt.Sprint(item.ItemID)
			//text := "Name: " + s.Character.Name + "(" + s.Character.UserID + ") give item(" + fmt.Sprint(item.ID) + ") ItemID: " + sItemID + " Quantity: " + fmt.Sprint(item.Quantity)
			//adminlogger.Println(text)
			ch.Socket.Write(*r)
			return nil, nil
		case "item":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			itemID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			quantity := int64(1)
			itemtype := 0
			judgestat := 0
			info := database.Items[itemID]
			if info.Timer > 0 {
				quantity = int64(info.Timer)
			}
			if len(parts) >= 3 {
				quantity, err = strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
			}
			ch := s.Character
			if s.User.UserType >= server.HGM_USER {
				if len(parts) >= 4 {
					chID, err := strconv.ParseInt(parts[3], 10, 64)
					if err == nil {
						chr, err := database.FindCharacterByID(int(chID))
						if err == nil {
							ch = chr
						}
					}
				}
			}

			item := &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity), ItemType: int16(itemtype), JudgementStat: int64(judgestat)}

			if info.GetType() == database.PET_TYPE {
				petInfo := database.Pets[itemID]
				expInfo := database.PetExps[petInfo.Level-1]
				targetExps := []int{expInfo.ReqExpEvo1, expInfo.ReqExpEvo2, expInfo.ReqExpEvo3, expInfo.ReqExpHt, expInfo.ReqExpDivEvo1, expInfo.ReqExpDivEvo2, expInfo.ReqExpDivEvo3}
				item.Pet = &database.PetSlot{
					Fullness: 100, Loyalty: 100,
					Exp:   uint64(targetExps[petInfo.Evolution-1]),
					HP:    petInfo.BaseHP,
					Level: byte(petInfo.Level),
					Name:  "",
					CHI:   petInfo.BaseChi}
			}

			r, _, err := ch.AddItem(item, -1, false)
			if err != nil {
				return nil, err
			}
			//sItemID := fmt.Sprint(item.ItemID)
			//text := "Name: " + s.Character.Name + "(" + s.Character.UserID + ") give item(" + fmt.Sprint(item.ID) + ") ItemID: " + sItemID + " Quantity: " + fmt.Sprint(item.Quantity)
			//adminlogger.Println(text)
			sItemID := fmt.Sprint(item.ItemID)
			text := "Name: " + s.Character.Name + "(" + s.Character.UserID + ") give item(" + fmt.Sprint(item.ID) + ") ItemID: " + sItemID + " Quantity: " + fmt.Sprint(item.Quantity)
			utils.NewLog("logs/admin_log.txt", text)
			ch.Socket.Write(*r)
			return nil, nil

		case "divine":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			data, levelUp := s.Character.AddExp(233332051410)
			if levelUp {
				statData, err := s.Character.GetStats()
				if err == nil && s.Character.Socket != nil {
					resp.Concat(statData)
				}
			}
			if s.Character.Socket != nil {
				resp.Concat(data)
			}
			s.Character.Class = 21
			s.Character.Update()
			gomap, _ := s.Character.ChangeMap(14, nil)
			resp.Concat(gomap)
			x := "261,420"
			coord := s.Character.Teleport(database.ConvertPointToLocation(x))
			resp.Concat(coord)
			return resp, nil
		case "class":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			id, _ := strconv.Atoi(parts[1])
			c, err := database.FindCharacterByID(int(id))
			if err != nil {
				return nil, err
			}

			t, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, err
			}
			c.Class = t
			c.Update()
			resp := utils.Packet{}
			resp = npc.JOB_PROMOTED
			resp[6] = byte(c.Class)
			return resp, nil
		case "gold":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			s.Character.LootGold(uint64(amount))
			h := &GetGoldHandler{}

			return h.Handle(s)
		case "resetalldungeons":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			for key := range database.DungeonEvents {
				char, err := database.FindCharacterByID(database.DungeonEvents[key].CharacterID)
				if err == nil && char.Map == dungeon.DungeonMapID {
					continue
				}
				delete(database.DungeonEvents, key)
			}

			resp = messaging.InfoMessage("dungeon time is reseted")
		case "clearbuffs":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			buffs, err := s.Character.FindAllRelevantBuffs()
			if err != nil {
				return nil, err
			}
			for _, buff := range buffs {
				buff.Duration = 0
				go buff.Update()
			}

			return messaging.InfoMessage(fmt.Sprintf("Buffs are cleared!")), nil
		case "sendcash":
			if s.Character.TradeID != "" {
				return messaging.SystemMessage(10053), nil //Cannot do that while trading
			}
			if len(parts) < 3 {
				return nil, nil
			}
			if s.Character.TradeID != "" {
				return messaging.SystemMessage(10053), nil //Cannot do that while trading
			}

			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			if s.User.NCash <= 0 {
				return nil, nil
			}
			if amount <= 0 {
				return nil, nil
			}
			if s.User.NCash < uint64(amount) {
				return messaging.InfoMessage(fmt.Sprintf("You have not enough Psi-Cash")), nil
			}
			c, err := database.FindCharacterByName(parts[2])
			if err != nil {
				return nil, err
			}
			if c == nil {
				return nil, nil
			}
			user, err := database.FindUserByID(c.UserID)
			if err != nil {
				return nil, err
			} else if user == nil {
				return nil, nil
			}
			s.User.NCash -= uint64(amount)
			s.User.Update()
			user.NCash += uint64(amount)
			user.Update()
			text := fmt.Sprintf("Name: "+s.Character.Name+"("+s.Character.UserID+") transfered nCash(%d) To: "+c.Name, amount)
			utils.NewLog("logs/nc_transfers.txt", text)
			c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("%d Psi-Cash received by %s.", amount, s.Character.Name)))
			return messaging.InfoMessage(fmt.Sprintf("%d Psi-Cash sent to %s.", amount, c.Name)), nil
		case "sendgold":
			if s.Character.TradeID != "" {
				return messaging.SystemMessage(10053), nil //Cannot do that while trading
			}
			if len(parts) < 3 {
				return nil, nil
			}
			if s.Character.TradeID != "" {
				return messaging.SystemMessage(10053), nil //Cannot do that while trading
			}
			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			if amount <= 0 {
				return nil, nil
			}
			if s.Character.Gold <= 0 {
				return nil, nil
			}
			if s.Character.Gold < uint64(amount) {
				return messaging.InfoMessage(fmt.Sprintf("You have not enough Gold")), nil
			}
			c, err := database.FindCharacterByName(parts[2])
			if err != nil {
				return nil, err
			}
			if c == nil {
				return nil, nil
			}

			s.Character.LootGold(uint64(-amount))
			s.Character.Update()
			s.Write(s.Character.GetGold())
			c.LootGold(uint64(amount))
			c.Update()
			c.Socket.Write(c.GetGold())

			text := fmt.Sprintf("Name: "+s.Character.Name+"("+s.Character.UserID+") transfered Gold(%d) To: "+c.Name, amount)
			utils.NewLog("logs/gold_transfers.txt", text)

			resp.Concat(messaging.InfoMessage(fmt.Sprintf("%d Gold sent to %s.", amount, c.Name)))
			c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("%d Gold received by %s.", amount, c.Name)))
			return resp, nil
		case "calculate":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}

			amount, err := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				return nil, err
			}
			bytefloat := utils.FloatToBytes(amount, 4, true)
			resp := utils.Packet(bytefloat)
			resp.Print()
		case "deposit":
			if s.Character.TradeID != "" {
				return messaging.SystemMessage(10053), nil //Cannot do that while trading
			}
			if s.Character.Gold <= 0 {
				return nil, nil
			}
			if len(parts) == 2 {
				a, err := strconv.ParseInt(parts[1], 10, 64)
				amount := uint64(a)
				if err != nil {
					return nil, err
				}
				if s.Character.Gold > uint64(amount) {
					s.Write(messaging.InfoMessage(fmt.Sprintf("Deposit %d Gold", amount)))
					s.Character.LootGold(-amount)
					s.User.BankGold += amount
				}
			} else if len(parts) == 1 {
				if s.Character.Gold <= 0 {
					return nil, nil
				}
				s.Write(messaging.InfoMessage(fmt.Sprintf("Deposit %d Gold", s.Character.Gold)))
				s.User.BankGold += s.Character.Gold
				s.Character.LootGold(-s.Character.Gold)
			}
			go s.User.Update()
			return s.Character.GetGold(), nil
		case "withdraw":
			if s.Character.TradeID != "" {
				return messaging.SystemMessage(10053), nil //Cannot do that while trading
			}
			if len(parts) == 2 {
				a, err := strconv.ParseInt(parts[1], 10, 64)
				amount := uint64(a)
				if err != nil {
					return nil, err
				}
				if s.User.BankGold > uint64(amount) {
					s.Write(messaging.InfoMessage(fmt.Sprintf("Withdraw %d Gold", amount)))
					s.Character.LootGold(amount)
					s.User.BankGold -= amount
				}
			} else if len(parts) == 1 {
				if s.User.BankGold <= 0 {
					return nil, nil
				}
				s.Write(messaging.InfoMessage(fmt.Sprintf("Withdraw %d Gold", s.User.BankGold)))
				s.Character.LootGold(s.User.BankGold)
				s.User.BankGold = 0
			}
			go s.User.Update()
			return s.Character.GetGold(), nil
		case "showdiff":
			c, err := database.FindCharacterByID(s.Character.ID)
			if err != nil {
				return nil, err
			}
			if !c.ShowStats {
				c.ShowStats = true
				resp.Concat(messaging.InfoMessage("Show Stat Updates is ON"))
			} else {
				c.ShowStats = false
				resp.Concat(messaging.InfoMessage("Show Stat Updates is OFF"))
			}
		case "buffs":
			buffs, err := s.Character.FindAllRelevantBuffs()
			var startTime int64
			var timeNow int64

			if err != nil {
				return nil, err
			}
			resp.Concat(messaging.InfoMessage("Active Buffs: "))
			for _, buff := range buffs {
				if buff.IsServerEpoch {
					timeNow = database.GetServerEpoch()
				} else {
					timeNow = s.Character.Epoch
				}

				duration := buff.Duration
				startTime = buff.StartedAt
				remainingTimeSeconds := startTime + duration - timeNow

				remainingHours := remainingTimeSeconds / 3600
				remainingMinutes := (remainingTimeSeconds % 3600) / 60

				remainingTimeHour := fmt.Sprintf("%d", remainingHours)
				remainingTimeMinutes := fmt.Sprintf("%d", remainingMinutes)

				resp.Concat(messaging.InfoMessage(fmt.Sprintf("%s will last for %sh:%sm.", buff.Name, remainingTimeHour, remainingTimeMinutes)))
			}
		case "starttrivia":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			database.TriviaEventStarted = true
			database.TriviaCanJoin = true
			newEvent := &database.ActiveTrivia{ActiveTriviaID: 1}
			database.ActiveEventTrivia[int(1)] = newEvent
			database.StartInTriviaTimer()
			return resp, nil
		case "endtrivia":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			itemID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			quantity := int64(1)
			if len(parts) >= 3 {
				quantity, err = strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
			}
			database.EndTriviaEvent(itemID, quantity)
		case "join":
			if database.TriviaCanJoin && !s.Character.IsInTraviaEvent {
				s.Character.IsInTraviaEvent = true
				database.ActiveEventTrivia[int(1)].AllPlayers = append(database.ActiveEventTrivia[int(1)].AllPlayers, s.Character)
				return s.Character.ChangeMap(int16(42), nil)
			}
		case "triviaonline":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			resp := utils.Packet{}
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("Stayed Players: %d Names: ", len(database.ActiveEventTrivia[int(1)].StayedPlayers))))
			for _, chars := range database.ActiveEventTrivia[int(1)].StayedPlayers {
				if chars == nil {
					continue
				}
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("%s.", chars.Name)))
			}
			s.Write(resp)
		case "nextquestion":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			itemID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			quantity := int64(1)
			if len(parts) >= 3 {
				quantity, err = strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
			}
			database.StartTriviaEvent(itemID, quantity)
		case "upgrade":
			if s.User.UserType < server.GM_USER || len(parts) < 3 {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			slots, err := s.Character.InventorySlots()
			if err != nil {
				return nil, err
			}

			slotID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			code, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}

			count := int64(1)
			if len(parts) > 3 {
				count, err = strconv.ParseInt(parts[3], 10, 64)
				if err != nil {
					return nil, err
				}
			}

			codes := []byte{}
			for i := 0; i < int(count); i++ {
				codes = append(codes, byte(code))
			}

			item := slots[slotID]
			return item.Upgrade(int16(slotID), codes...), nil

		case "exp":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			ch := s.Character
			if s.User.UserType >= server.HGM_USER {
				if len(parts) > 2 {
					chID, err := strconv.ParseInt(parts[2], 10, 64)
					if err == nil {
						chr, err := database.FindCharacterByID(int(chID))
						if err == nil {
							ch = chr
						}
					}
				}
			}
			data, levelUp := ch.AddExp(amount)
			if levelUp {
				statData, err := ch.GetStats()
				if err == nil && ch.Socket != nil {
					ch.Socket.Write(statData)
				}
			}

			if ch.Socket != nil {
				ch.Socket.Write(data)
			}

			return nil, nil
		case "petexp":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			amount, err := strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			slots, err := s.Character.InventorySlots()
			if err != nil {
				log.Println(err)
				return nil, nil
			}

			petSlot := slots[0x0A]
			pet := petSlot.Pet
			if pet == nil || petSlot.ItemID == 0 || !pet.IsOnline {
				return nil, nil
			}
			pet.AddExp(s.Character, amount)

		case "home":
			s.Respawn()
		case "map":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			mapID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			if len(parts) >= 3 {
				c, err := database.FindCharacterByName(parts[2])
				if err != nil {
					return nil, err
				}

				data, err := c.ChangeMap(int16(mapID), nil)
				if err != nil {
					return nil, err
				}

				database.GetSocket(c.UserID).Write(data)
				return nil, nil
			}

			return s.Character.ChangeMap(int16(mapID), nil)
		case "cash":

			var text string
			psi_community := 21
			psi_support := 28
			psi_titan := 42
			psi_ultimate := 56
			var finalAmount uint64
			var event_bonus int64

			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			User, err := database.FindUserByAnything(parts[2])
			if err != nil {
				return nil, err
			} else if User == nil {
				return nil, nil
			}

			if len(parts) == 4 {
				event_bonus, err = strconv.ParseInt(parts[3], 10, 64)
				if err != nil {
					return nil, err
				}
			} else {
				event_bonus = 0
				finalAmount = uint64(amount)
				goto SKIPPATREONANDBONUS
			}

			switch User.PatreonTier {
			case 1:
				finalAmount = uint64(float64(amount) * ((float64(psi_community)+float64(event_bonus))/100 + 1))
			case 2:
				finalAmount = uint64(float64(amount) * ((float64(psi_support)+float64(event_bonus))/100 + 1))
			case 3:
				finalAmount = uint64(float64(amount) * ((float64(psi_titan)+float64(event_bonus))/100 + 1))
			case 4:
				finalAmount = uint64(float64(amount) * ((float64(psi_ultimate)+float64(event_bonus))/100 + 1))
			default:
				finalAmount = uint64(float64(amount) * (float64(event_bonus)/100 + 1))
			}
			User.NCash += uint64(finalAmount)
			User.Update()
			text = fmt.Sprintf("Name: "+s.Character.Name+"("+s.Character.UserID+") give cash(%d) To: "+s.Character.Name, amount)
			utils.NewLog("logs/admin_log.txt", text)

			return messaging.InfoMessage(fmt.Sprintf("%d nCash to %s. With Patreon-Tier %d and event-bonus of %d is %d total.", amount, User.Username, User.PatreonTier, event_bonus, int64(finalAmount))), nil

		SKIPPATREONANDBONUS:
			User.NCash += uint64(finalAmount)
			User.Update()
			return messaging.InfoMessage(fmt.Sprintf("%d nCash to %s. Add a second number to add percentual bonus", int64(finalAmount), User.Username)), nil
		case "exprate":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			buffs, err := s.Character.FindAllRelevantBuffs()
			if err != nil {
				return nil, err
			}
			for _, buff := range buffs {
				if buff.ID == 21 {
					buff.Delete()
					break
				}
			}

			var name string
			var time int64
			var hour int64
			if len(parts) > 2 {

				if am, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					if am > 200 {
						s.Character.Socket.Write(messaging.InfoMessage(fmt.Sprintf("Are you sure you want to add a +%d%% EXP Bonus? 1 = +1%%.", am)))
						return nil, nil
					}
					percent := fmt.Sprintf("%d%%", am)
					name = fmt.Sprintf("EXP Event %s", percent)
				}
				hour, err = strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
				if hour > 200 {
					s.Character.Socket.Write(messaging.InfoMessage(fmt.Sprintf("Are you sure you want to schedule the event for %d hours? 1 = 1 Hour.", hour)))
					return nil, nil
				}
				time = hour * 60 * 60
			}
			s.Character.HandleBuffs()
			buff, err := s.Character.CraftBuff(name, 21, 0, time, true)
			if err != nil {
				return nil, err
			}
			buff.Create()
			buff.Update()
			onlineUsers, err := database.FindOnlineCharacters()
			if err != nil {
				return nil, err
			}
			for _, user := range onlineUsers {
				user.GetStats()
			}
			return messaging.InfoMessage(fmt.Sprintf("%s has been applied for %d hours.", name, hour)), nil
		case "dungeontime":
			if database.DungeonEvents[s.Character.ID] == nil {
				return messaging.InfoMessage("You can enter to dungeon"), nil
			}
			if time.Since(database.DungeonEvents[s.Character.ID].LastStartedTime.Time.Add(time.Hour*time.Duration(1))) >= 0 {
				return messaging.InfoMessage("You can enter to dungeon"), nil
			}
			durr := time.Since(database.DungeonEvents[s.Character.ID].LastStartedTime.Time.Add(time.Hour * time.Duration(1)))
			// durr to hour min and sec write in console
			return messaging.InfoMessage("You can enter dungeon in " + utils.DurationToString(durr)), nil
		case "droprate":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}
			if len(parts) > 2 {
				if s, err := strconv.ParseFloat(parts[1], 64); err == nil {
					database.DROP_RATE = s
				}
				minute, err := strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
				time.AfterFunc(time.Duration(minute)*time.Minute, func() {
					database.DROP_RATE = database.DEFAULT_DROP_RATE
				})
			}
			return messaging.InfoMessage(fmt.Sprintf("Drop Rate now: %f", database.DROP_RATE)), nil
		case "war":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			if len(parts) < 3 {
				return nil, nil
			}
			time, _ := strconv.ParseInt(parts[1], 10, 32)
			wartype, _ := strconv.ParseInt(parts[2], 10, 32)
			if !database.CanJoinWar && !database.WarStarted {
				database.CanJoinWar = true
				database.WarType = wartype
				database.StartWarTimer(int(time))
			}
		case "warplayer":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			number, _ := strconv.ParseInt(parts[1], 10, 32)
			database.WarRequirePlayers = int(number)
		case "debugdungeon":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			party := database.FindParty(s.Character)
			if party == nil {
				return nil, nil
			}
			if party.Leader == s.Character {
				if s.Character.PartyID == "" {
					return nil, nil
				}
				dungeons := funk.Values(database.GetActiveDungeons()).([]*database.Dungeon)
				filter := funk.Filter(dungeons, func(d *database.Dungeon) bool {
					return d.IsLoading
				}).([]*database.Dungeon)
				if database.DungeonLoading && len(filter) > 0 {
					s.Write(messaging.InfoMessage("Try again in a few seconds."))
					return nil, nil
				}
				if len(party.Members) < 1 { //UNTIL TEST 2 TODO: WRITE TO 4
					s.Write(messaging.InfoMessage("We do not have the right number of people in the group."))
					return nil, nil
				}
				leaderCanJoin := false
				if database.DungeonEvents[party.Leader.ID] == nil { //NEVER JOINED!!!
					leaderCanJoin = true
				} else if time.Since(database.DungeonEvents[party.Leader.ID].LastStartedTime.Time.Add(time.Hour*time.Duration(1))) >= 0 {
					leaderCanJoin = true
				}
				if !leaderCanJoin {
					return nil, nil
				}
				//GET TYPE OF PARTY LEADER
				dungeonType := 0 //RESERVED
				minLevel, maxLevel := 0, 0
				if party.Leader.Level <= 100 { //NON-DIVINE
					dungeonType = 1
					minLevel = 50
					maxLevel = 100
				} else if party.Leader.Level >= 101 && party.Leader.Level <= 200 { //DIVINE
					dungeonType = 2
					minLevel = 150
					maxLevel = 200
				} else if party.Leader.Level >= 201 && party.Leader.Level <= 300 { //DARKNESS
					dungeonType = 3
					minLevel = 235
					maxLevel = 300
				}
				_ = dungeonType //RESERVED
				allChars := funk.Values(party.Members)
				playersCount := len(funk.Filter(allChars, func(char *database.PartyMember) bool {
					canJoin := false
					if database.DungeonEvents[char.ID] == nil { //NEVER JOINED!!!
						canJoin = true
					} else if time.Since(database.DungeonEvents[char.ID].LastStartedTime.Time.Add(time.Hour*time.Duration(1))) >= 0 {
						canJoin = true
					}
					return char.Character.Level >= minLevel && char.Character.Level <= maxLevel && char.IsOnline && canJoin && char.Accepted
				}).([]*database.PartyMember))
				if playersCount != len(party.Members) {
					s.Write(messaging.InfoMessage("Someone in the group does not meet the criteria."))
					return nil, nil
				}
				database.DungeonLoading = true
				dungeon.StartDungeon(s)
			}
		case "debugarena":
			party := database.FindParty(s.Character)
			if party == nil {
				return nil, nil
			} else {
				if party.Leader == s.Character {
					if s.Character.PartyID == "" || database.FindPlayerInArena(s.Character) {
						return nil, nil
					}
					if party.ArenaFounded {
						s.Write(messaging.InfoMessage("You already have a party in the arena."))
						return nil, nil
					}

					if len(party.Members) < 1 { //UNTIL TEST 2 TODO: WRITE TO 4
						s.Write(messaging.InfoMessage("We do not have the right number of people in the group."))
						return nil, nil
					}

					//GET TYPE OF PARTY LEADER
					arenaType := database.GetArenaTypeByLevelType(s.Character.Level) //RESERVED
					if arenaType < 0 {
						s.Write(messaging.InfoMessage("Someone in the group does not meet the criteria."))
						return nil, nil
					}
					allChars := funk.Values(party.Members)
					playersCount := len(funk.Filter(allChars, func(char *database.PartyMember) bool {
						return database.GetArenaTypeByLevelType(char.Level) == database.GetArenaTypeByLevelType(party.Leader.Level) && char.IsOnline && char.Accepted
					}).([]*database.PartyMember))
					if playersCount != len(party.Members) {
						s.Write(messaging.InfoMessage("Someone in the group does not meet the criteria."))
						return nil, nil
					}
					s.Write(messaging.InfoMessage("Your team is in lobby."))
					database.JoinToArenaLobby(s.Character, arenaType, false)
					database.TryToCreateSession()
				}
			}
		case "acceptarena":
			if s.Character.Socket == nil || !s.Character.IsOnline {
				return nil, nil
			}
			party := database.FindParty(s.Character)
			if party == nil {
				return nil, nil
			}
			if party.ArenaFounded {
				if party.Leader == s.Character {
					s.Write(messaging.InfoMessage("You(Leader) have accepted the arena. Waiting for the other players to accept."))
					party.PartyLeaderAcceptArena = true
				} else {
					for _, arena := range party.GetMembers() {
						if arena.Character == s.Character {
							s.Write(messaging.InfoMessage("You have accepted the arena. Waiting for the other players to accept."))
							arena.AcceptedArena = true
						}
					}
				}
			}
		case "acceptwar":
			/*if funk.Contains(database.ActiveWars[1].ZuhangPlayers, s.Character) || funk.Contains(database.ActiveWars[1].ShaoPlayers, s.Character){
			fmt.Println("Accepted")
			s.Character.IsAcceptedWar = true
			}*/
		case "playerskillreset":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}
			skillresp, err := ResetPlayerSkillBook(c)
			if err != nil {
				fmt.Println(fmt.Sprintf("RESET BOOK ERROR: %s", err.Error()))
			}
			c.Socket.Write(skillresp)
		case "guildwar":
			if len(parts) < 2 {
				return nil, nil
			}
			if s.Character.GuildID == -1 {
				return nil, nil
			}
			guild, err := database.FindGuildByID(s.Character.GuildID)
			if err != nil {
				return nil, err
			}
			members, _ := guild.GetMembers()
			for _, m := range members {
				if m.Role != database.GROLE_LEADER {
					return nil, nil
				}
			}
			enemyguild, err := database.FindGuildByName(parts[1])
			if err != nil {
				return nil, err
			}

			startGuildWar(guild, enemyguild)
		case "addguild":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			guildid, err := strconv.ParseInt(parts[1], 10, 32)
			if err != nil {
				return nil, err
			}
			ch := s.Character
			if len(parts) >= 3 {
				c, err := database.FindCharacterByName(parts[2])
				if err != nil {
					return nil, err
				}
				ch = c
			}
			removeid := -1
			if int(guildid) == removeid {
				guild, err := database.FindGuildByID(ch.GuildID)
				err = guild.RemoveMember(ch.ID)
				if err != nil {
					return nil, err
				}
				go guild.Update()
				ch.GuildID = int(guildid)
			} else {
				guild, err := database.FindGuildByID(int(guildid))
				if err != nil {
					return nil, err
				}
				guild.AddMember(&database.GuildMember{ID: ch.ID, Role: database.GROLE_MEMBER})
				go guild.Update()
				ch.GuildID = int(guildid)
			}
			spawnData, err := s.Character.SpawnCharacter()
			if err == nil {
				p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.PLAYER_SPAWN, Data: spawnData}
				p.Cast()
				ch.Socket.Conn.Write(spawnData)
			}
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("Player new guild id: %d", ch.GuildID)))
			return resp, nil
		case "findguild":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}
			guild, err := database.FindGuildByName(parts[1])
			if err != nil {
				return nil, err
			}
			return messaging.InfoMessage(fmt.Sprintf("Clan ID: %d", guild.ID)), nil
		case "dungeon":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			data, err := s.Character.ChangeMap(243, nil)
			if err != nil {
				return nil, err
			}
			resp.Concat(data)
			x := "377,246"
			coord := s.Character.Teleport(database.ConvertPointToLocation(x))
			resp.Concat(coord)
			return resp, nil
		case "refresh":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			command := parts[1]
			switch command {
			case "items":
				database.RefreshAllItems()
			case "scripts":
				database.RefreshScripts()
			case "htshop":
				database.RefreshHTItems()
			case "buffinf":
				database.RefreshBuffInfections()
			case "advancedfusions":
				database.RefreshAdvancedFusions()
			case "gamblings":
				database.RefreshGamblingItems()
			case "productions":
				database.RefreshProductions()
			case "drops":
				database.RefreshAllDrops()
			case "shopitems":
				database.GetAllShopItems()
			case "shoptable":
				database.RefreshAllShops()
			case "events":
				database.GetAllEvents()
			case "exp":
				database.GetExps()
			case "skills":
				database.RefreshSkillInfos()
			case "npcs":
				database.RefreshAllNPCPos()
			case "quests":
				database.RefreshAllQuests()
			case "trivia":
				database.RefreshQuestionsItem()
			case "all":
				callBacks := []func() error{database.RefreshScripts, database.RefreshHTItems, database.RefreshBuffInfections, database.RefreshAdvancedFusions,
					database.RefreshGamblingItems, database.RefreshAllShops, database.RefreshAllDrops, database.RefreshProductions, database.RefreshQuestionsItem, database.GetExps, database.RefreshSkillInfos, database.RefreshAllQuests}
				for _, cb := range callBacks {
					if err := cb(); err != nil {
						fmt.Println("Error: ", err)
					}
				}
			}
		case "addbuff":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			number, _ := strconv.ParseInt(parts[1], 10, 32)
			buffinfo := database.BuffInfections[int(number)]
			buff := &database.Buff{ID: int(number), CharacterID: s.Character.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: s.Character.Epoch, Duration: 10, CanExpire: true}
			err := buff.Create()
			if err != nil {
				return nil, err
			}
		case "test":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			data, err := hex.DecodeString(parts[1])
			if err != nil {
				return nil, nil
			}
			log.Print(data)
			s.Character.Socket.Write(data)

		case "number":
			if len(parts) < 2 && s.Character.GeneratedNumber != 2 {
				return nil, nil
			}
			number, err := strconv.ParseInt(parts[1], 10, 32)
			if err != nil {
				return nil, err
			}
			if int(number) == s.Character.GeneratedNumber {
				s.Conn.Write(messaging.InfoMessage(fmt.Sprintf("You guessed right, Show the boss your power.")))
				s.Character.DungeonLevel++
			} else {
				s.Conn.Write(messaging.InfoMessage(fmt.Sprintf("You guessed poorly, survive & slay again!")))
				dungeon.MobsCreate([]int{40522}, s.User.ConnectedServer)
				s.Character.CanTip = 3
			}
		case "spawnmob":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			posId, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			npcPos := database.NPCPos[int(posId)]
			npc, ok := database.NPCs[npcPos.NPCID]
			if !ok {
				return nil, nil
			}
			//npcPos := &database.NpcPosition{ID: len(database.NPCPos), NPCID: int(action), MapID: s.Character.Map, Rotation: 0, Attackable: true, IsNPC: false, RespawnTime: 30, Count: 30, MinLocation: "120,120", MaxLocation: "150,150"}
			//newPos := database.NPCPos[int(action)]
			database.NPCPos = append(database.NPCPos, npcPos)
			database.AIMutex.Lock()
			allAI := database.AIs
			database.AIMutex.Unlock()
			r := funk.Map(allAI, func(k int, v *database.AI) int {
				return v.ID
			})
			maxAIID := funk.MaxInt(r.([]int)).(int)
			newai := &database.AI{ID: maxAIID + 1, HP: npc.MaxHp, Map: 229, PosID: npcPos.ID, RunningSpeed: 10, Server: 1, WalkingSpeed: 5, Once: true}
			newai.OnSightPlayers = make(map[int]interface{})
			coordinate := database.ConvertPointToLocation(s.Character.Coordinate)
			randomLocX := randFloats(coordinate.X, coordinate.X+30)
			randomLocY := randFloats(coordinate.Y, coordinate.Y+30)
			loc := utils.Location{X: randomLocX, Y: randomLocY}
			npcPos.MinLocation = fmt.Sprintf("%.1f,%.1f", randomLocX, randomLocY)
			maxX := randomLocX + 50
			maxY := randomLocY + 50
			npcPos.MaxLocation = fmt.Sprintf("%.1f,%.1f", maxX, maxY)
			newai.Coordinate = loc.String()
			fmt.Println(newai.Coordinate)
			newai.Handler = newai.AIHandler

			database.AIsByMap[newai.Server][newai.Map] = append(database.AIsByMap[newai.Server][newai.Map], newai)
			database.AIs[newai.ID] = newai
			server.GenerateIDForAI(newai)
			//ai.Init()
			if newai.WalkingSpeed > 0 {
				go newai.Handler()
			}
		case "float":
			number, err := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				return nil, err
			}
			posId, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}
			fmt.Println(utils.FloatToBytes(number, byte(posId), true))
		case "addmobs":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			npcId, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			count, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}
			mapID := s.Character.Map
			cmdSpawnMobs(int(count), int(npcId), int(mapID))
		case "event":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			cmdEvents(parts[1])
		case "eventprob":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			count, _ := strconv.ParseInt(parts[1], 10, 64)
			database.EventProb = int(count)
			s.Conn.Write(messaging.InfoMessage(fmt.Sprintf("Succesfully change the new value is %d !", count)))
		case "resetallmobs":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			for _, npcPos := range database.NPCPos {
				npc, ok := database.NPCs[npcPos.NPCID]
				if !ok {
					fmt.Println("Error")
					continue
				}
				for k := 1; k <= 4; k++ {
					for i := 0; i < int(npcPos.Count); i++ {
						if npc.ID == 0 || npcPos.IsNPC || !ok || !npcPos.Attackable {
							continue
						}
						minCoordinate := database.ConvertPointToLocation(npcPos.MinLocation)
						maxCoordinate := database.ConvertPointToLocation(npcPos.MaxLocation)
						targetX := utils.RandFloat(minCoordinate.X, maxCoordinate.X)
						targetY := utils.RandFloat(minCoordinate.Y, maxCoordinate.Y)
						target := utils.Location{X: targetX, Y: targetY}
						newai := &database.AI{ID: len(database.AIs), HP: npc.MaxHp, Map: npcPos.MapID, PosID: npcPos.ID, RunningSpeed: float64(npc.RunningSpeed), Server: k, WalkingSpeed: float64(npc.WalkingSpeed), Faction: npcPos.Faction, CanAttack: true}
						server.GenerateIDForAI(newai)
						newai.OnSightPlayers = make(map[int]interface{})
						newai.Coordinate = target.String()
						uploadAI := &database.AI{ID: len(database.AIs), PosID: npcPos.ID, Server: k, Faction: npcPos.Faction, Map: npcPos.MapID, Coordinate: newai.Coordinate, WalkingSpeed: float64(npc.WalkingSpeed), RunningSpeed: float64(npc.RunningSpeed), CanAttack: true}
						//fmt.Println(newai.Coordinate)
						aierr := uploadAI.Create()
						if aierr != nil {
							fmt.Println("Error: %s", aierr)
						}
						newai.Handler = newai.AIHandler
						database.AIsByMap[newai.Server][npcPos.MapID] = append(database.AIsByMap[newai.Server][npcPos.MapID], newai)
						database.AIs[newai.ID] = newai
						fmt.Println("New mob created", len(database.AIs))
						go newai.Handler()
					}
				}
			}
			fmt.Println("Finished")
			return nil, nil
		case "resetmobs":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			posId, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			npcPos := database.NPCPos[int(posId)]
			npc, ok := database.NPCs[npcPos.NPCID]
			if !ok {
				fmt.Println("Error")
			}
			for i := 0; i < int(npcPos.Count); i++ {
				if npc.ID == 0 {
					continue
				}

				newai := &database.AI{ID: len(database.AIs), HP: npc.MaxHp, Map: npcPos.MapID, PosID: npcPos.ID, RunningSpeed: 10, Server: 1, WalkingSpeed: 5, Once: true}
				server.GenerateIDForAI(newai)
				newai.OnSightPlayers = make(map[int]interface{})

				minLoc := database.ConvertPointToLocation(npcPos.MinLocation)
				maxLoc := database.ConvertPointToLocation(npcPos.MaxLocation)
				loc := utils.Location{X: utils.RandFloat(minLoc.X, maxLoc.X), Y: utils.RandFloat(minLoc.Y, maxLoc.Y)}
				newai.Coordinate = loc.String()
				fmt.Println(newai.Coordinate)
				newai.Handler = newai.AIHandler
				database.AIsByMap[newai.Server][npcPos.MapID] = append(database.AIsByMap[newai.Server][npcPos.MapID], newai)
				database.AIs[newai.ID] = newai
				fmt.Println("New mob created", len(database.AIs))
				newai.Create()
				go newai.Handler()
			}
			fmt.Println("Finished")
			return nil, nil
		case "mob":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			posId, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			npcPos := database.NPCPos[int(posId)]
			npc, ok := database.NPCs[npcPos.NPCID]
			if !ok {
				return nil, nil
			}
			database.AIMutex.Lock()
			allAI := database.AIs
			database.AIMutex.Unlock()
			r := funk.Map(allAI, func(k int, v *database.AI) int {
				return v.ID
			})
			maxAIID := funk.MaxInt(r.([]int)).(int)
			ai := &database.AI{ID: maxAIID + 1, HP: npc.MaxHp, Map: npcPos.MapID, PosID: npcPos.ID, RunningSpeed: 10, Server: 1, WalkingSpeed: 5, Once: true}
			server.GenerateIDForAI(ai)
			ai.OnSightPlayers = make(map[int]interface{})

			minLoc := database.ConvertPointToLocation(npcPos.MinLocation)
			maxLoc := database.ConvertPointToLocation(npcPos.MaxLocation)
			loc := utils.Location{X: utils.RandFloat(minLoc.X, maxLoc.X), Y: utils.RandFloat(minLoc.Y, maxLoc.Y)}

			ai.Coordinate = loc.String()
			fmt.Println(ai.Coordinate)
			ai.Handler = ai.AIHandler
			go ai.Handler()

			MakeAnnouncement(fmt.Sprintf("%s has been roaring.", npc.Name))

			database.AIsByMap[ai.Server][npcPos.MapID] = append(database.AIsByMap[ai.Server][npcPos.MapID], ai)
			database.AIs[ai.ID] = ai

		case "droplog":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("Today farmed relics: %d ea", len(database.RelicsLog))))
			for _, c := range database.RelicsLog {
				hour, min, sec := c.DropTime.Time.Hour(), c.DropTime.Time.Minute(), c.DropTime.Time.Second()
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("Character ID: %d dropped item id: %d at %d:%d:%d ", c.CharID, c.ItemID, hour, min, sec)))
			}

		case "relic":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			itemID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			ch := s.Character
			if len(parts) >= 3 {
				chID, err := strconv.ParseInt(parts[2], 10, 64)
				if err == nil {
					chr, err := database.FindCharacterByID(int(chID))
					if err == nil {
						ch = chr
					}
				}
			}

			slot, err := ch.FindFreeSlot()
			if err != nil {
				return nil, nil
			}

			itemData, _, _ := ch.AddItem(&database.InventorySlot{ItemID: itemID, Quantity: 1}, slot, true)
			if itemData != nil {
				ch.Socket.Write(*itemData)

				relicDrop := ch.RelicDrop(int64(itemID))
				p := nats.CastPacket{CastNear: false, Data: relicDrop, Type: nats.ITEM_DROP}
				p.Cast()
			}

		case "main":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			countMaintenance(60)
		case "restart":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			countRestart(60)
		case "mobbuff":
			/*now := time.Now()
			secs := now.Unix()
			infection := database.BuffInfections[257]
			ai,_ := database.GetFromRegister(s.User.ConnectedServer, s.Character.Map, uint16(s.Character.Selection)).(*database.AI)
			buff := &database.AiBuff{ID: 257, AiID: int(ai.PseudoID), Name: infection.Name, HPRecoveryRate: 100, StartedAt: secs,CharacterID: s.Character.ID, Duration: int64(1) * 10}
			err = buff.Create()
			if err != nil {
				fmt.Println(fmt.Sprintf("Error: %s", err.Error()))
				return nil, err
			}
			s.Character.DealPoisonDamageToAI(ai)*/
		case "ban":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			userID := parts[1]
			user, err := database.FindUserByAnything(userID)
			if err != nil {
				return nil, err
			} else if user == nil {
				return nil, nil
			}

			hours, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}

			user.UserType = 0
			user.DisabledUntil = null.NewTime(time.Now().Add(time.Hour*time.Duration(hours)), true)
			user.Update()

			database.GetSocket(userID).Conn.Close()
		case "resetdungeon":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			char, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}
			if database.DungeonEvents[char.ID] != nil {
				database.DungeonEvents[char.ID].Delete()
				delete(database.DungeonEvents, char.ID)
			}

			resp = messaging.InfoMessage(fmt.Sprintf("%s dungeon time is reseted ", char.Name))
		case "mute":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			dumb, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			server.MutedPlayers.Set(dumb.UserID, struct{}{})

		case "unmute":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			dumb, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			server.MutedPlayers.Remove(dumb.UserID)

		case "uid":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			} else if c == nil {
				return nil, nil
			}

			resp = messaging.InfoMessage(c.UserID)

		case "patreon":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			user, err := database.FindUserByName(parts[1])
			if err != nil {
				return nil, err
			} else if user == nil {
				return nil, nil
			}
			resp = messaging.InfoMessage(fmt.Sprintf("User %s has Patreon-Tier: %d ", user.Username, user.PatreonTier))

		case "uuid":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			user, err := database.FindUserByName(parts[1])
			if err != nil {
				return nil, err
			} else if user == nil {
				return nil, nil
			}

			resp = messaging.InfoMessage(user.ID)
		case "visibility":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}
			if parts[1] == "1" {
				data := database.BUFF_INFECTION
				data.Insert(utils.IntToBytes(uint64(70), 4, true), 6)     // infection id
				data.Insert(utils.IntToBytes(uint64(99999), 4, true), 11) // buff remaining time

				s.Conn.Write(data)
			} else {
				r := database.BUFF_EXPIRED
				r.Insert(utils.IntToBytes(uint64(70), 4, true), 6) // buff infection id
				r.Concat(data)

				s.Conn.Write(r)
			}
			s.Character.Invisible = parts[1] == "1"

		case "kick":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			dumb, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}
			database.GetSocket(dumb.UserID).Conn.Close()
		case "tp":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			x, err := strconv.ParseFloat(parts[1], 10)
			if err != nil {
				return nil, err
			}

			y, err := strconv.ParseFloat(parts[2], 10)
			if err != nil {
				return nil, err
			}

			return s.Character.Teleport(database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))), nil

		case "tpp":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}
			mapID, _ := s.Character.ChangeMap(c.Map, database.ConvertPointToLocation(c.Coordinate))
			s.Conn.Write(mapID)
			if c.Socket.User.ConnectedServer != s.User.ConnectedServer {
				s.User.ConnectedServer = c.Socket.User.ConnectedServer
			}
			return nil, nil
		case "summon":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}
			mapID, _ := c.ChangeMap(s.Character.Map, database.ConvertPointToLocation(s.Character.Coordinate))
			c.Socket.Write(mapID)
		case "speed":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			speed, err := strconv.ParseFloat(parts[1], 10)
			if err != nil {
				return nil, err
			}

			s.Character.RunningSpeed = speed

		case "online":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			characters, err := database.FindOnlineCharacters()
			if err != nil {
				return nil, err
			}

			online := funk.Values(characters).([]*database.Character)
			sort.Slice(online, func(i, j int) bool {
				return online[i].Name < online[j].Name
			})

			resp.Concat(messaging.InfoMessage(fmt.Sprintf("%d player(s) online.", len(characters))))

			for _, c := range online {
				u, _ := database.FindUserByID(c.UserID)
				if u == nil {
					continue
				}

				resp.Concat(messaging.InfoMessage(fmt.Sprintf("%s is in map %d (Dragon%d) at %s.", c.Name, c.Map, u.ConnectedServer, c.Coordinate)))
			}
		case "npc":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}
			npcID, _ := strconv.Atoi(parts[1])
			actID, _ := strconv.Atoi(parts[2])
			resp := npc.GetNPCMenu(npcID, 999993, 0, []int{actID})
			return resp, nil
		case "name":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			id, _ := strconv.Atoi(parts[1])
			c, err := database.FindCharacterByID(int(id))
			if err != nil {
				return nil, err
			}

			c2, err := database.FindCharacterByName(parts[2])
			if err != nil {
				return nil, err
			} else if c2 != nil {
				return nil, nil
			}

			c.Name = parts[2]
			c.Update()

		case "role":
			/*if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			id, _ := strconv.Atoi(parts[1])
			c, err := database.FindCharacterByID(int(id))
			if err != nil {
				return nil, err
			}

			user, err := database.FindUserByID(c.UserID)
			if err != nil {
				return nil, err
			}

			role, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, err
			}

			user.UserType = int8(role)
			user.Update()*/
		case "skillpoint":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			character, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}
			num, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, err
			}
			character.Socket.Skills.SkillPoints += num
			s.Conn.Write(character.GetExpAndSkillPts())
		case "type":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			go s.Character.AuthStatus(10)
			if s.Character.GMAuthenticated != database.GMPassword {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			id, _ := strconv.Atoi(parts[1])
			c, err := database.FindCharacterByID(int(id))
			if err != nil {
				return nil, err
			}

			t, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, err
			}

			c.Type = t
			c.Update()
		}

	}

	return resp, err
}
func ResetPlayerSkillBook(c *database.Character) ([]byte, error) {
	resp := utils.Packet{}
	var BookIDs []int64
	skills, err := database.FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}
	for j := 0; j < 5; j++ {
		if skillSlots.Slots[j].BookID != 0 {
			fmt.Println(fmt.Sprintf("BOOKID: %d"), skillSlots.Slots[j].BookID)
			BookIDs = append(BookIDs, skillSlots.Slots[j].BookID)
		}
	}
	//i := -1
	for k := 0; k < len(BookIDs); k++ {
		//fmt.Println(fmt.Sprintf("BOOKID: %d"), BookIDs[k])
		skillInfos := database.SkillInfosByBook[BookIDs[k]]
		set := &database.SkillSet{BookID: BookIDs[k]}
		c := 0
		for i := 1; i <= 24; i++ { // there should be 24 skills with empty ones
			fmt.Println(fmt.Sprintf("SkillLen: %d"), len(skillInfos))
			if len(skillInfos) <= c {
				set.Skills = append(set.Skills, &database.SkillTuple{SkillID: 0, Plus: 0})
			} else if si := skillInfos[c]; si.Slot == i {
				fmt.Println(fmt.Sprintf("skillID: %d"), si.ID)
				tuple := &database.SkillTuple{SkillID: si.ID, Plus: 0}
				set.Skills = append(set.Skills, tuple)
				c++
			} else {
				set.Skills = append(set.Skills, &database.SkillTuple{SkillID: 0, Plus: 0})
			}
		}
		skillSlots.Slots[k] = set
		skills.SetSkills(skillSlots)
		go skills.Update()
	}

	skillsData, err := skills.GetSkillsData()
	if err != nil {
		fmt.Println("SkillError: %s", err.Error())
		return nil, err
	}
	resp.Concat(skillsData)
	return resp, nil
}
func countMaintenance(cd int) {
	msg := fmt.Sprintf("There will be maintenance after %d seconds. Please log out in order to prevent any inconvenience.", cd)
	MakeAnnouncement(msg)

	if cd > 0 {
		time.AfterFunc(time.Second*10, func() {
			countMaintenance(cd - 10)
		})
	} else {
		os.Exit(0)
	}
}
func countRestart(cd int) {
	msg := fmt.Sprintf("There will be restart after %d seconds. Server should be quickly available after.", cd)
	MakeAnnouncement(msg)

	if cd > 0 {
		time.AfterFunc(time.Second*10, func() {
			countRestart(cd - 10)
		})
	} else {
		os.Exit(0)
	}
}

func cmdEvents(event string) {
	switch event {
	case "doublepoints":
		if dungeon.DungeonPointsReward == 1 {
			MakeAnnouncement("Double Dungeon Rewards event activated")
			dungeon.DungeonPointsReward = 2
		} else {
			MakeAnnouncement("Double Dungeon Rewards event deactivated")
			dungeon.DungeonPointsReward = 1
		}
	case "wyvern":
		if funk.Contains(database.EventItems, 13370000) {
			MakeAnnouncement("Wyvern drop event deactivated")
			RemoveEventItem(13370000)
		} else {
			MakeAnnouncement("Wyvern drop event activated")
			database.EventItems = append(database.EventItems, 13370000)
		}
	}
}

func RemoveEventItem(id int) {
	for i, other := range database.EventItems {
		if other == id {
			database.EventItems = append(database.EventItems[:i], database.EventItems[i+1:]...)
			break
		}
	}
}

func cmdSpawnMobs(count, npcID, mapID int) {
	NPCsSpawnPoint := []string{"97,339", "343,89", "211,235", "283,319", "393,383", "347,123", "413,365"}
	for i := 0; i < int(count); i++ {
		randomInt := rand.Intn(len(NPCsSpawnPoint))
		npcPos := &database.NpcPosition{ID: len(database.NPCPos), NPCID: int(npcID), MapID: int16(mapID), Rotation: 0, Attackable: true, IsNPC: false, RespawnTime: 30, Count: 30, MinLocation: "120,120", MaxLocation: "150,150"}
		database.NPCPos = append(database.NPCPos, npcPos)
		npcPos.Create()
		npc, _ := database.NPCs[npcID]
		newai := &database.AI{ID: len(database.AIs), HP: npc.MaxHp, Map: int16(mapID), PosID: npcPos.ID, RunningSpeed: 10, Server: 1, WalkingSpeed: 5, Once: false}
		newai.OnSightPlayers = make(map[int]interface{})
		coordinate := database.ConvertPointToLocation(NPCsSpawnPoint[randomInt])
		randomLocX := randFloats(coordinate.X, coordinate.X+30)
		randomLocY := randFloats(coordinate.Y, coordinate.Y+30)
		loc := utils.Location{X: randomLocX, Y: randomLocY}
		npcPos.MinLocation = fmt.Sprintf("%.1f,%.1f", randomLocX, randomLocY)
		maxX := randomLocX + 50
		maxY := randomLocY + 50
		npcPos.MaxLocation = fmt.Sprintf("%.1f,%.1f", maxX, maxY)
		newai.Coordinate = loc.String()
		fmt.Println(newai.Coordinate)
		newai.Handler = newai.AIHandler
		database.AIsByMap[newai.Server][newai.Map] = append(database.AIsByMap[newai.Server][newai.Map], newai)
		database.DungeonsAiByMap[newai.Server][newai.Map] = append(database.AIsByMap[newai.Server][newai.Map], newai)
		database.AIs[newai.ID] = newai
		DungeonCount := database.DungeonsByMap[newai.Server][newai.Map] + 1
		database.DungeonsByMap[newai.Server][newai.Map] = DungeonCount
		fmt.Println("Mobs Count: ", DungeonCount)
		server.GenerateIDForAI(newai)
		newai.Create()
		//ai.Init()
		if newai.WalkingSpeed > 0 {
			go newai.Handler()
		}
	}
}
func startGuildWar(sourceG, enemyG *database.Guild) []byte {
	challengerGuild := sourceG
	enemyGuild := enemyG
	MakeAnnouncement(fmt.Sprintf("%s has declare war to %s.", challengerGuild.Name, enemyGuild.Name))

	return nil
}

func randFloats(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func RemoveIndex(a []string, index int) []string {
	a[index] = a[len(a)-1] // Copy last element to index i.
	a[len(a)-1] = ""       // Erase last element (write zero value).
	a = a[:len(a)-1]       // Truncate slice.
	return a
}
