package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"PsiHero/messaging"
	"PsiHero/utils"

	"github.com/thoas/go-funk"
	gorp "gopkg.in/gorp.v1"
)

const (
	SERVER_COUNT = 100
)

var (
	servers []*Server
)

type Server struct {
	ID       int    `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	MaxUsers int    `db:"max_users" json:"max_users"`
	Epoch    int64  `db:"epoch" json:"epoch"`
}

type ServerItem struct {
	Server
	ConnectedUsers int `json:"conn_users"`
}

func (t *Server) Create() error {
	return db.Insert(t)
}

func (t *Server) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(t)
}

func (t *Server) Update() error {
	_, err := db.Update(t)
	return err
}

func (t *Server) Delete() error {
	_, err := db.Delete(t)
	return err
}
func ServerExploreWorld() {
	ServerExploreDungeons()
	ServerExploreNonSocketPlayers()
	ServerExploreArenas()
	ServerExploreQueues()
	ServerExploreLobbies()
}
func ServerExploreNonSocketPlayers() {
	characterMutex.Lock()
	allChars := funk.Values(characters)
	characterMutex.Unlock()
	characters := funk.Filter(allChars, func(character *Character) bool {

		user, err := FindUserByID(character.UserID)
		if err != nil || user == nil {
			return false
		}

		return character.IsOnline || !character.IsOnline && character.IsinWar
	}).([]*Character)

	for _, char := range characters {
		if char.IsinWar && (!char.IsOnline || char.Socket == nil) {
			if char.Faction == 1 {
				delete(OrderCharacters, char.ID)
			} else {
				delete(ShaoCharacters, char.ID)
			}
		}
		if char.IsOnline && char.Socket == nil {
			fmt.Println("Character with no socket: ", char.Name)
			tryKick := func() {
				//char.Socket.OnClose()
				defer func() {
					if err := recover(); err != nil {
						log.Println(err)
					}
				}()
				char.Socket.OnClose()
				panic("Can't kick character with no socket!")
			}
			tryKick()
		}
	}
}
func ServerExploreQueues() {
	queuesMutex.Lock()
	defer queuesMutex.Unlock()
	//create int array with all queues ids
	var deleteQueues []int

	for _, queue := range Queues {
		queue.CheckingLobby()
		if queue.RemainingTime <= 0 {
			deleteQueues = append(deleteQueues, queue.ID)
		}
		queue.RemainingTime--
	}
	for _, queueID := range deleteQueues {
		Queues[queueID].PartyA.PartyLeaderAcceptArena = false
		Queues[queueID].PartyB.PartyLeaderAcceptArena = false
		//for queue party members
		for _, member := range Queues[queueID].PartyA.Members {
			if member == nil {
				continue
			}
			member.AcceptedArena = false
		}
		for _, member := range Queues[queueID].PartyB.Members {
			if member == nil {
				continue
			}
			member.AcceptedArena = false
		}
		delete(Queues, queueID)
	}
}
func ServerExploreLobbies() {
	var deleteParty []string
	for _, lobby := range LevelTypeArenaLobby {
		lobby.Partylock.Lock()
		defer lobby.Partylock.Unlock()
		for _, party := range lobby.Parties {
			if party.Leader == nil || party.Leader.Socket == nil {
				deleteParty = append(deleteParty, party.Leader.UserID)
				continue
			}
		}
	}

	for _, partyID := range deleteParty {
		if LevelTypeArenaLobby[0].Parties[partyID] != nil {
			delete(LevelTypeArenaLobby[0].Parties, partyID)
		} else if LevelTypeArenaLobby[1].Parties[partyID] != nil {
			delete(LevelTypeArenaLobby[1].Parties, partyID)
		} else if LevelTypeArenaLobby[2].Parties[partyID] != nil {
			delete(LevelTypeArenaLobby[2].Parties, partyID)
		}
	}
}
func ServerExploreArenas() {
	arenaMutex.Lock()
	getArenasLen := len(Arenas)
	arenaMutex.Unlock()
	for i := 0; i < getArenasLen; i++ {
		if Arenas[i] == nil || Arenas[i].WaitForNextRound || Arenas[i].IsFinished {
			continue
		}
		teamADead := 0
		teamBDead := 0
		ArenaWinnedByATeam := false
		ArenaWinnedByBTeam := false
		Arenas[i].RoundTimer()
		//For each arena team members
		for _, member := range Arenas[i].TeamA.players {
			//If member is not online DISABLED UNTIL TEST
			/*if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
				Arenas[i].TeamA.RemovePlayer(member, Arenas[i].TeamA)
			}*/
			//if players hp is less or equal to 0 then the round is winned by enemy team
			if member.Socket.Stats.HP <= 0 {
				teamADead++
			}
		}
		for _, member := range Arenas[i].TeamB.players {
			//If member is not online DISABLED UNTIL TEST
			/*if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
				Arenas[i].TeamB.RemovePlayer(member, Arenas[i].TeamB)
			}*/
			if member.Socket.Stats.HP <= 0 {
				teamBDead++
			}
		}
		fmt.Println("ROUNDTIME: ", Arenas[i].RoundTime)
		//END TIME
		if Arenas[i].RoundTime <= 0 {
			teamBPercent, teamAPercent := 0, 0
			for _, member := range Arenas[i].TeamA.players {
				if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
					continue
				}
				hpFloat := float64(member.Socket.Stats.HP / member.Socket.Stats.MaxHP)
				hpPercent := int(hpFloat) * 100
				teamAPercent += hpPercent
			}
			for _, member := range Arenas[i].TeamB.players {
				if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
					continue
				}
				hpFloat := float64(member.Socket.Stats.HP / member.Socket.Stats.MaxHP)
				hpPercent := int(hpFloat) * 100
				teamBPercent += hpPercent
			}
			fmt.Println("Times is up. HPA: ", teamAPercent, " HPB: ", teamBPercent)
			if teamAPercent > teamBPercent {
				Arenas[i].TeamA.Wins++
				Arenas[i].GameNumber++
				if Arenas[i].TeamA.Wins < ArenaWinNumber && !ArenaWinnedByBTeam && Arenas[i].GameNumber <= 5 {
					Arenas[i].WaitForNextRound = true
					go Arenas[i].StartNextRound(10)
				} else if Arenas[i].TeamA.Wins >= ArenaWinNumber { //The teamB wins the arena
					ArenaWinnedByATeam = true
				}
			} else if teamBPercent > teamAPercent {
				Arenas[i].TeamB.Wins++
				Arenas[i].GameNumber++
				if Arenas[i].TeamB.Wins < ArenaWinNumber && !ArenaWinnedByATeam && Arenas[i].GameNumber <= 5 {
					Arenas[i].WaitForNextRound = true
					go Arenas[i].StartNextRound(10)
				} else if Arenas[i].TeamB.Wins >= ArenaWinNumber { //The teamB wins the arena
					ArenaWinnedByBTeam = true
				}
			} else {
				Arenas[i].TeamB.Wins++
				Arenas[i].TeamA.Wins++
				Arenas[i].GameNumber++
				if Arenas[i].TeamB.Wins < ArenaWinNumber && Arenas[i].TeamA.Wins < ArenaWinNumber && Arenas[i].GameNumber <= 5 {
					Arenas[i].WaitForNextRound = true
					go Arenas[i].StartNextRound(10)
				} else if Arenas[i].TeamB.Wins >= ArenaWinNumber && Arenas[i].TeamA.Wins < ArenaWinNumber { //The teamB wins the arena
					ArenaWinnedByBTeam = true
					Arenas[i].TeamB.TeamGiveWinReward()
					Arenas[i].TeamA.TeamGiveLoseReward()
					party := Arenas[i].TeamB.GetPlayerParty()
					if party != nil {
						party.ArenaFounded = false
					}
					party = Arenas[i].TeamA.GetPlayerParty()
					if party != nil {
						party.ArenaFounded = false
					}
					//teamB wins
					for _, member := range Arenas[i].TeamB.players {
						if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
							continue
						}
						member.Socket.Write(messaging.InfoMessage("Your team won the arena!"))
					}
					if ArenaWinnedByBTeam {
						for _, member := range Arenas[i].TeamA.players {
							if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
								continue
							}
							member.Socket.Write(messaging.InfoMessage("Your team lose the arena!"))
						}
					}
					Arenas[i].IsFinished = true
					Arenas[i].ArenaEndTimer(10)
					continue
				} else if Arenas[i].TeamA.Wins >= ArenaWinNumber && Arenas[i].TeamB.Wins < ArenaWinNumber { //The teamA wins the arena
					ArenaWinnedByATeam = true
					Arenas[i].TeamA.TeamGiveWinReward()
					Arenas[i].TeamB.TeamGiveLoseReward()
					party := Arenas[i].TeamB.GetPlayerParty()
					if party != nil {
						party.ArenaFounded = false
					}
					party = Arenas[i].TeamA.GetPlayerParty()
					if party != nil {
						party.ArenaFounded = false
					}
					//teamA wins
					for _, member := range Arenas[i].TeamA.players {
						if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
							continue
						}
						member.Socket.Write(messaging.InfoMessage("Your team won the arena!"))
					}
					if ArenaWinnedByATeam {
						for _, member := range Arenas[i].TeamB.players {
							if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
								continue
							}
							member.Socket.Write(messaging.InfoMessage("Your team lose the arena!"))
						}
					}
					Arenas[i].IsFinished = true
					Arenas[i].ArenaEndTimer(10)
					continue
				}
			}
		}

		//WINS BY B TEAM
		//if teamA players is dead
		if teamADead >= len(Arenas[i].TeamA.players) {
			//teamB wins the round
			for _, member := range Arenas[i].TeamB.players {
				if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
					continue
				}
				member.Socket.Write(messaging.InfoMessage("Your team won the round!"))
			}
			Arenas[i].TeamB.Wins++
			Arenas[i].GameNumber++
			if Arenas[i].TeamB.Wins < ArenaWinNumber && !ArenaWinnedByATeam && Arenas[i].GameNumber <= 5 {
				Arenas[i].WaitForNextRound = true
				go Arenas[i].StartNextRound(10)
			} else if Arenas[i].TeamB.Wins >= ArenaWinNumber { //The teamB wins the arena
				ArenaWinnedByBTeam = true
			}
		}
		//WINS BY A TEAM
		//if teamB players is dead
		if teamBDead >= len(Arenas[i].TeamB.players) {
			//teamA wins the round
			for _, member := range Arenas[i].TeamA.players {
				if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
					continue
				}
				member.Socket.Write(messaging.InfoMessage("Your team won the round!"))
			}
			Arenas[i].TeamA.Wins++
			Arenas[i].GameNumber++
			if Arenas[i].TeamA.Wins < ArenaWinNumber && !ArenaWinnedByBTeam && Arenas[i].GameNumber <= 5 {
				Arenas[i].WaitForNextRound = true
				go Arenas[i].StartNextRound(10)

			} else if Arenas[i].TeamA.Wins >= ArenaWinNumber { //The teamA wins the arena
				ArenaWinnedByATeam = true
			}
		}

		//if len teamA players is less or equal to 0
		if len(Arenas[i].TeamA.players) <= 0 || ArenaWinnedByBTeam {
			Arenas[i].TeamB.TeamGiveWinReward()
			Arenas[i].TeamA.TeamGiveLoseReward()
			party := Arenas[i].TeamB.GetPlayerParty()
			if party != nil {
				party.ArenaFounded = false
			}
			party = Arenas[i].TeamA.GetPlayerParty()
			if party != nil {
				party.ArenaFounded = false
			}
			//teamB wins
			for _, member := range Arenas[i].TeamB.players {
				if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
					continue
				}
				member.Socket.Write(messaging.InfoMessage("Your team won the arena!"))
			}
			if ArenaWinnedByBTeam {
				for _, member := range Arenas[i].TeamA.players {
					if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
						continue
					}
					member.Socket.Write(messaging.InfoMessage("Your team lose the arena!"))
				}
			}
			Arenas[i].IsFinished = true
			Arenas[i].ArenaEndTimer(10)
			//Delete arena
			//delete(Arenas, i)
			continue
		} else if len(Arenas[i].TeamB.players) <= 0 || ArenaWinnedByATeam {
			Arenas[i].TeamA.TeamGiveWinReward()
			Arenas[i].TeamB.TeamGiveLoseReward()
			party := Arenas[i].TeamB.GetPlayerParty()
			if party != nil {
				party.ArenaFounded = false
			}
			party = Arenas[i].TeamA.GetPlayerParty()
			if party != nil {
				party.ArenaFounded = false
			}
			//teamA wins
			for _, member := range Arenas[i].TeamA.players {
				if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
					continue
				}
				member.Socket.Write(messaging.InfoMessage("Your team won the arena!"))
			}
			if ArenaWinnedByATeam {
				for _, member := range Arenas[i].TeamB.players {
					if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
						continue
					}
					member.Socket.Write(messaging.InfoMessage("Your team lose the arena!"))
				}
			}
			Arenas[i].IsFinished = true
			Arenas[i].ArenaEndTimer(10)
			//Delete arena
			//delete(Arenas, i)
			continue
		}
		//MORE THEN 5 ROUNDS
		if Arenas[i].GameNumber > 5 {
			if Arenas[i].TeamA.Wins > Arenas[i].TeamB.Wins {
				Arenas[i].TeamA.TeamGiveWinReward()
				Arenas[i].TeamB.TeamGiveLoseReward()
				party := Arenas[i].TeamB.GetPlayerParty()
				if party != nil {
					party.ArenaFounded = false
				}
				party = Arenas[i].TeamA.GetPlayerParty()
				if party != nil {
					party.ArenaFounded = false
				}
				//teamA wins
				for _, member := range Arenas[i].TeamA.players {
					if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
						continue
					}
					member.Socket.Write(messaging.InfoMessage("Your team won the arena!"))
				}
				if ArenaWinnedByATeam {
					for _, member := range Arenas[i].TeamB.players {
						if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
							continue
						}
						member.Socket.Write(messaging.InfoMessage("Your team lose the arena!"))
					}
				}
				Arenas[i].IsFinished = true
				Arenas[i].ArenaEndTimer(10)
				continue
			} else if Arenas[i].TeamB.Wins > Arenas[i].TeamA.Wins {
				Arenas[i].TeamB.TeamGiveWinReward()
				Arenas[i].TeamA.TeamGiveLoseReward()
				party := Arenas[i].TeamB.GetPlayerParty()
				if party != nil {
					party.ArenaFounded = false
				}
				party = Arenas[i].TeamA.GetPlayerParty()
				if party != nil {
					party.ArenaFounded = false
				}
				//teamB wins
				for _, member := range Arenas[i].TeamB.players {
					if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
						continue
					}
					member.Socket.Write(messaging.InfoMessage("Your team won the arena!"))
				}
				if ArenaWinnedByBTeam {
					for _, member := range Arenas[i].TeamA.players {
						if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
							continue
						}
						member.Socket.Write(messaging.InfoMessage("Your team lose the arena!"))
					}
				}
				Arenas[i].IsFinished = true
				Arenas[i].ArenaEndTimer(10)
				continue
			} else {
				party := Arenas[i].TeamB.GetPlayerParty()
				if party != nil {
					party.ArenaFounded = false
				}
				party = Arenas[i].TeamA.GetPlayerParty()
				if party != nil {
					party.ArenaFounded = false
				}
				//teamB wins
				for _, member := range Arenas[i].TeamB.players {
					if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
						continue
					}
					member.Socket.Write(messaging.InfoMessage("The arena was draw!"))
				}
				if ArenaWinnedByBTeam {
					for _, member := range Arenas[i].TeamA.players {
						if member == nil || member.Socket == nil || !member.IsOnline || !funk.Contains(ArenaZones, member.Map) {
							continue
						}
						member.Socket.Write(messaging.InfoMessage("The arena was draw!"))
					}
				}
				Arenas[i].IsFinished = true
				Arenas[i].ArenaEndTimer(10)
				continue
			}
		}
	}
}
func ServerExploreDungeons() {
	dungeons := GetActiveDungeons()
	for i := 0; i < len(dungeons); i++ {
		if dungeons[i] == nil || dungeons[i].IsLoading || dungeons[i].IsDeleting {
			continue
		}
		//IF DUNGEON EMPTY NEED TO DELETE IT!!
		chars, err := FindCharactersInServer(dungeons[i].ServerID)
		if err != nil {
			continue
		}
		if len(chars) == 0 {
			dungeons[i].IsDeleting = true
			DeleteDungeonMobs(dungeons[i].ServerID)
			delete(dungeons, i)
			continue
		}
		//TIME IS ENDED SO PLAYERS NEED TO TELEPORT OUT AND MOBS NEED TO DELETE
		if time.Since(dungeons[i].DungeonStartedTime.Add(time.Minute*30)) >= 0 {
			for _, member := range chars {
				if int16(246) != member.Map {
					continue
				}
				resp := utils.Packet{}
				resp.Concat(messaging.InfoMessage("The time is up. Teleporting to safe zone."))
				if member.Socket == nil {
					continue
				}
				member.Socket.User.ConnectedServer = 1
				data, _ := member.ChangeMap(1, nil)
				resp.Concat(data)
				member.Socket.Write(resp)
			}
			DeleteDungeonMobs(dungeons[i].ServerID)
			delete(dungeons, i)
		}
	}
}
func GetServers() ([]*ServerItem, error) {

	var (
		items []*ServerItem
	)

	if len(servers) == 0 {
		query := `select * from hops.servers`

		if _, err := db.Select(&servers, query); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, fmt.Errorf("GetServers: %s", err.Error())
		}
	}

	socketMutex.RLock()
	sArr := funk.Values(Sockets)
	socketMutex.RUnlock()

	for _, s := range servers {

		i := &ServerItem{*s, 0}
		count := len(funk.Filter(sArr, func(socket *Socket) bool {
			if socket.User == nil {
				return false
			}

			return socket.User.ConnectedServer == s.ID
		}).([]*Socket))

		i.ConnectedUsers = int(count)
		items = append(items, i)
	}

	return items, nil
}

func GetServerByID(id string) (*ServerItem, error) {
	var (
		server = &Server{}
	)

	query := `select * from hops.servers where id = $1`

	if err := db.SelectOne(&server, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("GetServerByID: %s", err.Error())
	}

	i := &ServerItem{*server, 0}
	query = `select count(*) from hops.users where server = $1`
	count, err := db.SelectInt(query, server.ID)
	if err != nil {
		return nil, fmt.Errorf("GetConnectedUserCount: %s", err.Error())
	}

	i.ConnectedUsers = int(count)
	return i, nil
}

func EpochHandler() {
	server, err := GetServerByID("1")
	if err != nil {
		log.Print(err)
	}
	if server != nil {
		server.Epoch++
		server.Update()
	}
	time.AfterFunc(time.Second, EpochHandler)
}

func GetServerEpoch() int64 {
	server, err := GetServerByID("1")
	if err != nil {
		log.Print(err)
	}
	return server.Epoch
}
