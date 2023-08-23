package event

import (
	"PsiHero/database"
	"PsiHero/server"
	"PsiHero/utils"
	"fmt"
	"math/rand"
)

func randFloats(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}
func MobsCreate(mobsID []int, serverID int, mapID int, minlocation string, maxlocation string, respawntime int64, eventid int) {
	minCoordinate := database.ConvertPointToLocation(minlocation)
	maxCoordinate := database.ConvertPointToLocation(maxlocation)
	for _, action := range mobsID {
		fmt.Println("RespawnTime: ", respawntime)
		npcPos := &database.NpcPosition{ID: len(database.NPCPos), NPCID: int(action), MapID: int16(mapID), Rotation: 0, Attackable: true, IsNPC: false, RespawnTime: int(respawntime), Count: 30, MinLocation: minlocation, MaxLocation: maxlocation}
		database.NPCPos[npcPos.ID] = npcPos
		npc, _ := database.NPCs[action]
		database.MakeAnnouncement(fmt.Sprintf("%s has been roaring.", npc.Name))
		newai := &database.AI{ID: len(database.AIs), HP: npc.MaxHp, Map: int16(mapID), PosID: npcPos.ID, RunningSpeed: 10, Server: serverID, WalkingSpeed: 5, Once: false, EventID: eventid}
		newai.OnSightPlayers = make(map[int]interface{})
		randomLocX := randFloats(minCoordinate.X, maxCoordinate.X)
		randomLocY := randFloats(minCoordinate.Y, maxCoordinate.Y)
		loc := utils.Location{X: randomLocX, Y: randomLocY}
		npcPos.MinLocation = fmt.Sprintf("%.1f,%.1f", randomLocX, randomLocY)
		maxX := randomLocX + 50
		maxY := randomLocY + 50
		npcPos.MaxLocation = fmt.Sprintf("%.1f,%.1f", maxX, maxY)
		newai.Coordinate = loc.String()
		server.GenerateIDForAI(newai)
		newai.Handler = newai.AIHandler

		database.AIsByMap[newai.Server][newai.Map] = append(database.AIsByMap[newai.Server][newai.Map], newai)
		database.AIs[newai.ID] = newai
		go newai.Handler()

	}
}
