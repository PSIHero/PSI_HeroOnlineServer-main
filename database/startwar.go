package database

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"PsiHero/messaging"

	"PsiHero/nats"
	"PsiHero/utils"
)

var (
	ANNOUNCEMENT    = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x06, 0x00, 0x55, 0xAA}
	START_WAR       = utils.Packet{0xaa, 0x55, 0x23, 0x00, 0x65, 0x01, 0x00, 0x00, 0x17, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0d, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xb0, 0x04, 0x00, 0x00, 0x55, 0xaa}
	OrderCharacters = make(map[int]*Character)
	ShaoCharacters  = make(map[int]*Character)
	LobbyCharacters = make(map[int]*Character)

	WarRequirePlayers = 10
	OrderPoints       = 10000
	ShaoPoints        = 10000
	CanJoinWar        = false
	WarType           = int64(0)
	WarStarted        = false
	WarStonesIDs      = []uint16{}
	WarStones         = make(map[int]*WarStone)
	ActiveWars        = make(map[int]*ActiveWar)

	// War Automation stuff
	fileReaded        = false
	battlegroundTime  [10]int
	nextWarTime       = 0
	WarAutomationIsOn = true
)

type WarStone struct {
	PseudoID      uint16 `db:"-" json:"-"`
	NpcID         int    `db:"-" json:"-"`
	ConqueredID   int    `db:"-" json:"-"`
	ConquereValue int    `db:"-" json:"-"`
	NearbyZuhang  int    `db:"-" json:"-"`
	NearbyShao    int    `db:"-" json:"-"`
	NearbyZuhangV []int  `db:"-" json:"-"`
	NearbyShaoV   []int  `db:"-" json:"-"`
}
type ActiveWar struct {
	WarID         uint16       `db:"-" json:"-"`
	ZuhangPlayers []*Character `db:"-" json:"-"`
	ShaoPlayers   []*Character `db:"-" json:"-"`
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
func JoinToWarLobby(char *Character) {
	LobbyCharacters[char.ID] = char
	warReady := false
	zuhangPlayers := 0
	shaoPlayers := 0
	if len(LobbyCharacters) >= WarRequirePlayers {
		newWar := &ActiveWar{WarID: 1}
		ActiveWars[int(1)] = newWar
		for _, char := range LobbyCharacters {
			if char.Faction == 1 {
				if zuhangPlayers < WarRequirePlayers/2 {
					zuhangPlayers++
					ActiveWars[int(1)].ZuhangPlayers = append(ActiveWars[int(1)].ZuhangPlayers, char)
				}
			} else {
				if shaoPlayers < WarRequirePlayers/2 {
					shaoPlayers++
					ActiveWars[int(1)].ShaoPlayers = append(ActiveWars[int(1)].ShaoPlayers, char)
				}
			}
			if zuhangPlayers >= WarRequirePlayers/2 && shaoPlayers >= WarRequirePlayers/2 {
				warReady = true
				continue
			}
		}
	}
	if warReady {
		for _, char := range ActiveWars[int(1)].ShaoPlayers {
			char.Socket.Write(messaging.InfoMessage(fmt.Sprintf("Your war is ready. /accept war ")))
		}
		for _, char := range ActiveWars[int(1)].ZuhangPlayers {
			char.Socket.Write(messaging.InfoMessage(fmt.Sprintf("Your war is ready. /accept war ")))
		}
		CheckALlPlayersReady()
	}
}
func secondsToMinutes(inSeconds int) (int, int) {
	minutes := inSeconds / 60
	seconds := inSeconds % 60
	return minutes, seconds
}

func WarAutomation() {

	if !fileReaded { // Read times from txt file and save it to array and close it
		BattlegroundSchedule, err := os.Open("BattlegroundSchedule.txt")
		if err != nil {
			panic(err)
		}
		scanner := bufio.NewScanner(BattlegroundSchedule)
		for i := 0; scanner.Scan(); i++ {
			line := scanner.Text()
			v, _ := strconv.ParseInt(line, 10, 64)
			battlegroundTime[i] = int(v)
		}
		fileReaded = true
		BattlegroundSchedule.Close()
	}

	warTimeSelected := false
	warProcessStarted := false
	announcementSended := false
	func() { // Do loop every 10 seconds
		currentTime := 0
		for {
			t := time.Now()
			t2 := (t.Format("1504"))
			t3, _ := strconv.ParseInt(t2, 10, 64)
			currentTime = int(t3)

			if !warTimeSelected { // Select next War Time
				selection := currentTime
				for { //Start from current time and scroll
					selection++
					for i := 0; i != len(battlegroundTime); i++ { //Scroll through BattlegroundSchedule and find next match
						if battlegroundTime[i] == 0 {
							break
						}
						if battlegroundTime[i] == selection {
							nextWarTime = selection
							warTimeSelected = true
							fmt.Println("Next war is at: ", nextWarTime)
							break
						}
					}

					if selection > 2400 { // Day is over, start from 0
						selection = 0
					}

					if warTimeSelected { //Break loop if Selected
						break
					}
				}
			}

			if nextWarTime == currentTime {
				warProcessStarted = true
			}

			if warProcessStarted {
				for i := 1; i < 4; {
					if !WarStarted && !CanJoinWar {
						time.Sleep(time.Second * 10)
						if !WarStarted && !CanJoinWar {
							OrderPoints = 10000
							ShaoPoints = 10000
							CanJoinWar = true
							WarType = int64(i)
							warmUpTime := 600
							if i > 1 {
								warmUpTime = 300
							}
							StartWarTimer(warmUpTime)
							i++
						}
					}

					if !WarAutomationIsOn { // Stop WarProcess if automation deactivated with command
						break
					}
				}
				time.AfterFunc(time.Second*60, func() { // Remove stucked Announcement
					empty := fmt.Sprintf(" ")
					MakeAnnouncement(empty)
					time.Sleep(time.Second * 3)
					MakeAnnouncement(empty)
					time.Sleep(time.Second * 3)
					MakeAnnouncement(empty)
				})
				warProcessStarted = false
				warTimeSelected = false
			}

			if currentTime%100 == 0 && !announcementSended {
				announcementSended = true
				s := fmt.Sprintf("%04d", nextWarTime)
				v := strings.SplitAfter(s, "")
				msg := fmt.Sprintf("Next Battleground will start at: %s%s:%s%s (Servertime)", v[0], v[1], v[2], v[3])
				MakeAnnouncement(msg)
				/*time.AfterFunc(time.Second*300, func() { // Remove stucked Announcement
					empty := fmt.Sprintf(" ")
					MakeAnnouncement(empty)
					time.Sleep(time.Second * 3)
					MakeAnnouncement(empty)
					time.Sleep(time.Second * 3)
					MakeAnnouncement(empty)
				})*/
			} else if currentTime%100 != 0 {
				announcementSended = false
			}

			time.Sleep(10 * time.Second)
		}
	}()
}

func StartWarTimer(prepareWarStart int) {
	min, sec := secondsToMinutes(prepareWarStart)
	var msg string
	var msg2 string
	switch WarType {
	case 1:
		msg = fmt.Sprintf("In %dm %ds the Non-Divine Battleground will start.", min, sec)
		msg2 = fmt.Sprintf("Participate Battle by Ascension Battle Guard")
	case 2:
		msg = fmt.Sprintf("In %dm %ds the Divine Battleground will start.", min, sec)
		msg2 = fmt.Sprintf("Participate Battle by Ascension Battle Guard")
	case 3:
		msg = fmt.Sprintf("In %dm %ds the Dark Battleground will start.", min, sec)
		msg2 = fmt.Sprintf("Participate Battle by Ascension Battle Guard")
	}
	MakeAnnouncement(msg)
	MakeAnnouncement(msg2)
	if prepareWarStart > 0 {
		time.AfterFunc(time.Second*10, func() {
			StartWarTimer(prepareWarStart - 10)
		})
	} else {
		StartWar()
	}
}

func ResetWar() {
	time.AfterFunc(time.Second*10, func() {
		for _, char := range OrderCharacters {
			if char.IsOnline {

			} else {
				delete(OrderCharacters, char.ID)
			}
			char.WarKillCount = 0
			char.WarContribution = 0
			char.IsinWar = false

			char.Socket.Respawn()
			delete(OrderCharacters, char.ID)
		}
		for _, char := range ShaoCharacters {
			if char.IsOnline {

			} else {
				delete(OrderCharacters, char.ID)
			}
			char.WarKillCount = 0
			char.WarContribution = 0
			char.IsinWar = false

			char.Socket.Respawn()
			delete(ShaoCharacters, char.ID)
		}
		for _, stones := range WarStones {
			stones.ConquereValue = 0
			stones.ConqueredID = 0
		}
		WarStarted = false
	})
	OrderPoints = 10000
	ShaoPoints = 10000
}

func StartWar() {
	resp := START_WAR
	byteOrders := utils.IntToBytes(uint64(len(OrderCharacters)), 4, false)
	byteShaos := utils.IntToBytes(uint64(len(ShaoCharacters)), 4, false)
	resp.Overwrite(byteOrders, 8)
	resp.Overwrite(byteShaos, 22)
	for _, char := range OrderCharacters {
		if char.Socket == nil || !char.IsOnline {
			delete(OrderCharacters, char.ID)
		}
		char.Socket.Write(resp)
	}
	for _, char := range ShaoCharacters {
		if char.Socket == nil || !char.IsOnline {
			char.Socket.Write(resp)
		}
	}
	if len(OrderCharacters) < 1 && len(ShaoCharacters) < 1 {
		CanJoinWar = false
		WarStarted = false
		ResetWar() //Players need to go map 1 if nobody is in the war
	} else {
		CanJoinWar = false
		WarStarted = true
		StartInWarTimer()
	}
}

func (self *WarStone) RemoveZuhang(id int) {
	for i, other := range self.NearbyZuhangV {
		if other == id {
			self.NearbyZuhangV = append(self.NearbyZuhangV[:i], self.NearbyZuhangV[i+1:]...)
			break
		}
	}
}

func (self *WarStone) RemoveShao(id int) {
	for i, other := range self.NearbyShaoV {
		if other == id {
			self.NearbyShaoV = append(self.NearbyShaoV[:i], self.NearbyShaoV[i+1:]...)
			break
		}
	}
}
