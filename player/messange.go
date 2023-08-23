package player

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"PsiHero/database"
	"PsiHero/messaging"
	"PsiHero/utils"

	"gopkg.in/guregu/null.v3"
)

type (
	OpenMessagerHandler      struct{}
	OpenAMessageHandler      struct{}
	DeleteMessageHandler     struct{}
	ReceiveItemsHandler      struct{}
	SendMessageHandler       struct{}
	ItemAddMessageHandler    struct{}
	ItemRemoveMessageHandler struct{}
)

var (
	MESSANGE_MENU       = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x8b, 0x01, 0x0a, 0x00, 0x00, 0x00, 0x55, 0xAA}
	OPEN_A_MESSANGE     = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x8b, 0x02, 0x0a, 0x00, 0x55, 0xAA}
	ADDITEM_MESSANGE    = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x8b, 0x06, 0x0a, 0x00, 0x55, 0xAA}
	REMOVEITEM_MESSANGE = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x8b, 0x07, 0x0a, 0x00, 0x55, 0xAA}
	MAIL_RECEIVED       = utils.Packet{0xaa, 0x55, 0x02, 0x00, 0x8b, 0x08, 0x55, 0xaa}
)

func (h *OpenMessagerHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	s.Character.MessageItems = nil
	resp := MESSANGE_MENU
	charactermails, err := database.FindMailsByCharacterID(s.Character.ID)
	if err != nil {
		return nil, err
	}
	if len(charactermails) <= 0 {
		return MESSANGE_MENU, nil
	}
	index := 8
	resp.Insert(utils.IntToBytes(uint64(len(charactermails)), 2, true), index)
	index += 2
	for _, mail := range charactermails {
		currentTime := time.Now()
		diff := mail.ExpiresAt.Time.Sub(currentTime)
		if diff < 0 {
			MessageExpired(mail.ID)
			continue
		}
		resp.Insert(utils.IntToBytes(uint64(len(mail.Title)), 2, true), index)
		index += 2
		resp.Insert([]byte(mail.Title), index)
		index += len(mail.Title)
		char, err := database.FindCharacterByID(mail.SenderID)
		if err != nil {
			return nil, nil
		}
		if char == nil { //KI KELL JAVÍTANI, HA TÖRLIK A KARAKTERT
			fmt.Println("Character not found")
			return nil, nil
		}
		resp.Insert(utils.IntToBytes(uint64(len(char.Name)), 2, true), index)
		index += 2
		resp.Insert([]byte(char.Name), index)
		index += len(char.Name)
		resp.Insert(utils.IntToBytes(uint64(mail.ID), 4, true), index)
		index += 4
		bitSet := mail.IsReceived
		bitSetVar := int8(1)
		if bitSet {
			bitSetVar = 0
		}
		resp.Insert([]byte{byte(bitSetVar)}, index) //TÍPUS
		index++
		resp.Insert(utils.IntToBytes(uint64(diff.Minutes()), 4, true), index)
		index += 4
		bitSet = mail.IsOpened
		bitSetVar = int8(0)
		if bitSet {
			bitSetVar = 1
		}
		resp.Insert([]byte{byte(bitSetVar), 0x00}, index) //TÍPUS
		index += 2
	}
	resp.SetLength(int16(binary.Size(resp) - 6))
	return resp, nil
}

func MessageExpired(messageid int) error {
	mail, err := database.FindMailByID(int(messageid))
	if err != nil {
		return err
	}
	allItems := mail.GetItems()
	for _, itemid := range allItems {
		if itemid == 0 {
			continue
		}
		item, err := database.FindInventorySlotByID(itemid)
		if err != nil {
			return err
		}
		if item == nil {
			return err
		}
		item.Delete()
	}
	mail.Delete()
	return nil
}
func (h *OpenAMessageHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	mailID := int(utils.BytesToInt(data[6:10], true))
	mail, err := database.FindMailByID(mailID)
	if err != nil || mail == nil {
		return nil, nil
	}
	resp := OPEN_A_MESSANGE
	index := 8
	resp.Insert(utils.IntToBytes(uint64(len(mail.Title)), 2, true), index)
	index += 2
	resp.Insert([]byte(mail.Title), index)
	index += len(mail.Title)
	resp.Insert(utils.IntToBytes(uint64(len(mail.Content)), 2, true), index)
	index += 2
	if len(mail.Content) > 0 {
		resp.Insert([]byte(mail.Content), index)
		index += len(mail.Content)
	}
	resp.Insert(utils.IntToBytes(uint64(mail.Gold), 5, true), index)
	index += 5
	resp.Insert([]byte{0x00, 0x00, 0x00, 0x00}, index)
	index += 4
	char, err := database.FindCharacterByID(mail.SenderID)
	if err != nil {
		return nil, nil
	}
	if char == nil { //KI KELL JAVÍTANI, HA TÖRLIK A KARAKTERT
		//fmt.Println("Character not found")
		//return nil, nil
		char.Name = "Deleted"
	}
	resp.Insert(utils.IntToBytes(uint64(len(char.Name)), 2, true), index)
	index += 2
	resp.Insert([]byte(char.Name), index)
	index += len(char.Name)
	resp.Insert([]byte{0x00, 0x00}, index)
	index += 2
	itemscountlength := index - 2
	itemscount := 0

	allItems := mail.GetItems()
	for _, itemid := range allItems {
		if itemid == 0 {
			continue
		}
		item, err := database.FindInventorySlotByID(itemid)
		if err != nil {
			return nil, nil
		}
		if item == nil {
			log.Print("FindItemError: ", err)
			log.Print("ItemID: ", itemid)
			return nil, nil
		}
		iteminfo, ok := database.GetItemInfo(item.ItemID)
		if !ok {
			log.Print("Item not found: ", item.ItemID)
			return nil, nil
		}
		resp.Insert([]byte{0x06}, index)
		index++
		itemscount++
		resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), index)
		index += 4
		resp.Insert([]byte{0x0c, 0x00, 0x00, 0x00, 0x77, 0x0b, 0x15, 0x02}, index)
		index += 8
		resp.Insert(utils.IntToBytes(uint64(item.Quantity), 2, true), index)
		index += 2
		if iteminfo.GetType() == database.PET_TYPE { // pet
			pet := &database.PetSlot{}
			isPet := iteminfo.GetType() == database.PET_TYPE
			if isPet {
				json.Unmarshal(item.PetInfo, pet)
				//resp.Insert([]byte{0x00}, index)
				//index++
				resp.Insert([]byte{pet.Level}, index) // pet level, loyalty and fullness
				index++
				resp.Insert(utils.IntToBytes(uint64(pet.HP), 2, true), index) // pet hp
				index += 2
				resp.Insert(utils.IntToBytes(uint64(pet.CHI), 2, true), index) // pet hp
				index += 2
				resp.Insert(utils.IntToBytes(uint64(pet.Exp), 8, true), index) // pet exp
				index += 8
				resp.Insert([]byte{0, 0, 0, 0}, index) // padding
				index += 4
				if pet.Name != "" {
					resp.Insert([]byte(pet.Name), index)
					index += len(pet.Name)
				}
				resp.Insert([]byte{0x00, 0x00, 0x00, 0x7b, 0x94, 0x63, 0x00}, index)
				index += 7
			}
		} else {
			if item.Plus > 0 || item.SocketCount > 0 {
				resp.Insert([]byte{0xA2}, index)
			} else {
				resp.Insert([]byte{0xA1}, index)
			}
			index++
			resp.Insert(item.GetUpgrades(), index) // item upgrades
			index += 15
			resp.Insert(utils.IntToBytes(uint64(item.SocketCount), 1, true), index)
			index++
			resp.Insert(item.GetSockets(), index)
			index += 14
			if item.ItemType != 0 {
				resp.Overwrite(utils.IntToBytes(uint64(item.ItemType), 1, true), index-5)
				if item.ItemType == 2 {
					resp.Overwrite(utils.IntToBytes(uint64(item.JudgementStat), 4, true), index-4)
				}
			}
			resp.Insert(utils.IntToBytes(uint64(item.Appearance), 4, true), index)
			index += 4

			resp.Insert([]byte{0x7b, 0x94, 0x63, 0x00}, index)
			index += 4
		}
	}
	mail.IsOpened = true
	mail.Update()
	resp.Overwrite(utils.IntToBytes(uint64(itemscount), 2, true), itemscountlength)
	resp.SetLength(int16(binary.Size(resp) - 6))
	return resp, nil
}

func (h *ItemAddMessageHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	slotID := int(utils.BytesToInt(data[6:8], true))
	itemID := utils.BytesToInt(data[8:12], true)
	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, nil
	}
	item := slots[slotID]
	itemInfo := database.Items[itemID]
	if !itemInfo.Tradable || item.InUse {
		return nil, nil
	}
	s.Character.MessageItems = append(s.Character.MessageItems, &database.MessageItems{ID: item.ID, SlotID: slotID, ItemID: itemID})
	return ADDITEM_MESSANGE, nil
}
func RemoveMessageItem(id int16, char *database.Character) error {
	for i, other := range char.MessageItems {
		if int16(other.SlotID) == id {
			char.MessageItems = append(char.MessageItems[:i], char.MessageItems[i+1:]...)
			break
		}
	}
	return nil
}

func (h *ItemRemoveMessageHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	slotID := int16(utils.BytesToInt(data[6:8], true))
	itemID := utils.BytesToInt(data[8:12], true)
	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, nil
	}
	item := slots[slotID]
	if itemID != item.ItemID && slotID != item.SlotID {
		return nil, nil
	}
	err = RemoveMessageItem(slotID, s.Character)
	if err != nil {
		return nil, nil
	}
	return REMOVEITEM_MESSANGE, nil
}

func (h *DeleteMessageHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	mailCount := int(data[6])
	index := 7
	sendresp := utils.Packet(data)
	respindex := 9
	sendresp.Insert([]byte{0x0a, 0x00}, 6)
	for i := 1; i <= mailCount; i++ {
		mailID := utils.BytesToInt(data[index*i:index*i+4], true)
		sendresp.Insert([]byte{0x0a, 0x00}, respindex)
		respindex += 6
		mail, err := database.FindMailByID(int(mailID))
		if err != nil {
			return nil, err
		}
		mail.Delete()
	}
	sendresp.SetLength(int16(binary.Size(sendresp) - 6))
	return sendresp, nil
}

func (h *SendMessageHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.Character.TradeID != "" {
		info := messaging.SystemMessage(messaging.CANNOT_MOVE_ITEM_IN_TRADE)
		return info, nil
	}
	content := ""
	titleLength := int(utils.BytesToInt(data[6:8], true))
	index := 8
	title := string(data[index : titleLength+index])
	index += titleLength
	contentLength := int(utils.BytesToInt(data[index:index+2], true))
	index += 2
	if contentLength > 0 {
		content = string(data[index : contentLength+index])
		index += contentLength
	}
	//index++
	gold := utils.BytesToInt(data[index:index+5], true)
	index += 8
	charnameLength := int(utils.BytesToInt(data[index:index+2], true))
	index += 2
	charname := string(data[index : charnameLength+index])
	char, err := database.FindCharacterByName(charname)
	if err != nil {
		return nil, nil
	}
	if char == nil {
		fmt.Println("Character not found")
		return nil, nil
	}
	itemsArr := "{0,0,0,0,0}"
	exp := time.Now().UTC().Add(time.Hour * 24 * 10)
	message := &database.MailMessage{
		SenderID:   s.Character.ID,
		ReceiverID: char.ID,
		Gold:       gold,
		Content:    content,
		Title:      title,
		ItemsArr:   itemsArr,
		IsOpened:   false,
		ExpiresAt:  null.TimeFrom(exp),
	}
	resp := utils.Packet{}
	itemslength := 0
	for i, other := range s.Character.MessageItems {
		message.SetMessageItems(i, other.ID)
		removeresp := database.ITEM_REMOVED
		removeresp.Insert(utils.IntToBytes(uint64(other.ItemID), 4, true), 9)  // item id
		removeresp.Insert(utils.IntToBytes(uint64(other.SlotID), 2, true), 13) // slot id
		resp.Concat(removeresp)
		itemslot, err := database.FindInventorySlotByID(other.ID)
		if itemslot.ItemID != other.ItemID && itemslot.SlotID != int16(other.SlotID) {
			s.Character.MessageItems = nil
			return nil, nil
		}
		if err != nil {
			s.Character.MessageItems = nil
			return nil, nil
		}
		slots, err := s.Character.InventorySlots()
		if err != nil {
			return nil, err
		}

		item := slots[other.SlotID]
		newItem := database.NewSlot()
		*newItem = *item
		newItem.SlotID = -1
		newItem.UserID = null.StringFromPtr(nil)
		newItem.CharacterID = null.IntFromPtr(nil)
		newItem.Update()
		database.InventoryItems.Add(newItem.ID, newItem)
		*item = *database.NewSlot()
		itemslength++
	}
	if s.Character.Gold < uint64(gold) {
		return nil, nil
	}
	commersion := uint64(1000)
	if itemslength > 0 {
		commersion = commersion * uint64(itemslength)
	}
	resp.Concat(s.Character.LootGold(-commersion))
	resp.Concat(s.Character.LootGold(-uint64(gold)))
	s.Character.Update()
	err = message.Create()
	if err != nil {
		fmt.Println("SendMessageError: ", err)
	}
	fmt.Println("Sender: ", message.SenderID, ". Receiver: ", message.ReceiverID, ". Mail gold: ", message.Gold, ", mail items: ", message.ItemsArr, ". Time: ", time.Now())
	s.Character.MessageItems = nil
	sendresp := utils.Packet(data)
	sendresp.Insert([]byte{0x0a, 0x00}, 6)
	sendresp.SetLength(int16(binary.Size(sendresp) - 6))
	resp.Concat(sendresp)
	s.Write(resp)
	if char.IsOnline {
		char.Socket.Write(MAIL_RECEIVED)
	}
	return nil, nil
}

func (h *ReceiveItemsHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	mailID := int(utils.BytesToInt(data[6:10], true))
	mail, err := database.FindMailByID(mailID)
	if err != nil || mail == nil {
		return nil, nil
	}
	allItems := mail.GetItems()
	getitems := 0
	for _, itemid := range allItems {
		if itemid == 0 {
			continue
		}
		getitems++
	}
	freeslots, err := s.Character.FindFreeSlots(getitems)
	if err != nil {
		return nil, err
	}
	resp := utils.Packet(data)
	resp.Insert([]byte{0x0a, 0x00}, 6)
	resp.SetLength(int16(binary.Size(resp) - 6))
	slotid := int16(0)
	for _, itemid := range allItems {
		if itemid == 0 {
			continue
		}
		item, err := database.FindInventorySlotByID(itemid)
		if err != nil {
			return nil, nil
		}
		if item == nil {
			return nil, nil
		}
		slots, err := s.Character.InventorySlots()
		if err != nil {
			return nil, err
		}
		freeslot := freeslots[slotid]
		newItem := database.NewSlot()
		*newItem = *item
		newItem.UserID = null.StringFrom(s.Character.UserID)
		newItem.CharacterID = null.IntFrom(int64(s.Character.ID))
		newItem.SlotID = freeslot

		err = newItem.Update()
		if err != nil {
			return nil, err
		}

		*slots[freeslot] = *newItem
		database.InventoryItems.Add(newItem.ID, slots[freeslot])

		resp.Concat(item.GetData(freeslot))
		slotid++
	}
	fmt.Println("Receiver: ", mail.ReceiverID, ". Sender: ", mail.SenderID, ". Mail gold: ", mail.Gold, ", nail items: ", mail.ItemsArr, ". Opened or received? ", mail.IsOpened, mail.IsReceived, ". Time: ", time.Now())
	s.Character.Gold += uint64(mail.Gold)
	goldresp := GET_GOLD
	goldresp.Insert(utils.IntToBytes(uint64(s.Character.Gold), 8, true), 6)
	resp.Concat(goldresp)
	resp.Concat([]byte{0xaa, 0x55, 0x04, 0x00, 0x8b, 0x05, 0x0a, 0x00, 0x55, 0xaa})
	mail.Gold = 0
	mail.ItemsArr = "{0,0,0,0,0}"
	mail.IsReceived = true
	mail.Update()
	return resp, nil
}
