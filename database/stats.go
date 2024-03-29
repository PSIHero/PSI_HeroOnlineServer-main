package database

import (
	"database/sql"
	"fmt"
	"sync"

	gorp "gopkg.in/gorp.v1"
)

var (
	stats         = make(map[int]*Stat)
	stMutex       sync.RWMutex
	startingStats = map[int]*Stat{
		50: {STR: 10, DEX: 18, INT: 2, HP: 66, MaxHP: 66, CHI: 18, MaxCHI: 18, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // BEAST MALE
		51: {STR: 10, DEX: 18, INT: 2, HP: 66, MaxHP: 66, CHI: 18, MaxCHI: 18, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // BEAST FEMALE
		52: {STR: 13, DEX: 12, INT: 8, HP: 72, MaxHP: 72, CHI: 48, MaxCHI: 48, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Monk
		53: {STR: 12, DEX: 12, INT: 6, HP: 90, MaxHP: 90, CHI: 30, MaxCHI: 30, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Male Blade
		54: {STR: 11, DEX: 13, INT: 6, HP: 90, MaxHP: 90, CHI: 30, MaxCHI: 30, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Female Blade
		56: {STR: 15, DEX: 10, INT: 5, HP: 96, MaxHP: 96, CHI: 24, MaxCHI: 24, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Axe
		57: {STR: 14, DEX: 11, INT: 5, HP: 96, MaxHP: 96, CHI: 24, MaxCHI: 24, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Female Spear
		59: {STR: 10, DEX: 18, INT: 2, HP: 84, MaxHP: 84, CHI: 36, MaxCHI: 36, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Dual Sword

		60: {STR: 10, DEX: 18, INT: 2, HP: 66, MaxHP: 66, CHI: 18, MaxCHI: 18, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // BEAST MALE
		61: {STR: 10, DEX: 18, INT: 2, HP: 66, MaxHP: 66, CHI: 18, MaxCHI: 18, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // BEAST FEMALE
		62: {STR: 13, DEX: 12, INT: 8, HP: 72, MaxHP: 72, CHI: 48, MaxCHI: 48, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Divine Monk
		63: {STR: 12, DEX: 12, INT: 6, HP: 90, MaxHP: 90, CHI: 30, MaxCHI: 30, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Divine Male Blade
		64: {STR: 11, DEX: 13, INT: 6, HP: 90, MaxHP: 90, CHI: 30, MaxCHI: 30, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Divine Female Blade
		66: {STR: 15, DEX: 10, INT: 5, HP: 96, MaxHP: 96, CHI: 24, MaxCHI: 24, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Divine Axe
		67: {STR: 14, DEX: 11, INT: 5, HP: 96, MaxHP: 96, CHI: 24, MaxCHI: 24, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Divine Female Spear
		69: {STR: 10, DEX: 18, INT: 2, HP: 84, MaxHP: 84, CHI: 36, MaxCHI: 36, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Divine Dual Sword

		70: {STR: 10, DEX: 18, INT: 2, HP: 66, MaxHP: 66, CHI: 18, MaxCHI: 18, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // BEAST MALE
		71: {STR: 10, DEX: 18, INT: 2, HP: 66, MaxHP: 66, CHI: 18, MaxCHI: 18, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // BEAST FEMALE
		72: {STR: 13, DEX: 12, INT: 8, HP: 72, MaxHP: 72, CHI: 48, MaxCHI: 48, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Dark Monk
		73: {STR: 12, DEX: 12, INT: 6, HP: 90, MaxHP: 90, CHI: 30, MaxCHI: 30, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Dark Male Blade
		74: {STR: 11, DEX: 13, INT: 6, HP: 90, MaxHP: 90, CHI: 30, MaxCHI: 30, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Dark Female Blade
		76: {STR: 15, DEX: 10, INT: 5, HP: 96, MaxHP: 96, CHI: 24, MaxCHI: 24, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Dark Axe
		77: {STR: 14, DEX: 11, INT: 5, HP: 96, MaxHP: 96, CHI: 24, MaxCHI: 24, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Dark Female Spear
		79: {STR: 10, DEX: 18, INT: 2, HP: 84, MaxHP: 84, CHI: 36, MaxCHI: 36, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Dark Dual Sword
	}
)

type Stat struct {
	ID              int     `db:"id"`
	HP              int     `db:"hp"`
	MaxHP           int     `db:"max_hp"`
	HPRecoveryRate  int     `db:"hp_recovery_rate"`
	CHI             int     `db:"chi"`
	MaxCHI          int     `db:"max_chi"`
	CHIRecoveryRate int     `db:"chi_recovery_rate"`
	STR             int     `db:"str"`
	DEX             int     `db:"dex"`
	INT             int     `db:"int"`
	STRBuff         int     `db:"str_buff"`
	DEXBuff         int     `db:"dex_buff"`
	INTBuff         int     `db:"int_buff"`
	StatPoints      int     `db:"stat_points"`
	Honor           int     `db:"honor"`
	MinATK          int     `db:"min_atk"`
	MaxATK          int     `db:"max_atk"`
	ATKRate         int     `db:"atk_rate"`
	MinArtsATK      int     `db:"min_arts_atk"`
	MaxArtsATK      int     `db:"max_arts_atk"`
	ArtsATKRate     int     `db:"arts_atk_rate"`
	DEF             int     `db:"def"`
	DefRate         int     `db:"def_rate"`
	ArtsDEF         int     `db:"arts_def"`
	ArtsDEFRate     int     `db:"arts_def_rate"`
	Accuracy        int     `db:"accuracy"`
	Dodge           int     `db:"dodge"`
	PoisonATK       int     `db:"poison_atk"`
	ParalysisATK    int     `db:"paralysis_atk"`
	ConfusionATK    int     `db:"confusion_atk"`
	PoisonDEF       int     `db:"poison_def"`
	ParalysisDEF    int     `db:"paralysis_def"`
	ConfusionDEF    int     `db:"confusion_def"`
	Wind            int     `db:"wind"`
	WindBuff        int     `db:"wind_buff"`
	Water           int     `db:"water"`
	WaterBuff       int     `db:"water_buff"`
	Fire            int     `db:"fire"`
	FireBuff        int     `db:"fire_buff"`
	NaturePoints    int     `db:"nature_points"`
	Paratime        int     `db:"paratime"`
	PoisonTime      int     `db:"poisontime"`
	ConfusionTime   int     `db:"confusiontime"`
	AttackSpeed     int     `db:"attack_speed"`
	GoldRate        float64 `db:"gold_rate"`
	ExpRate         float64 `db:"exp_rate"`
	DropRate        float64 `db:"drop_rate"`
}

func (t *Stat) Create(c *Character) error {
	t = startingStats[c.Type]
	t.ID = c.ID
	t.StatPoints = 4
	t.NaturePoints = 0
	t.Honor = 10000
	return db.Insert(t)
}

func (t *Stat) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(t)
}

func (t *Stat) Update() error {
	_, err := db.Update(t)
	return err
}

func (t *Stat) Delete() error {
	stMutex.Lock()
	delete(stats, t.ID)
	stMutex.Unlock()

	_, err := db.Delete(t)
	return err
}

func (t *Stat) Calculate() error {

	c, err := FindCharacterByID(t.ID)
	if err != nil {
		return err
	} else if c == nil {
		return nil
	}

	temp := *t
	stStat := startingStats[c.Type]
	temp.MaxHP = stStat.MaxHP + 1730*temp.STR
	temp.MaxCHI = stStat.MaxCHI
	temp.STRBuff = 0
	temp.DEXBuff = 0
	temp.INTBuff = 0
	temp.WindBuff = 0
	temp.WaterBuff = 0
	temp.FireBuff = 0
	temp.MinATK = temp.STR
	temp.MaxATK = temp.STR
	temp.ATKRate = 0
	temp.MinArtsATK = temp.STR
	temp.MaxArtsATK = temp.STR
	temp.ArtsATKRate = 0
	temp.DEF = 5 * temp.DEX
	temp.AttackSpeed = stStat.AttackSpeed
	temp.DefRate = 0
	temp.ArtsDEF = 4*temp.INT + 4*temp.DEX
	temp.ArtsDEFRate = 0
	temp.Accuracy = int(float32(temp.STR) * 3)
	temp.Dodge = 0
	temp.PoisonATK = 0
	temp.PoisonDEF = 0
	temp.ParalysisATK = 0
	temp.ParalysisDEF = 0
	temp.ConfusionATK = 0
	temp.ConfusionDEF = 0
	temp.PoisonTime = 0
	temp.Paratime = 0
	temp.ConfusionTime = 0
	temp.CHIRecoveryRate = 10
	temp.HPRecoveryRate = 10 + 250*temp.STR
	temp.GoldRate = 0
	temp.ExpRate = 0
	temp.DropRate = 0

	if c != nil {
		if c.Socket != nil {
			if c.Socket.User != nil {
				if c.Socket.User.UserType < 2 {
					c.RunningSpeed = 5.6
					skills, err := FindSkillsByID(c.ID)
					if err == nil {
						skillSlots, err := skills.GetSkills()
						if err == nil {
							set := skillSlots.Slots[7]
							if set != nil && set.BookID != 0 {
								c.RunningSpeed = 10.0 + (float64(set.Skills[0].Plus) * 0.2)
							}
						}
					}
				}
			}
		}
	}

	c.BuffEffects(&temp)
	c.JobPassives(&temp)

	c.AdditionalDropMultiplier = 0
	c.AdditionalExpMultiplier = 0
	c.AdditionalRunningSpeed = 0

	c.ItemEffects(&temp, 0, 9)         // NORMAL ITEMS
	c.ItemEffects(&temp, 307, 315)     // HT ITEMS
	c.ItemEffects(&temp, 0x0B, 0x43)   // INV BUFFS1
	c.ItemEffects(&temp, 0x155, 0x18D) // INV BUFFS2
	c.ItemEffects(&temp, 397, 399)     // MARBLES(1-3)
	c.RebornEffects(&temp)
	//totalDEX := temp.DEX + temp.DEXBuff
	totalWind := temp.Wind + temp.WindBuff
	totalWater := temp.Water + temp.WaterBuff
	totalFire := temp.Fire + temp.FireBuff

	temp.DEF += temp.DEXBuff + 1*totalWind + 3*totalWater + 1*totalFire
	temp.DEF += temp.DEF * temp.DefRate / 156
	temp.ArtsDEF += 2*temp.INTBuff + temp.DEXBuff + 1*totalWind + 3*totalWater + 1*totalFire
	temp.ArtsDEF += temp.ArtsDEF * temp.ArtsDEFRate / 200

	c.ItemEffects(&temp, 400, 401) // MARBLES(4-5)

	totalSTR := temp.STR + temp.STRBuff
	totalINT := temp.INT + temp.INTBuff

	temp.MaxHP += 14*totalSTR + 1400*totalWind
	temp.MaxCHI += 3*totalINT + 200*totalWater

	temp.MinATK += temp.STRBuff + 1*totalWind + 1*totalWater + 2*totalFire
	temp.MinATK += temp.MinATK * temp.ATKRate / 100
	temp.MaxATK += temp.STRBuff + 1*totalWind + 1*totalWater + 2*totalFire
	temp.MaxATK += temp.MaxATK * temp.ATKRate / 100

	temp.MinArtsATK += temp.STRBuff + 2*totalINT + int(float32(totalINT*temp.MinATK)/515)
	temp.MinArtsATK += temp.MinArtsATK * temp.ArtsATKRate / 200
	temp.MaxArtsATK += temp.STRBuff + 2*totalINT + int(float32(totalINT*temp.MaxATK)/515)
	temp.MaxArtsATK += temp.MaxArtsATK * temp.ArtsATKRate / 200

	temp.Accuracy += int(float32(temp.STRBuff) * 0.925)
	//temp.Dodge += temp.DEXBuff

	c.SpecialBuffEffects(&temp)

	*t = temp
	go t.Update()
	return nil
}

func (t *Stat) ResetStats() error {

	c, err := FindCharacterByID(t.ID)
	if err != nil {
		return err
	} else if c == nil {
		return fmt.Errorf("CalculateTotalStatPoints: character not found")
	}

	stat := startingStats[c.Type]
	statPoints := 0
	naturePoints := 0

	for i := 1; i <= c.Level; i++ {
		statPoints += EXPs[int16(i)].StatPoints
		naturePoints += EXPs[int16(i)].NaturePoints
	}

	t.STR = stat.STR
	t.DEX = stat.DEX
	t.INT = stat.INT
	t.Wind = 0
	t.Water = 0
	t.Fire = 0
	t.StatPoints = statPoints
	t.NaturePoints = naturePoints
	return t.Update()
}

func FindStatByID(id int) (*Stat, error) {

	stMutex.RLock()
	s, ok := stats[id]
	stMutex.RUnlock()
	if ok {
		return s, nil
	}

	stat := &Stat{}
	query := `select * from hops.stats where id = $1`

	if err := db.SelectOne(&stat, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindStatByID: %s", err.Error())
	}

	stMutex.Lock()
	defer stMutex.Unlock()
	stats[stat.ID] = stat

	return stat, nil
}

func DeleteStatFromCache(id int) {
	delete(stats, id)
}
