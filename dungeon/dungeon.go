package dungeon

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"PsiHero/database"
	"PsiHero/messaging"
	"PsiHero/server"
	"PsiHero/utils"

	"github.com/thoas/go-funk"
	"gopkg.in/guregu/null.v3"
)

var (
	DungeonCharacters        = make(map[int]*database.Character)
	ActiveDungeons           = make(map[int]*database.Dungeon)
	DungeonCharactersByparty []map[int16]int
	DungeonPointsReward      = 1
	DungeonMapID             = int16(246)
	mu                       sync.Mutex
)

/*
------------------------//<> PSI Dungeon by ePecetek<>\\----------------------------------------
- 3 dungeons total, 1 for non-divine, 1 for divine, 1 for darkness. Bosses drop certain tokens (I can replace that, can be just garnet crystals for me to replace)
- Have to go in as a group of 4, has a 6 hour timer in between.
- Multiple groups can go in separate instances. So, 1 group of 4 can go in, then another can go in, and divine groups can go in and darkness groups can go in.
---------------------------//<> PSI Dungeon by ePecetek<>\\--------------------------------------
*/
func init() {
	database.GetActiveDungeons = func() map[int]*database.Dungeon {
		return ActiveDungeons
	}
	database.DeleteDungeonMobs = func(server int) {
		allMobs := database.AIsByMap[server][DungeonMapID]
		//database.AIsByMap[server][DungeonMapID] = nil
		for _, mobs := range allMobs {
			database.AIMutex.Lock()
			delete(database.AIs, mobs.ID)
			database.AIMutex.Unlock()
			//database.AIs[mobs.ID] = nil
			mobs.IsDead = true
			mobs.Handler = nil
			delete(database.DungeonsAiByMap[server], mobs.Map)
			DungeonCount := database.DungeonsByMap[server][DungeonMapID] - 1
			database.DungeonsByMap[server][DungeonMapID] -= DungeonCount
		}
	}
}

func DeleteDungeonMobs(server int) {
	allMobs := database.AIsByMap[server][DungeonMapID]
	//database.AIsByMap[server][DungeonMapID] = nil
	for _, mobs := range allMobs {
		database.AIMutex.Lock()
		delete(database.AIs, mobs.ID)
		database.AIMutex.Unlock()
		//database.AIs[mobs.ID] = nil
		mobs.IsDead = true
		mobs.Handler = nil
		delete(database.DungeonsAiByMap[server], mobs.Map)
		DungeonCount := database.DungeonsByMap[server][DungeonMapID] - 1
		database.DungeonsByMap[server][DungeonMapID] -= DungeonCount
	}
}
func FindEmptyServer() int {
	for i := 1; i < database.SERVER_COUNT; i++ {
		allMobs := database.DungeonsByMap[i][DungeonMapID]
		allMobsCount := database.AIsByMap[i][DungeonMapID]
		if allMobs == 0 && len(allMobsCount) == 0 {
			return i
		}
	}
	return 0
}

func StartDungeon(char *database.Socket) {
	server := FindEmptyServer()
	if server != 0 {
		char.Character.IsDungeon = true
		char.Character.DungeonLevel = 1
		char.Character.GeneratedNumber = 0
		char.Character.CanTip = 1
		char.Character.Socket.User.ConnectedServer = server
		DeleteDungeonMobs(server)
		dungeon := &database.Dungeon{ServerID: server, DungeonLeader: char.Character, DungeonStartedTime: time.Now(), IsLoading: true}
		ActiveDungeons[len(ActiveDungeons)] = dungeon
		data, _ := char.Character.ChangeMap(DungeonMapID, nil)
		char.Conn.Write(data)
		char.Conn.Write(messaging.InfoMessage("Welcome to Pecetek's Dungeon. You have 30 minutes, Survive & Slay the Monsters."))
		if database.DungeonEvents[char.Character.ID] == nil {
			database.DungeonEvents[char.Character.ID] = &database.DungeonEvent{CharacterID: char.Character.ID}
			database.DungeonEvents[char.Character.ID].Create()
		}
		database.DungeonEvents[char.Character.ID].LastStartedTime = null.NewTime(time.Now(), true) //THERE CAN BE PROBLEM!!
		database.DungeonEvents[char.Character.ID].Update()
		DUNGEON_TIMER := utils.Packet{0xAA, 0x55, 0x06, 0x00, 0xC0, 0x17, 0x09, 0x07, 0x00, 0x00, 0x55, 0xAA}
		char.Conn.Write(DUNGEON_TIMER)
		party := database.FindParty(char.Character)
		for _, member := range party.Members {
			if member.Socket == nil || !member.Accepted {
				continue
			}
			if database.DungeonEvents[member.ID] == nil {
				database.DungeonEvents[member.ID] = &database.DungeonEvent{CharacterID: member.ID}
				database.DungeonEvents[member.ID].Create()
			}
			database.DungeonEvents[member.ID].LastStartedTime = null.NewTime(time.Now(), true) //THERE CAN BE PROBLEM!!
			err := database.DungeonEvents[member.ID].Update()
			if err != nil {
				fmt.Println("ERROR WHILE DUNGEONEVENTS UPDATE:", err)
			}
			member.Character.IsDungeon = true
			member.Character.DungeonLevel = 1
			member.Socket.User.ConnectedServer = server
			data, _ := member.ChangeMap(246, nil)
			member.Socket.Write(data)
			member.Socket.Write(DUNGEON_TIMER)
			member.Socket.Write(messaging.InfoMessage("Welcome to Pecetek's Dungeon. You have 30 minutes, Survive & Slay the Monsters."))
		}
		CreateMobsForNewDungeon(server, char)
		database.DungeonLoading = false
		dungeon.IsLoading = false
	} else {
		database.DungeonLoading = false
	}
}

func DeleteMobs(server int) {
	allMobs := database.AIsByMap[server][DungeonMapID]
	//database.AIsByMap[server][DungeonMapID] = nil
	for _, mobs := range allMobs {
		//delete(database.AIs, mobs.ID)
		//database.AIs[mobs.ID] = nil
		mobs.IsDead = true
		mobs.Handler = nil
		delete(database.DungeonsAiByMap[server], mobs.Map)
		DungeonCount := database.DungeonsByMap[server][DungeonMapID] - 1
		database.DungeonsByMap[server][DungeonMapID] -= DungeonCount
	}
}

func findRightMobForPlayer(char *database.Socket) []int {
	NPCArray := []int{}
	if char.Character.Level >= 50 && char.Character.Level < 60 {
		return []int{13000230, 13000231, 13000232, 13000233} //13000234 BOSS
	} else if char.Character.Level >= 60 && char.Character.Level < 70 {
		return []int{13000235, 13000236, 13000237, 13000238} //13000239
	} else if char.Character.Level >= 70 && char.Character.Level < 80 {
		return []int{13000240, 13000241, 13000242, 13000243} //13000244
	} else if char.Character.Level >= 80 && char.Character.Level < 90 {
		return []int{13000245, 13000246, 13000247, 13000248} //13000249
	} else if char.Character.Level >= 90 && char.Character.Level < 100 {
		return []int{13000250, 13000251, 13000252, 13000253} //13000254
	}
	return NPCArray
}
func findRightMobByDivineType(char *database.Socket) []*database.NpcPosition {
	minLevel, maxLevel := int16(0), int16(0)
	if char.Character.Level <= 100 { //NON-DIVINE
		minLevel = 0
		maxLevel = 100
	} else if char.Character.Level >= 101 && char.Character.Level <= 200 { //DIVINE
		minLevel = 101
		maxLevel = 200
	} else if char.Character.Level >= 201 && char.Character.Level <= 300 { //DARKNESS
		minLevel = 201
		maxLevel = 300
	}

	mobArray := funk.Filter(database.NPCPos, func(npcPos *database.NpcPosition) bool {
		npcTable := database.NPCs[npcPos.NPCID]
		return !npcPos.IsNPC && npcPos.Attackable && npcPos.MapID == DungeonMapID && npcTable.Level >= minLevel && npcTable.Level <= maxLevel
	}).([]*database.NpcPosition)

	return mobArray
}
func CreateMobsForNewDungeon(serverID int, char *database.Socket) {
	mu.Lock()
	defer mu.Unlock()
	NPCsTest := findRightMobByDivineType(char)
	for _, npcPos := range NPCsTest {
		for i := 0; i < int(npcPos.Count); i++ { //SPAWN HOW MANY IS IN THE COUNT!!
			npc, _ := database.NPCs[npcPos.NPCID]
			if npc == nil {
				continue
			}
			database.AIMutex.Lock()
			allAI := database.AIs
			database.AIMutex.Unlock()
			r := funk.Map(allAI, func(k int, v *database.AI) int {
				return v.ID
			})
			maxAIID := funk.MaxInt(r.([]int)).(int)
			newai := &database.AI{ID: maxAIID + 1, HP: npc.MaxHp, Map: DungeonMapID, PosID: npcPos.ID, RunningSpeed: 10, Server: serverID, WalkingSpeed: 5, Once: true}
			newai.OnSightPlayers = make(map[int]interface{})
			minCoordinate := database.ConvertPointToLocation(npcPos.MinLocation)
			maxCoordinate := database.ConvertPointToLocation(npcPos.MaxLocation)
			randomLocX := randFloats(minCoordinate.X, maxCoordinate.X)
			randomLocY := randFloats(minCoordinate.Y, maxCoordinate.Y)
			loc := utils.Location{X: randomLocX, Y: randomLocY}
			newai.Coordinate = loc.String()
			server.GenerateIDForAI(newai)
			newai.Handler = newai.AIHandler
			database.AIsByMap[newai.Server][newai.Map] = append(database.AIsByMap[newai.Server][newai.Map], newai)
			database.DungeonsAiByMap[serverID][newai.Map] = append(database.DungeonsAiByMap[serverID][newai.Map], newai)
			database.DungeonsTest[newai.ID] = newai
			database.AIMutex.Lock()
			database.AIs[newai.ID] = newai
			database.AIMutex.Unlock()
			DungeonCount := database.DungeonsByMap[newai.Server][newai.Map] + 1
			database.DungeonsByMap[newai.Server][newai.Map] = DungeonCount
			go newai.Handler()
		}
	}
}
func Createmob(serverID int, char *database.Socket) {
	NPCsSpawnPoint := []string{"591,267", "381,251", "201,241", "253,293", "177,463", "213,591", "135,651"}
	NPCsTest := findRightMobForPlayer(char)
	for _, action := range NPCsTest {
		for i := 0; i < int(20); i++ {
			randomInt := rand.Intn(len(NPCsSpawnPoint))
			npcPos := &database.NpcPosition{ID: len(database.NPCPos), NPCID: int(action), MapID: DungeonMapID, Rotation: 0, Attackable: true, IsNPC: false, RespawnTime: 30, Count: 1, MinLocation: "120,120", MaxLocation: "150,150"}
			database.NPCPos = append(database.NPCPos, npcPos)
			npc, _ := database.NPCs[action]
			newai := &database.AI{ID: len(database.AIs), HP: npc.MaxHp, Map: DungeonMapID, PosID: npcPos.ID, RunningSpeed: 10, Server: serverID, WalkingSpeed: 5, Once: true}
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
			server.GenerateIDForAI(newai)
			newai.Handler = newai.AIHandler
			database.AIsByMap[newai.Server][newai.Map] = append(database.AIsByMap[newai.Server][newai.Map], newai)
			database.DungeonsAiByMap[serverID][newai.Map] = append(database.DungeonsAiByMap[serverID][newai.Map], newai)
			database.DungeonsTest[newai.ID] = newai
			database.AIMutex.Lock()
			database.AIs[newai.ID] = newai
			database.AIMutex.Unlock()
			DungeonCount := database.DungeonsByMap[newai.Server][newai.Map] + 1
			database.DungeonsByMap[newai.Server][newai.Map] = DungeonCount
			go newai.Handler()
		}
	}
}

func randFloats(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}
func findRightBoss(char *database.Character) int {
	if char.Level >= 50 && char.Level < 60 {
		return 13000234
	} else if char.Level >= 60 && char.Level < 70 {
		return 13000239 //13000239
	} else if char.Level >= 70 && char.Level < 80 {
		return 13000244 //13000244
	} else if char.Level >= 80 && char.Level < 90 {
		return 13000249 //13000249
	} else if char.Level >= 90 && char.Level < 100 {
		return 13000254 //13000254
	}
	return 0
}
func BossSpawn(serverID int, char *database.Character) {
	NPCsSpawnPoint := "211,235"
	mobInt := findRightBoss(char)
	npcPos := &database.NpcPosition{ID: len(database.NPCPos), NPCID: int(mobInt), MapID: DungeonMapID, Rotation: 0, Attackable: true, IsNPC: false, RespawnTime: 30, Count: 30, MinLocation: "120,120", MaxLocation: "150,150"}
	database.NPCPos = append(database.NPCPos, npcPos)
	npc, _ := database.NPCs[mobInt]
	newai := &database.AI{ID: len(database.AIs), HP: npc.MaxHp, Map: DungeonMapID, PosID: npcPos.ID, RunningSpeed: 10, Server: serverID, WalkingSpeed: 5, Once: true}
	newai.OnSightPlayers = make(map[int]interface{})
	coordinate := database.ConvertPointToLocation(NPCsSpawnPoint)
	randomLocX := randFloats(coordinate.X, coordinate.X+30)
	randomLocY := randFloats(coordinate.Y, coordinate.Y+30)
	loc := utils.Location{X: randomLocX, Y: randomLocY}
	npcPos.MinLocation = fmt.Sprintf("%.1f,%.1f", randomLocX, randomLocY)
	maxX := randomLocX + 50
	maxY := randomLocY + 50
	npcPos.MaxLocation = fmt.Sprintf("%.1f,%.1f", maxX, maxY)
	newai.Coordinate = loc.String()
	server.GenerateIDForAI(newai)
	newai.Handler = newai.AIHandler
	database.AIsByMap[newai.Server][newai.Map] = append(database.AIsByMap[newai.Server][newai.Map], newai)
	database.DungeonsAiByMap[serverID][newai.Map] = append(database.DungeonsAiByMap[serverID][newai.Map], newai)
	database.DungeonsTest[newai.ID] = newai
	database.AIMutex.Lock()
	database.AIs[newai.ID] = newai
	database.AIMutex.Unlock()
	DungeonCount := database.DungeonsByMap[newai.Server][newai.Map] + 1
	database.DungeonsByMap[newai.Server][newai.Map] = DungeonCount
	go newai.Handler()
}

func FindTheNumber(s *database.Socket) {
	s.Character.CanTip = 2
	s.Conn.Write(messaging.InfoMessage(fmt.Sprintf("Congratulations! Now guess the Boss's Favourite Number [1 - 10] Type:/number [no]")))
	if s.Character.GeneratedNumber == 0 {
		min := 1
		max := 10
		s.Character.GeneratedNumber = rand.Intn(max-min) + min
	}

}

func MobsCreate(mobsID []int, serverID int) {
	NPCsSpawnPoint := []string{"97,339", "343,89", "211,235", "283,319", "393,383", "347,123", "413,365"}
	for _, action := range mobsID {
		randomInt := rand.Intn(len(NPCsSpawnPoint))
		for i := 0; i < int(20); i++ {
			npcPos := &database.NpcPosition{ID: len(database.NPCPos), NPCID: int(action), MapID: DungeonMapID, Rotation: 0, Attackable: true, IsNPC: false, RespawnTime: 30, Count: 30, MinLocation: "120,120", MaxLocation: "150,150"}
			database.NPCPos = append(database.NPCPos, npcPos)
			npc, _ := database.NPCs[action]
			newai := &database.AI{ID: len(database.AIs), HP: npc.MaxHp, Map: DungeonMapID, PosID: npcPos.ID, RunningSpeed: 10, Server: serverID, WalkingSpeed: 5, Once: true}
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
			server.GenerateIDForAI(newai)
			newai.Handler = newai.AIHandler
			database.AIsByMap[newai.Server][newai.Map] = append(database.AIsByMap[newai.Server][newai.Map], newai)
			database.DungeonsAiByMap[serverID][newai.Map] = append(database.DungeonsAiByMap[serverID][newai.Map], newai)
			database.DungeonsTest[newai.ID] = newai
			database.AIMutex.Lock()
			database.AIs[newai.ID] = newai
			database.AIMutex.Unlock()
			DungeonCount := database.DungeonsByMap[newai.Server][newai.Map] + 1
			database.DungeonsByMap[newai.Server][newai.Map] = DungeonCount
			go newai.Handler()
		}
	}
}

func MobScrollUse(mobInt int, c *database.Character) {
	npcPos := &database.NpcPosition{ID: len(database.NPCPos), NPCID: mobInt, MapID: c.Map, Rotation: 0, Attackable: true, IsNPC: false, RespawnTime: 30, Count: 1, MinLocation: "120,120", MaxLocation: "150,150"}
	database.NPCPos = append(database.NPCPos, npcPos)
	npc, _ := database.NPCs[mobInt]
	database.AIMutex.Lock()
	allAI := database.AIs
	database.AIMutex.Unlock()
	r := funk.Map(allAI, func(k int, v *database.AI) int {
		return v.ID
	})
	maxAIID := funk.MaxInt(r.([]int)).(int)
	newai := &database.AI{ID: maxAIID + 1, HP: npc.MaxHp, Map: c.Map, PosID: npcPos.ID, RunningSpeed: 10, Server: c.Socket.User.ConnectedServer, WalkingSpeed: 5, Once: true}
	newai.OnSightPlayers = make(map[int]interface{})
	ownerPos := database.ConvertPointToLocation(c.Coordinate)
	npcPos.MinLocation = fmt.Sprintf("%.1f,%.1f", ownerPos.X+10, ownerPos.Y)
	maxX := ownerPos.X + 20
	maxY := ownerPos.Y + 20
	npcPos.MaxLocation = fmt.Sprintf("%.1f,%.1f", maxX, maxY)
	newai.Coordinate = npcPos.MinLocation
	server.GenerateIDForAI(newai)
	newai.Handler = newai.AIHandler
	database.AIsByMap[newai.Server][newai.Map] = append(database.AIsByMap[newai.Server][newai.Map], newai)
	database.AIMutex.Lock()
	database.AIs[newai.ID] = newai
	database.AIMutex.Unlock()
	go newai.Handler()
}
