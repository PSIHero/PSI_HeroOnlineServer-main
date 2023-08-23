package database

import "PsiHero/utils"

type Duel struct {
	EnemyID    int
	Coordinate utils.Location
	Started    bool
}
