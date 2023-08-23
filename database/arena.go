package database

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"PsiHero/messaging"
	"PsiHero/nats"
	"PsiHero/utils"

	"github.com/thoas/go-funk"
)

var lowRewardsForNonDivine = [...]*InventorySlot{{ItemID: 221, Quantity: 1}, {ItemID: 221, Quantity: 2}}    //YOU CAN WRITE MORE
var mediumRewardsForNonDivine = [...]*InventorySlot{{ItemID: 222, Quantity: 1}, {ItemID: 222, Quantity: 2}} //YOU CAN WRITE MORE
var highRewardsForNonDivine = [...]*InventorySlot{{ItemID: 223, Quantity: 1}, {ItemID: 223, Quantity: 2}}   //YOU CAN WRITE MORE
var lowRewardsForDivine = [...]*InventorySlot{{ItemID: 224, Quantity: 1}, {ItemID: 224, Quantity: 2}}       //YOU CAN WRITE MORE
var mediumRewardsForDivine = [...]*InventorySlot{{ItemID: 225, Quantity: 1}, {ItemID: 225, Quantity: 2}}    //YOU CAN WRITE MORE
var highRewardsForDivine = [...]*InventorySlot{{ItemID: 226, Quantity: 1}, {ItemID: 226, Quantity: 2}}      //YOU CAN WRITE MORE
var lowRewardsForDarkness = [...]*InventorySlot{{ItemID: 224, Quantity: 1}, {ItemID: 224, Quantity: 2}}     //YOU CAN WRITE MORE
var mediumRewardsForDarkness = [...]*InventorySlot{{ItemID: 224, Quantity: 1}, {ItemID: 224, Quantity: 2}}  //YOU CAN WRITE MORE
var highRewardsForDarkness = [...]*InventorySlot{{ItemID: 225, Quantity: 1}, {ItemID: 225, Quantity: 2}}    //YOU CAN WRITE MORE

var RankForNonDivine = [...]*Ranks{{ID: 50, RankPoints: 1100}, {ID: 30, RankPoints: 1250}, {ID: 14, RankPoints: 1500}, {ID: 4, RankPoints: 1750}, {ID: 2, RankPoints: 2100}, {ID: 1, RankPoints: 2400}}
var RankForDivine = [...]*Ranks{{ID: 50, RankPoints: 1100}, {ID: 30, RankPoints: 1250}, {ID: 14, RankPoints: 1500}, {ID: 4, RankPoints: 1750}, {ID: 2, RankPoints: 2100}, {ID: 1, RankPoints: 2400}}
var RankForDarkness = [...]*Ranks{{ID: 50, RankPoints: 1100}, {ID: 30, RankPoints: 1250}, {ID: 14, RankPoints: 1500}, {ID: 4, RankPoints: 1750}, {ID: 2, RankPoints: 2100}, {ID: 1, RankPoints: 2400}}

const (
	//NON-DIVINE
	NonDivineMediumRewardsPoints = 1000 //IF PLAYER POINTS BIGGER THEN 1000 THEN IT WILL WIN THE MEDIUM REWARDS
	NonDivineHighRewardsPoints   = 2000
	//DIVINE
	DivineMediumRewardsPoints = 1000
	DivineHighRewardsPoints   = 2000
	//DARKNESS
	DarknessMediumRewardsPoints = 1000
	DarknessHighRewardsPoints   = 2000

	//ROUND TIMER
	RoundInSec = 120 //2 min for 1 round

)

var (
	ARENA_TIMER = utils.Packet{0xAA, 0x55, 0x08, 0x00, 0x65, 0x03, 0x00, 0x00, 0x55, 0xAA}
	ARENA_END   = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x65, 0x06, 0x0a, 0x00, 0x00, 0x00, 0x55, 0xAA}
)

type Ranks struct {
	ID         int
	RankPoints int
	Name       string
}
type Arena struct {
	ID               int
	ServerID         int
	TeamA            *Team
	TeamB            *Team
	GameNumber       int
	WaitForNextRound bool
	IsFinished       bool
	RoundTime        int
}
type Team struct {
	players   []*Character
	Wins      int
	AvgPoints int
}

type ArenaLobby struct {
	Characters map[int]*Character
	Parties    map[string]*Party
	Charlock   sync.Mutex
	Partylock  sync.Mutex
}
type Queue struct {
	ID               int
	PartyA           *Party
	PartyB           *Party
	PartyACharacters []*Character
	PartyBCharacters []*Character
	AvgPointsA       int
	AvgPointsB       int
	RemainingTime    int
}

var (
	Arenas              = make(map[int]*Arena)
	Queues              = make(map[int]*Queue)
	LevelTypeArenaLobby = make(map[int]*ArenaLobby) //0: NON-DIVINE, 1: DIVINE, 2: DARKNESS
	needInLobby         = 4                         // 6 because 3-3 each team // FOR TEST I MADE IT 2
	ArenaWinNumber      = 3                         //3 wins need to somebody win the arena
	arenaPlayerMutex    sync.RWMutex
	arenaMutex          sync.RWMutex
	queuesMutex         sync.RWMutex
	ArenaTeamSize       = 2 //Players in one team if 2 then 2v2
	PARTY_REQUEST       = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x52, 0x04, 0x0A, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x04, 0xFF, 0xFF, 0xFF, 0xFF, 0x03, 0x55, 0xAA}
	ArenaZones          = []int16{253}
	HonorPointsChange   = 50 //HERE YOU CAN CHANGE HONOR POINTS FOR WINNING/LOSING THE ARENA
	DeserterPunishment  = 100
)

func init() {
	LevelTypeArenaLobby[0] = &ArenaLobby{}
	LevelTypeArenaLobby[1] = &ArenaLobby{}
	LevelTypeArenaLobby[2] = &ArenaLobby{}
	LevelTypeArenaLobby[0].Characters = make(map[int]*Character)
	LevelTypeArenaLobby[1].Characters = make(map[int]*Character)
	LevelTypeArenaLobby[2].Characters = make(map[int]*Character)
	LevelTypeArenaLobby[0].Parties = make(map[string]*Party)
	LevelTypeArenaLobby[1].Parties = make(map[string]*Party)
	LevelTypeArenaLobby[2].Parties = make(map[string]*Party)
}

func (char *Character) GiveRankByHonorPoints() (int, int) {
	switch true {
	case char.Level < 101:

		r := funk.Filter(RankForNonDivine, func(x *Ranks) bool {
			return char.Socket.Stats.Honor >= x.RankPoints
		}).([]*Ranks)
		sort.Slice(r, func(i, j int) bool {
			return r[i].RankPoints > r[j].RankPoints
		})
		if len(r) > 0 {
			return 50, r[0].ID
		} else {
			return 50, 0
		}

	case char.Level >= 101 && char.Level < 201: //DIVINE
		r := funk.Filter(RankForDivine, func(x *Ranks) bool {
			return char.Socket.Stats.Honor >= x.RankPoints
		}).([]*Ranks)
		sort.Slice(r, func(i, j int) bool {
			return r[i].RankPoints > r[j].RankPoints
		})
		if len(r) > 0 {
			return 60, r[0].ID
		} else {
			return 60, 0
		}
	case char.Level >= 201: //DARKNESS

		r := funk.Filter(RankForDarkness, func(x *Ranks) bool {
			return char.Socket.Stats.Honor >= x.RankPoints
		}).([]*Ranks)
		sort.Slice(r, func(i, j int) bool {
			return r[i].RankPoints > r[j].RankPoints
		})
		if len(r) > 0 {
			return 70, r[0].ID
		} else {
			return 70, 0
		}

	}
	return 50, 0
}
func (a *Team) RemovePlayer(char *Character, team *Team) {
	//delete player from a.players []*Character
	a.players = funk.Filter(a.players, func(c *Character) bool { return c.ID != char.ID }).([]*Character)
}

//RemovePlayerFromArena
func RemovePlayerFromArena(char *Character) {
	arenaMutex.Lock()
	defer arenaMutex.Unlock()
	allarena := funk.Values(Arenas).([]*Arena)
	filter := funk.Filter(allarena, func(a *Arena) bool {
		return funk.Contains(a.TeamA.players, char) || funk.Contains(a.TeamB.players, char)
	}).([]*Arena)
	if len(filter) > 0 {
		arena := filter[0]
		if funk.Contains(arena.TeamA.players, char) {
			arena.TeamA.RemovePlayer(char, arena.TeamA)
		} else {
			arena.TeamB.RemovePlayer(char, arena.TeamB)
		}
	}
}

//FIND PLAYER IN ARENA
func FindPlayerInArena(char *Character) bool {
	arenaMutex.Lock()
	defer arenaMutex.Unlock()
	allarena := funk.Values(Arenas).([]*Arena)
	filter := funk.Filter(allarena, func(a *Arena) bool {
		return funk.Contains(a.TeamA.players, char) || funk.Contains(a.TeamB.players, char)
	}).([]*Arena)
	return len(filter) > 0
}

//FIND PLAYER ARENA
func FindPlayerArena(char *Character) *Arena {
	arenaMutex.Lock()
	defer arenaMutex.Unlock()
	allarena := funk.Values(Arenas).([]*Arena)
	filter := funk.Filter(allarena, func(a *Arena) bool {
		return funk.Contains(a.TeamA.players, char) || funk.Contains(a.TeamB.players, char)
	}).([]*Arena)
	return filter[0]
}

func (a *Team) GetPlayerParty() *Party {
	for _, c := range a.players {
		party := FindParty(c)
		if party == nil {
			continue
		}
		return party
	}
	return nil
}
func RemovePartyFromLobby(char *Character) {
	if char != nil {
		arenaType := GetArenaTypeByLevelType(char.Level)
		if arenaType >= 0 {
			LevelTypeArenaLobby[arenaType].Partylock.Lock()
			defer LevelTypeArenaLobby[arenaType].Partylock.Unlock()
			delete(LevelTypeArenaLobby[arenaType].Parties, char.PartyID)
			fmt.Println("Removed from arenaType: ", arenaType)
		}
	}
}
func FindEmptyServerArena() int {
	for i := 1; i < SERVER_COUNT; i++ {
		charsInServer, _ := FindCharactersInServer(i)
		if len(charsInServer) == 0 {
			return i
		}
	}
	return 0
}

//Team win the arena
func (t *Team) TeamGiveWinReward() {
	//PLAYERS WHO WIN THE ARENA
	for _, player := range t.players {
		if player == nil || player.Socket == nil || !player.IsOnline {
			continue
		}
		player.Socket.Stats.Honor += HonorPointsChange
		player.Update()
		player.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You acquired %d Honor points.", HonorPointsChange)))
		stat, _ := player.GetStats()
		player.Socket.Write(stat)
		winnedItems := t.TeamGiveItemReward(player)
		if len(winnedItems) > 0 {
			slots, err := player.InventorySlots()
			if err != nil {
				fmt.Println("Give Team Reward item give fail: ", err)
				return
			}
			resp := utils.Packet{}
			for i := 0; i < len(winnedItems); i++ {
				reward := winnedItems[i]
				_, slot, _ := player.AddItem(reward, -1, true)
				if slot > 0 {
					resp.Concat(slots[slot].GetData(slot))
					player.Socket.Write(resp)
				} else {
					player.Socket.Write(messaging.InfoMessage("Not enough space in your inventory."))
				}
			}
		}
	}

}
func (t *Team) TeamGiveItemReward(player *Character) []*InventorySlot {
	arenaType := GetArenaTypeByLevelType(player.Level)
	if arenaType >= 0 {
		switch arenaType {
		case 0: //LEVEL 50+
			if player.Socket.Stats.Honor < NonDivineMediumRewardsPoints {
				return lowRewardsForNonDivine[:]
			} else if player.Socket.Stats.Honor >= NonDivineMediumRewardsPoints && player.Socket.Stats.Honor < NonDivineHighRewardsPoints {
				return mediumRewardsForNonDivine[:]
			} else if player.Socket.Stats.Honor >= NonDivineHighRewardsPoints {
				return highRewardsForNonDivine[:]
			}
		case 1:
			if player.Socket.Stats.Honor < DivineMediumRewardsPoints {
				return lowRewardsForDivine[:]
			} else if player.Socket.Stats.Honor >= DivineMediumRewardsPoints && player.Socket.Stats.Honor < DivineHighRewardsPoints {
				return mediumRewardsForDivine[:]
			} else if player.Socket.Stats.Honor >= DivineHighRewardsPoints {
				return highRewardsForDivine[:]
			}
		case 2:
			if player.Socket.Stats.Honor < DarknessMediumRewardsPoints {
				return lowRewardsForDarkness[:]
			} else if player.Socket.Stats.Honor >= DarknessMediumRewardsPoints && player.Socket.Stats.Honor < DarknessHighRewardsPoints {
				return mediumRewardsForDarkness[:]
			} else if player.Socket.Stats.Honor >= DarknessHighRewardsPoints {
				return highRewardsForDarkness[:]
			}
		}
	}
	return []*InventorySlot{}
}

//Team lose the arena
func (t *Team) TeamGiveLoseReward() {
	//PLAYERS WHO LOSE THE ARENA
	for _, player := range t.players {
		if player == nil {
			continue
		}
		if player.Socket.Stats.Honor-HonorPointsChange < 0 {
			player.Socket.Stats.Honor = 0
		} else {
			player.Socket.Stats.Honor -= HonorPointsChange
		}
		player.Update()
		player.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have lost %d Honor points.", HonorPointsChange)))
		stat, _ := player.GetStats()
		player.Socket.Write(stat)
	}
}
func PlayerDeserterPunishment(player *Character) {
	player.Socket.Stats.Honor -= DeserterPunishment
	player.Update()
}

func (a *Arena) StartNextRound(remainingTime int) {

	var msg string
	if remainingTime == 0 {
		a.StartRound()

	} else {
		msg = fmt.Sprintf("Your round will be start %d seconds later.", remainingTime)
		time.AfterFunc(time.Second, func() {
			if a.WaitForNextRound {
				a.StartNextRound(remainingTime - 1)
			}
		})
	}

	info := MakeAnnouncementForArena(msg)
	for _, c := range a.TeamA.players {
		if c.Socket != nil {
			c.Socket.Write(info)
		}
	}
	for _, c := range a.TeamB.players {
		if c.Socket != nil {
			c.Socket.Write(info)
		}
	}

}

func MakeAnnouncementForArena(msg string) []byte {
	length := int16(len(msg) + 3)

	resp := ANNOUNCEMENT
	resp.SetLength(length)
	resp[6] = byte(len(msg))
	resp.Insert([]byte(msg), 7)
	return resp
}

//teleport players out from arena func
func (a *Arena) ArenaEndTeleportOut() {
	for _, c := range a.TeamA.players {
		if c == nil || c.Socket == nil || !c.IsOnline || !funk.Contains(ArenaZones, c.Map) {
			continue
		}
		if c.Socket != nil {
			resp := utils.Packet{}
			c.Socket.User.ConnectedServer = 1
			c.IsActive = false
			c.Socket.Stats.HP = c.Socket.Stats.MaxHP
			c.Socket.Stats.CHI = c.Socket.Stats.MaxCHI
			c.Respawning = false
			hpData := c.GetHPandChi()
			resp.Concat(hpData)
			c.Socket.Write(ARENA_END)
			tp, _ := c.ChangeMap(1, nil)
			resp.Concat(tp)
			c.Socket.Write(resp)
			p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.PLAYER_RESPAWN}
			p.Cast()
		}
	}
	for _, c := range a.TeamB.players {
		if c == nil || c.Socket == nil || !c.IsOnline || !funk.Contains(ArenaZones, c.Map) {
			continue
		}
		if c.Socket != nil {
			resp := utils.Packet{}
			c.Socket.User.ConnectedServer = 1
			c.IsActive = false
			c.Socket.Stats.HP = c.Socket.Stats.MaxHP
			c.Socket.Stats.CHI = c.Socket.Stats.MaxCHI
			c.Respawning = false
			hpData := c.GetHPandChi()
			resp.Concat(hpData)
			c.Socket.Write(ARENA_END)
			tp, _ := c.ChangeMap(1, nil)
			resp.Concat(tp)
			c.Socket.Write(resp)
			p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.PLAYER_RESPAWN}
			p.Cast()
		}
	}
	arenaMutex.Lock()
	defer arenaMutex.Unlock()
	delete(Arenas, a.ID)
}

func (a *Arena) ArenaEndTimer(remainingTime int) {

	var msg string
	if remainingTime == 0 {
		a.ArenaEndTeleportOut()

	} else {
		// you will be teleported out from arena in 10 seconds fmt.sprintf
		msg = fmt.Sprintf("You will be teleported out from arena in %d seconds.", remainingTime)

		time.AfterFunc(time.Second, func() {
			a.ArenaEndTimer(remainingTime - 1)
		})
	}

	info := MakeAnnouncementForArena(msg)
	for _, c := range a.TeamA.players {
		if c.Socket != nil {
			c.Socket.Write(info)
		}
	}
	for _, c := range a.TeamB.players {
		if c.Socket != nil {
			c.Socket.Write(info)
		}
	}

}

func (a *Arena) RoundTimer() {
	a.RoundTime -= 1
	resp := ARENA_TIMER
	index := 8
	resp.Insert(utils.IntToBytes(uint64(a.TeamA.Wins), 4, true), index)
	index += 4
	resp.Insert(utils.IntToBytes(uint64(2), 4, true), index)
	index += 4
	resp.Insert(utils.IntToBytes(uint64(0), 6, true), index)
	index += 6
	resp.Insert(utils.IntToBytes(uint64(a.TeamB.Wins), 4, true), index)
	index += 4
	resp.Insert(utils.IntToBytes(uint64(2), 4, true), index)
	index += 4
	resp.Insert(utils.IntToBytes(uint64(0), 5, true), index)
	index += 5
	resp.Insert(utils.IntToBytes(uint64(a.RoundTime), 4, true), index)
	resp.SetLength(int16(binary.Size(resp) - 6))
	for _, c := range a.TeamA.players {
		if c.Socket != nil {
			c.Socket.Write(resp)
		}
	}
	for _, c := range a.TeamB.players {
		if c.Socket != nil {
			c.Socket.Write(resp)
		}
	}

}

//Start arena round
func (a *Arena) StartRound() {
	if a.TeamA == nil || a.TeamB == nil {
		fmt.Println("Arena team is nil")
		return
	}

	rmessage := utils.Packet{}
	rmessage.Concat(MakeAnnouncementForArena(" "))
	rmessage.Concat(MakeAnnouncementForArena(" "))
	rmessage.Concat(MakeAnnouncementForArena(" "))
	teamALocation := &utils.Location{X: 0, Y: 0}
	teamBLocation := &utils.Location{X: 0, Y: 0}
	if a.GameNumber%2 == 0 {
		teamALocation = &utils.Location{X: 255, Y: 165}
		teamBLocation = &utils.Location{X: 255, Y: 353}
	} else {
		teamALocation = &utils.Location{X: 255, Y: 353}
		teamBLocation = &utils.Location{X: 255, Y: 165}
	}
	a.WaitForNextRound = false
	a.RoundTime = RoundInSec
	//for A team players
	for _, player := range a.TeamA.players {
		resp := utils.Packet{}
		resp.Concat(rmessage)
		player.Socket.User.ConnectedServer = a.ServerID
		player.IsActive = false
		player.Socket.Stats.HP = player.Socket.Stats.MaxHP
		player.Socket.Stats.CHI = player.Socket.Stats.MaxCHI
		player.Respawning = false
		hpData := player.GetHPandChi()
		resp.Concat(hpData)
		tp, _ := player.ChangeMap(253, teamALocation)
		resp.Concat(tp)
		player.Socket.Write(resp)
		p := nats.CastPacket{CastNear: true, CharacterID: player.ID, Type: nats.PLAYER_RESPAWN}
		p.Cast()
	}
	//for B team players
	for _, player := range a.TeamB.players {
		resp := utils.Packet{}
		resp.Concat(rmessage)
		player.Socket.User.ConnectedServer = a.ServerID
		player.IsActive = false
		player.Socket.Stats.HP = player.Socket.Stats.MaxHP
		player.Socket.Stats.CHI = player.Socket.Stats.MaxCHI
		player.Respawning = false
		hpData := player.GetHPandChi()
		resp.Concat(hpData)
		tp, _ := player.ChangeMap(253, teamBLocation)
		resp.Concat(tp)
		player.Socket.Write(resp)
		p := nats.CastPacket{CastNear: true, CharacterID: player.ID, Type: nats.PLAYER_RESPAWN}
		p.Cast()
	}
	a.RoundTimer()
}
func JoinToArenaLobby(char *Character, arenaType int, isSolo bool) {
	if isSolo {
		if _, ok := LevelTypeArenaLobby[arenaType].Characters[char.ID]; !ok {
			LevelTypeArenaLobby[arenaType].Charlock.Lock()
			defer LevelTypeArenaLobby[arenaType].Charlock.Unlock()
			LevelTypeArenaLobby[arenaType].Characters[char.ID] = char
		}
	} else {
		//HERE WE CAN MAKE THE PARTY JOIN
		party := FindParty(char)
		if party == nil {
			return
		}
		if _, ok := LevelTypeArenaLobby[arenaType].Parties[char.PartyID]; !ok { //PARTY LEADER!!!
			LevelTypeArenaLobby[arenaType].Partylock.Lock()
			defer LevelTypeArenaLobby[arenaType].Partylock.Unlock()
			LevelTypeArenaLobby[arenaType].Parties[char.PartyID] = party
		}
	}

}
func GetArenaTypeByLevelType(level int) int {
	if level >= 50 && level <= 100 {
		return 0
	} else if level > 100 && level <= 200 {
		return 1
	} else if level > 200 && level <= 300 {
		return 2
	}
	return -1
}
func TryToCreateSession() {
	fmt.Println("Characters: ", len(LevelTypeArenaLobby[0].Characters), " Parties: ", len(LevelTypeArenaLobby[0].Parties))
	if len(LevelTypeArenaLobby[0].Characters) >= needInLobby {
		_, err := FindOpponentsTest(LevelTypeArenaLobby[0].Characters)
		if err != nil {
			fmt.Println("Error: ", err)
		}
	}
	for i := 0; i < len(LevelTypeArenaLobby); i++ {
		if len(LevelTypeArenaLobby[i].Parties) >= 2 {
			FindOpponentsToParty(LevelTypeArenaLobby[i].Parties)
		}
	}
	//HERE WE TRY TO MAKE THE PARTY PVP

}

func sumPoints(players []*Character) int {
	var totalPoints int
	for _, player := range players {
		totalPoints += int(player.Socket.Stats.Honor)
	}
	return totalPoints
}
func avaragePoints(players []*Character) int {
	var totalPoints int
	for _, player := range players {
		totalPoints += int(player.Socket.Stats.Honor)
	}
	return totalPoints / len(players)
}
func (p *Party) GetCharactersFromParty() interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	//get the characters from p.Members
	characters := make(map[int]*Character)
	for _, member := range p.Members {
		characters[member.ID] = member.Character
	}
	characters[p.Leader.ID] = p.Leader
	return characters
}

func FindOpponentsToSolo(characters map[int]*Character) (*Arena, error) {

	arenaPlayerMutex.RLock()
	players := funk.Values(characters).([]*Character)
	arenaPlayerMutex.RUnlock()
	sort.Slice(players, func(i, j int) bool {
		return players[i].Socket.Stats.Honor > players[j].Socket.Stats.Honor
	})

	n := len(players)
	if n < ArenaTeamSize {
		return nil, fmt.Errorf("Need at least %d players to make a match", ArenaTeamSize)
	}

	var teams []*Team
	var team *Team
	for i := 0; i < n; i += ArenaTeamSize {
		fmt.Println("I: ", i)
		team = &Team{}
		for j := i; j < i+ArenaTeamSize; j++ {
			fmt.Println("J: ", j)
			team.players = append(team.players, players[j])
		}
		teams = append(teams, team)
	}

	for _, t := range teams {
		for i := 0; i < ArenaTeamSize; i++ {
			for j := i + 1; j < ArenaTeamSize; j++ {
				if math.Abs(float64(t.players[i].Socket.Stats.Honor)-float64(t.players[j].Socket.Stats.Honor)) > 100 {
					return nil, fmt.Errorf("Could not find a match with less than 100 rank point difference within a team.")
				}
			}
		}
	}
	SendInvitesToPlayers(teams[0].players, teams[1].players, false, 0, 0)
	return nil, nil
}
func FindOpponentsTest(characters map[int]*Character) (*Arena, error) {
	arenaPlayerMutex.RLock()
	sortedPlayers := funk.Values(characters).([]*Character)
	arenaPlayerMutex.RUnlock()
	sort.Slice(sortedPlayers, func(i, j int) bool {
		return sortedPlayers[i].Socket.Stats.Honor > sortedPlayers[j].Socket.Stats.Honor
	})
	for len(sortedPlayers) > 0 {
		team1 := &Team{}
		team2 := &Team{}
		team1Points := 0
		team2Points := 0
		for i := 0; i < ArenaTeamSize*2; i++ {
			if len(sortedPlayers) == 0 {
				break
			}
			player := sortedPlayers[0]
			if i%2 == 0 {
				team1.players = append(team1.players, player)
			} else {
				team2.players = append(team2.players, player)

			}
			sortedPlayers = sortedPlayers[1:]
		}
		team1Points = avaragePoints(team1.players)
		team2Points = avaragePoints(team2.players)
		fmt.Println("TeamPoints1: ", team1Points, " TeamPoints2: ", team2Points)
		if math.Abs(float64(team1Points)-float64(team2Points)) <= 100 {
			SendInvitesToPlayers(team1.players, team2.players, false, team1Points, team2Points)
			return nil, fmt.Errorf("Good")
		} else {
			return nil, fmt.Errorf("Could not find a match with less than 100 rank point difference within a team.")
		}
	}
	return nil, nil
}
func FindOpponentsToParty(partiesMap map[string]*Party) (*Arena, error) {

	arenaPlayerMutex.RLock()
	parties := funk.Values(partiesMap).([]*Party)
	arenaPlayerMutex.RUnlock()
	sort.Slice(parties, func(i, j int) bool { //SORT PARTIES BY POINTS
		characterMapI := parties[i].GetCharactersFromParty()
		parties[i].mutex.RLock()
		charactersI := funk.Values(characterMapI).([]*Character)
		parties[i].mutex.RUnlock()
		characterMapJ := parties[j].GetCharactersFromParty()
		parties[j].mutex.RLock()
		charactersJ := funk.Values(characterMapJ).([]*Character)
		parties[j].mutex.RUnlock()
		return avaragePoints(charactersI) > avaragePoints(charactersJ)
	})
	for len(parties) > 0 {
		team1 := &Team{}
		team2 := &Team{}
		team1Points := 0
		team2Points := 0
		team1.players = append(team1.players, parties[0].Leader)
		team2.players = append(team2.players, parties[1].Leader)
		for i := 0; i < 2; i++ {
			for _, member := range parties[i].Members {
				if member.Socket == nil || !member.Accepted || !member.IsOnline {
					continue
				}
				if i == 0 {
					team1.players = append(team1.players, member.Character)
				} else {
					team2.players = append(team2.players, member.Character)
				}
			}

		}

		team1Points = avaragePoints(team1.players)
		team2Points = avaragePoints(team2.players)
		fmt.Println("TeamPoints1: ", team1Points, " TeamPoints2: ", team2Points)
		if math.Abs(float64(team1Points)-float64(team2Points)) <= 2000 {
			SendInvitesToPlayers(team1.players, team2.players, true, team1Points, team2Points)
			return nil, fmt.Errorf("Good")
		} else {
			return nil, fmt.Errorf("Could not find a match with less than 100 rank point difference within a team.")
		}
	}
	return nil, nil
}

func SendInvitesToPlayers(teamA, teamB []*Character, teamJoin bool, teamAPoints, teamBPoints int) {
	teamAParty := &Party{}
	teamBParty := &Party{}

	for i, char := range teamA { //NON-DIVINE
		fmt.Println("SendInvitesA: ", char.Name, " HonorPoints: ", char.Socket.Stats.Honor)
		party := FindParty(char)
		if party == nil && !teamJoin {
			if i == 0 {
				party = &Party{}
				party.Leader = char
				char.PartyID = char.UserID
				party.ArenaParty = true
				party.Create()
				party.PartyMode = party.Leader.PartyMode
				party.PartyLeaderAccept = false
				teamAParty = party
				m := &PartyMember{Character: char, Accepted: false}
				teamAParty.AddMember(m)
			} else {

				char.PartyID = teamAParty.Leader.UserID
				m := &PartyMember{Character: char, Accepted: false}
				teamAParty.AddMember(m)
			}
			resp := PARTY_REQUEST
			length := int16(len("[SERVER]Arena") + 6)
			resp.SetLength(length)
			resp[8] = byte(len("[SERVER]Arena"))
			resp[9] = byte(0x21)
			resp.Insert([]byte("[SERVER]Arena"), 9)
			char.Socket.Write(resp)
		} else if teamJoin && party != nil {
			teamAParty = party
			party.ArenaFounded = true
			char.Socket.Write(messaging.InfoMessage("We found a match for you! Write /acceptarena to join the match!"))
		}
	}

	/*TEAM B */
	for i, char := range teamB { //NON-DIVINE
		fmt.Println("SendInvitesB: ", char.Name, " HonorPoints: ", char.Socket.Stats.Honor)
		party := FindParty(char)
		if party == nil && !teamJoin {
			if i == 0 {
				party = &Party{}
				party.Leader = char
				char.PartyID = char.UserID
				party.ArenaParty = true
				party.Create()
				party.PartyMode = party.Leader.PartyMode
				party.PartyLeaderAccept = false
				teamBParty = party
				m := &PartyMember{Character: char, Accepted: false}
				teamBParty.AddMember(m)
			} else {

				char.PartyID = teamBParty.Leader.UserID
				m := &PartyMember{Character: char, Accepted: false}
				teamBParty.AddMember(m)
			}
			resp := PARTY_REQUEST
			length := int16(len("[SERVER]Arena") + 6)
			resp.SetLength(length)
			resp[8] = byte(len("[SERVER]Arena"))
			resp[9] = byte(0x21)
			resp.Insert([]byte("[SERVER]Arena"), 9)
			char.Socket.Write(resp)
		} else if teamJoin && party != nil {
			teamBParty = party
			party.ArenaFounded = true
			char.Socket.Write(messaging.InfoMessage("We found a match for you! Write /acceptarena to join the match!"))

		}
	}
	if !teamJoin { //SOLO JOIN
		time.AfterFunc(20*time.Second, func() {
			allAccepted := true
			for _, m := range teamAParty.Members { //NON-DIVINE
				if !m.Accepted || !teamAParty.PartyLeaderAccept {
					fmt.Println("Somebody not accept")
					teamAParty.RemoveMember(m)
					DeleteLobbyNotAllPlayerAccept(teamAParty, teamBParty, m.Character)
					allAccepted = false
					break
				}
			}
			for _, m := range teamBParty.Members { //NON-DIVINE
				if !m.Accepted || !teamBParty.PartyLeaderAccept {
					fmt.Println("Somebody not accept")
					teamBParty.RemoveMember(m)
					DeleteLobbyNotAllPlayerAccept(teamAParty, teamBParty, m.Character)
					allAccepted = false
					break
				}
			}
			if len(teamAParty.Members) < 2 || len(teamBParty.Members) < 2 || !teamBParty.PartyLeaderAccept || !teamAParty.PartyLeaderAccept {
				DeleteLobbyNotAllPlayerAccept(teamAParty, teamBParty)
				fmt.Println("Somebody not accept")
				allAccepted = false
			}
			if allAccepted {

				TeleportToMap(teamAParty, teamBParty)
			}
		})
	} else if teamJoin {
		queuesMutex.Lock()
		defer queuesMutex.Unlock()
		queue := &Queue{RemainingTime: 20, PartyA: teamAParty, PartyB: teamBParty, AvgPointsA: teamAPoints, AvgPointsB: teamBPoints, PartyACharacters: teamA, PartyBCharacters: teamB, ID: len(Queues)}
		Queues[len(Queues)] = queue
	}

}

//CHECKING LOBBY ALL PLAYERS ACCEPT IF NOT DELETE LOBBY
func (q *Queue) CheckingLobby() {
	allAccepted := true
	for _, m := range q.PartyA.Members { //NON-DIVINE
		if !m.AcceptedArena || !q.PartyA.PartyLeaderAcceptArena {
			allAccepted = false
			break
		}
	}
	for _, m := range q.PartyB.Members { //NON-DIVINE
		if !m.AcceptedArena || !q.PartyB.PartyLeaderAcceptArena {
			allAccepted = false
			break
		}
	}

	fmt.Println("Remaining time: ", q.RemainingTime)
	if q.RemainingTime == 0 {
		for _, m := range q.PartyA.Members { //NON-DIVINE
			if !m.AcceptedArena || !q.PartyA.PartyLeaderAcceptArena {
				fmt.Println("Somebody not accept(A TEAM)")
				DeleteTeamFromLobby(q.PartyA, q.PartyB, q.PartyA)
				allAccepted = false
				break
			}
		}
		for _, m := range q.PartyB.Members { //NON-DIVINE
			if !m.AcceptedArena || !q.PartyB.PartyLeaderAcceptArena {
				fmt.Println("Somebody not accept(B TEAM)")
				DeleteTeamFromLobby(q.PartyA, q.PartyB, q.PartyB)
				allAccepted = false
				break
			}
		}
		if q.PartyA == nil || q.PartyB == nil || !q.PartyB.PartyLeaderAcceptArena || !q.PartyA.PartyLeaderAcceptArena {
			DeleteTeamFromLobby(q.PartyA, q.PartyB)
			fmt.Println("Somebody not accept(ALL TEAM)")
			allAccepted = false
		}
		q.RemainingTime = 0

	} else if allAccepted {
		arenaServer := FindEmptyServerArena()
		if arenaServer == 0 { //ALL SERVER ARE FULL OR SOMETHING IS WRONG
			return
		}
		RemovePartyFromLobby(q.PartyA.Leader)
		RemovePartyFromLobby(q.PartyB.Leader)
		team1 := &Team{players: q.PartyACharacters, Wins: 0} //AVG POINTS DISABLED YET
		team2 := &Team{players: q.PartyBCharacters, Wins: 0} //AVG POINTS DISABLED YET

		//write len teamA and len teamB
		arena := &Arena{TeamA: team1, TeamB: team2, GameNumber: 1, ServerID: arenaServer, ID: len(Arenas), IsFinished: false, WaitForNextRound: true}
		arenaMutex.Lock()
		Arenas[len(Arenas)] = arena
		arenaMutex.Unlock()
		arena.TeleportToRestZone()
		q.RemainingTime = 0
	}

}
func (a *Arena) TeleportToRestZone() {
	teamALocation := &utils.Location{X: 257, Y: 73}
	teamBLocation := &utils.Location{X: 257, Y: 430}
	//for A team players
	for _, player := range a.TeamA.players {
		player.DisableCharacterMorphed()
		resp := utils.Packet{}
		player.Socket.User.ConnectedServer = a.ServerID
		player.IsActive = false
		player.Socket.Stats.HP = player.Socket.Stats.MaxHP
		player.Socket.Stats.CHI = player.Socket.Stats.MaxCHI
		player.Respawning = false
		hpData := player.GetHPandChi()
		resp.Concat(hpData)
		tp, _ := player.ChangeMap(253, teamALocation)
		resp.Concat(tp)
		player.Socket.Write(resp)
		p := nats.CastPacket{CastNear: true, CharacterID: player.ID, Type: nats.PLAYER_RESPAWN}
		p.Cast()
	}
	//for B team players
	for _, player := range a.TeamB.players {
		player.DisableCharacterMorphed()
		resp := utils.Packet{}
		player.Socket.User.ConnectedServer = a.ServerID
		player.IsActive = false
		player.Socket.Stats.HP = player.Socket.Stats.MaxHP
		player.Socket.Stats.CHI = player.Socket.Stats.MaxCHI
		player.Respawning = false
		hpData := player.GetHPandChi()
		resp.Concat(hpData)
		tp, _ := player.ChangeMap(253, teamBLocation)
		resp.Concat(tp)
		player.Socket.Write(resp)
		p := nats.CastPacket{CastNear: true, CharacterID: player.ID, Type: nats.PLAYER_RESPAWN}
		p.Cast()
	}

	bLeader := a.TeamB.GetPlayerParty().Leader
	aLeader := a.TeamA.GetPlayerParty().Leader
	fmt.Println("PartyLeader remove: ", bLeader.Name, " A leader:", aLeader.Name)
	RemovePartyFromLobby(bLeader)
	RemovePartyFromLobby(aLeader)
	a.StartNextRound(30)
}

func DeleteLobbyNotAllPlayerAccept(partyA, partyB *Party, deserter ...*Character) { //MAYBE WE CAN'T SHOW THE TEAMMATES
	if partyA.Leader != nil {
		if partyA.Leader.PartyID != "" {
			partyA.Leader.LeaveParty()
		}
	}
	if partyB.Leader != nil {
		if partyB.Leader.PartyID != "" {
			partyB.Leader.LeaveParty()
		}
	}
	if len(deserter) > 0 {
		if deserter[0] != nil {
			if deserter[0].Socket != nil {
				deserter[0].Socket.Write(messaging.InfoMessage("You didn't accept the Arena. You kicked from lobby!"))
			}
			arenaType := GetArenaTypeByLevelType(deserter[0].Level)
			delete(LevelTypeArenaLobby[arenaType].Characters, deserter[0].ID)
		}
	}

}
func DeleteTeamFromLobby(partyA, partyB *Party, deserter ...*Party) { //MAYBE WE CAN'T SHOW THE TEAMMATES
	if partyA != nil {
		partyA.PartyLeaderAcceptArena = false
		partyA.ArenaFounded = false
		for _, c := range partyA.Members {
			c.AcceptedArena = false
		}
	}

	if partyB != nil {
		partyB.PartyLeaderAcceptArena = false
		partyB.ArenaFounded = false
		for _, c := range partyB.Members {
			c.AcceptedArena = false
		}
	}
	if len(deserter) > 0 {
		if deserter[0] != nil {
			for _, c := range deserter[0].Members {
				c.Socket.Write(messaging.InfoMessage("Your team not accept the Arena. Your team kicked from lobby!"))
			}
			arenaType := GetArenaTypeByLevelType(deserter[0].Leader.Level)
			delete(LevelTypeArenaLobby[arenaType].Parties, deserter[0].Leader.PartyID)
		}
	}

}
func TeleportToMap(teamA, teamB *Party) {
	fmt.Println("Players teleport to arena")
	resp := utils.Packet{}
	//resp.Concat(GetPartyMemberData(teamA.Leader)) // get party leader
	teamA.ArenaFounded = false
	teamB.ArenaFounded = false
	tpA, _ := teamA.Leader.ChangeMap(253, &utils.Location{X: 255, Y: 165})
	teamA.Leader.Socket.Write(tpA)
	for _, member := range teamA.Members { // get all party members for TeamA
		resp.Concat(GetPartyMemberData(member.Character))
		tp, _ := member.ChangeMap(253, &utils.Location{X: 255, Y: 165})
		member.Socket.Write(tp)
	}
	SendInformationToParty(teamA, resp)
	resp = utils.Packet{}
	//resp.Concat(GetPartyMemberData(teamB.Leader)) // get party leader
	tpB, _ := teamB.Leader.ChangeMap(253, &utils.Location{X: 255, Y: 353})
	teamB.Leader.Socket.Write(tpB)
	for _, member := range teamB.Members { // get all party members for TeamB
		resp.Concat(GetPartyMemberData(member.Character))
		tp, _ := member.ChangeMap(253, &utils.Location{X: 255, Y: 353})
		member.Socket.Write(tp)
	}

	SendInformationToParty(teamB, resp)
}

func SendInformationToParty(team *Party, data []byte) {
	if team.PartyLeaderAccept && team.Leader != nil && team.Leader.IsOnline && team.Leader.Socket != nil {
		team.Leader.Socket.Write(data)
	}
	for _, member := range team.Members {
		if member == nil || !member.IsOnline || member.Socket == nil {
			continue
		}
		member.Socket.Write(data)
	}
}
