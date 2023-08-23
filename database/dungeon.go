package database

import "time"

type Dungeon struct {
	ServerID           int
	DungeonLeader      *Character
	DungeonStartedTime time.Time
	IsLoading          bool
	IsDeleting         bool
}

var (
	GetActiveDungeons func() map[int]*Dungeon
	DeleteDungeonMobs func(int)
	DungeonLoading    = false
)
