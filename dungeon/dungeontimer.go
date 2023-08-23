package dungeon

import (
	"fmt"
	"os"
	"time"

	"PsiHero/database"
	"PsiHero/messaging"
	"PsiHero/utils"

	"github.com/thoas/go-funk"
)

var (
	TIMER_MENU    = utils.Packet{0xAA, 0x55, 0x08, 0x00, 0x65, 0x03, 0x00, 0x00, 0x00, 0x55, 0xAA}
	DungeonMinute = time.Duration(time.Minute * 30) //HERE YOU CAN CHANGE HOW MANY TIMES THEY CAN BE IN DUNGEON //NOW IT'S 30 MINUTE
)

func findRightReward(char *database.Character) int64 {
	if char.Level >= 50 && char.Level < 70 {
		return 17700580
	} else if char.Level >= 70 && char.Level < 90 {
		return 17700581
	} else if char.Level >= 90 && char.Level < 100 {
		return 17700582
	}
	return 0
}

func StartTimer(char *database.Socket) {
	timein := time.Now().Add(time.Minute * 30)
	deadtime := timein.Format(time.RFC3339)

	v, err := time.Parse(time.RFC3339, deadtime)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for range time.Tick(1 * time.Second) {
		timeRemaining := getTimeRemaining(v)
		if char.Character.DungeonLevel == 5 {
			ResetDungeon(char.Character)
			break
		}
		if timeRemaining.t <= 0 || char.Character.Map != 229 || !char.Character.IsOnline {
			resp := utils.Packet{}
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("You have failed. Come again when you are stronger. Teleporting to safe zone.")))
			DeleteMobs(char.User.ConnectedServer)
			char.Character.Socket.User.ConnectedServer = 1
			data, _ := char.Character.ChangeMap(1, nil)
			resp.Concat(data)
			char.Conn.Write(resp)
			break
		}
		mobscount := 0
		database.DungeonsByMap[char.User.ConnectedServer][char.Character.Map] = mobscount
		for _, mobs := range database.DungeonsTest {
			if mobs.Handler == nil && mobs.Server == char.User.ConnectedServer {
				delete(database.DungeonsTest, mobs.ID)
				database.DungeonsByMap[char.User.ConnectedServer][char.Character.Map] = len(database.DungeonsTest)
			}
			if mobs.Server == char.User.ConnectedServer {
				mobscount++
				database.DungeonsByMap[char.User.ConnectedServer][char.Character.Map] = mobscount
			}
		}
		if len(database.DungeonsTest) == 0 {
			database.DungeonsByMap[char.User.ConnectedServer][char.Character.Map] = mobscount
			/*if char.Character.DungeonLevel == 3 {
				char.Character.DungeonLevel = 4
			}*/
		}
		//for _, char := range DungeonCharacters {
		if char.Character.IsDungeon {
			if funk.Contains(database.DungeonZones, char.Character.Map) { //len(DungeonsByMap[ai.Server][ai.Map])
				if database.DungeonsByMap[char.Character.Socket.User.ConnectedServer][char.Character.Map] == 0 {
					switch char.Character.DungeonLevel {
					case 1:
						if char.Character.GeneratedNumber == 0 || char.Character.CanTip == 1 {
							//	if char.PartyID != "" {
							//		party := database.FindParty(char)
							//		if party.Leader.ID == char.ID {
							//			dungeon.FindTheNumber(s)
							//		}
							//	} else {
							FindTheNumber(char.Character.Socket)
							//	}
						}
						if char.Character.CanTip == 3 {
							char.Character.CanTip = 1
						}
					case 2:
						BossSpawn(char.Character.Socket.User.ConnectedServer, char.Character)
						char.Character.DungeonLevel++
						/*	if char.PartyID != "" {
							party := database.FindParty(char)
							for _, member := range party.Members {
								member.DungeonLevel++
							}
						}*/
						//char.DungeonLevel++
					case 3:
						/*if char.PartyID != "" {
							party := database.FindParty(char)
							for _, member := range party.Members {
								member.Socket.Write(messaging.InfoMessage(fmt.Sprintf("Dungeon finished. You will teleport out from this arena after 10 sec ")))
								member.IsDungeon = false
								member.Character.DungeonLevel = 1
							}
						}*/
						resp := messaging.InfoMessage(fmt.Sprintf("Dungeon finished. You will teleport out from this arena after 10 sec "))
						char.Character.Socket.Write(resp)
						char.Character.DungeonLevel++
						ResetDungeon(char.Character)
						break
					}
				}
			} else {
				char.Character.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You teleported out of the dungeon")))
				char.Character.IsDungeon = false
				char.Character.DungeonLevel = 1
				DeleteMobs(char.Character.Socket.User.ConnectedServer)
				/*for k := range database.Dungeon {
					delete(database.Dungeon, k)
					fmt.Println("Mobok törölve")
				}*/
			}
			//}
		}
	}
}

type countdown struct {
	t int
	d int
	h int
	m int
	s int
}

func ResetDungeon(char *database.Character) {
	char.IsDungeon = false
	char.DungeonLevel = 1
	newItemID := findRightReward(char)
	itemData, _, _ := char.AddItem(&database.InventorySlot{ItemID: newItemID, Quantity: uint(DungeonPointsReward)}, -1, false)
	char.Socket.Write(*itemData)
	time.AfterFunc(time.Second*10, func() {
		/*	party := database.FindParty(char)
			for _, member := range party.Members {
				gomap, _ := member.ChangeMap(1, nil)
				member.Socket.Write(gomap)
			}*/
		char.Socket.User.ConnectedServer = 1
		char.Socket.Respawn()
	})
}

func deleteDeadMobs(char *database.Socket, id int) {
	for i, other := range database.DungeonsAiByMap[char.User.ConnectedServer][char.Character.Map] {
		if other.ID == id {
			database.DungeonsAiByMap[char.User.ConnectedServer][char.Character.Map] = append(database.DungeonsAiByMap[char.User.ConnectedServer][char.Character.Map], database.DungeonsAiByMap[char.User.ConnectedServer][char.Character.Map][i+1:]...)
			break
		}
	}
}

func divmod(numerator, denominator int64) (quotient, remainder int64) {
	quotient = numerator / denominator // integer division, decimals are truncated
	remainder = numerator % denominator
	return
}

func getTimeRemaining(t time.Time) countdown {
	currentTime := time.Now()
	difference := t.Sub(currentTime)

	total := int(difference.Seconds())
	days := int(total / (60 * 60 * 24))
	hours := int(total / (60 * 60) % 24)
	minutes := int(total/60) % 60
	seconds := int(total % 60)
	return countdown{
		t: total,
		d: days,
		h: hours,
		m: minutes,
		s: seconds,
	}
}
