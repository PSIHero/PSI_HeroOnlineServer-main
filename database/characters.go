package database

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"reflect"
	"regexp"
	dbg "runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"PsiHero/logging"
	"PsiHero/messaging"
	"PsiHero/nats"
	"PsiHero/utils"

	"github.com/osamingo/boolconv"
	"github.com/thoas/go-funk"
	gorp "gopkg.in/gorp.v1"
	null "gopkg.in/guregu/null.v3"
)

const (
	BEAST_KING            = 0x32
	EMPRESS               = 0x33
	MONK                  = 0x34
	MALE_BLADE            = 0x35
	FEMALE_BLADE          = 0x36
	AXE                   = 0x38
	FEMALE_ROD            = 0x39
	DUAL_BLADE            = 0x3B
	DIVINE_BEAST_KING     = 0x3C
	DIVINE_EMPRESS        = 0x3D
	DIVINE_MONK           = 0x3E
	DIVINE_MALE_BLADE     = 0x3F
	DIVINE_FEMALE_BLADE   = 0x40
	DIVINE_AXE            = 0x42
	DIVINE_FEMALE_ROD     = 0x43
	DIVINE_DUAL_BLADE     = 0x45
	DARKNESS_BEAST_KING   = 0x46
	DARKNESS_EMPRESS      = 0x47
	DARKNESS_MONK         = 0x48
	DARKNESS_MALE_BLADE   = 0x49
	DARKNESS_FEMALE_BLADE = 0x4A
	DARKNESS_AXE          = 0x4C
	DARKNESS_FEMALE_ROD   = 0x4D
	DARKNESS_DUAL_BLADE   = 0x4F
)

var (
	GMPassword           = "loveuniverse"
	characters           = make(map[int]*Character)
	characterMutex       sync.RWMutex
	GenerateID           func(*Character) error
	GeneratePetID        func(*Character, *PetSlot)
	challengerGuild      = &Guild{}
	enemyGuild           = &Guild{}
	GMRanks              = []int16{2, 3, 4}
	Characterspawntester = 50
	DEAL_POISON_DAMAGE   = utils.Packet{0xAA, 0x55, 0x2e, 0x00, 0x16, 0xFE, 0xFF, 0xFF, 0xFF, 0x01, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1c, 0x57, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}
	DEAL_DAMAGE          = utils.Packet{0xAA, 0x55, 0x1C, 0x00, 0x16, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	DEAL_BUFF_AI         = utils.Packet{0xaa, 0x55, 0x1e, 0x00, 0x16, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}
	BAG_EXPANDED         = utils.Packet{0xAA, 0x55, 0x17, 0x00, 0xA3, 0x02, 0x01, 0x32, 0x30, 0x32, 0x30, 0x2D, 0x30, 0x33, 0x2D, 0x31, 0x37, 0x20, 0x31, 0x31, 0x3A, 0x32, 0x32, 0x3A, 0x30, 0x31, 0x00, 0x55, 0xAA}
	BANK_ITEMS           = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x57, 0x05, 0x01, 0x02, 0x55, 0xAA}
	CHARACTER_DIED       = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x12, 0x01, 0x55, 0xAA}
	BANK_EXPANDED        = utils.Packet{0xAA, 0x55, 0x17, 0x00, 0xA3, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x30, 0x31, 0x00, 0x55, 0xAA}
	/*CHARACTER_SPAWNED  = utils.Packet{0xAA, 0x55, 0x00, 0x00, , 0xD7, 0xEF, 0xE6, 0x00, 0x03, 0x01, 0x00, 0x00, 0x00, 0x00, 0xC9, 0x00, 0x00, 0x00,
	0x49, 0x2A, 0xFE, 0x00, 0x20, 0x1C, 0x00, 0x00, 0x02, 0xD2, 0x7E, 0x7F, 0xBF, 0xCD, 0x1A, 0x86, 0x3D, 0x33, 0x33, 0x6B, 0x41, 0xFF, 0xFF, 0x10, 0x27,
	0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0xC4, 0x0E, 0x00, 0x00, 0xC8, 0xBB, 0x30, 0x00, 0x00, 0x03, 0xF3, 0x03, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
	0x00, 0x10, 0x27, 0x00, 0x00, 0x49, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x55, 0xAA}
	*/
	CHARACTER_SPAWNED    = utils.Packet{0xaa, 0x55, 0xbd, 0x00, 0x21, 0x01, 0x00, 0x00, 0x36, 0xff, 0xff, 0xff, 0xff, 0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x64, 0xff, 0xff, 0xff, 0xff, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}
	EXP_SKILL_PT_CHANGED = utils.Packet{0xAA, 0x55, 0x0D, 0x00, 0x13, 0x55, 0xAA}

	HP_CHI = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}

	/*HP_CHI = utils.Packet{0xaa, 0x55, 0x2e, 0x00, 0x16, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x26, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xaa}*/

	RESPAWN_COUNTER  = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x12, 0x02, 0x01, 0x00, 0x00, 0x55, 0xAA}
	SHOW_ITEMS       = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x59, 0x05, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA}
	MEDITATION_MODE  = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x82, 0x05, 0x00, 0x55, 0xAA}
	TELEPORT_PLAYER  = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x24, 0x55, 0xAA}
	ITEM_COUNT       = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x59, 0x04, 0x0A, 0x00, 0x55, 0xAA}
	GREEN_ITEM_COUNT = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0x59, 0x19, 0x0A, 0x00, 0x55, 0xAA}
	ITEM_EXPIRED     = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x69, 0x03, 0x55, 0xAA}
	ITEM_ADDED       = utils.Packet{0xaa, 0x55, 0x2e, 0x00, 0x57, 0x0a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x83, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}
	ITEM_LOOTED      = utils.Packet{0xAA, 0x55, 0x33, 0x00, 0x59, 0x01, 0x0A, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x21, 0x11, 0x55, 0xAA}

	PTS_CHANGED = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0xA2, 0x04, 0x55, 0xAA}
	GOLD_LOOTED = utils.Packet{0xAA, 0x55, 0x0D, 0x00, 0x59, 0x01, 0x0A, 0x00, 0x02, 0x55, 0xAA}
	GET_GOLD    = utils.Packet{0xAA, 0x55, 0x12, 0x00, 0x63, 0x01, 0x55, 0xAA}

	MAP_CHANGED = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x2B, 0x01, 0x55, 0xAA, 0xAA, 0x55, 0x0E, 0x00, 0x73, 0x00, 0x00, 0x00, 0x7A, 0x44, 0x55, 0xAA,
		0xAA, 0x55, 0x07, 0x00, 0x01, 0xB9, 0x0A, 0x00, 0x00, 0x01, 0x00, 0x55, 0xAA, 0xAA, 0x55, 0x09, 0x00, 0x24, 0x55, 0xAA,
		0xAA, 0x55, 0x03, 0x00, 0xA6, 0x00, 0x00, 0x55, 0xAA, 0xAA, 0x55, 0x02, 0x00, 0xAD, 0x01, 0x55, 0xAA}

	ITEM_REMOVED = utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x59, 0x02, 0x0A, 0x00, 0x01, 0x55, 0xAA}
	SELL_ITEM    = utils.Packet{0xAA, 0x55, 0x16, 0x00, 0x58, 0x02, 0x0A, 0x00, 0x20, 0x1C, 0x00, 0x00, 0x55, 0xAA}
	//2. SOR MIKOR A SOK NULLAT VALTOZTATTAM AZ A PVP
	GET_STATS = utils.Packet{0xAA, 0x55, 0xDE, 0x00, 0x14, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x66, 0x66, 0xC6, 0x40,
		0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x00, 0x00, 0x00, 0x20, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x30, 0x30, 0x31, 0x2D, 0x30, 0x31, 0x2D, 0x30,
		0x31, 0x20, 0x30, 0x30, 0x3A, 0x30, 0x30, 0x3A, 0x30, 0x30, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x00, 0x64, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0xff, 0xff, 0xff,
		0x00, 0x00, 0xF0, 0x3F, 0x10, 0x27, 0x80, 0x3F, 0x55, 0xAA}

	ITEM_REPLACEMENT   = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x59, 0x03, 0x0A, 0x00, 0x55, 0xAA}
	ITEM_SWAP          = utils.Packet{0xAA, 0x55, 0x15, 0x00, 0x59, 0x07, 0x0A, 0x00, 0x00, 0x55, 0xAA}
	HT_UPG_FAILED      = utils.Packet{0xAA, 0x55, 0x31, 0x00, 0x54, 0x02, 0xA7, 0x0F, 0x01, 0x00, 0xA3, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	UPG_FAILED         = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA2, 0x0F, 0x00, 0x55, 0xAA}
	PRODUCTION_SUCCESS = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x04, 0x08, 0x10, 0x01, 0x55, 0xAA}
	PRODUCTION_FAILED  = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x04, 0x09, 0x10, 0x00, 0x55, 0xAA}
	FUSION_SUCCESS     = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x09, 0x08, 0x10, 0x01, 0x55, 0xAA}
	FUSION_FAILED      = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x09, 0x09, 0x10, 0x00, 0x55, 0xAA}
	DISMANTLE_SUCCESS  = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x54, 0x05, 0x68, 0x10, 0x01, 0x00, 0x55, 0xAA}
	EXTRACTION_SUCCESS = utils.Packet{0xAA, 0x55, 0xB7, 0x00, 0x54, 0x06, 0xCC, 0x10, 0x01, 0x00, 0xA2, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	HOLYWATER_FAILED   = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x10, 0x32, 0x11, 0x00, 0x55, 0xAA}
	HOLYWATER_SUCCESS  = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x10, 0x31, 0x11, 0x01, 0x55, 0xAA}
	ITEM_REGISTERED    = utils.Packet{0xAA, 0x55, 0x43, 0x00, 0x3D, 0x01, 0x0A, 0x00, 0x00, 0x80, 0x1A, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x00,
		0x00, 0x00, 0x63, 0x99, 0xEA, 0x00, 0x00, 0xA1, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	CLAIM_MENU              = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x3D, 0x03, 0x0A, 0x00, 0x55, 0xAA}
	CONSIGMENT_ITEM_BOUGHT  = utils.Packet{0xAA, 0x55, 0x08, 0x00, 0x3D, 0x02, 0x0A, 0x00, 0x55, 0xAA}
	CONSIGMENT_ITEM_SOLD    = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x3F, 0x00, 0x55, 0xAA}
	CONSIGMENT_ITEM_CLAIMED = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0x3D, 0x04, 0x0A, 0x00, 0x01, 0x00, 0x55, 0xAA}
	SKILL_UPGRADED          = utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x81, 0x02, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	SKILL_DOWNGRADED        = utils.Packet{0xAA, 0x55, 0x0E, 0x00, 0x81, 0x03, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	SKILL_REMOVED           = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x81, 0x06, 0x0A, 0x00, 0x00, 0x55, 0xAA}
	PASSIVE_SKILL_UGRADED   = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x82, 0x02, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA}
	PASSIVE_SKILL_REMOVED   = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x82, 0x04, 0x0A, 0x00, 0x00, 0x55, 0xAA}
	SKILL_CASTED            = utils.Packet{0xAA, 0x55, 0x1D, 0x00, 0x42, 0x0A, 0x00, 0x00, 0x00, 0x01, 0x01, 0x55, 0xAA}
	TRADE_CANCELLED         = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x53, 0x03, 0xD5, 0x07, 0x7E, 0x02, 0x55, 0xAA}
	SKILL_BOOK_EXISTS       = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}
	INVALID_CHARACTER_TYPE  = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xF2, 0x03, 0x55, 0xAA}
	NO_SLOTS_FOR_SKILL_BOOK = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xF3, 0x03, 0x55, 0xAA}
	OPEN_SALE               = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x55, 0x01, 0x0A, 0x00, 0x55, 0xAA}
	GET_SALE_ITEMS          = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x55, 0x03, 0x0A, 0x00, 0x00, 0x55, 0xAA}
	CLOSE_SALE              = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x55, 0x02, 0x0A, 0x00, 0x55, 0xAA}
	BOUGHT_SALE_ITEM        = utils.Packet{0xAA, 0x55, 0x39, 0x00, 0x53, 0x10, 0x0A, 0x00, 0x01, 0x00, 0xA2, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	SOLD_SALE_ITEM          = utils.Packet{0xAA, 0x55, 0x10, 0x00, 0x55, 0x07, 0x0A, 0x00, 0x55, 0xAA}
	BUFF_INFECTION          = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x4D, 0x02, 0x0A, 0x01, 0x55, 0xAA}
	BUFF_EXPIRED            = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x4D, 0x03, 0x55, 0xAA}

	SPLIT_ITEM = utils.Packet{0xAA, 0x55, 0x5C, 0x00, 0x59, 0x09, 0x0A, 0x00, 0x00, 0xA1, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA1, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}

	QUEST_HANDLER = utils.Packet{0xaa, 0x55, 0x30, 0x00, 0x57, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}

	RELIC_DROP            = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x10, 0x00, 0x55, 0xAA}
	PVP_FINISHED          = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x2A, 0x05, 0x55, 0xAA}
	FORM_ACTIVATED        = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x37, 0x55, 0xAA}
	FORM_DEACTIVATED      = utils.Packet{0xAA, 0x55, 0x01, 0x00, 0x38, 0x55, 0xAA}
	CHANGE_RANK           = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x2F, 0xF1, 0x36, 0x55, 0xAA}
	DARK_MODE_ACTIVE      = utils.Packet{0xaa, 0x55, 0x0a, 0x00, 0xad, 0x02, 0x9a, 0x99, 0x99, 0x3f, 0x66, 0x66, 0x66, 0x3f, 0x55, 0xaa}
	Beast_King_Infections = []int16{277, 307, 368, 283, 319, 382, 291, 333, 398, 297, 351, 418}
	Empress_Infections    = []int16{280, 313, 375, 287, 326, 390, 294, 342, 408, 302, 359, 429}
)

type Target struct {
	Damage    int  `db:"-" json:"damage"`
	AI        *AI  `db:"-" json:"ai"`
	Skill     bool `default:"false db:"-" json:"skill"`
	Critical  bool `default:"false db:"-" json:"critical"`
	Reflected bool `default:"false db:"-" json:"reflected"`
}

type AidSettings struct {
	PetFood1ItemID  int64 `db:"-" json:"petfood1"`
	PetFood1Percent uint  `db:"-" json:"petfood1percent"`
	PetChiItemID    int64 `db:"-" json:"petchi"`
	PetChiPercent   uint  `db:"-" json:"petchipercent"`
}
type PlayerTarget struct {
	Damage    int        `db:"-" json:"damage"`
	Enemy     *Character `db:"-" json:"ai"`
	Skill     bool       `default:"false db:"-" json:"skill"`
	Critical  bool       `default:"false db:"-" json:"critical"`
	Reflected bool       `default:"false db:"-" json:"reflected"`
}
type MessageItems struct {
	ID     int   `db:"-" json:"id"`
	SlotID int   `db:"-" json:"slotid"`
	ItemID int64 `db:"-" json:"itemid"`
}

type Character struct {
	ID                       int        `db:"id" json:"id"`
	UserID                   string     `db:"user_id" json:"user_id"`
	Name                     string     `db:"name" json:"name"`
	Epoch                    int64      `db:"epoch" json:"epoch"`
	Type                     int        `db:"type" json:"type"`
	Faction                  int        `db:"faction" json:"faction"`
	Height                   int        `db:"height" json:"height"`
	Level                    int        `db:"level" json:"level"`
	Class                    int        `db:"class" json:"class"`
	IsOnline                 bool       `db:"is_online" json:"is_online"`
	IsActive                 bool       `db:"is_active" json:"is_active"`
	Gold                     uint64     `db:"gold" json:"gold"`
	Coordinate               string     `db:"coordinate" json:"coordinate"`
	Map                      int16      `db:"map" json:"map"`
	Exp                      int64      `db:"exp" json:"exp"`
	HTVisibility             int        `db:"ht_visibility" json:"ht_visibility"`
	WeaponSlot               int        `db:"weapon_slot" json:"weapon_slot"`
	RunningSpeed             float64    `db:"running_speed" json:"running_speed"`
	GuildID                  int        `db:"guild_id" json:"guild_id"`
	ExpMultiplier            float64    `db:"exp_multiplier" json:"exp_multiplier"`
	DropMultiplier           float64    `db:"drop_multiplier" json:"drop_multiplier"`
	Slotbar                  []byte     `db:"slotbar" json:"slotbar"`
	CreatedAt                null.Time  `db:"created_at" json:"created_at"`
	AdditionalExpMultiplier  float64    `db:"additional_exp_multiplier" json:"additional_exp_multiplier"`
	AdditionalDropMultiplier float64    `db:"additional_drop_multiplier" json:"additional_drop_multiplier"`
	AidMode                  bool       `db:"aid_mode" json:"aid_mode"`
	AidTime                  uint32     `db:"aid_time" json:"aid_time"`
	Injury                   float64    `db:"injury" json:"injury"`
	HeadStyle                int64      `db:"headstyle" json:"headstyle"`
	FaceStyle                int64      `db:"facestyle" json:"facestyle"`
	HonorRank                int64      `db:"rank" json:"rank"`
	AddingExp                sync.Mutex `db:"-" json:"-"`
	AddingGold               sync.Mutex `db:"-" json:"-"`
	Looting                  sync.Mutex `db:"-" json:"-"`
	AdditionalRunningSpeed   float64    `db:"-" json:"-"`
	InvMutex                 sync.Mutex `db:"-"`
	Socket                   *Socket    `db:"-" json:"-"`
	ExploreWorld             func()     `db:"-" json:"-"`
	HasLot                   bool       `db:"-" json:"-"`
	LastRoar                 time.Time  `db:"-" json:"-"`
	Meditating               bool       `db:"-"`
	MovementToken            int64      `db:"-" json:"-"`
	PseudoID                 uint16     `db:"-" json:"pseudo_id"`
	PTS                      int        `db:"-" json:"pts"`
	OnSight                  struct {
		Drops       map[int]interface{} `db:"-" json:"drops"`
		DropsMutex  sync.RWMutex
		Mobs        map[int]interface{} `db:"-" json:"mobs"`
		MobMutex    sync.RWMutex        `db:"-"`
		NPCs        map[int]interface{} `db:"-" json:"npcs"`
		NpcMutex    sync.RWMutex        `db:"-"`
		Pets        map[int]interface{} `db:"-" json:"pets"`
		PetsMutex   sync.RWMutex        `db:"-"`
		Players     map[int]interface{} `db:"-" json:"players"`
		PlayerMutex sync.RWMutex        `db:"-"`
	} `db:"-" json:"on_sight"`
	PrevInvisible        map[int]bool     `db:"-"`
	PartyID              string           `db:"-"`
	Selection            int              `db:"-" json:"selection"`
	Targets              []*Target        `db:"-" json:"target"`
	TamingAI             *AI              `db:"-" json:"-"`
	PlayerTargets        []*PlayerTarget  `db:"-" json:"player_targets"`
	TradeID              string           `db:"-" json:"trade_id"`
	Invisible            bool             `db:"-" json:"-"`
	DetectionMode        bool             `db:"-" json:"-"`
	Poisoned             bool             `db:"-" json:"-"`
	SufferedPoison       int              `db:"-" json:"-"`
	PoisonSource         *Character       `db:"-" json:"-"`
	Confusioned          bool             `db:"-" json:"-"`
	ConfusionSource      *Character       `db:"-" json:"-"`
	Paralysised          bool             `db:"-" json:"-"`
	ParaSource           *Character       `db:"-" json:"-"`
	VisitedSaleID        uint16           `db:"-" json:"-"`
	DuelID               int              `db:"-" json:"-"`
	LastAttackPacketTime time.Time        `db:"-" json:"-"`
	DuelStarted          bool             `db:"-" json:"-"`
	IsinWar              bool             `db:"-" json:"-"`
	Respawning           bool             `db:"-" json:"-"`
	SkillHistory         utils.SMap       `db:"-" json:"-"`
	Morphed              bool             `db:"-" json:"-"`
	MorphedNPCID         int              `db:"-" json:"-"`
	IsDungeon            bool             `db:"-" json:"-"`
	IsMounting           bool             `db:"-" json:"-"`
	DungeonLevel         int16            `db:"-" json:"-"`
	CanTip               int16            `db:"-" json:"-"`
	GeneratedNumber      int              `db:"-" json:"-"`
	WarKillCount         int              `db:"-" json:"-"`
	WarContribution      int              `db:"-" json:"-"`
	TriviaSelected       int              `db:"-" json:"-"`
	PartyMode            int              `db:"-" json:"-"`
	IsAcceptedWar        bool             `db:"-" json:"-"`
	IsInTraviaEvent      bool             `db:"-" json:"-"`
	IsQuestMenuOpened    bool             `db:"-" json:"-"`
	QuestActions         []int            `db:"-" json:"-"`
	LastNPCAction        int64            `db:"-" json:"-"`
	InjuryCount          float64          `db:"-" json:"-"`
	HandlerCB            func()           `db:"-"`
	PetHandlerCB         func()           `db:"-"`
	questMobsIDs         []int            `db:"-" json:"-"`
	MessageItems         []*MessageItems  `db:"-" json:"-"`
	inventory            []*InventorySlot `db:"-" json:"-"`
	PlayerAidSettings    *AidSettings     `db:"-" json:"-"`
	KilledByCharacter    *Character       `db:"-" json:"-"`
	PacketSended         bool             `db:"-" json:"-"`
	UsedPotion           bool             `db:"-" json:"-"`

	AntiDupeMutex   sync.RWMutex `db:"-"`
	ArrangeCooldown uint16       `db:"-"`

	GMAuthenticated string `db:"-"`

	ShowStats bool `db:"-" json:"-"`
}

func (t *Character) PreInsert(s gorp.SqlExecutor) error {
	now := time.Now().UTC()
	t.CreatedAt = null.TimeFrom(now)
	return nil
}

func (t *Character) SetCoordinate(coordinate *utils.Location) {
	t.Coordinate = fmt.Sprintf("(%.1f,%.1f)", coordinate.X, coordinate.Y)
}

func (t *Character) Create() error {
	return db.Insert(t)
}

func (t *Character) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(t)
}

func (t *Character) PreUpdate(s gorp.SqlExecutor) error {
	if int64(t.Gold) < 0 {
		t.Gold = 0
	}
	return nil
}

func (t *Character) Update() error {
	_, err := db.Update(t)
	if err != nil {
		fmt.Println("Updating Character Socket failed")
		log.Println(err)
	}
	return err
}

func (t *Character) Delete() error {
	characterMutex.Lock()
	defer characterMutex.Unlock()

	delete(characters, t.ID)
	_, err := db.Delete(t)
	return err
}

func (t *Character) InventorySlots() ([]*InventorySlot, error) {

	if len(t.inventory) > 0 {
		return t.inventory, nil
	}

	inventory := make([]*InventorySlot, 450)

	for i := range inventory {
		inventory[i] = NewSlot()
	}

	slots, err := FindInventorySlotsByCharacterID(t.ID)
	if err != nil {

		return nil, err
	}

	bankSlots, err := FindBankSlotsByUserID(t.UserID)
	if err != nil {
		return nil, err
	}

	for _, s := range slots {
		inventory[s.SlotID] = s
	}

	for _, s := range bankSlots {
		inventory[s.SlotID] = s
	}

	t.inventory = inventory

	return inventory, nil
}

func (t *Character) SetInventorySlots(slots []*InventorySlot) { // FIX HERE
	t.inventory = slots
}

func (t *Character) CopyInventorySlots() []*InventorySlot {
	slots := []*InventorySlot{}
	for _, s := range t.inventory {
		copySlot := *s
		slots = append(slots, &copySlot)
	}

	return slots
}

func RefreshAIDs() error {
	query := `update hops.characters SET aid_time = 9999999`
	_, err := db.Exec(query)
	if err != nil {
		return err
	}
	characterMutex.RLock()
	allChars := funk.Values(characters).([]*Character)
	characterMutex.RUnlock()
	for _, c := range allChars {
		c.AidTime = 9999999
	}

	return err
}
func RefreshOnline() error {
	query := `update hops.users SET loginfrompanel = false`
	_, err := db.Exec(query)
	if err != nil {
		return err
	}

	return err
}

func FindCharactersByUserID(userID string) ([]*Character, error) {
	characterMutex.Lock()
	defer characterMutex.Unlock()
	charMap := make(map[int]*Character)
	for _, c := range characters {
		if c.UserID == userID {
			charMap[c.ID] = c
		}
	}

	var arr []*Character
	query := `select * from hops.characters where user_id = $1`

	if _, err := db.Select(&arr, query, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindCharactersByUserID: %s", err.Error())
	}

	var chars []*Character
	for _, c := range arr {
		char, ok := charMap[c.ID]
		if ok {
			chars = append(chars, char)
		} else {
			characters[c.ID] = c
			chars = append(chars, c)
		}
	}

	return chars, nil
}

func IsValidUsername(name string) (bool, error) {

	var (
		count int64
		err   error
		query string
	)

	re := regexp.MustCompile("^[a-zA-Z0-9]{4,18}$")
	if !re.MatchString(name) {
		return false, nil
	}

	query = `select count(*) from hops.characters where lower(name) = $1`

	if count, err = db.SelectInt(query, strings.ToLower(name)); err != nil {
		return false, fmt.Errorf("IsValidUsername: %s", err.Error())
	}

	return count == 0, nil
}

func FindCharacterByName(name string) (*Character, error) {

	for _, c := range characters {
		if c.Name == name {
			return c, nil
		}
	}

	character := &Character{}
	query := `select * from hops.characters where name = $1`

	if err := db.SelectOne(&character, query, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindCharacterByName: %s", err.Error())
	}

	characterMutex.Lock()
	defer characterMutex.Unlock()
	characters[character.ID] = character

	return character, nil
}

func FindAllCharacter() ([]*Character, error) {

	charMap := make(map[int]*Character)

	var arr []*Character
	query := `select * from hops.characters`

	if _, err := db.Select(&arr, query); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindAllCharacter: %s", err.Error())
	}

	characterMutex.Lock()
	defer characterMutex.Unlock()

	var chars []*Character
	for _, c := range arr {
		char, ok := charMap[c.ID]
		if ok {
			chars = append(chars, char)
		} else {
			characters[c.ID] = c
			chars = append(chars, c)
		}
	}

	return chars, nil
}

func FindCharacterByID(id int) (*Character, error) {

	characterMutex.RLock()
	c, ok := characters[id]
	characterMutex.RUnlock()

	if ok {
		return c, nil
	}

	character := &Character{}
	query := `select * from hops.characters where id = $1`

	if err := db.SelectOne(&character, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindCharacterByID: %s", err.Error())
	}

	characterMutex.Lock()
	defer characterMutex.Unlock()
	characters[character.ID] = character

	return character, nil
}

func (c *Character) GetAppearingItemSlots() []int {

	helmSlot := 0
	if c.HTVisibility&0x01 != 0 {
		helmSlot = 0x0133
	}

	maskSlot := 1
	if c.HTVisibility&0x02 != 0 {
		maskSlot = 0x0134
	}

	armorSlot := 2
	if c.HTVisibility&0x04 != 0 {
		armorSlot = 0x0135
	}

	bootsSlot := 9
	if c.HTVisibility&0x10 != 0 {
		bootsSlot = 0x0136
	}

	armorSlot2 := 2
	if c.HTVisibility&0x08 != 0 {
		armorSlot2 = 0x0137
	}

	if armorSlot2 != 2 {
		armorSlot = armorSlot2
	}

	return []int{helmSlot, maskSlot, armorSlot, 3, 4, 5, 6, 7, 8, bootsSlot, 10}
}

func (c *Character) GetEquipedItemSlots() []int {
	return []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 307, 309, 310, 312, 313, 314, 315}
}

func (c *Character) GetAllEquipedSlots() []int {
	return []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 307, 309, 310, 312, 313, 314, 315, 317, 318, 319}
}
func (c *Character) Logout() {
	c.IsOnline = false
	c.IsActive = false
	c.IsDungeon = false
	c.OnSight.Drops = map[int]interface{}{}
	c.OnSight.Mobs = map[int]interface{}{}
	c.OnSight.NPCs = map[int]interface{}{}
	c.OnSight.Pets = map[int]interface{}{}
	c.OnSight.Players = map[int]interface{}{}
	c.ExploreWorld = nil
	c.HandlerCB = nil
	c.PetHandlerCB = nil
	c.PTS = 0
	c.TradeID = ""
	c.GMAuthenticated = ""
	c.LeaveParty()
	c.EndPvP()
	sale := FindSale(c.PseudoID)
	if sale != nil {
		sale.Delete()
	}

	if trade := FindTrade(c); trade != nil {
		c.CancelTrade()
	}

	if FindPlayerInArena(c) && !FindPlayerArena(c).IsFinished {
		PlayerDeserterPunishment(c)
		RemovePlayerFromArena(c)
	}
	if c.GuildID > 0 {
		guild, err := FindGuildByID(c.GuildID)
		if err == nil && guild != nil {
			guild.InformMembers(c)
		}
	}
	if c.IsinWar {
		c.IsinWar = false
		if c.Faction == 1 {
			delete(OrderCharacters, c.ID)
		} else {
			delete(ShaoCharacters, c.ID)
		}
		c.Map = 1
	}
	//DELETE FIVE TEMPLE BUFF
	LogoutFiveBuffDelete(c)

	friends, _ := FindAllCharacterByFriendID(c.ID)
	for _, friend := range friends {
		char, err := FindCharacterByID(friend.CharacterID)
		if err != nil {
			continue
		}
		if char == nil {
			continue
		}
		index := 6
		resp := MODIFY_FRIEND
		resp.Insert(utils.IntToBytes(uint64(friend.ID), 4, true), index)
		index += 4
		online, err := boolconv.NewBoolByInterface(c.IsOnline)
		if err != nil {
			log.Println("error should not be nil")
		}
		resp.Overwrite(online.Bytes(), index)
		resp.SetLength(int16(binary.Size(resp) - 6))
		char.Socket.Write(resp)
	}
	err := c.Update()
	if err != nil {
		log.Println("Logout character sql error: ", err)
	}
	c.Socket.User.Update()
	RemoveFromRegister(c)
	RemovePetFromRegister(c)
	DeleteCharacterFromCache(c.ID)
	//DeleteStatFromCache(c.ID)
}

func (c *Character) EndPvP() {
	if c.DuelID > 0 {
		op, _ := FindCharacterByID(c.DuelID)
		if op != nil {
			op.Socket.Write(PVP_FINISHED)
			op.DuelID = 0
			op.DuelStarted = false
		}
		c.DuelID = 0
		c.DuelStarted = false
		c.Socket.Write(PVP_FINISHED)
	}
}

func DeleteCharacterFromCache(id int) {
	characterMutex.Lock()
	defer characterMutex.Unlock()

	delete(characters, id)
}

func (c *Character) GetNearbyCharacters() ([]*Character, error) {
	var (
		distance = float64(150)
	)

	u, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	}

	myCoordinate := ConvertPointToLocation(c.Coordinate)
	characterMutex.RLock()
	allChars := funk.Values(characters)
	characterMutex.RUnlock()
	characters := funk.Filter(allChars, func(character *Character) bool {

		user, err := FindUserByID(character.UserID)
		if err != nil || user == nil {
			return false
		}

		characterCoordinate := ConvertPointToLocation(character.Coordinate)

		return character.IsOnline && (user.ConnectedServer == u.ConnectedServer || character.Map == 1 || character.IsinWar) && character.Map == c.Map &&
			(!character.Invisible || c.DetectionMode) && utils.CalculateDistance(characterCoordinate, myCoordinate) <= distance
	}).([]*Character)

	return characters, nil
}

func (c *Character) GetNearbyAIIDs() ([]int, error) {

	var (
		distance = 100.0
		ids      []int
	)
	if funk.Contains(DungeonZones, c.Map) {
		distance = 100.0
	}
	if c.IsinWar {
		distance = 100.0
	}

	user, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	candidates := AIsByMap[user.ConnectedServer][c.Map]
	filtered := funk.Filter(candidates, func(ai *AI) bool {

		characterCoordinate := ConvertPointToLocation(c.Coordinate)
		aiCoordinate := ConvertPointToLocation(ai.Coordinate)

		return utils.CalculateDistance(characterCoordinate, aiCoordinate) <= distance
	})

	for _, ai := range filtered.([]*AI) {
		ids = append(ids, ai.ID)
	}

	return ids, nil
}

func (c *Character) GetQuestNPCID(sNPCID int64) (int, error) {

	var (
		//distance = 50.0
		ids int
	)

	user, err := FindUserByID(c.UserID)
	if err != nil {
		return 0, err
	} else if user == nil {
		return 0, nil
	}

	filtered := funk.Filter(NPCPos, func(pos *NpcPosition) bool {
		return c.Map == pos.MapID && pos.IsNPC && !pos.Attackable && pos.NPCID == int(sNPCID)
	})

	for _, pos := range filtered.([]*NpcPosition) {
		return pos.ID, nil
	}

	return ids, nil
}

func (c *Character) GetWholeMapNPCIDs() ([]int, error) {

	var (
		//distance = 50.0
		ids []int
	)

	user, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	filtered := funk.Filter(NPCPos, func(pos *NpcPosition) bool {
		return c.Map == pos.MapID && pos.IsNPC && !pos.Attackable
	})

	for _, pos := range filtered.([]*NpcPosition) {
		ids = append(ids, pos.ID)
	}

	return ids, nil
}

func (c *Character) GetNearbyNPCIDs() ([]int, error) {

	var (
		distance = 100.0
		ids      []int
	)

	user, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	filtered := funk.Filter(NPCPos, func(pos *NpcPosition) bool {

		characterCoordinate := ConvertPointToLocation(c.Coordinate)
		minLocation := ConvertPointToLocation(pos.MinLocation)
		maxLocation := ConvertPointToLocation(pos.MaxLocation)

		npcCoordinate := &utils.Location{X: (minLocation.X + maxLocation.X) / 2, Y: (minLocation.Y + maxLocation.Y) / 2}
		return c.Map == pos.MapID && utils.CalculateDistance(characterCoordinate, npcCoordinate) <= distance && pos.IsNPC
	})

	for _, pos := range filtered.([]*NpcPosition) {
		ids = append(ids, pos.ID)
	}

	return ids, nil
}

func (c *Character) GetNearbyDrops() ([]int, error) {

	var (
		distance = 50.0
		ids      []int
	)

	user, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	allDrops := GetDropsInMap(user.ConnectedServer, c.Map)
	filtered := funk.Filter(allDrops, func(drop *Drop) bool {

		characterCoordinate := ConvertPointToLocation(c.Coordinate)

		return utils.CalculateDistance(characterCoordinate, &drop.Location) <= distance
	})

	for _, d := range filtered.([]*Drop) {
		ids = append(ids, d.ID)
	}

	return ids, nil
}

func (c *Character) SpawnCharacter() ([]byte, error) {

	if c == nil {
		return nil, nil
	}
	if c.Socket == nil {
		return nil, nil
	}
	resp := CHARACTER_SPAWNED
	index := 6
	//testIndex := len(resp) - 9
	//resp.Overwrite([]byte{0xFF, 0xFF, 0xFF, 0xFF}, testIndex)
	resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // character pseudo id
	index += 2
	resp.Insert([]byte{0xee, 0x22, 0x00, 0x00}, index)
	index += 4
	if c.IsActive {
		resp.Insert([]byte{0x03, 0x00, 0x00, 0x00, 0x00}, index)
	} else {
		resp.Insert([]byte{0x04, 0x00, 0x00, 0x00, 0x00}, index)
	}
	index += 5

	if c.DuelID > 0 || funk.Contains(PvPZones, c.Map) || funk.Contains(PVPServers, int16(c.Socket.User.ConnectedServer)) || funk.Contains(ArenaZones, c.Map) {
		resp.Overwrite(utils.IntToBytes(500, 2, true), 13) // duel state
	}

	resp.Insert(utils.IntToBytes(uint64(len(c.Name)), 1, true), index)
	index++
	resp.Insert([]byte(c.Name), index) // character name
	index += len(c.Name)
	resp.Insert(utils.IntToBytes(uint64(c.Level), 4, true), index)
	index += 4
	resp.Insert(utils.IntToBytes(uint64(c.Type), 1, true), index) // character type
	index++
	resp.Insert([]byte{0x1b, 0x00, 0x00, 0x20, 0x1c, 0x00, 0x00, 0x00}, index)
	index += 8

	coordinate := ConvertPointToLocation(c.Coordinate)
	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), index) // coordinate-x
	index += 4

	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index) // coordinate-y
	index += 4

	resp.Insert([]byte{0x00, 0x00, 0x00, 0x00}, index)
	index += 4

	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), index) // coordinate-x
	index += 4

	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index) // coordinate-y
	index += 4
	resp.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff}, index)
	index += 10
	resp.Insert(utils.IntToBytes(uint64(10000), 4, true), index) // HONOR
	index += 4
	resp.Insert([]byte{0xc8, 0x00, 0x00, 0x00}, index)
	index += 4

	resp.Insert(utils.IntToBytes(uint64(c.Socket.Stats.HP), 4, true), index) // hp
	index += 4
	typeHonor, honorRank := c.GiveRankByHonorPoints()
	resp.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xfe, 0x03, 0x00, 0x00, 0x00, 0x05}, index)
	idgindex := index + 12
	resp.Insert(utils.IntToBytes(uint64(honorRank), 4, true), idgindex)
	resp.Insert([]byte{byte(typeHonor)}, idgindex+4)
	resp.Insert([]byte{0xff, 0xff, 0xff, 0xff, 0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x64, 0xff, 0xff, 0xff, 0xff, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}, idgindex+5)
	index += 6
	resp[index] = byte(c.WeaponSlot) // weapon slot
	index += 6                       //16 volt
	if c.Morphed {
		resp.Insert(utils.IntToBytes(uint64(c.MorphedNPCID), 4, true), index)
		index += 4
	} else {
		resp.Insert([]byte{0x00, 0x00, 0x00, 0x00}, index)
		index += 4
	}
	index += 5
	resp.Overwrite(utils.IntToBytes(uint64(c.GuildID), 4, true), index) // guild id
	index += 8

	resp[index] = byte(c.Faction) // character faction
	index += 10
	items, err := c.ShowItems()
	if err != nil {
		return nil, err
	}

	itemsData := items[11 : len(items)-2]
	sale := FindSale(c.PseudoID)
	if sale != nil {
		itemsData = []byte{0x05, 0xAA, 0x45, 0xF1, 0x00, 0x00, 0x00, 0xA1, 0x00, 0x00, 0x00, 0x00, 0xB4, 0x6C, 0xF1, 0x00, 0x01, 0x00, 0xA1, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	}

	resp.Overwrite(itemsData, index)
	index += len(itemsData)

	if sale != nil {
		resp.Insert([]byte{0x02}, index) // sale indicator
		index++

		resp.Insert([]byte{byte(len(sale.Name))}, index) // sale name length
		index++

		resp.Insert([]byte(sale.Name), index) // sale name
		index += len(sale.Name)

		resp.Insert([]byte{0x00}, index)
		index++
	}
	resp.SetLength(int16(binary.Size(resp) - 6))

	resp.Concat(items) // FIX => workaround for weapon slot

	if c.GuildID > 0 {
		guild, err := FindGuildByID(c.GuildID)
		if err == nil && guild != nil {
			resp.Concat(guild.GetInfo())
		}
	}

	STYLE_MENU := utils.Packet{0xaa, 0x55, 0x0d, 0x00, 0x01, 0xb5, 0x0a, 0x00, 0x00, 0x55, 0xaa}
	styleresp := STYLE_MENU
	styleresp[8] = byte(0x02)
	index = 9
	styleresp.Insert(utils.IntToBytes(uint64(c.HeadStyle), 4, true), index)
	index += 4
	styleresp.Insert(utils.IntToBytes(uint64(c.FaceStyle), 4, true), index)
	index += 4
	resp.Concat(styleresp)
	return resp, nil
}

func (c *Character) ShowItems() ([]byte, error) {

	if c == nil {
		return nil, nil
	}

	slots := c.GetAppearingItemSlots()
	inventory, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	helm := inventory[slots[0]]
	mask := inventory[slots[1]]
	armor := inventory[slots[2]]
	weapon1 := inventory[slots[3]]
	weapon2 := inventory[slots[4]]
	boots := inventory[slots[9]]
	pet := inventory[slots[10]].Pet
	petid := inventory[slots[10]].ItemID
	count := byte(4)
	if weapon1.ItemID > 0 {
		count++
	}
	if weapon2.ItemID > 0 {
		count++
	}
	//if pet != nil && pet.IsOnline {
	count++
	//}
	weapon1ID := int64(0)
	if weapon1.Appearance != 0 {
		weapon1ID = weapon1.Appearance
	}
	weapon2ID := int64(0)
	if weapon2.Appearance != 0 {
		weapon2ID = weapon2.Appearance
	}
	helmID := int64(0)
	if slots[0] == 0 && helm.Appearance != 0 {
		helmID = helm.Appearance
	}
	if helmID == 0 {
		helmID = int64(0)
	}
	maskID := int64(0)
	if slots[1] == 1 && mask.Appearance != 0 {
		maskID = mask.Appearance
	}
	if maskID == 0 {
		maskID = int64(0)
	}
	armorID := int64(0)
	if slots[2] == 2 && armor.Appearance != 0 {
		armorID = armor.Appearance
	}
	bootsID := int64(0)
	if slots[9] == 9 && boots.Appearance != 0 {
		bootsID = boots.Appearance
	}
	resp := SHOW_ITEMS
	resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 8) // character pseudo id
	resp[10] = byte(c.WeaponSlot)                                 // character weapon slot
	resp[11] = count

	index := 12
	resp.Insert(utils.IntToBytes(uint64(helm.ItemID), 4, true), index) // helm id
	index += 4

	resp.Insert(utils.IntToBytes(uint64(slots[0]), 2, true), index) // helm slot
	resp.Insert([]byte{0xA2}, index+2)
	index += 3

	resp.Insert(utils.IntToBytes(uint64(helm.Plus), 1, true), index) // helm plus
	resp.Insert(utils.IntToBytes(uint64(helmID), 4, true), index+1)  // Kinézet
	index += 5

	resp.Insert(utils.IntToBytes(uint64(mask.ItemID), 4, true), index) // mask id
	index += 4

	resp.Insert(utils.IntToBytes(uint64(slots[1]), 2, true), index) // mask slot
	resp.Insert([]byte{0xA2}, index+2)
	index += 3

	resp.Insert(utils.IntToBytes(uint64(mask.Plus), 1, true), index) // mask plus
	resp.Insert(utils.IntToBytes(uint64(maskID), 4, true), index+1)  // Kinézet
	index += 5

	resp.Insert(utils.IntToBytes(uint64(armor.ItemID), 4, true), index) // armor id
	index += 4

	resp.Insert(utils.IntToBytes(uint64(slots[2]), 2, true), index) // armor slot
	resp.Insert([]byte{0xA2}, index+2)
	index += 3

	resp.Insert(utils.IntToBytes(uint64(armor.Plus), 1, true), index) // armor plus
	resp.Insert(utils.IntToBytes(uint64(armorID), 4, true), index+1)  // Kinézet
	index += 5

	if weapon1.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(weapon1.ItemID), 4, true), index) // weapon1 id
		index += 4

		resp.Insert([]byte{0x03, 0x00}, index) // weapon1 slot
		resp.Insert([]byte{0xA2}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(weapon1.Plus), 1, true), index) // weapon1 plus
		resp.Insert(utils.IntToBytes(uint64(weapon1ID), 4, true), index+1)  // Kinézet
		index += 5
	}

	if weapon2.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(weapon2.ItemID), 4, true), index) // weapon2 id
		index += 4

		resp.Insert([]byte{0x04, 0x00}, index) // weapon2 slot
		resp.Insert([]byte{0xA2}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(weapon2.Plus), 1, true), index) // weapon2 plus
		resp.Insert(utils.IntToBytes(uint64(weapon2ID), 4, true), index+1)  // Kinézet
		index += 5
	}

	resp.Insert(utils.IntToBytes(uint64(boots.ItemID), 4, true), index) // boots id
	index += 4

	resp.Insert(utils.IntToBytes(uint64(slots[9]), 2, true), index) // boots slot
	resp.Insert([]byte{0xA2}, index+2)
	index += 3

	resp.Insert(utils.IntToBytes(uint64(boots.Plus), 1, true), index) // boots plus
	resp.Insert(utils.IntToBytes(uint64(bootsID), 4, true), index+1)  // Kinézet
	index += 5

	//
	resp.Insert(utils.IntToBytes(uint64(inventory[10].ItemID), 4, true), index) // pet id
	index += 4

	resp.Insert(utils.IntToBytes(uint64(slots[10]), 2, true), index) // pet slot
	index += 2
	if pet != nil {
		resp.Insert(utils.IntToBytes(uint64(pet.Level), 1, true), index) // pet plus ?
		index++
	} else {
		resp.Insert(utils.IntToBytes(uint64(0), 1, true), index) // pet plus ?
		index++
	}
	resp.Insert([]byte{0x05, 0x00, 0x00, 0x00}, index)
	index += 4
	if pet != nil {
		petInfo, ok := Pets[inventory[10].ItemID]
		if ok {
			c.Socket.Write(petInfo.LoadPetsSkills())
		}
	} else {
		if petid != 0 {
			c.Socket.Write(RemovePetsSkills(petid))
		}

	}

	resp.SetLength(int16(binary.Size(resp) - 6))
	return resp, nil
}

func (c *Character) ShowItemsByCharacter() ([]byte, error) {

	if c == nil {
		return nil, nil
	}

	slots := c.GetAppearingItemSlots()
	inventory, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	helm := inventory[slots[0]]
	mask := inventory[slots[1]]
	armor := inventory[slots[2]]
	weapon1 := inventory[slots[3]]
	weapon2 := inventory[slots[4]]
	boots := inventory[slots[9]]
	pet := inventory[slots[10]].Pet

	count := 0
	if weapon1.ItemID > 0 {
		count++
	}
	if weapon2.ItemID > 0 {
		count++
	}
	if helm.ItemID > 0 {
		count++
	}
	if mask.ItemID > 0 {
		count++
	}
	if armor.ItemID > 0 {
		count++
	}
	if boots.ItemID > 0 {
		count++
	}
	if pet != nil {
		count++
	}

	weapon1ID := int64(0)
	if weapon1.Appearance != 0 {
		weapon1ID = weapon1.Appearance
	}
	weapon2ID := int64(0)
	if weapon2.Appearance != 0 {
		weapon2ID = weapon2.Appearance
	}
	helmID := int64(0)
	if slots[0] == 0 && helm.Appearance != 0 {
		helmID = helm.Appearance
	}
	if helmID == 0 {
		helmID = int64(0)
	}
	maskID := int64(0)
	if slots[1] == 1 && mask.Appearance != 0 {
		maskID = mask.Appearance
	}
	if maskID == 0 {
		maskID = int64(0)
	}
	armorID := int64(0)
	if slots[2] == 2 && armor.Appearance != 0 {
		armorID = armor.Appearance
	}
	bootsID := int64(0)
	if slots[9] == 9 && boots.Appearance != 0 {
		bootsID = boots.Appearance
	}
	resp := utils.Packet{}
	index := 0
	resp.Insert(utils.IntToBytes(uint64(count), 1, true), index) // count
	index++
	if helm.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(helm.ItemID), 4, true), index) // helm id
		index += 4

		resp.Insert(utils.IntToBytes(uint64(slots[0]), 2, true), index) // helm slot
		resp.Insert([]byte{0xA4}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(helm.Plus), 1, true), index) // helm plus
		resp.Insert(utils.IntToBytes(uint64(helmID), 4, true), index+1)  // Kinézet
		index += 5
	}
	if mask.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(mask.ItemID), 4, true), index) // mask id
		index += 4

		resp.Insert(utils.IntToBytes(uint64(slots[1]), 2, true), index) // mask slot
		resp.Insert([]byte{0xA4}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(mask.Plus), 1, true), index) // mask plus
		resp.Insert(utils.IntToBytes(uint64(maskID), 4, true), index+1)  // Kinézet
		index += 5
	}
	if armor.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(armor.ItemID), 4, true), index) // armor id
		index += 4

		resp.Insert(utils.IntToBytes(uint64(slots[2]), 2, true), index) // armor slot
		resp.Insert([]byte{0xA4}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(armor.Plus), 1, true), index) // armor plus
		resp.Insert(utils.IntToBytes(uint64(armorID), 4, true), index+1)  // Kinézet
		index += 5
	}
	if weapon1.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(weapon1.ItemID), 4, true), index) // weapon1 id
		index += 4

		resp.Insert([]byte{0x03, 0x00}, index) // weapon1 slot
		resp.Insert([]byte{0xA4}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(weapon1.Plus), 1, true), index) // weapon1 plus
		resp.Insert(utils.IntToBytes(uint64(weapon1ID), 4, true), index+1)  // Kinézet
		index += 5
	}
	if weapon2.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(weapon2.ItemID), 4, true), index) // weapon2 id
		index += 4

		resp.Insert([]byte{0x04, 0x00}, index) // weapon2 slot
		resp.Insert([]byte{0xA4}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(weapon2.Plus), 1, true), index) // weapon2 plus
		resp.Insert(utils.IntToBytes(uint64(weapon2ID), 4, true), index+1)  // Kinézet
		index += 5
	}
	if boots.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(boots.ItemID), 4, true), index) // boots id
		index += 4

		resp.Insert(utils.IntToBytes(uint64(slots[9]), 2, true), index) // boots slot
		resp.Insert([]byte{0xA4}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(boots.Plus), 1, true), index) // boots plus
		resp.Insert(utils.IntToBytes(uint64(bootsID), 4, true), index+1)  // Kinézet
		index += 5
	}
	if pet != nil {
		resp.Insert(utils.IntToBytes(uint64(inventory[10].ItemID), 4, true), index) // pet id
		index += 4

		resp.Insert(utils.IntToBytes(uint64(slots[10]), 2, true), index) // pet slot
		index += 2
		if pet != nil {
			resp.Insert(utils.IntToBytes(uint64(pet.Level), 1, true), index) // pet plus ?
			index++
		} else {
			resp.Insert(utils.IntToBytes(uint64(0), 1, true), index) // pet plus ?
			index++
		}
		resp.Insert([]byte{0x05, 0x00, 0x00, 0x00}, index)
		index += 4
	}

	//resp.SetLength(int16(binary.Size(resp) - 6))
	return resp, nil
}

func FindOnlineCharacterByUserID(userID string) (*Character, error) {

	var id int
	query := `select id from hops.characters where user_id = $1 and is_online = true`

	if err := db.SelectOne(&id, query, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindOnlineCharacterByUserID: %s", err.Error())
	}

	return FindCharacterByID(id)
}

func FindCharactersInServer(server int) (map[int]*Character, error) {

	characterMutex.RLock()
	allChars := funk.Values(characters).([]*Character)
	characterMutex.RUnlock()

	allChars = funk.Filter(allChars, func(c *Character) bool {
		if c.Socket == nil {
			return false
		}
		user := c.Socket.User
		if user == nil {
			return false
		}

		return user.ConnectedServer == server && c.IsOnline
	}).([]*Character)

	candidates := make(map[int]*Character)
	for _, c := range allChars {
		candidates[c.ID] = c
	}

	return candidates, nil
}

func FindOnlineCharacters() (map[int]*Character, error) {

	characters := make(map[int]*Character)
	users := AllUsers()
	users = funk.Filter(users, func(u *User) bool {
		return u.ConnectedIP != "" && u.ConnectedServer > 0
	}).([]*User)

	for _, u := range users {
		c, _ := FindOnlineCharacterByUserID(u.ID)
		if c == nil {
			continue
		}

		characters[c.ID] = c
	}

	return characters, nil
}
func (c *Character) FindItemInUsed(itemIDs []int64) (bool, error) {
	slots, err := c.InventorySlots()
	if err != nil {
		return false, err
	}

	for index, slot := range slots {
		if ok, _ := utils.Contains(itemIDs, slot.ItemID); ok {
			if index >= 0x43 && index <= 0x132 {
				continue
			}
			infoItem := Items[slot.ItemID]
			if slots[index].InUse && infoItem.HtType != 21 {
				return true, nil
			}
		}
	}
	return false, nil
}

func (c *Character) FindItemInInventory(callback func(*InventorySlot) bool, itemIDs ...int64) (int16, *InventorySlot, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return -1, nil, err
	}

	for index, slot := range slots {
		if ok, _ := utils.Contains(itemIDs, slot.ItemID); ok {
			if index >= 0x43 && index <= 0x132 {
				continue
			}

			if callback == nil || callback(slot) {
				return int16(index), slot, nil
			}
		}
	}

	return -1, nil, nil
}

func (c *Character) FindItemInInventoryForProduction(callback func(*InventorySlot) bool, itemIDs ...int64) (int16, *InventorySlot, error) {

	gearSlots := c.GetAllEquipedSlots()

	slots, err := c.InventorySlots()
	if err != nil {
		return -1, nil, err
	}

	for index, slot := range slots {
		if ok, _ := utils.Contains(itemIDs, slot.ItemID); ok {
			isEquipped := false
			for s := range gearSlots {
				if s == slot.ID {
					isEquipped = true
				}
			}

			if isEquipped {
				continue
			}

			if index >= 0x43 && index <= 0x132 {
				continue
			}

			if callback == nil || callback(slot) {
				return int16(index), slot, nil
			}
		}
	}

	return -1, nil, nil
}

func (c *Character) FindItemInInventoryByType(callback func(*InventorySlot) bool, itemTypes ...int16) (int16, *InventorySlot, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return -1, nil, err
	}

	for index, slot := range slots {
		if ok, _ := utils.Contains(itemTypes, slot.ItemType); ok {
			if index >= 0x43 && index <= 0x132 {
				continue
			}

			if callback == nil || callback(slot) {
				return int16(index), slot, nil
			}
		}
	}

	return -1, nil, nil
}

func (c *Character) FindAllItemsInInventoryByType(callback func(*InventorySlot) bool, itemTypes ...int16) ([]int16, []*InventorySlot, error) {
	var inventorySlots []int16
	var inventorySlotsInfo []*InventorySlot
	slots, err := c.InventorySlots()
	if err != nil {
		return inventorySlots, inventorySlotsInfo, err
	}

	for index, slot := range slots {
		if ok, _ := utils.Contains(itemTypes, slot.ItemType); ok {
			if index >= 0x43 && index <= 0x132 {
				continue
			}

			if callback == nil || callback(slot) {
				inventorySlots = append(inventorySlots, int16(index))
				inventorySlotsInfo = append(inventorySlotsInfo, slot)
				//return int16(index), slot, nil
			}
		}
	}
	if len(inventorySlots) > 0 {
		return inventorySlots, inventorySlotsInfo, nil
	}
	return inventorySlots, inventorySlotsInfo, nil
}

func (c *Character) DecrementItem(slotID int16, amount uint) *utils.Packet {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil
	}

	slot := slots[slotID]
	if slot == nil || slot.ItemID == 0 || slot.Quantity < amount {
		return nil
	}

	slot.Quantity -= amount

	info := Items[slot.ItemID]
	resp := utils.Packet{}

	if info.TimerType == 3 {
		resp = GREEN_ITEM_COUNT
		resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 8)         // slot id
		resp.Insert(utils.IntToBytes(uint64(slot.Quantity), 4, true), 10) // item quantity
	} else {
		resp = ITEM_COUNT
		resp.Insert(utils.IntToBytes(uint64(slot.ItemID), 4, true), 8)    // item id
		resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 12)        // slot id
		resp.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 14) // item quantity
	}

	if slot.Quantity == 0 {
		err = slot.Delete()
		if err != nil {
			log.Print(err)
		}
		*slot = *NewSlot()
	} else {
		err = slot.Update()
		if err != nil {
			log.Print(err)
		}
	}

	return &resp
}

func (c *Character) FindFreeSlot() (int16, error) {

	slotID := 11
	slots, err := c.InventorySlots()
	if err != nil {
		return -1, err
	}

	for ; slotID <= 66; slotID++ {
		slot := slots[slotID]
		if slot.ItemID == 0 {
			return int16(slotID), nil
		}
	}

	//if c.DoesInventoryExpanded() { //Quoted out because there is no such item in use yet
	slotID = 341
	for ; slotID <= 396; slotID++ {
		slot := slots[slotID]
		if slot.ItemID == 0 {
			return int16(slotID), nil
		}
	}
	//}

	return -1, nil
}

func (c *Character) FindFreeSlots(count int) ([]int16, error) {

	var slotIDs []int16
	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	for slotID := int16(11); slotID <= 66; slotID++ {
		slot := slots[slotID]
		if slot.ItemID == 0 {
			slotIDs = append(slotIDs, slotID)
		}
		if len(slotIDs) == count {
			return slotIDs, nil
		}
	}

	//if c.DoesInventoryExpanded() { //Quoted out because there is no such item in use yet
	for slotID := int16(341); slotID <= 396; slotID++ {
		slot := slots[slotID]
		if slot.ItemID == 0 {
			slotIDs = append(slotIDs, slotID)
		}
		if len(slotIDs) == count {
			return slotIDs, nil
		}
	}
	//}

	return nil, fmt.Errorf("not enough inventory space")
}

func (c *Character) DoesInventoryExpanded() bool {
	buffs, err := FindBuffsByCharacterID(c.ID)
	if err != nil || len(buffs) == 0 {
		return false
	}

	buffs = funk.Filter(buffs, func(b *Buff) bool {
		return b.BagExpansion
	}).([]*Buff)

	return len(buffs) > 0
}
func (c *Character) FindItemInInventoryByPlus(callback func(*InventorySlot) bool, itemPlus uint8, itemIDs ...int64) (int16, *InventorySlot, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return -1, nil, err
	}

	for index, slot := range slots {
		ok, _ := utils.Contains(itemIDs, slot.ItemID)
		//ok2, _ := utils.Contains(itemPlus, slot.Plus)
		if ok && itemPlus == slot.Plus {
			if index >= 0x43 && index <= 0x132 {
				continue
			}

			if callback == nil || callback(slot) {
				return int16(index), slot, nil
			}
		}
	}

	return -1, nil, nil
}
func (c *Character) FindStackableItemInInventory(callback func(*InventorySlot) bool, item *InventorySlot) (int16, *InventorySlot, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return -1, nil, err
	}

	for index, slot := range slots {
		if slot.ItemID == item.ItemID {
			if item.Plus != slot.Plus {
				continue
			}
			if index >= 0x43 && index <= 0x132 || index >= 402 {
				continue
			}
			if (callback == nil || callback(slot)) && slot.Quantity < 10000 {

				return int16(index), slot, nil
			}
		}
	}

	return -1, nil, nil
}

/*
	func (c *Character) AddItem(itemToAdd *InventorySlot, slotID int16, lootingDrop bool) (*utils.Packet, int16, error) {
		var (
			item *InventorySlot
		)

		if itemToAdd == nil {
			return nil, -1, nil
		}

		itemToAdd.CharacterID = null.IntFrom(int64(c.ID))
		itemToAdd.UserID = null.StringFrom(c.UserID)

		i := Items[itemToAdd.ItemID]
		stackable := FindStackableByUIF(i.UIF)
		//	fmt.Println("Stackable: ", stackable, " UIF: ", i.UIF)
		slots, err := c.InventorySlots()
		if err != nil {
			return nil, -1, err
		}

		stacking := false
		resp := utils.Packet{}
		if slotID == -1 {
			if stackable != nil { // stackable item
				if itemToAdd.Plus > 0 {
					slotID, item, err = c.FindItemInInventoryByPlus(nil, itemToAdd.Plus, itemToAdd.ItemID)
				} else {
					slotID, item, err = c.FindItemInInventory(nil, itemToAdd.ItemID)
				}
				if err != nil {
					return nil, -1, err
				} else if slotID == -1 { // no same item found => find free slot
					slotID, err = c.FindFreeSlot()
					if err != nil {
						return nil, -1, err
					} else if slotID == -1 { // no free slot
						return nil, -1, nil
					}
					stacking = false
				} else if item.ItemID != itemToAdd.ItemID { // slot is not available
					return nil, -1, nil
				} else if item != nil { // can be stacked
					itemToAdd.Quantity += item.Quantity
					stacking = true
				}
			} else { // not stackable item => find free slot
				slotID, err = c.FindFreeSlot()
				if err != nil {
					return nil, -1, err
				} else if slotID == -1 {
					return nil, -1, nil
				}
			}
		}

		itemToAdd.SlotID = slotID
		slot := slots[slotID]
		id := slot.ID
		*slot = *itemToAdd
		slot.ID = id

		if !stacking && stackable == nil {
			//for j := 0; j < int(itemToAdd.Quantity); j++ {

			if lootingDrop {
				r := ITEM_LOOTED
				r.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 9) // item id
				r[14] = 0xA1
				if itemToAdd.Plus > 0 || itemToAdd.SocketCount > 0 {
					r[14] = 0xA2
				}

				r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 15) // item count
				r.Insert(utils.IntToBytes(uint64(slotID), 2, true), 17)        // slot id
				r.Insert(itemToAdd.GetUpgrades(), 19)                          // item upgrades
				resp.Concat(r)
			} else {
				resp = ITEM_ADDED
				resp.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 6) // item id
				resp.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 12)   // item quantity
				resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 14)          // slot id
				resp.Insert(utils.IntToBytes(uint64(0), 8, true), 20)               // ures
				info := Items[itemToAdd.ItemID]
				if info.GetType() == PET_TYPE && slot.Pet.Name != "" {
					resp.Overwrite([]byte(slot.Pet.Name), 32)
				}

				resp.Overwrite(utils.IntToBytes(uint64(itemToAdd.ItemType), 1, true), 41) // ures
				resp.Overwrite(utils.IntToBytes(uint64(itemToAdd.JudgementStat), 4, true), 42)
				if slot.Appearance != 0 {
					resp.Overwrite(utils.IntToBytes(uint64(slot.Appearance), 4, true), 46) // KINÉZET id 16 volt
				}
				//r.Insert(utils.IntToBytes(uint64(c.Gold), 8, true), 20) // gold
				//resp.Concat(r)
			}
			resp.Concat(c.GetGold())
			/*
				slotID, err = c.FindFreeSlot()
				if err != nil || slotID == -1 {
					break
				}

				slot = slots[slotID]

			//}
		} else {

			if lootingDrop {
				r := ITEM_LOOTED
				r.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 9) // item id
				r[14] = 0xA1
				r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 15) // item count
				r.Insert(utils.IntToBytes(uint64(slotID), 2, true), 17)        // slot id
				r.Insert(itemToAdd.GetUpgrades(), 19)                          // item upgrades
				resp.Concat(r)
			} else if stacking {
				resp = ITEM_COUNT
				resp.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 8)    // item id
				resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 12)             // slot id
				resp.Insert(utils.IntToBytes(uint64(itemToAdd.Quantity), 2, true), 14) // item quantity
			} else if !stacking {
				slot := slots[slotID]
				slot.ItemID = itemToAdd.ItemID
				slot.Quantity = itemToAdd.Quantity
				slot.Plus = itemToAdd.Plus
				slot.UpgradeArr = itemToAdd.UpgradeArr

				resp = ITEM_ADDED
				resp.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 6)    // item id
				resp.Insert(utils.IntToBytes(uint64(itemToAdd.Quantity), 2, true), 12) // item quantity
				resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 14)             // slot id
				resp.Insert(utils.IntToBytes(uint64(0), 8, true), 20)                  // gold
				info := Items[itemToAdd.ItemID]
				if info.GetType() == PET_TYPE && slot.Pet.Name != "" {
					resp.Overwrite([]byte(slot.Pet.Name), 32)
				}
				resp.Overwrite(utils.IntToBytes(uint64(itemToAdd.ItemType), 1, true), 41) // ures
				resp.Overwrite(utils.IntToBytes(uint64(itemToAdd.JudgementStat), 4, true), 42)
				if slot.Appearance != 0 {
					resp.Overwrite(utils.IntToBytes(uint64(slot.Appearance), 4, true), 46) // KINÉZET id 16 volt
				}
			}
			resp.Concat(c.GetGold())
		}

		if slot.ID > 0 {
			err = slot.Update()
		} else {
			err = slot.Insert()
		}

		if err != nil {
			*slot = *NewSlot()
			resp = utils.Packet{}
			resp.Concat(slot.GetData(slotID))
			return &resp, -1, nil
		}

		InventoryItems.Add(slot.ID, slot)
		resp.Concat(slot.GetData(slotID))
		return &resp, slotID, nil
	}
*/
func (c *Character) AddItem(itemToAdd *InventorySlot, slotID int16, lootingDrop bool) (*utils.Packet, int16, error) {

	var (
		item *InventorySlot
	)

	if itemToAdd == nil {
		return nil, -1, nil
	}

	itemToAdd.CharacterID = null.IntFrom(int64(c.ID))
	itemToAdd.UserID = null.StringFrom(c.UserID)

	iteminfo, ok := GetItemInfo(itemToAdd.ItemID)
	if !ok || iteminfo == nil {
		return nil, -1, nil
	}
	stackableUIF := FindStackableByUIF(iteminfo.UIF)
	stackable := stackableUIF != nil
	slots, err := c.InventorySlots()
	if err != nil {
		return nil, -1, err
	}

	stacking := false
	resp := utils.Packet{}
	if slotID == -1 {
		if stackable { // stackable item
			slotID, item, err = c.FindStackableItemInInventory(nil, itemToAdd)
			if err != nil {
				return nil, -1, err
			} else if slotID == -1 { // no same item found => find free slot
				slotID, err = c.FindFreeSlot()
				if err != nil {
					return nil, -1, err
				} else if slotID == -1 { // no free slot

					return nil, -1, nil
				}
				stacking = false
			} else if item.ItemID != itemToAdd.ItemID { // slot is not available

				return nil, -1, nil

				/*} else if item.Quantity >= 10000 { // max items stockable 999
				slotID, item, err = c.FindStockableItemInInventory(nil, itemToAdd)
				if err != nil {
					return nil, -1, nil
				}
				if slotID == -1 {
					slotID, err = c.FindFreeSlot()
					if err != nil {
						return nil, -1, err
					} else if slotID == -1 {
						return nil, -1, nil
					}
				} else {
					itemToAdd.Quantity += item.Quantity
					stacking = true
				}*/

			} else if item != nil { // can be stacked
				itemToAdd.Quantity += item.Quantity
				stacking = true
			}
		} else { // not stackable item => find free slot
			slotID, err = c.FindFreeSlot()
			if err != nil {
				return nil, -1, err
			} else if slotID == -1 {
				return nil, -1, nil
			}
		}
	}

	itemToAdd.SlotID = slotID
	slot := slots[slotID]
	id := slot.ID
	*slot = *itemToAdd
	slot.ID = id

	if !stacking && !stackable {

		if lootingDrop {
			r := ITEM_LOOTED
			r.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 9) // item id
			r[14] = 0xA1
			if itemToAdd.Plus > 0 || itemToAdd.SocketCount > 0 {
				r[14] = 0xA2
			}

			r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 15) // item count
			r.Insert(utils.IntToBytes(uint64(slotID), 2, true), 17)        // slot id
			r.Insert(itemToAdd.GetUpgrades(), 19)                          // item upgrades
			resp.Concat(r)
		} else {
			resp = ITEM_ADDED
			resp.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 6) // item id
			resp.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 12)   // item quantity
			resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 14)          // slot id
			resp.Insert(utils.IntToBytes(uint64(0), 8, true), 20)               // ures
			//info := Items[itemToAdd.ItemID]
			//if info.GetType() == PET_TYPE && slot.Pet.Name != "" {
			//resp.Overwrite([]byte(slot.Pet.Name), 32)
			//}

			resp.Overwrite(utils.IntToBytes(uint64(itemToAdd.ItemType), 1, true), 41) // ures
			resp.Overwrite(utils.IntToBytes(uint64(itemToAdd.JudgementStat), 4, true), 42)
			if slot.Appearance != 0 {
				resp.Overwrite(utils.IntToBytes(uint64(slot.Appearance), 4, true), 46) // KINÉZET id 16 volt
			}

			resp.Concat(messaging.InfoMessage(fmt.Sprintf("You acquired %s.", iteminfo.Name)))
		}
		resp.Concat(c.GetGold())
	} else {

		if lootingDrop {
			r := ITEM_LOOTED
			r.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 9) // item id
			r[14] = 0xA1
			r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 15) // item count
			r.Insert(utils.IntToBytes(uint64(slotID), 2, true), 17)        // slot id
			r.Insert(itemToAdd.GetUpgrades(), 19)                          // item upgrades
			resp.Concat(r)
		} else if stacking {
			resp = ITEM_COUNT
			resp.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 8)    // item id
			resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 12)             // slot id
			resp.Insert(utils.IntToBytes(uint64(itemToAdd.Quantity), 2, true), 14) // item quantity
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("You acquired %s.", iteminfo.Name)))
		} else if !stacking {
			slot := slots[slotID]
			slot.ItemID = itemToAdd.ItemID
			slot.Quantity = itemToAdd.Quantity
			slot.Plus = itemToAdd.Plus
			slot.UpgradeArr = itemToAdd.UpgradeArr

			resp = ITEM_ADDED
			resp.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 6)    // item id
			resp.Insert(utils.IntToBytes(uint64(itemToAdd.Quantity), 2, true), 12) // item quantity
			resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 14)             // slot id
			resp.Insert(utils.IntToBytes(uint64(0), 8, true), 20)                  // gold
			info, _ := GetItemInfo(itemToAdd.ItemID)
			if info.GetType() == PET_TYPE && slot.Pet.Name != "" {
				resp.Overwrite([]byte(slot.Pet.Name), 32)
			}
			resp.Overwrite(utils.IntToBytes(uint64(itemToAdd.ItemType), 1, true), 41) // ures
			resp.Overwrite(utils.IntToBytes(uint64(itemToAdd.JudgementStat), 4, true), 42)
			if slot.Appearance != 0 {
				resp.Overwrite(utils.IntToBytes(uint64(slot.Appearance), 4, true), 46) // KINÉZET id 16 volt
			}
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("You acquired %s.", iteminfo.Name)))
		}
		resp.Concat(c.GetGold())
	}

	if slot.ID > 0 {
		err = slot.Update()
	} else {
		err = slot.Insert()
	}

	if err != nil {
		*slot = *NewSlot()
		resp = utils.Packet{}
		resp.Concat(slot.GetData(slotID, c.ID))
		return &resp, -1, nil
	}

	InventoryItems.Add(slot.ID, slot)
	resp.Concat(slot.GetData(slotID, c.ID))
	return &resp, slotID, nil
}

func (c *Character) ReplaceItem(itemID int, where, to int16) ([]byte, error) {

	c.AntiDupeMutex.Lock()
	defer c.AntiDupeMutex.Unlock()
	sale := FindSale(c.PseudoID)
	if sale != nil {
		return nil, fmt.Errorf("cannot replace item on sale")
	} else if c.TradeID != "" {
		return nil, fmt.Errorf("cannot replace item on trade")
	}
	invSlots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}
	/*if to == 10 && (c.Class != 25 || c.Class != 26 || c.Class != 13 || c.Class != 33) {
		return invSlots[where].GetData(int16(where)), nil

	}*/

	whereItem := invSlots[where]
	if whereItem.ItemID == 0 {
		return nil, nil
	}

	toItem := invSlots[to]
	whereInfoItem := Items[whereItem.ItemID]
	toInfoItem := Items[toItem.ItemID]
	slots := c.GetAllEquipedSlots()
	useItem, _ := utils.Contains(slots, int(to))
	isWeapon := false
	if useItem {
		if !c.CanUse(whereInfoItem.CharacterType) {
			return nil, errors.New("Cheat Warning")
		}
		if whereInfoItem.MinLevel > c.Level || (whereInfoItem.MaxLevel > 0 && whereInfoItem.MaxLevel < c.Level) {
			resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xF0, 0x03, 0x55, 0xAA} // inappropriate level
			return resp, nil
		}
		if whereInfoItem.Slot == 3 || whereInfoItem.Slot == 4 {
			if int(to) == 4 || int(to) == 3 {
				isWeapon = true
			}
		}
		if int(to) != whereInfoItem.Slot && isWeapon == false {
			return nil, errors.New("Cheat Warning")
		}
	}

	if (where >= 317 && where <= 319) && (to >= 317 && to <= 319) || where == 10 && to == 10 {
		if whereInfoItem.Slot != toInfoItem.Slot {
			return nil, errors.New("Cheat Warning")
		}
	}
	if (where >= 0x0043 && where <= 0x132) && (to >= 0x0043 && to <= 0x132) && toItem.ItemID == 0 { // From: Bank, To: Bank
		whereItem.SlotID = to
		*toItem = *whereItem
		*whereItem = *NewSlot()

	} else if (where >= 0x0043 && where <= 0x132) && (to < 0x0043 || to > 0x132) && toItem.ItemID == 0 { // From: Bank, To: Inventory
		whereItem.SlotID = to
		whereItem.CharacterID = null.IntFrom(int64(c.ID))
		*toItem = *whereItem
		*whereItem = *NewSlot()

	} else if (to >= 0x0043 && to <= 0x132) && (where < 0x0043 || where > 0x132) && toItem.ItemID == 0 &&
		!whereItem.Activated && !whereItem.InUse && whereInfoItem.Tradable { // From: Inventory, To: Bank
		whereItem.SlotID = to
		whereItem.CharacterID = null.IntFromPtr(nil)
		*toItem = *whereItem
		*whereItem = *NewSlot()

	} else if ((to < 0x0043 || to > 0x132) && (where < 0x0043 || where > 0x132)) && toItem.ItemID == 0 { // From: Inventory, To: Inventory
		whereItem.SlotID = to
		*toItem = *whereItem
		*whereItem = *NewSlot()

	} else {
		return nil, nil
	}

	toItem.Update()
	InventoryItems.Add(toItem.ID, toItem)

	resp := ITEM_REPLACEMENT
	resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 8) // item id
	resp.Insert(utils.IntToBytes(uint64(where), 2, true), 12) // where slot id
	resp.Insert(utils.IntToBytes(uint64(to), 2, true), 14)    // to slot id

	whereAffects, toAffects := DoesSlotAffectStats(where), DoesSlotAffectStats(to)

	info := Items[int64(itemID)]
	if whereAffects {
		if info != nil && info.Timer > 0 {
			toItem.InUse = false
		}
	}
	if toAffects {
		if info != nil && info.Timer > 0 {
			toItem.InUse = true
		}
	}

	if whereAffects || toAffects {
		statData, err := c.GetStats()
		if err != nil {
			return nil, err
		}

		resp.Concat(statData)

		itemsData, err := c.ShowItems()
		if err != nil {
			return nil, err
		}

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.SHOW_ITEMS, Data: itemsData}
		if err = p.Cast(); err != nil {
			return nil, err
		}

		resp.Concat(itemsData)
	}

	if to == 0x0A {
		resp.Concat(invSlots[to].GetPetStats(c))
		showpet, _ := c.ShowItems()
		resp.Concat(showpet)
	} else if where == 0x0A {
		c.Socket.Write(RemovePetsSkills(int64(itemID)))
		resp.Concat(DISMISS_PET)
		showpet, _ := c.ShowItems()
		resp.Concat(showpet)
	}

	if (where >= 317 && where <= 319) || (to >= 317 && to <= 319) {
		resp.Concat(c.GetPetStats())
	}

	return resp, nil
}

func (c *Character) SwapItems(where, to int16) ([]byte, error) {

	c.AntiDupeMutex.Lock()
	defer c.AntiDupeMutex.Unlock()
	sale := FindSale(c.PseudoID)
	if sale != nil {
		return nil, fmt.Errorf("cannot swap items on sale")
	} else if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to swap items in trade"
		utils.NewLog("logs/cheat_alert.txt", text)
		return nil, fmt.Errorf("cannot swap item on trade")
	}

	invSlots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	whereItem := invSlots[where]
	toItem := invSlots[to]

	if whereItem.ItemID == 0 || toItem.ItemID == 0 {
		return nil, nil
	}
	whereInfoItem := Items[whereItem.ItemID]
	toInfoItem := Items[toItem.ItemID]
	slots := c.GetAllEquipedSlots()
	useItem, _ := utils.Contains(slots, int(to))
	useItem2, _ := utils.Contains(slots, int(where))
	isWeapon := false
	if useItem || useItem2 {
		if !c.CanUse(toInfoItem.CharacterType) {
			return nil, errors.New("Cheat Warning")
		}
		if (where >= 317 && where <= 319) || (to >= 317 && to <= 319) || to == 10 || where == 10 {
			if whereInfoItem.Slot != toInfoItem.Slot {
				return nil, errors.New("Cheat Warning")
			}
		}
		if whereInfoItem.MinLevel > c.Level || toInfoItem.MinLevel > c.Level || (toInfoItem.MaxLevel > 0 && toInfoItem.MaxLevel < c.Level) || (whereInfoItem.MaxLevel > 0 && whereInfoItem.MaxLevel < c.Level) {
			resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xF0, 0x03, 0x55, 0xAA} // inappropriate level
			return resp, nil
		}
		if whereInfoItem.Slot == 3 || whereInfoItem.Slot == 4 || toInfoItem.Slot == 3 || toInfoItem.Slot == 4 {
			if int(to) == 4 || int(to) == 3 {
				isWeapon = true
			}
		}
		if int(to) != whereInfoItem.Slot && isWeapon == false {
			return nil, errors.New("Cheat Warning")
		}
	}
	if (where >= 0x0043 && where <= 0x132) && (to >= 0x0043 && to <= 0x132) { // From: Bank, To: Bank
		temp := *toItem
		*toItem = *whereItem
		*whereItem = temp
		toItem.SlotID = to
		whereItem.SlotID = where

	} else if (where >= 0x0043 && where <= 0x132) && (to < 0x0043 || to > 0x132) &&
		!toItem.Activated && !toItem.InUse { // From: Bank, To: Inventory

		temp := *toItem
		*toItem = *whereItem
		*whereItem = temp
		toItem.SlotID = to
		whereItem.SlotID = where

	} else if (to >= 0x0043 && to <= 0x132) && (where < 0x0043 || where > 0x132) &&
		!whereItem.Activated && !whereItem.InUse && whereInfoItem.Tradable && toInfoItem.Tradable { // From: Inventory, To: Bank

		temp := *toItem
		*toItem = *whereItem
		*whereItem = temp
		toItem.SlotID = to
		whereItem.SlotID = where

	} else if (to < 0x0043 || to > 0x132) && (where < 0x0043 || where > 0x132) { // From: Inventory, To: Inventory
		temp := *toItem
		*toItem = *whereItem
		*whereItem = temp
		toItem.SlotID = to
		whereItem.SlotID = where

	} else {
		return nil, nil
	}

	whereItem.Update()
	toItem.Update()
	InventoryItems.Add(whereItem.ID, whereItem)
	InventoryItems.Add(toItem.ID, toItem)

	resp := ITEM_SWAP
	resp.Insert(utils.IntToBytes(uint64(where), 4, true), 9)  // where slot
	resp.Insert(utils.IntToBytes(uint64(where), 2, true), 13) // where slot
	resp.Insert(utils.IntToBytes(uint64(to), 2, true), 15)    // to slot
	resp.Insert(utils.IntToBytes(uint64(to), 4, true), 17)    // to slot
	resp.Insert(utils.IntToBytes(uint64(to), 2, true), 21)    // to slot
	resp.Insert(utils.IntToBytes(uint64(where), 2, true), 23) // where slot

	whereAffects, toAffects := DoesSlotAffectStats(where), DoesSlotAffectStats(to)

	if whereAffects {
		item := whereItem // new item
		info := Items[int64(item.ItemID)]
		if info != nil && info.Timer > 0 {
			item.InUse = true
		}

		item = toItem // old item
		info = Items[int64(item.ItemID)]
		if info != nil && info.Timer > 0 {
			item.InUse = false
		}
	}

	if toAffects {
		item := whereItem // old item
		info := Items[int64(item.ItemID)]
		if info != nil && info.Timer > 0 {
			item.InUse = false
		}

		item = toItem // new item
		info = Items[int64(item.ItemID)]
		if info != nil && info.Timer > 0 {
			item.InUse = true
		}
	}

	if whereAffects || toAffects {

		statData, err := c.GetStats()
		if err != nil {
			return nil, err
		}

		resp.Concat(statData)

		itemsData, err := c.ShowItems()
		if err != nil {
			return nil, err
		}

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: itemsData, Type: nats.SHOW_ITEMS}
		if err = p.Cast(); err != nil {
			return nil, err
		}

		resp.Concat(itemsData)
	}

	if to == 0x0A {
		resp.Concat(invSlots[to].GetPetStats(c))
		showpet, _ := c.ShowItems()
		resp.Concat(showpet)
	}

	if (where >= 317 && where <= 319) || (to >= 317 && to <= 319) {
		resp.Concat(c.GetPetStats())
	}

	return resp, nil
}

func (c *Character) SplitItem(where, to, quantity uint16) ([]byte, error) {

	c.AntiDupeMutex.Lock()
	defer c.AntiDupeMutex.Unlock()
	sale := FindSale(c.PseudoID)
	if sale != nil {
		return nil, fmt.Errorf("cannot split item on sale")
	} else if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to split items in trade"
		utils.NewLog("logs/cheat_alert.txt", text)
		return nil, fmt.Errorf("cannot split item on trade")
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	whereItem := slots[where]
	toItem := slots[to]

	if quantity > 0 {

		if whereItem.Quantity >= uint(quantity) {
			*toItem = *whereItem
			toItem.SlotID = int16(to)
			toItem.Quantity = uint(quantity)
			c.DecrementItem(int16(where), uint(quantity))

		} else {
			return nil, nil
		}

		toItem.Insert()
		InventoryItems.Add(toItem.ID, toItem)

		resp := SPLIT_ITEM
		resp.Insert(utils.IntToBytes(uint64(toItem.ItemID), 4, true), 8)       // item id
		resp.Insert(utils.IntToBytes(uint64(whereItem.Quantity), 2, true), 14) // remaining quantity
		resp.Insert(utils.IntToBytes(uint64(where), 2, true), 16)              // where slot id

		resp.Insert(utils.IntToBytes(uint64(toItem.ItemID), 4, true), 52) // item id
		resp.Insert(utils.IntToBytes(uint64(quantity), 2, true), 58)      // new quantity
		resp.Insert(utils.IntToBytes(uint64(to), 2, true), 60)            // to slot id
		resp.Concat(toItem.GetData(int16(to)))
		if whereItem.Quantity > 9999 {
			c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("Remaining in stack: %d", whereItem.Quantity)))
		}
		return resp, nil
	}

	return nil, nil
}

func (c *Character) GetHPandChi() []byte {
	hpChi := HP_CHI
	stat := c.Socket.Stats

	hpChi.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 5)
	hpChi.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 7)
	hpChi.Insert(utils.IntToBytes(uint64(stat.HP), 4, true), 9)
	hpChi.Insert(utils.IntToBytes(uint64(stat.CHI), 4, true), 13)

	count := 0
	buffs, _ := FindBuffsByCharacterID(c.ID)
	index := 22
	for _, buff := range buffs {

		_, ok := BuffInfections[buff.ID]
		if !ok {
			continue
		}
		hpChi.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), index)
		index += 4
		hpChi.Insert(utils.IntToBytes(uint64(buff.SkillPlus), 1, false), index)
		index++
		hpChi.Insert([]byte{0x01}, index)
		index++
		count++
	}

	/*if c.AidMode {
		hpChi.Insert(utils.IntToBytes(11121, 4, true), index)
		hpChi.Insert([]byte{0x00, 0x00}, index)
		count++
	}*/
	hpChi[19] = byte(0x02)
	hpChi[21] = byte(count) // buff count
	index += 5
	//hpChi[index] = byte(15)
	hpChi.SetLength(int16(binary.Size(hpChi) - 6))
	return hpChi
}

func (c *Character) Handler() {

	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			log.Printf("handler error: %+v", string(dbg.Stack()))
			c.HandlerCB = nil
			c.Socket.Conn.Close()
		}
	}()

	st := c.Socket.Stats
	c.Epoch++

	if st.HP > 0 && c.Epoch%2 == 0 {
		hp, chi := st.HP, st.CHI
		if st.HP += st.HPRecoveryRate; st.HP > st.MaxHP {
			st.HP = st.MaxHP
		}

		if st.CHI += st.CHIRecoveryRate; st.CHI > st.MaxCHI {
			st.CHI = st.MaxCHI
		}

		if c.Meditating {
			if st.HP += st.MaxHP / 7; st.HP > st.MaxHP {
				st.HP = st.MaxHP
			}

			if st.CHI += st.MaxCHI / 7; st.CHI > st.MaxCHI {
				st.CHI = st.MaxCHI
			}
		}

		if hp != st.HP || chi != st.CHI {
			c.Socket.Write(c.GetHPandChi()) // hp-chi packet
		}

	} else if st.HP <= 0 && !c.Respawning { // dead
		c.Respawning = true
		st.HP = 0
		c.Socket.Write(c.GetHPandChi())
		c.Socket.Write(CHARACTER_DIED)
		go c.RespawnCounter(10)

		if c.DuelID > 0 { // lost pvp
			opponent, _ := FindCharacterByID(c.DuelID)

			c.DuelID = 0
			c.DuelStarted = false
			c.Socket.Write(PVP_FINISHED)

			opponent.DuelID = 0
			opponent.DuelStarted = false
			opponent.Socket.Write(PVP_FINISHED)
		} /*else if c.DuelID == 0 && c.KilledByCharacter == nil {
			randInt := utils.RandInt(1, 3)
			seed := utils.RandInt(1, 1000)
			if seed > 500 {
				c.LosePlayerExp(int(randInt))
			}
		}*/
	}
	if c.Poisoned == true && c.Epoch%1 == 0 && c.IsOnline {
		c.DealPoisonDamageToPlayer(c.PoisonSource, c.SufferedPoison)
	}
	if c.AidTime <= 0 && c.AidMode {

		c.AidTime = 0
		c.AidMode = false
		c.Socket.Write(c.AidStatus())

		tpData, _ := c.ChangeMap(c.Map, nil)
		c.Socket.Write(tpData)
	}

	if c.AidMode && !c.HasAidBuff() {
		c.AidTime--
		if c.AidTime%60 == 0 {
			stData, _ := c.GetStats()
			c.Socket.Write(stData)
		}
	}

	if !c.AidMode && c.Epoch%2 == 0 && c.AidTime < 7200 {
		c.AidTime++
		if c.AidTime%60 == 0 {
			stData, _ := c.GetStats()
			c.Socket.Write(stData)
		}
	}

	if c.PartyID != "" {
		c.UpdatePartyStatus()
	}

	c.HandleBuffs()
	c.HandleLimitedItems()
	//c.KilledByCharacter = nil
	go c.Update()
	go st.Update()
	time.AfterFunc(time.Second, func() {
		if c.HandlerCB != nil {
			c.HandlerCB()
		}
	})
}

func (c *Character) PetHandler() {

	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			log.Printf("%+v", string(dbg.Stack()))
		}
	}()

	{
		slots, err := c.InventorySlots()
		if err != nil {
			log.Println(err)
			goto OUT
		}

		petSlot := slots[0x0A]
		pet := petSlot.Pet
		if pet == nil || petSlot.ItemID == 0 || !pet.IsOnline {
			return
		}

		petInfo, ok := Pets[petSlot.ItemID]
		if !ok {
			return
		}

		if pet.HP <= 0 {
			resp := utils.Packet{}
			resp.Concat(c.GetPetStats())
			resp.Concat(DISMISS_PET)
			showpet, _ := c.ShowItems()
			resp.Concat(showpet)
			c.Socket.Write(resp)
			c.IsMounting = false
			pet.IsOnline = false
			return
		}
		if c.AidMode {
			if c.PlayerAidSettings.PetFood1ItemID != 0 && pet.IsOnline {
				slotID, item, err := c.FindItemInInventory(nil, c.PlayerAidSettings.PetFood1ItemID)
				if err != nil {
					//return nil, -1, err
				}
				percent := float32(c.PlayerAidSettings.PetFood1Percent) / float32(100)
				minPetHP := float32(pet.MaxHP) * percent
				if float32(pet.HP) <= minPetHP {
					petresp, err := c.UseConsumable(item, slotID)
					if err == nil {
						c.Socket.Write(petresp)
					} else {
						fmt.Println(fmt.Sprintf("PetError: %s", err.Error()))
					}
				}

			}
			if c.PlayerAidSettings.PetChiPercent != 0 && pet.IsOnline {
				slotID, item, err := c.FindItemInInventory(nil, c.PlayerAidSettings.PetChiItemID)
				if err != nil {
					//return nil, -1, err
				}
				percent := float32(c.PlayerAidSettings.PetChiPercent) / float32(100)
				minPetChi := float32(pet.MaxCHI) * percent
				if float32(pet.CHI) <= minPetChi {
					petresp, err := c.UseConsumable(item, slotID)
					if err == nil {
						c.Socket.Write(petresp)
					} else {
						fmt.Println(fmt.Sprintf("PetError: %s", err.Error()))
					}
				}

			}
		}

		if petInfo.Combat && pet.Target == 0 && pet.Loyalty >= 10 {
			pet.Target, err = pet.FindTargetMobID(c) // 75% chance to trigger
			if err != nil {
				log.Println("AIHandler error:", err)
			}
		}

		if pet.Target > 0 {
			pet.IsMoving = false
		}

		if c.Epoch%60 == 0 {
			if pet.Fullness > 1 {
				pet.Fullness--
			}
			if pet.Fullness < 25 && pet.Loyalty > 1 {
				pet.Loyalty--
			} else if pet.Fullness >= 25 && pet.Loyalty < 100 {
				pet.Loyalty++
			}
		}
		cPetLevel := int(pet.Level)
		if c.Epoch%20 == 0 {
			if pet.HP < pet.MaxHP {
				pet.HP = int(math.Min(float64(pet.HP+cPetLevel*3), float64(pet.MaxHP)))
			}
			if pet.CHI < pet.MaxCHI {
				pet.CHI = int(math.Min(float64(pet.CHI+cPetLevel*2), float64(pet.CHI)))
			}
			pet.RefreshStats = true
		}

		if pet.RefreshStats {
			pet.RefreshStats = false
			c.Socket.Write(c.GetPetStats())
		}
		if !petInfo.Combat {
			pet.Target = 0
			goto OUT
		}
		if pet.IsMoving || pet.Casting {
			goto OUT
		}

		if pet.Loyalty < 10 {
			pet.Target = 0
		}

	BEGIN:
		ownerPos := ConvertPointToLocation(c.Coordinate)
		ownerdistance := utils.CalculateDistance(ownerPos, &pet.Coordinate)
		if pet.PetCombatMode == 2 && ownerdistance <= 10 && c.Selection > 0 {
			pet.Target = c.Selection
		} else if ownerdistance > 10 {
			pet.Target = 0
		}

		if petInfo.Combat && pet.Target == 0 && pet.Loyalty >= 10 {
			pet.Target, err = pet.FindTargetMobID(c) // 75% chance to trigger
			if err != nil {
				log.Println("AIHandler error:", err)
			}
		}

		if pet.Target == 0 { // Idle mode

			ownerPos := ConvertPointToLocation(c.Coordinate)
			distance := utils.CalculateDistance(ownerPos, &pet.Coordinate)

			if distance > 10 { // Pet is so far from his owner
				pet.IsMoving = true
				targetX := utils.RandFloat(ownerPos.X-5, ownerPos.X+5)
				targetY := utils.RandFloat(ownerPos.Y-5, ownerPos.Y+5)

				target := utils.Location{X: targetX, Y: targetY}
				pet.TargetLocation = target
				speed := float64(10.0)

				token := pet.MovementToken
				for token == pet.MovementToken {
					pet.MovementToken = utils.RandInt(1, math.MaxInt64)
				}

				go pet.MovementHandler(pet.MovementToken, &pet.Coordinate, &target, speed)
			}

		} else { // Target mode
			target := GetFromRegister(c.Socket.User.ConnectedServer, c.Map, uint16(pet.Target))
			if _, ok := target.(*AI); ok { // attacked to ai
				mob, ok := GetFromRegister(c.Socket.User.ConnectedServer, c.Map, uint16(pet.Target)).(*AI)
				if !ok || mob == nil {
					pet.Target = 0
					goto OUT

				} else if mob.HP <= 0 {
					pet.Target = 0
					time.Sleep(time.Second)
					goto BEGIN
				}

				aiCoordinate := ConvertPointToLocation(mob.Coordinate)
				distance := utils.CalculateDistance(&pet.Coordinate, aiCoordinate)

				if distance <= 3 && pet.LastHit%2 == 0 { // attack
					seed := utils.RandInt(1, 1000)
					r := utils.Packet{}
					skillIds := petInfo.GetSkills()
					skillsCount := len(skillIds) - 1
					randomSkill := utils.RandInt(0, int64(skillsCount))
					skillID := skillIds[randomSkill]
					skill, ok := SkillInfos[skillID]
					if seed < 500 && ok && pet.CHI >= skill.BaseChi && skillID != 0 {
						r.Concat(pet.CastSkill(c, skillID))
					} else {
						r.Concat(pet.Attack(c))
					}

					p := nats.CastPacket{CastNear: true, PetID: pet.PseudoID, Data: r, Type: nats.MOB_ATTACK}
					p.Cast()
					pet.LastHit++

				} else if distance > 3 && distance <= 50 { // chase
					pet.IsMoving = true
					target := GeneratePoint(aiCoordinate)
					pet.TargetLocation = target
					speed := float64(10.0)

					token := pet.MovementToken
					for token == pet.MovementToken {
						pet.MovementToken = utils.RandInt(1, math.MaxInt64)
					}

					go pet.MovementHandler(pet.MovementToken, &pet.Coordinate, &target, speed)
					pet.LastHit = 0

				} else {
					pet.LastHit++
				}
			} else { // FIX => attacked to player
				mob := FindCharacterByPseudoID(c.Socket.User.ConnectedServer, uint16(pet.Target))
				if mob.Socket.Stats.HP <= 0 || !c.CanAttack(mob) {
					pet.Target = 0
					time.Sleep(time.Second)
					goto BEGIN
				}
				aiCoordinate := ConvertPointToLocation(mob.Coordinate)
				distance := utils.CalculateDistance(&pet.Coordinate, aiCoordinate)

				if distance <= 3 && pet.LastHit%2 == 0 { // attack
					seed := utils.RandInt(1, 1000)
					r := utils.Packet{}
					skillIds := petInfo.GetSkills()
					skillsCount := len(skillIds) - 1
					randomSkill := utils.RandInt(0, int64(skillsCount))
					skillID := skillIds[randomSkill]
					skill, ok := SkillInfos[skillID]
					if seed < 500 && ok && pet.CHI >= skill.BaseChi && skillID != 0 {
						r.Concat(pet.CastSkill(c, skillID))
					} else {
						r.Concat(pet.PlayerAttack(c))
					}

					p := nats.CastPacket{CastNear: true, PetID: pet.PseudoID, Data: r, Type: nats.MOB_ATTACK}
					p.Cast()
					pet.LastHit++

				} else if distance > 3 && distance <= 50 { // chase
					pet.IsMoving = true
					target := GeneratePoint(aiCoordinate)
					pet.TargetLocation = target
					speed := float64(10.0)

					token := pet.MovementToken
					for token == pet.MovementToken {
						pet.MovementToken = utils.RandInt(1, math.MaxInt64)
					}

					go pet.MovementHandler(pet.MovementToken, &pet.Coordinate, &target, speed)
					pet.LastHit = 0

				} else {
					pet.LastHit++
				}
			}
			petSlot.Update()
		}
	}

OUT:
	time.AfterFunc(time.Second, func() {
		if c.PetHandlerCB != nil {
			c.PetHandlerCB()
		}
	})
}

func (c *Character) HandleBuffs() {
	allBuffs, err := c.FindAllRelevantBuffs()
	if err != nil {
		return
	}
	var activeBuffs []*Buff
	stat := c.Socket.Stats
	for _, buff := range allBuffs {
		if ((!buff.IsServerEpoch && buff.StartedAt+buff.Duration <= c.Epoch) || (buff.IsServerEpoch && buff.StartedAt+buff.Duration <= GetServerEpoch())) && buff.CanExpire { // buff expired
			stat.MinATK -= buff.ATK
			stat.MaxATK -= buff.ATK
			stat.ATKRate -= buff.ATKRate
			stat.DEF -= buff.DEF
			stat.DefRate -= buff.DEFRate

			stat.MinArtsATK -= buff.ArtsATK
			stat.MaxArtsATK -= buff.ArtsATK
			stat.ArtsATKRate -= buff.ArtsATKRate
			stat.ArtsDEF -= buff.ArtsDEF
			stat.ArtsDEFRate -= buff.ArtsDEFRate

			stat.MaxCHI -= buff.MaxCHI
			stat.MaxHP -= buff.MaxHP
			stat.HPRecoveryRate -= buff.HPRecoveryRate
			stat.CHIRecoveryRate -= buff.CHIRecoveryRate
			stat.Dodge -= buff.Dodge
			stat.Accuracy -= buff.Accuracy

			stat.ParalysisDEF -= buff.ParalysisDEF
			stat.PoisonDEF -= buff.PoisonDEF
			stat.ConfusionDEF -= buff.ConfusionDEF

			stat.DEXBuff -= buff.DEX
			stat.INTBuff -= buff.INT
			stat.STRBuff -= buff.STR
			stat.FireBuff -= buff.Fire
			stat.WaterBuff -= buff.Water
			stat.WindBuff -= buff.Wind

			stat.ExpRate -= float64(buff.ExpRate) / 1000
			stat.GoldRate -= float64(buff.GoldRate) / 1000
			stat.DropRate -= float64(buff.DropRate) / 1000
			c.RunningSpeed -= buff.RunningSpeed
			data, _ := c.GetStats()

			r := BUFF_EXPIRED
			r.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), 6) // buff infection id
			r.Concat(data)

			c.Socket.Write(r)
			buff.Delete()

			p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: c.GetHPandChi()}
			p.Cast()

			if buff.ID == 241 || buff.ID == 244 || buff.ID == 139 || buff.ID == 50 { // invisibility
				c.Invisible = false
				if c.DuelID > 0 {
					opponent, _ := FindCharacterByID(c.DuelID)
					sock := opponent.Socket
					if sock != nil {
						time.AfterFunc(time.Second*1, func() {
							sock.Write(opponent.OnDuelStarted())
						})
					}
				}

			} else if buff.ID == 242 || buff.ID == 245 || buff.ID == 105 || buff.ID == 53 || buff.ID == 59 || buff.ID == 142 || buff.ID == 164 || buff.ID == 214 || buff.ID == 217 { // detection arts
				c.DetectionMode = false
			}

			if buff.ID == 140 && buff.CanExpire {
			}
			if buff.ID == 141 && buff.CanExpire {
			}
			if buff.ID == 14009 && buff.CanExpire {
			}
			//if buff.ID == 257 {
			//c.Poisoned = false
			//}
			if buff.ID == 258 {
				c.Confusioned = false
			}
			if buff.ID == 259 {
				c.Paralysised = false
			}
			if len(allBuffs) == 1 {
				allBuffs = []*Buff{}
			} else {
				allBuffs = allBuffs[1:]
			}
		} else {
			activeBuffs = append(activeBuffs, buff)
		}
	}
	for _, buff := range activeBuffs {
		var remainingTime int64
		id := buff.ID
		//if buff.ID == 257 {
		//c.Poisoned = true
		//}
		if buff.ID == 258 {
			c.Confusioned = true
		}
		if buff.ID == 259 {
			c.Paralysised = true
		}
		infection, ok := BuffInfections[id]
		if !ok {
			continue
		}
		if buff.IsServerEpoch {
			remainingTime = buff.StartedAt + buff.Duration - GetServerEpoch()
		} else {
			remainingTime = buff.StartedAt + buff.Duration - c.Epoch
		}
		data := BUFF_INFECTION
		data.Overwrite(utils.IntToBytes(uint64(buff.SkillPlus), 1, false), 6)
		data.Insert(utils.IntToBytes(uint64(infection.ID), 4, true), 6)   // infection id
		data.Insert(utils.IntToBytes(uint64(remainingTime), 4, true), 11) // buff remaining time

		c.Socket.Write(data)
	}
}

func (c *Character) HandleLimitedItems() {

	invSlots, err := c.InventorySlots()
	if err != nil {
		return
	}

	slotIDs := []int16{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0133, 0x0134, 0x0135, 0x0136, 0x0137, 0x0138, 0x0139, 0x013A, 0x013B}

	for _, slotID := range slotIDs {
		slot := invSlots[slotID]
		item := Items[slot.ItemID]
		if item != nil && (item.TimerType == 1 || item.TimerType == 3) { // time limited item
			if c.Epoch%60 == 0 {
				data := c.DecrementItem(slotID, 1)
				c.Socket.Write(*data)
			}
			if slot.Quantity == 0 {
				data := ITEM_EXPIRED
				data.Insert(utils.IntToBytes(uint64(item.ID), 4, true), 6)

				removeData, _ := c.RemoveItem(slotID)
				data.Concat(removeData)

				statData, _ := c.GetStats()
				data.Concat(statData)
				c.Socket.Write(data)
			}
		}
	}

	starts, ends := []int16{0x0B, 0x0155}, []int16{0x043, 0x018D}
	for j := 0; j < 2; j++ {
		start, end := starts[j], ends[j]
		for slotID := start; slotID <= end; slotID++ {
			slot := invSlots[slotID]
			item := Items[slot.ItemID]
			if slot.Activated {
				if c.Epoch%60 == 0 {
					data := c.DecrementItem(slotID, 1)
					c.Socket.Write(*data)
				}
				if slot.Quantity == 0 { // item expired
					data := ITEM_EXPIRED
					data.Insert(utils.IntToBytes(uint64(item.ID), 4, true), 6)

					c.RemoveItem(slotID)
					data.Concat(slot.GetData(slotID))

					statData, _ := c.GetStats()
					data.Concat(statData)
					c.Socket.Write(data)

					if slot.ItemID == 100080000 { // eyeball of divine
						c.DetectionMode = false
					}

					if item.GetType() == FORM_TYPE {
						c.Morphed = false
						c.MorphedNPCID = 0
						c.Socket.Write(FORM_DEACTIVATED)
						characters, err := c.GetNearbyCharacters()
						if err != nil {
							log.Println(err)
							//return
						}
						//test := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x21, 0x02, 0x00, 0x55, 0xAA}
						for _, chars := range characters {
							delete(chars.OnSight.Players, c.ID)
						}
					}

				} else { // item not expired
					if slot.ItemID == 100080000 && !c.DetectionMode { // eyeball of divine
						c.DetectionMode = true
					} else if item.GetType() == FORM_TYPE && !c.Morphed {
						c.Morphed = true
						c.MorphedNPCID = item.NPCID
						r := FORM_ACTIVATED
						r.Insert(utils.IntToBytes(uint64(item.NPCID), 4, true), 5) // form npc id
						c.Socket.Write(r)
						characters, err := c.GetNearbyCharacters()
						if err != nil {
							log.Println(err)
							return
						}
						//test := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x21, 0x02, 0x00, 0x55, 0xAA}
						for _, chars := range characters {
							delete(chars.OnSight.Players, c.ID)
						}
					}
				}
			}
		}
	}
}

func (c *Character) makeCharacterMorphed(npcID uint64, activateState bool) []byte {

	resp := FORM_ACTIVATED
	resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 5) // form npc id
	c.Socket.Write(resp)

	return resp
}
func (c *Character) DisableCharacterMorphed() []byte { //byte maybe useable later
	invSlots, err := c.InventorySlots()
	if err != nil {
		return nil
	}
	resp := utils.Packet{}
	starts, ends := []int16{0x0B, 0x0155}, []int16{0x043, 0x018D}
	for j := 0; j < 2; j++ {
		start, end := starts[j], ends[j]
		for slotID := start; slotID <= end; slotID++ {
			slot := invSlots[slotID]
			item := Items[slot.ItemID]
			if slot.Activated {
				if item.GetType() == FORM_TYPE {
					slot.Activated = false
					c.Morphed = false
					c.MorphedNPCID = 0
					c.Socket.Write(FORM_DEACTIVATED)
					characters, err := c.GetNearbyCharacters()
					if err != nil {
						log.Println(err)
						//return
					}
					//test := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x21, 0x02, 0x00, 0x55, 0xAA}
					for _, chars := range characters {
						delete(chars.OnSight.Players, c.ID)
					}
					go item.Update()

					statData, err := c.GetStats()
					if err != nil {
						return nil
					}
					resp.Concat(*c.DecrementItem(slotID, 0))
					resp.Concat(statData)
				}
			}
		}
	}
	c.Socket.Write(resp)
	return nil
}
func (c *Character) RespawnCounter(seconds byte) {

	resp := RESPAWN_COUNTER
	resp[7] = seconds
	c.Socket.Write(resp)

	if seconds > 0 {
		time.AfterFunc(time.Second, func() {
			c.RespawnCounter(seconds - 1)
		})
	}
}

func (c *Character) Teleport(coordinate *utils.Location) []byte {

	c.SetCoordinate(coordinate)

	resp := TELEPORT_PLAYER
	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), 5) // coordinate-x
	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 9) // coordinate-x

	return resp
}

func (c *Character) ActivityStatus(remainingTime int) {

	var msg string
	if c.IsActive || remainingTime == 0 {
		msg = "Your character has been activated."
		c.IsActive = true

		data, err := c.SpawnCharacter()
		if err != nil {
			return
		}

		p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: data, Type: nats.PLAYER_SPAWN}
		if err = p.Cast(); err != nil {
			return
		}

	} else {
		msg = fmt.Sprintf("Your character will be activated %d seconds later.", remainingTime)

		if c.IsOnline {
			time.AfterFunc(time.Second, func() {
				if !c.IsActive {
					c.ActivityStatus(remainingTime - 1)
				}
			})
		}
	}

	info := messaging.InfoMessage(msg)
	if c.Socket != nil {
		c.Socket.Write(info)
	}
}

func (c *Character) AuthStatus(remainingTime int) {

	var msg string
	if c.GMAuthenticated == GMPassword {
		msg = "Your GM Account is authenticated."
		info := messaging.InfoMessage(msg)
		if c.Socket != nil {
			c.Socket.Write(info)
		}
	} else if remainingTime == 0 {
		c.Socket.Conn.Close()
		text := "Name: " + c.Name + "(" + c.Socket.User.ID + ") hasnt anwsered in time"
		utils.NewLog("logs/gm_auth.txt", text)
	} else {
		time.AfterFunc(time.Second, func() {
			c.AuthStatus(remainingTime - 1)
		})
	}
}

func contains(v int64, a []int64) bool {
	for _, i := range a {
		if i == v {
			return true
		}
	}
	return false
}

func unorderedEqual(first, second []int64, count int) bool {
	exists := make(map[int64]bool)
	match := 0
	for _, value := range first {
		exists[value] = true
	}
	for _, value := range second {
		if match >= count {
			return true
		}
		if !exists[value] {
			return false
		}
		match++
	}
	return true
}

func BonusActive(first, second []int64) bool {
	exists := make(map[int64]bool)
	for _, value := range first {
		exists[value] = true
	}
	for _, value := range second {
		if !exists[value] {
			return false
		}
	}
	return true
}

func (c *Character) ItemSetEffects(indexes []int16) []int64 {
	slots, _ := c.InventorySlots()
	playerItems := []int64{}
	for _, i := range indexes {
		if (i == 3 && c.WeaponSlot == 4) || (i == 4 && c.WeaponSlot == 3) {
			continue
		}
		item := slots[i]
		playerItems = append(playerItems, item.ItemID)
	}
	setEffect := []int64{}
	for _, i := range playerItems {
		for _, sets := range ItemSets {
			if contains(i, sets.GetSetItems()) {
				if unorderedEqual(playerItems, sets.GetSetItems(), sets.SetItemCount) {
					buffEffects := sets.GetSetBonus()
					for _, effect := range buffEffects {
						if effect == 0 || effect == 616 || effect == 617 {
							continue
						}
						if !contains(effect, setEffect) {
							setEffect = append(setEffect, effect)
						}
					}
				}
			}
		}
	}
	return setEffect
}
func (c *Character) applySetEffect(bonuses []int64, st *Stat) {
	additionalDropMultiplier, additionalExpMultiplier, additionalRunningSpeed := float64(0), float64(0), float64(0)
	for _, id := range bonuses {
		item := Items[id]
		if item == nil {
			continue
		}

		st.STRBuff += item.STR
		st.DEXBuff += item.DEX
		st.INTBuff += item.INT
		st.WindBuff += item.Wind
		st.WaterBuff += item.Water
		st.FireBuff += item.Fire

		st.DEF += item.Def + ((item.BaseDef1 + item.BaseDef2 + item.BaseDef3) / 3)
		st.DefRate += item.DefRate

		st.ArtsDEF += item.ArtsDef
		st.ArtsDEFRate += item.ArtsDefRate

		st.MaxHP += item.MaxHp
		st.MaxCHI += item.MaxChi

		st.Accuracy += item.Accuracy
		st.Dodge += item.Dodge

		st.MinATK += item.BaseMinAtk + item.MinAtk
		st.MaxATK += item.BaseMaxAtk + item.MaxAtk
		st.ATKRate += item.AtkRate

		st.MinArtsATK += item.MinArtsAtk
		st.MaxArtsATK += item.MaxArtsAtk
		st.ArtsATKRate += item.ArtsAtkRate
		additionalExpMultiplier += item.ExpRate / 100
		additionalDropMultiplier += item.DropRate / 100
		additionalRunningSpeed += item.RunningSpeed

		st.HPRecoveryRate += item.HPRecoveryRate
		st.CHIRecoveryRate += item.CHIRecoveryRate
	}
	c.AdditionalExpMultiplier += additionalExpMultiplier
	c.AdditionalDropMultiplier += additionalDropMultiplier
	c.AdditionalRunningSpeed += additionalRunningSpeed
}
func (c *Character) applyJudgementEffect(bonusID int64, st *Stat) {
	item := ItemJudgements[int(bonusID)]
	st.STRBuff += item.StrPlus
	st.DEXBuff += item.DexPlus
	st.INTBuff += item.IntPlus
	st.WindBuff += item.WindPlus
	st.WaterBuff += item.WaterPlus
	st.FireBuff += item.FirePlus

	st.DEF += item.Def
	//st.DEF += item.Def + ((item.BaseDef1 + item.BaseDef2 + item.BaseDef3) / 3)
	//st.DefRate += item.Def

	st.ArtsDEF += item.ExtraArtsDef
	//st.ArtsDEFRate += item.ArtsDefRate

	st.MaxHP += item.MaxHP
	st.MaxCHI += item.MaxChi

	st.Accuracy += item.AccuracyPlus
	st.Dodge += item.ExtraDodge
	st.AttackSpeed += item.ExtraAttackSpeed

	st.MinATK += item.AttackPlus
	st.MaxATK += item.AttackPlus

	//st.MinArtsATK += item.MinArtsAtk
	//st.MaxArtsATK += item.MaxArtsAtk
	//st.ArtsATKRate += item.ArtsAtkRate
}
func (c *Character) ItemEffects(st *Stat, start, end int16) error {

	slots, err := c.InventorySlots()
	if err != nil {
		return err
	}

	indexes := []int16{}

	for i := start; i <= end; i++ {
		slot := slots[i]
		if start == 0x0B || start == 0x155 {
			if slot != nil && slot.Activated && slot.InUse {
				indexes = append(indexes, i)
			}
		} else {
			indexes = append(indexes, i)
		}
	}

	additionalDropMultiplier, additionalExpMultiplier, additionalRunningSpeed := float64(0), float64(0), float64(0)
	maxPoison, maxConfusion, maxPara := 0, 0, 0
	setEffects := c.ItemSetEffects(indexes)
	c.applySetEffect(setEffects, st)
	for _, i := range indexes {
		if (i == 3 && c.WeaponSlot == 4) || (i == 4 && c.WeaponSlot == 3) {
			continue
		}

		item := slots[i]

		if item.ItemID != 0 {

			info := Items[item.ItemID]
			slotId := i
			if slotId == 4 {
				slotId = 3
			}

			if (info == nil || slotId != int16(info.Slot) || c.Level < info.MinLevel || (info.MaxLevel > 0 && c.Level > info.MaxLevel)) &&
				!(start == 0x0B || start == 0x155) {
				continue
			}

			ids := []int64{item.ItemID}
			if item.ItemType != 0 {
				//fmt.Println("BONUSID: ", item.JudgementStat)
				c.applyJudgementEffect(item.JudgementStat, st)
			}
			for _, u := range item.GetUpgrades() {
				if u == 0 {
					break
				}
				ids = append(ids, int64(u))
			}

			for _, s := range item.GetSockets() {
				if s == 0 {
					break
				}
				ids = append(ids, int64(s))
			}
			for _, id := range ids {
				item := Items[id]
				if item == nil {
					continue
				}

				st.STRBuff += item.STR
				st.DEXBuff += item.DEX
				st.INTBuff += item.INT
				st.WindBuff += item.Wind
				st.WaterBuff += item.Water
				st.FireBuff += item.Fire

				st.DEF += item.Def + ((item.BaseDef1 + item.BaseDef2 + item.BaseDef3) / 3)
				st.DefRate += item.DefRate

				st.ArtsDEF += item.ArtsDef
				st.ArtsDEFRate += item.ArtsDefRate

				st.MaxHP += item.MaxHp
				st.MaxCHI += item.MaxChi

				st.Accuracy += item.Accuracy
				st.Dodge += item.Dodge

				st.MinATK += item.BaseMinAtk + item.MinAtk
				st.MaxATK += item.BaseMaxAtk + item.MaxAtk
				st.ATKRate += item.AtkRate

				st.PoisonATK += item.PoisonATK
				st.PoisonDEF += item.PoisonDEF
				st.ParalysisATK += item.ParaATK
				st.ParalysisDEF += item.ParaDEF
				st.ConfusionATK += item.ConfusionATK
				st.ConfusionDEF += item.ConfusionDEF

				st.MinArtsATK += item.MinArtsAtk
				st.MaxArtsATK += item.MaxArtsAtk
				st.ArtsATKRate += item.ArtsAtkRate
				additionalExpMultiplier += item.ExpRate / 100
				additionalDropMultiplier += item.DropRate / 100
				additionalRunningSpeed += item.RunningSpeed
				st.HPRecoveryRate += item.HPRecoveryRate
				st.CHIRecoveryRate += item.CHIRecoveryRate

				if item.PoisonTime > maxPoison {
					maxPoison = item.PoisonTime
					st.PoisonTime = item.PoisonTime / 1000
				}
				if item.ParaTime > maxPara {
					maxPara = item.ParaTime
					st.Paratime = item.ParaTime / 1000
				}
				if item.ConfusionTime > maxConfusion {
					maxConfusion = item.ConfusionTime
					st.ConfusionTime = item.ConfusionTime / 1000
				}
			}

		}
	}

	c.AdditionalExpMultiplier += additionalExpMultiplier
	c.AdditionalDropMultiplier += additionalDropMultiplier
	c.AdditionalRunningSpeed += additionalRunningSpeed
	return nil
}

func (c *Character) GetExpAndSkillPts() []byte {

	resp := EXP_SKILL_PT_CHANGED
	resp.Insert(utils.IntToBytes(uint64(c.Exp), 8, true), 5)                        // character exp
	resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points
	return resp
}

func (c *Character) GetPTS() []byte {

	resp := PTS_CHANGED
	resp.Insert(utils.IntToBytes(uint64(c.PTS), 4, true), 6) // character pts
	return resp
}

func (c *Character) LootGold(amount uint64) []byte {

	c.AddingGold.Lock()
	defer c.AddingGold.Unlock()

	c.Gold += amount
	resp := GOLD_LOOTED
	resp.Insert(utils.IntToBytes(uint64(c.Gold), 8, true), 9) // character gold

	return resp
}

func (c *Character) AddExp(amount int64) ([]byte, bool) {
	c.AddingExp.Lock()
	defer c.AddingExp.Unlock()
	var exp int64

	// 21% EXP for PVP Servers. Custom for PSI
	serverBonus := float64(0)
	user, _ := FindUserByID(c.UserID)
	if user.ConnectedServer == 6 || user.ConnectedServer == 7 {
		serverBonus = 0.21
	} else {
		serverBonus = 0
	}

	expMultipler := c.ExpMultiplier + c.AdditionalExpMultiplier + c.Socket.Stats.ExpRate + serverBonus
	exp = c.Exp + int64(float64(amount)*(expMultipler*EXP_RATE))

	spIndex := utils.SearchUInt64(SkillPoints, uint64(c.Exp))
	canLevelUp := true
	if exp > EXPs[100].Exp && c.Level <= 100 && c.Type <= 59 {
		exp = EXPs[100].Exp
		//canLevelUp = false
		//return nil, false
	}
	if exp > EXPs[200].Exp && c.Level <= 200 && c.Type <= 69 {
		exp = EXPs[200].Exp
		//canLevelUp = false
		//return nil, false
	}
	c.Exp = exp
	spIndex2 := utils.SearchUInt64(SkillPoints, uint64(c.Exp))

	//resp := c.GetExpAndSkillPts()

	st := c.Socket.Stats
	if st == nil {
		return nil, false
	}

	levelUp := false
	level := int16(c.Level)
	targetExp := EXPs[level].Exp
	skPts, sp := 0, 0
	np := 0                                              //nature pts
	for exp >= targetExp && level <= 299 && canLevelUp { // Levelling up && level < 100
		if c.Type <= 59 && level >= 100 {
			level = 100
			exp = EXPs[100].Exp
			canLevelUp = false
		} else if c.Type <= 69 && level >= 200 {
			level = 200
			exp = EXPs[200].Exp
			canLevelUp = false
		} else {
			level++
			st.HP = st.MaxHP
			if EXPs[level] != nil {
				sp += EXPs[level].StatPoints
				np += EXPs[level].NaturePoints

			}
			targetExp = EXPs[level].Exp
			levelUp = true
		}

	}
	c.Level = int(level)
	resp := EXP_SKILL_PT_CHANGED

	skPts = spIndex2 - spIndex
	c.Socket.Skills.SkillPoints += skPts
	if levelUp {
		//LOAD QUESTS
		//c.AddPlayerQuests()

		st.StatPoints += sp
		st.NaturePoints += np
		resp.Insert(utils.IntToBytes(uint64(exp), 8, true), 5)                          // character exp
		resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points
		if c.GuildID > 0 {
			guild, err := FindGuildByID(c.GuildID)
			if err == nil && guild != nil {
				guild.InformMembers(c)
			}
		}

		resp.Concat(messaging.SystemMessage(messaging.LEVEL_UP))
		resp.Concat(messaging.SystemMessage(messaging.LEVEL_UP_SP))
		resp.Concat(messaging.InfoMessage(c.GetLevelText()))

		spawnData, err := c.SpawnCharacter()
		if err == nil {
			c.Socket.Write(spawnData)
			c.Update()
			//resp.Concat(spawnData)
		}
	} else {
		resp.Insert(utils.IntToBytes(uint64(exp), 8, true), 5)                          // character exp
		resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points
	}
	go c.Socket.Skills.Update()
	return resp, levelUp
}

func (c *Character) AddPlayerExp(amount int64) ([]byte, bool) {

	c.AddingExp.Lock()
	defer c.AddingExp.Unlock()

	exp := c.Exp + int64(float64(amount))
	spIndex := utils.SearchUInt64(SkillPoints, uint64(c.Exp))
	canLevelUp := true
	if exp > 233332051410 && c.Level <= 100 {
		exp = 233332051410
	}
	if exp > 544951059310 && c.Level <= 200 {
		exp = 544951059310
	}
	c.Exp = exp
	spIndex2 := utils.SearchUInt64(SkillPoints, uint64(c.Exp))

	//resp := c.GetExpAndSkillPts()

	st := c.Socket.Stats
	if st == nil {
		return nil, false
	}

	levelUp := false
	level := int16(c.Level)
	targetExp := EXPs[level].Exp
	skPts, sp := 0, 0
	np := 0                                             //nature pts
	for exp >= targetExp && level < 299 && canLevelUp { // Levelling up && level < 100
		//skPts += EXPs[level].SkillPoints
		if c.Type <= 59 && level >= 100 {
			level = 100
			canLevelUp = false
		} else if c.Type <= 69 && level >= 200 {
			level = 200
			canLevelUp = false
		} else {
			level++
			st.HP = st.MaxHP
			sp += int(level/10) + 4

			targetExp = EXPs[level].Exp
			levelUp = true
		}
		if level >= 101 && level < 299 { //divine nature stats
			np += 21
			skPts = spIndex2 - spIndex
		}
	}
	c.Level = int(level)
	resp := EXP_SKILL_PT_CHANGED
	if level < 101 { //divine nature stats
		skPts = spIndex2 - spIndex
	}
	skPts = spIndex2 - spIndex
	c.Socket.Skills.SkillPoints += skPts
	if levelUp {
		//LOAD QUESTS
		c.AddPlayerQuests()

		st.StatPoints += sp
		st.NaturePoints += np
		resp.Insert(utils.IntToBytes(uint64(exp), 8, true), 5)                          // character exp
		resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points
		if c.GuildID > 0 {
			guild, err := FindGuildByID(c.GuildID)
			if err == nil && guild != nil {
				guild.InformMembers(c)
			}
		}

		resp.Concat(messaging.SystemMessage(messaging.LEVEL_UP))
		resp.Concat(messaging.SystemMessage(messaging.LEVEL_UP_SP))
		resp.Concat(messaging.InfoMessage(c.GetLevelText()))

		spawnData, err := c.SpawnCharacter()
		if err == nil {
			c.Socket.Write(spawnData)
			c.Update()
			//resp.Concat(spawnData)
		}
	} else {
		resp.Insert(utils.IntToBytes(uint64(exp), 8, true), 5)                          // character exp
		resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points
	}
	go c.Socket.Skills.Update()
	return resp, levelUp
}

func (c *Character) LosePlayerExp(percent int) (int64, error) {
	level := int16(c.Level)
	expminus := int64(0)
	if level >= 10 {
		oldExp := EXPs[level-1].Exp
		resp := EXP_SKILL_PT_CHANGED
		if oldExp <= c.Exp {
			per := float64(percent) / 100
			expLose := float64(c.Exp) * float64(1-per)
			if int64(expLose) >= oldExp {
				exp := c.Exp - int64(expLose)
				expminus = int64(float64(exp) * float64(1-0.30))
				c.Exp = int64(expLose)
			} else {
				exp := c.Exp - oldExp
				expminus = int64(float64(exp) * float64(1-0.30))
				c.Exp = oldExp
			}
		}
		resp.Insert(utils.IntToBytes(uint64(c.Exp), 8, true), 5)                        // character exp
		resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points
		go c.Socket.Skills.Update()
		c.Socket.Write(resp)
	}
	return expminus, nil
}

func (c *Character) AddPlayerQuests() error {
	qList, err := GetQuestForPlayer(c)
	if err != nil {
		fmt.Println("Error with load: ", err)
		return err
	}
	for _, allquest := range qList {
		theQuest, _ := FindPlayerQuestByID(allquest.ID, c.ID)
		if theQuest == nil {
			if allquest.PrevMissionID != 0 {
				prevQuest, _ := FindPlayerQuestByID(allquest.PrevMissionID, c.ID)
				if prevQuest != nil {
					if prevQuest.QuestState == 2 {
						addPlayerQuest := &Quest{ID: allquest.ID, CharacterID: c.ID, QuestState: 3}
						err := addPlayerQuest.Create()
						if err != nil {
							fmt.Println("Error with load: ", err)
						}
						//return nil
					}
				}
			} else {
				addPlayerQuest := &Quest{ID: allquest.ID, CharacterID: c.ID, QuestState: 3}
				err := addPlayerQuest.Create()
				if err != nil {
					fmt.Println("Error with load: ", err)
				}
				//return nil
			}
		}
	}
	return nil
}
func (c *Character) CombineItems(where, to int16) (int64, int16, error) {

	c.AntiDupeMutex.Lock()
	defer c.AntiDupeMutex.Unlock()
	invSlots, err := c.InventorySlots()
	if err != nil {
		return 0, 0, err
	}

	whereItem := invSlots[where]
	toItem := invSlots[to]

	if toItem.ItemID == whereItem.ItemID {
		toItem.Quantity += whereItem.Quantity

		go toItem.Update()
		whereItem.Delete()
		*whereItem = *NewSlot()

	} else {
		return 0, 0, nil
	}

	return toItem.ItemID, int16(toItem.Quantity), nil
}

func (c *Character) BankItems() []byte {

	bankSlots, err := c.InventorySlots()
	if err != nil {
		return nil
	}

	bankSlots = bankSlots[0x43:0x133]
	resp := BANK_ITEMS

	index, length := 8, int16(4)
	for i, slot := range bankSlots {
		if slot.ItemID == 0 {
			continue
		}

		resp.Insert(utils.IntToBytes(uint64(slot.ItemID), 4, true), index) // item id
		index += 4

		resp.Insert([]byte{0x00, 0xA1, 0x01, 0x00}, index)
		index += 4

		test := 67 + i
		resp.Insert(utils.IntToBytes(uint64(test), 2, true), index) // slot id
		index += 2

		resp.Insert([]byte{0x00, 0x00, 0x00, 0x00}, index)
		index += 4
		length += 14
	}

	resp.SetLength(length)
	return resp
}

func (c *Character) GetGold() []byte {

	user, err := FindUserByID(c.UserID)
	if err != nil || user == nil {
		return nil
	}

	resp := GET_GOLD
	resp.Insert(utils.IntToBytes(uint64(c.Gold), 8, true), 6)         // gold
	resp.Insert(utils.IntToBytes(uint64(user.BankGold), 8, true), 14) // bank gold

	return resp
}
func (c *Character) GetMapQuestMobs() {
	questItems, _ := FindQuestByMapID(int(c.Map))
	questItems = funk.Filter(questItems, func(item *QuestList) bool {
		return item.DropFromMobs
	}).([]*QuestList)
	if len(questItems) > 0 {
		questbymap, _ := FindQuestsAcceptedByID(c.ID)
		for _, quest := range questItems {
			questID := int(quest.ID)
			if funk.Contains(questbymap, questID) {
				mQuest := QuestsList[int(quest.ID)]
				questMaterials, _ := mQuest.GetQuestReqItems()
				for _, reqItems := range questMaterials {
					c.questMobsIDs = append(c.questMobsIDs, reqItems.DropMobID)
				}
			}
		}
	}
}

func (c *Character) GetQuestItemsDrop(npcID int) (int64, int, int) {
	questItems, _ := FindQuestByMapID(int(c.Map))
	questItems = funk.Filter(questItems, func(item *QuestList) bool {
		return item.DropFromMobs
	}).([]*QuestList)
	questbymap, _ := FindQuestsAcceptedByID(c.ID)
	for _, quest := range questItems {
		questID := int(quest.ID)
		if funk.Contains(questbymap, questID) {
			mQuest := QuestsList[int(quest.ID)]
			questMaterials, _ := mQuest.GetQuestReqItems()
			for _, reqItems := range questMaterials {
				if reqItems.DropMobID == npcID {
					return reqItems.ItemID, reqItems.ItemCount, questID
				}
			}
		}
	}
	return 0, 0, 0
}
func (c *Character) ChangeMap(mapID int16, coordinate *utils.Location, args ...interface{}) ([]byte, error) {

	if !funk.Contains(unlockedMaps, mapID) {
		return nil, nil
	}
	if c.TradeID != "" && c.Map != 243 {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to change map while trading"
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}
	resp, r := MAP_CHANGED, utils.Packet{}
	if FindPlayerInArena(c) && !funk.Contains(ArenaZones, mapID) && !FindPlayerArena(c).IsFinished {
		PlayerDeserterPunishment(c)
		RemovePlayerFromArena(c)
	}
	c.Map = mapID
	c.EndPvP()
	if !c.IsinWar {
		if coordinate == nil { // if no coordinate then teleport home
			d := SavePoints[uint8(mapID)]
			if d == nil {
				d = &SavePoint{Point: "(100.0,100.0)"}
			}
			coordinate = ConvertPointToLocation(d.Point)
		}
	} else {
		if c.Faction == 1 && c.Map != 230 {
			delete(OrderCharacters, c.ID)
			c.IsinWar = false
		} else if c.Faction == 2 && c.Map != 230 {
			delete(ShaoCharacters, c.ID)
			c.IsinWar = false
		}
	}
	//LoadQuestItems
	c.GetMapQuestMobs()
	//END
	qList, _ := FindQuestsByCharacterID(c.ID)
	for _, quest := range qList {
		if quest.QuestState != 2 {
			questresp, _ := c.LoadReturnQuests(quest.ID, quest.QuestState)
			resp.Concat(questresp)
		}

	}

	if funk.Contains(sharedMaps, mapID) && !funk.Contains(PVPServers, int16(c.Socket.User.ConnectedServer)) || c.IsinWar { // shared map
		//c.Socket.User.ConnectedServer = 1 // No more Server hopping god dammit
	}

	if c.GuildID > 0 {
		guild, err := FindGuildByID(c.GuildID)
		if err == nil && guild != nil {
			guild.InformMembers(c)
		}
	}

	consItems, _ := FindConsignmentItemsBySellerID(c.ID)
	consItems = (funk.Filter(consItems, func(item *ConsignmentItem) bool {
		return item.IsSold
	}).([]*ConsignmentItem))
	if len(consItems) > 0 {
		r.Concat(CONSIGMENT_ITEM_SOLD)
	}

	slots, err := c.InventorySlots()
	if err == nil {
		pet := slots[0x0A].Pet
		if pet != nil && pet.IsOnline {
			pet.IsOnline = false
			r.Concat(DISMISS_PET)
			showpet, _ := c.ShowItems()
			resp.Concat(showpet)
			c.IsMounting = false
		}
	}

	if c.AidMode {
		c.AidMode = false
		r.Concat(c.AidStatus())
	}

	RemovePetFromRegister(c)
	//RemoveFromRegister(c)
	//GenerateID(c)

	c.SetCoordinate(coordinate)

	if len(args) == 0 { // not logging in
		c.OnSight.DropsMutex.Lock()
		c.OnSight.Drops = map[int]interface{}{}
		c.OnSight.DropsMutex.Unlock()

		c.OnSight.MobMutex.Lock()
		c.OnSight.Mobs = map[int]interface{}{}
		c.OnSight.MobMutex.Unlock()

		c.OnSight.NpcMutex.Lock()
		c.OnSight.NPCs = map[int]interface{}{}
		c.OnSight.NpcMutex.Unlock()

		c.OnSight.PetsMutex.Lock()
		c.OnSight.Pets = map[int]interface{}{}
		c.OnSight.PetsMutex.Unlock()
	}

	resp[13] = byte(mapID)                                     // map id
	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), 14) // coordinate-x
	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 18) // coordinate-y
	resp[36] = byte(mapID)                                     // map id
	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), 46) // coordinate-x
	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 50) // coordinate-y
	resp[61] = byte(mapID)                                     // map id

	spawnData, _ := c.SpawnCharacter()
	r.Concat(spawnData)
	resp.Concat(r)
	resp.Concat(c.Socket.User.GetTime())
	if funk.Contains(DarkZones, mapID) {
		resp.Concat(DARK_MODE_ACTIVE)
	}
	return resp, nil
}

func DoesSlotAffectStats(slotNo int16) bool {
	return slotNo < 0x0B || (slotNo >= 0x0133 && slotNo <= 0x013B) || (slotNo >= 0x18D && slotNo <= 0x192)
}

func (c *Character) RemoveItem(slotID int16) ([]byte, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	item := slots[slotID]

	resp := ITEM_REMOVED
	resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
	resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 13)     // slot id

	affects, activated := DoesSlotAffectStats(slotID), item.Activated
	if affects || activated {
		item.Activated = false
		item.InUse = false

		statData, err := c.GetStats()
		if err != nil {
			return nil, err
		}

		resp.Concat(statData)
	}

	info := Items[item.ItemID]
	if activated {
		if item.ItemID == 100080000 { // eyeball of divine
			c.DetectionMode = false
		}

		if info != nil && info.GetType() == FORM_TYPE {
			c.Morphed = false
			c.MorphedNPCID = 0
			resp.Concat(FORM_DEACTIVATED)
		}

		data := ITEM_EXPIRED
		data.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 6)
		resp.Concat(data)
	}

	if affects {
		itemsData, err := c.ShowItems()
		if err != nil {
			return nil, err
		}

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: itemsData, Type: nats.SHOW_ITEMS}
		if err = p.Cast(); err != nil {
			return nil, err
		}

		resp.Concat(itemsData)
	}

	consItem, _ := FindConsignmentItemByID(item.ID)
	if consItem == nil { // item is not in consigment table
		err = item.Delete()
		if err != nil {
			return nil, err
		}

	} else { // seller did not claim the consigment item
		newItem := NewSlot()
		*newItem = *item
		newItem.UserID = null.StringFromPtr(nil)
		newItem.CharacterID = null.IntFromPtr(nil)
		newItem.Update()
		InventoryItems.Add(consItem.ID, newItem)
	}

	*item = *NewSlot()
	return resp, nil
}

func (c *Character) SellItem(itemID, slot, quantity int, unitPrice uint64) ([]byte, error) {

	c.AntiDupeMutex.Lock()
	defer c.AntiDupeMutex.Unlock()

	if c.TradeID != "" {
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	c.LootGold(unitPrice * uint64(quantity))
	_, err := c.RemoveItem(int16(slot))
	if err != nil {
		return nil, err
	}

	resp := SELL_ITEM
	resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 8)  // item id
	resp.Insert(utils.IntToBytes(uint64(slot), 2, true), 12)   // slot id
	resp.Insert(utils.IntToBytes(uint64(c.Gold), 8, true), 14) // character gold

	return resp, nil
}

func (c *Character) GetStats() ([]byte, error) {

	if c == nil {
		log.Println("c is nil")
		return nil, nil

	} else if c.Socket == nil {
		log.Println("socket is nil")
		return nil, nil
	}

	st := c.Socket.Stats
	if st == nil {
		return nil, nil
	}

	prevStat := *st

	// -- Refresh Longterm Buffs --
	allBuffs, err := c.FindAllRelevantBuffs()
	if err != nil {
		return nil, err
	}
	for _, buff := range allBuffs {
		if buff.ID == 10098 || buff.ID == 10100 { // Water and Fire Statue Buffs
			if buff.IsPercent && (!buff.IsServerEpoch && buff.StartedAt+buff.Duration > c.Epoch || buff.IsServerEpoch && buff.StartedAt+buff.Duration > GetServerEpoch()) {
				var remainingTime int64
				if buff.IsServerEpoch {
					elapsedTime := GetServerEpoch() - buff.StartedAt
					remainingTime = buff.Duration - elapsedTime
				} else {
					elapsedTime := c.Epoch - buff.StartedAt
					remainingTime = buff.Duration - elapsedTime
				}
				buff.Delete()

				err = st.Calculate()
				if err != nil {
					return nil, err
				}

				buff, _ := c.CraftBuff(buff.Name, buff.ID, buff.Plus, remainingTime, buff.IsServerEpoch)
				buff.Create()
				buff.Update()
			}
		}
	}

	err = st.Calculate()
	if err != nil {
		return nil, err
	}
	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}
	resp := GET_STATS

	index := 5
	resp.Insert(utils.IntToBytes(uint64(c.Level), 4, true), index) // character level
	index += 4

	duelState := 0
	if c.DuelID > 0 && c.DuelStarted {
		duelState = 500
	}

	resp.Insert(utils.IntToBytes(uint64(duelState), 2, true), index) // duel state
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.StatPoints), 2, true), index) // stat points
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.NaturePoints), 2, true), index) // divine stat points
	index += 2

	resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), index) // character skill points
	index += 6

	resp.Insert(utils.IntToBytes(uint64(c.Exp), 8, true), index) // character experience
	index += 8

	resp.Insert(utils.IntToBytes(uint64(c.AidTime), 4, true), index) // remaining aid
	index += 4
	index++

	targetExp := EXPs[int16(c.Level)].Exp
	resp.Insert(utils.IntToBytes(uint64(targetExp), 8, true), index) // character target experience
	index += 8

	resp.Insert(utils.IntToBytes(uint64(st.STR), 2, true), index) // character str
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.STR+st.STRBuff), 2, true), index) // character str buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.DEX), 2, true), index) // character dex
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.DEX+st.DEXBuff), 2, true), index) // character dex buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.INT), 2, true), index) // character int
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.INT+st.INTBuff), 2, true), index) // character int buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Wind), 2, true), index) // character wind
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Wind+st.WindBuff), 2, true), index) // character wind buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Water), 2, true), index) // character water
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Water+st.WaterBuff), 2, true), index) // character water buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Fire), 2, true), index) // character fire
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Fire+st.FireBuff), 2, true), index) // character fire buff
	index += 7

	resp.Insert(utils.FloatToBytes(c.RunningSpeed+c.AdditionalRunningSpeed, 4, true), index) // character running speed
	index += 4                                                                               //10 volt
	resp.Insert(utils.IntToBytes(uint64(st.AttackSpeed), 2, true), index)
	index += 4
	weapon := slots[c.WeaponSlot]
	if weapon.ItemID != 0 {
		itemInfo := Items[weapon.ItemID]
		if itemInfo.Type == 105 || itemInfo.Type == 108 {
			resp.Insert([]byte{0x03, 0x41}, index)
		} else {
			resp.Insert([]byte{0x00, 0x40}, index)
		}
	} else {
		resp.Insert([]byte{0x00, 0x40}, index)
	}
	index += 2
	resp.Insert(utils.IntToBytes(uint64(st.MaxHP), 4, true), index) // character max hp
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.MaxCHI), 4, true), index) // character max chi
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.MinATK), 2, true), index) // character min atk
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.MaxATK), 2, true), index) // character max atk
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.DEF), 4, true), index) // character def
	index += 4
	resp.Insert(utils.IntToBytes(uint64(st.DEF), 4, true), index) // character def
	index += 4
	resp.Insert(utils.IntToBytes(uint64(st.DEF), 4, true), index) // character def
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.MinArtsATK), 4, true), index) // character min arts atk
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.MaxArtsATK), 4, true), index) // character max arts atk
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.ArtsDEF), 4, true), index) // character arts def
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.Accuracy), 2, true), index) // character accuracy
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Dodge), 2, true), index) // character dodge
	index += 2
	//resp.Overwrite(utils.IntToBytes(uint64(0), 2, true), index) // character dodge
	index += 2
	resp.Insert(utils.IntToBytes(uint64(st.PoisonATK), 2, true), index) // character PoisonDamage
	index += 2
	resp.Insert(utils.IntToBytes(uint64(st.PoisonDEF), 2, true), index) // character PoisonDEF
	index += 2
	index++
	resp.Insert(utils.IntToBytes(uint64(st.ConfusionATK), 2, true), index) // character ParaATK
	index += 2
	resp.Insert(utils.IntToBytes(uint64(st.ConfusionDEF), 2, true), index) // character ParaDEF
	index += 2
	index++
	resp.Insert(utils.IntToBytes(uint64(st.ParalysisATK), 2, true), index) // character ConfusionATK
	index += 2
	resp.Insert(utils.IntToBytes(uint64(st.ParalysisDEF), 2, true), index) // character ConfusionDef
	index += 2
	//IDE JÖNNEK A PVP STATOK!
	index += 18
	resp.Insert([]byte{0x00, 0x00, 0x00, 0x00}, index)
	index = binary.Size(resp) - 2
	resp.Insert([]byte{0x00, 0x00}, index)
	resp.SetLength(int16(binary.Size(resp) - 6))
	resp.Concat(c.GetHPandChi()) // hp and chi

	if c.ShowStats {
		stat := *st
		v := reflect.ValueOf(stat)
		prevV := reflect.ValueOf(prevStat)
		c.Socket.Write(messaging.InfoMessage("--- STAT DIFFERENCE STARTS HERE ---"))
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).Interface() != prevV.Field(i).Interface() {
				var percentageDiff float64
				if v.Type().Field(i).Name == "STRBuff" {
					percentageDiff = (float64((v.Field(i).Int()+int64(stat.STR))-(prevV.Field(i).Int()+int64(prevStat.STR))) / float64((prevV.Field(i).Int() + int64(prevStat.STR)))) * 100
					c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("STR changed by %.2f%%\n", percentageDiff)))
				} else if v.Type().Field(i).Name == "INTBuff" {
					percentageDiff = (float64((v.Field(i).Int()+int64(stat.INT))-(prevV.Field(i).Int()+int64(prevStat.INT))) / float64((prevV.Field(i).Int() + int64(prevStat.INT)))) * 100
					c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("INT changed by %.2f%%\n", percentageDiff)))
				} else if v.Type().Field(i).Name == "DEXBuff" {
					percentageDiff = (float64((v.Field(i).Int()+int64(stat.DEX))-(prevV.Field(i).Int()+int64(prevStat.DEX))) / float64((prevV.Field(i).Int() + int64(prevStat.DEX)))) * 100
					c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("DEX changed by %.2f%%\n", percentageDiff)))
				} else if v.Type().Field(i).Name == "WindBuff" {
					percentageDiff = (float64((v.Field(i).Int()+int64(stat.Wind))-(prevV.Field(i).Int()+int64(prevStat.Wind))) / float64((prevV.Field(i).Int() + int64(prevStat.Wind)))) * 100
					c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("Wind changed by %.2f%%\n", percentageDiff)))
				} else if v.Type().Field(i).Name == "WaterBuff" {
					percentageDiff = (float64((v.Field(i).Int()+int64(stat.Water))-(prevV.Field(i).Int()+int64(prevStat.Water))) / float64((prevV.Field(i).Int() + int64(prevStat.Water)))) * 100
					c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("Water changed by %.2f%%\n", percentageDiff)))
				} else if v.Type().Field(i).Name == "FireBuff" {
					percentageDiff = (float64((v.Field(i).Int()+int64(stat.Fire))-(prevV.Field(i).Int()+int64(prevStat.Fire))) / float64((prevV.Field(i).Int() + int64(prevStat.Fire)))) * 100
					c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("Fire changed by %.2f%%\n", percentageDiff)))
				} else if v.Type().Field(i).Name == "GoldRate" || v.Type().Field(i).Name == "ExpRate" || v.Type().Field(i).Name == "DropRate" {
					continue
				} else {
					percentageDiff = (float64(v.Field(i).Int()-prevV.Field(i).Int()) / float64(prevV.Field(i).Int())) * 100
					c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("%s changed by %.2f%%\n", v.Type().Field(i).Name, percentageDiff)))
				}
			}
		}
	}

	return resp, nil
}

func (c *Character) BSUpgrade(slotID int64, stones []*InventorySlot, luck, protection *InventorySlot, stoneSlots []int64, luckSlot, protectionSlot int64) ([]byte, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to plus items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	item := slots[slotID]
	if item.Plus >= 15 { // cannot be upgraded more
		resp := utils.Packet{0xAA, 0x55, 0x31, 0x00, 0x54, 0x02, 0xA6, 0x0F, 0x01, 0x00, 0xA3, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
		resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
		resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 17)     // slot id
		resp.Insert(item.GetUpgrades(), 19)                            // item upgrades
		resp[34] = byte(item.SocketCount)                              // socket count
		resp.Insert(item.GetSockets(), 35)                             // item sockets
		c := 35 + 15
		if item.ItemType != 0 {
			resp.Overwrite(utils.IntToBytes(uint64(item.ItemType), 1, true), c-6)
			if item.ItemType == 2 {
				resp.Overwrite(utils.IntToBytes(uint64(item.JudgementStat), 4, true), c-5)
			}
		}

		return resp, nil
	}

	info := Items[item.ItemID]
	cost := (info.BuyPrice / 10) * int64(item.Plus+1) * int64(math.Pow(2, float64(len(stones)-1)))

	if uint64(cost) > c.Gold {
		resp := messaging.SystemMessage(messaging.INSUFFICIENT_GOLD)
		return resp, nil

	} else if len(stones) == 0 {
		resp := messaging.SystemMessage(messaging.INCORRECT_GEM_QTY)
		return resp, nil
	}
	if protection != nil && item.Plus > 4 {
		if protection.ItemID == 97700564 {
			resp := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA6, 0x0F, 0x00, 0x55, 0xAA}
			resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
			return resp, nil
		}
	}
	stone := stones[0]
	stoneInfo := Items[stone.ItemID]

	if int16(item.Plus) < stoneInfo.MinUpgradeLevel || stoneInfo.ID > 255 {
		resp := messaging.SystemMessage(messaging.INCORRECT_GEM)
		return resp, nil
	}

	itemType := info.GetType()
	//log.Printf("ItemType: %d StoneType: %d", info.Type, stoneInfo.Type)
	typeMatch := (stoneInfo.Type == 190 && itemType == PET_ITEM_TYPE) || (stoneInfo.Type == 191 && itemType == HT_ARMOR_TYPE) ||
		(stoneInfo.Type == 192 && (itemType == ACC_TYPE || itemType == MASTER_HT_ACC)) && item.ItemType == 0 || (stoneInfo.Type == 194 && itemType == WEAPON_TYPE && item.ItemType == 0) || (stoneInfo.Type == 195 && itemType == ARMOR_TYPE && item.ItemType == 0) ||
		//DISC ITEMS
		(stoneInfo.Type == 229 && stoneInfo.HtType == 36 && itemType == WEAPON_TYPE && item.ItemType == 2) || (stoneInfo.Type == 229 && stoneInfo.HtType == 37 && itemType == ARMOR_TYPE && item.ItemType == 2) || (stoneInfo.Type == 229 && stoneInfo.HtType == 38 && (itemType == ACC_TYPE || itemType == MASTER_HT_ACC) && item.ItemType == 2)

	if !typeMatch {

		resp := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA4, 0x0F, 0x00, 0x55, 0xAA}
		resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
		return resp, nil
	}

	rate := float64(STRRates[item.Plus] * len(stones))
	plus := item.Plus + 1

	if stone.Plus > 0 { // Precious Pendent or Ghost Dagger or Dragon Scale
		for i := 0; i < len(stones); i++ {
			for j := i; j < len(stones); j++ {
				if stones[i].Plus != stones[j].Plus { // mismatch stone plus
					resp := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA4, 0x0F, 0x00, 0x55, 0xAA}
					resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
					return resp, nil
				}
			}
		}

		plus = item.Plus + stone.Plus
		if plus > 15 {
			plus = 15
		}

		rate = float64(STRRates[plus-1] * len(stones))
	}

	if luck != nil {
		luckInfo := Items[luck.ItemID]
		if luckInfo.Type == 164 { // charm of luck
			k := float64(luckInfo.SellPrice) / 100
			rate += rate * k / float64(len(stones))

		} else if luckInfo.Type == 219 { // bagua
			if byte(luckInfo.SellPrice) != item.Plus { // item plus not matching with bagua
				resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x02, 0xB6, 0x0F, 0x55, 0xAA}
				return resp, nil

			} else if len(stones) < 3 {
				resp := messaging.SystemMessage(messaging.INCORRECT_GEM_QTY)
				return resp, nil
			}

			rate = 1000
			bagRates := []int{luckInfo.HolyWaterUpg3, luckInfo.HolyWaterRate1, luckInfo.HolyWaterRate2, luckInfo.HolyWaterRate3}
			seed := utils.RandInt(0, 100)

			for i := 0; i < len(bagRates); i++ {
				if int(seed) > bagRates[i] {
					plus++
				}
			}
		}
	}

	protectionInfo := &Item{}
	if protection != nil {
		protectionInfo = Items[protection.ItemID]
	}

	resp := utils.Packet{}
	c.LootGold(-uint64(cost))
	resp.Concat(c.GetGold())

	//chance := rate / 10
	//c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("Chance to success was: %.2f%%\n", chance)))

	seed := int(utils.RandInt(0, 1000))
	c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You rolled: %d of 1000. Must be lower then %d.", seed, int(rate))))
	if float64(seed) < rate { // upgrade successful
		var codes []byte
		for i := item.Plus; i < plus; i++ {
			codes = append(codes, byte(stone.ItemID))
		}

		before := item.GetUpgrades()
		resp.Concat(item.Upgrade(int16(slotID), codes...))
		logger.Log(logging.ACTION_UPGRADE_ITEM, c.ID, fmt.Sprintf("Item (%d) upgraded: %s -> %s", item.ID, before, item.GetUpgrades()), c.UserID)

	} else if itemType == HT_ARMOR_TYPE || itemType == PET_ITEM_TYPE ||
		(protection != nil && protectionInfo.GetType() == SCALE_TYPE) { // ht or pet item failed or got protection

		if protectionInfo.GetType() == SCALE_TYPE { // if scale
			if item.Plus < uint8(protectionInfo.SellPrice) {
				item.Plus = 0
			} else {
				item.Plus -= uint8(protectionInfo.SellPrice)
			}
		} else {
			if item.Plus < stone.Plus {
				item.Plus = 0
			} else {
				item.Plus -= stone.Plus
			}
		}

		upgs := item.GetUpgrades()
		for i := int(item.Plus); i < len(upgs); i++ {
			item.SetUpgrade(i, 0)
		}

		r := HT_UPG_FAILED
		r.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
		r.Insert(utils.IntToBytes(uint64(slotID), 2, true), 17)     // slot id
		r.Insert(item.GetUpgrades(), 19)                            // item upgrades
		r[34] = byte(item.SocketCount)                              // socket count
		r.Insert(item.GetSockets(), 35)                             // item sockets

		resp.Concat(r)
		logger.Log(logging.ACTION_UPGRADE_ITEM, c.ID, fmt.Sprintf("Item (%d) upgrade failed but not vanished", item.ID), c.UserID)

	} else { // casual item failed so destroy it
		r := UPG_FAILED
		r.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
		resp.Concat(r)

		itemsData, err := c.RemoveItem(int16(slotID))
		if err != nil {
			return nil, err
		}

		resp.Concat(itemsData)
		logger.Log(logging.ACTION_UPGRADE_ITEM, c.ID, fmt.Sprintf("Item (%d) upgrade failed and destroyed", item.ID), c.UserID)
	}

	for _, slot := range stoneSlots {
		resp.Concat(*c.DecrementItem(int16(slot), 1))
	}

	if luck != nil {
		resp.Concat(*c.DecrementItem(int16(luckSlot), 1))
	}

	if protection != nil {
		resp.Concat(*c.DecrementItem(int16(protectionSlot), 1))
	}

	err = item.Update()
	if err != nil {
		return nil, err
	}

	return resp, nil
}
func (c *Character) BSProduction(book *InventorySlot, materials []*InventorySlot, special *InventorySlot, prodSlot int16, bookSlot, specialSlot int16, materialSlots []int16, materialCounts []uint) ([]byte, error) {

	production := Productions[int(book.ItemID)]
	prodMaterials, err := production.GetMaterials()
	if err != nil {
		return nil, err
	}

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to use composition while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	canProduce := true

	for i := 0; i < len(materials); i++ {
		if materials[i].Quantity < uint(prodMaterials[i].Count) || int(materials[i].ItemID) != prodMaterials[i].ID {
			canProduce = false
			break
		}
	}

	if prodMaterials[2].ID > 0 && (special.Quantity < uint(prodMaterials[2].Count) || int(special.ItemID) != prodMaterials[2].ID) {
		canProduce = false
	}

	cost := uint64(production.Cost)
	if cost > c.Gold || !canProduce {
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x04, 0x07, 0x10, 0x55, 0xAA}
		return resp, nil
	}

	c.LootGold(-cost)
	luckRate := float64(1)
	if special != nil {
		specialInfo := Items[special.ItemID]
		luckRate = float64(specialInfo.SellPrice+100) / 100
	}

	resp := &utils.Packet{}
	seed := int(utils.RandInt(0, 1000))
	if float64(seed) < float64(production.Probability)*luckRate { // Success
		itemInfo := Items[int64(production.Production)]
		quantity := 1
		if itemInfo.Timer > 0 {
			quantity = itemInfo.Timer
		}
		resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(production.Production), Quantity: uint(quantity)}, prodSlot, false)
		if err != nil {
			return nil, err
		} else if resp == nil {
			return nil, nil
		}

		resp.Concat(PRODUCTION_SUCCESS)
		logger.Log(logging.ACTION_PRODUCTION, c.ID, fmt.Sprintf("Production (%d) success", book.ItemID), c.UserID)

	} else { // Failed
		resp.Concat(PRODUCTION_FAILED)
		resp.Concat(c.GetGold())
		logger.Log(logging.ACTION_PRODUCTION, c.ID, fmt.Sprintf("Production (%d) failed", book.ItemID), c.UserID)
	}

	resp.Concat(*c.DecrementItem(int16(bookSlot), 1))

	for i := 0; i < len(materialSlots); i++ {
		resp.Concat(*c.DecrementItem(int16(materialSlots[i]), uint(materialCounts[i])))
	}

	if special != nil {
		resp.Concat(*c.DecrementItem(int16(specialSlot), 1))
	}

	return *resp, nil
}

func (c *Character) AdvancedFusion(items []*InventorySlot, special *InventorySlot, prodSlot int16) ([]byte, bool, error) {

	if len(items) < 3 {
		return nil, false, nil
	}

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to fusion items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), false, nil //Cannot do that while trading
	}

	fusion := Fusions[items[0].ItemID]
	seed := int(utils.RandInt(0, 1000))

	cost := uint64(fusion.Cost)
	if c.Gold < cost {
		return FUSION_FAILED, false, nil
	}

	if items[0].ItemID != fusion.Item1 || items[1].ItemID != fusion.Item2 || items[2].ItemID != fusion.Item3 {
		return FUSION_FAILED, false, nil
	}

	c.LootGold(-cost)
	rate := float64(fusion.Probability)
	if special != nil {
		info := Items[special.ItemID]
		rate *= float64(info.SellPrice+100) / 100
	}

	if float64(seed) < rate { // Success
		resp := utils.Packet{}
		quantity := 1
		iteminfo := Items[fusion.Production]
		if iteminfo.Timer > 0 {
			quantity = iteminfo.Timer
		}
		itemData, _, err := c.AddItem(&InventorySlot{ItemID: fusion.Production, Quantity: uint(quantity)}, prodSlot, false)
		if err != nil {
			return nil, false, err
		} else if itemData == nil {
			return nil, false, nil
		}

		resp.Concat(*itemData)
		resp.Concat(FUSION_SUCCESS)
		logger.Log(logging.ACTION_ADVANCED_FUSION, c.ID, fmt.Sprintf("Advanced fusion (%d) success", items[0].ItemID), c.UserID)
		return resp, true, nil

	} else { // Failed
		resp := FUSION_FAILED
		resp.Concat(c.GetGold())
		logger.Log(logging.ACTION_ADVANCED_FUSION, c.ID, fmt.Sprintf("Advanced fusion (%d) failed", items[0].ItemID), c.UserID)
		return resp, false, nil
	}
}

func (c *Character) Dismantle(item, special *InventorySlot) ([]byte, bool, error) {

	melting := Meltings[int(item.ItemID)]
	cost := uint64(melting.Cost)

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to dismintle items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), false, nil //Cannot do that while trading
	}

	if c.Gold < cost {
		return nil, false, nil
	}

	meltedItems, err := melting.GetMeltedItems()
	if err != nil {
		return nil, false, err
	}

	itemCounts, err := melting.GetItemCounts()
	if err != nil {
		return nil, false, err
	}

	c.LootGold(-cost)

	info := Items[item.ItemID]

	profit := utils.RandFloat(1, melting.ProfitMultiplier) * float64(info.BuyPrice*2)
	c.LootGold(uint64(profit))

	resp := utils.Packet{}
	r := DISMANTLE_SUCCESS
	r.Insert(utils.IntToBytes(uint64(profit), 8, true), 9) // profit

	count, index := 0, 18
	for i := 0; i < 3; i++ {
		id := meltedItems[i]
		if id == 0 {
			continue
		}

		maxCount := int64(itemCounts[i])
		meltedCount := utils.RandInt(0, maxCount+1)
		if meltedCount == 0 {
			continue
		}

		count++
		r.Insert(utils.IntToBytes(uint64(id), 4, true), index) // melted item id
		index += 4

		r.Insert([]byte{0x00, 0xA2}, index)
		index += 2

		r.Insert(utils.IntToBytes(uint64(meltedCount), 2, true), index) // melted item count
		index += 2

		freeSlot, err := c.FindFreeSlot()
		if err != nil {
			return nil, false, err
		}

		r.Insert(utils.IntToBytes(uint64(freeSlot), 2, true), index) // free slot id
		index += 2

		r.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, index) // upgrades
		index += 34

		itemData, _, err := c.AddItem(&InventorySlot{ItemID: int64(id), Quantity: uint(meltedCount)}, freeSlot, false)
		if err != nil {
			return nil, false, err
		} else if itemData == nil {
			return nil, false, nil
		}

		resp.Concat(*itemData)
	}

	r[17] = byte(count)
	length := int16(44*count) + 14

	if melting.SpecialItem > 0 {
		seed := int(utils.RandInt(0, 1000))

		if seed < melting.SpecialProbability {

			freeSlot, err := c.FindFreeSlot()
			if err != nil {
				return nil, false, err
			}

			r.Insert([]byte{0x01}, index)
			index++

			r.Insert(utils.IntToBytes(uint64(melting.SpecialItem), 4, true), index) // special item id
			index += 4

			r.Insert([]byte{0x00, 0xA2, 0x01, 0x00}, index)
			index += 4

			r.Insert(utils.IntToBytes(uint64(freeSlot), 2, true), index) // free slot id
			index += 2

			r.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, index) // upgrades
			index += 34

			itemData, _, err := c.AddItem(&InventorySlot{ItemID: int64(melting.SpecialItem), Quantity: 1}, freeSlot, false)
			if err != nil {
				return nil, false, err
			} else if itemData == nil {
				return nil, false, nil
			}

			resp.Concat(*itemData)
			length += 45
		}
	}

	r.SetLength(length)
	resp.Concat(r)
	resp.Concat(c.GetGold())
	logger.Log(logging.ACTION_DISMANTLE, c.ID, fmt.Sprintf("Dismantle (%d) success with %d gold", item.ID, c.Gold), c.UserID)
	return resp, true, nil
}

func (c *Character) Extraction(item, special *InventorySlot, itemSlot int16) ([]byte, bool, error) {

	info, ok := GetItemInfo(item.ItemID)
	if !ok {
		return nil, false, nil
	}
	code := int(item.GetUpgrades()[item.Plus-1])
	cost := uint64(info.SellPrice) * uint64(HaxCodes[code].ExtractionMultiplier) / 1000

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to extract items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), false, nil //Cannot do that while trading
	}
	if c.Gold < cost {
		return nil, false, nil
	}

	c.LootGold(-cost)
	item.Plus--
	item.SetUpgrade(int(item.Plus), 0)

	resp := utils.Packet{}
	r := EXTRACTION_SUCCESS
	r.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9)    // item id
	r.Insert(utils.IntToBytes(uint64(item.Quantity), 2, true), 15) // item quantity
	r.Insert(utils.IntToBytes(uint64(itemSlot), 2, true), 17)      // item slot
	r.Insert(item.GetUpgrades(), 19)                               // item upgrades
	r[34] = byte(item.SocketCount)                                 // item socket count
	r.Insert(item.GetUpgrades(), 35)                               // item sockets

	count := 1          //int(utils.RandInt(1, 4))
	r[53] = byte(count) // stone count

	index, length := 54, int16(51)
	for i := 0; i < count; i++ {

		freeSlot, err := c.FindFreeSlot()
		if err != nil {
			return nil, false, err
		}

		id := int64(HaxCodes[code].ExtractedItem)
		itemData, _, err := c.AddItem(&InventorySlot{ItemID: id, Quantity: 1}, freeSlot, false)
		if err != nil {
			return nil, false, err
		} else if itemData == nil {
			return nil, false, nil
		}

		resp.Concat(*itemData)

		r.Insert(utils.IntToBytes(uint64(id), 4, true), index) // extracted item id
		index += 4

		r.Insert([]byte{0x00, 0xA2, 0x01, 0x00}, index)
		index += 4

		r.Insert(utils.IntToBytes(uint64(freeSlot), 2, true), index) // free slot id
		index += 2

		r.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, index) // upgrades
		index += 34

		length += 44
	}

	r.SetLength(length)
	resp.Concat(r)
	resp.Concat(c.GetGold())

	err := item.Update()
	if err != nil {
		return nil, false, err
	}

	logger.Log(logging.ACTION_EXTRACTION, c.ID, fmt.Sprintf("Extraction success for item (%d)", item.ID), c.UserID)
	return resp, true, nil
}

func (c *Character) CreateSocket(item, special *InventorySlot, itemSlot, specialSlot int16) ([]byte, error) {

	info, ok := GetItemInfo(item.ItemID)
	if !ok {
		return nil, nil
	}
	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to create socket on items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)

	}

	cost := uint64(info.SellPrice * 164)
	if c.Gold < cost {
		return nil, nil
	}

	if item.SocketCount > 0 && special != nil && special.ItemID == 17200186 { // socket init
		resp := c.DecrementItem(specialSlot, 1)
		resp.Concat(item.CreateSocket(itemSlot, 0))
		return *resp, nil

	} else if item.SocketCount > 0 { // item already has socket
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x16, 0x0B, 0xCF, 0x55, 0xAA}
		return resp, nil

	} else if item.SocketCount == 0 && special != nil && special.ItemID == 17200186 { // socket init with no sockets
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x16, 0x0A, 0xCF, 0x55, 0xAA}
		return resp, nil
	}

	seed := utils.RandInt(0, 1000)
	socketCount := int8(1)
	if seed >= 850 {
		socketCount = 4
	} else if seed >= 650 {
		socketCount = 3
	} else if seed >= 350 {
		socketCount = 2
	}

	c.LootGold(-cost)
	resp := utils.Packet{}
	if special != nil {
		if special.ItemID == 17200185 { // +1 miled stone
			socketCount++

		} else if special.ItemID == 15710239 { // +2 miled stone
			socketCount += 2
			if socketCount > 5 {
				socketCount = 5
			}

		}

		resp.Concat(*c.DecrementItem(specialSlot, 1))
	}
	item.SocketCount = socketCount
	item.Update()
	resp.Concat(item.CreateSocket(itemSlot, socketCount))
	resp.Concat(c.GetGold())
	return resp, nil
}

func (c *Character) UpgradeSocket(item, socket, special, edit *InventorySlot, itemSlot, socketSlot, specialSlot, editSlot int16, locks []bool) ([]byte, error) {

	info, ok := GetItemInfo(item.ItemID)
	if !ok {
		return nil, nil
	}
	cost := uint64(info.SellPrice * 164)
	if c.Gold < cost {
		return nil, nil
	}
	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to upgrade sockets on items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	if item.SocketCount == 0 { // No socket on item
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x16, 0x10, 0xCF, 0x55, 0xAA}
		return resp, nil
	}

	if socket.Plus < uint8(item.SocketCount) { // Insufficient socket
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x17, 0x0D, 0xCF, 0x55, 0xAA}
		return resp, nil
	}

	stabilize := special != nil && special.ItemID == 17200187

	if edit != nil {
		if edit.ItemID < 17503030 && edit.ItemID > 17503032 {
			return nil, nil
		}
	}

	upgradesArray := bytes.Join([][]byte{ArmorUpgrades, WeaponUpgrades, AccUpgrades}, []byte{})
	sockets := make([]byte, item.SocketCount)
	socks := item.GetSockets()
	for i := int8(0); i < item.SocketCount; i++ {
		if locks[i] {
			sockets[i] = socks[i]
			continue
		}

		seed := utils.RandInt(0, int64(len(upgradesArray)+1))
		code := upgradesArray[seed]
		if stabilize && code%5 > 0 {
			code++
		} else if !stabilize && code%5 == 0 {
			code--
		}

		sockets[i] = code
	}

	c.LootGold(-cost)
	resp := utils.Packet{}
	resp.Concat(item.UpgradeSocket(itemSlot, sockets))
	resp.Concat(c.GetGold())
	resp.Concat(*c.DecrementItem(socketSlot, 1))

	if special != nil {
		resp.Concat(*c.DecrementItem(specialSlot, 1))
	}

	if edit != nil {
		resp.Concat(*c.DecrementItem(editSlot, 1))
	}

	return resp, nil
}

func (c *Character) CoProduction(craftID, bFinished int) ([]byte, error) {
	resp := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x20, 0x0a, 0x00, 0x00, 0x55, 0xAA}
	resp.Concat(utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0x2d, 0x03, 0x2d, 0x03, 0xfe, 0x9a, 0x00, 0x00, 0x83, 0x1b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x54, 0x83, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA})

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to use co-production while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	if bFinished == 1 {
		production, ok := CraftItems[int(craftID)]
		if !ok {
			return nil, nil
		}
		var prodMaterials []int
		var prodQty []int
		var probabilities []int
		var craftedItems []int
		var canBreak bool

		prodMaterials = append(prodMaterials, production.Material1)
		prodMaterials = append(prodMaterials, production.Material2)
		prodMaterials = append(prodMaterials, production.Material3)
		prodMaterials = append(prodMaterials, production.Material4)
		prodMaterials = append(prodMaterials, production.Material5)
		prodMaterials = append(prodMaterials, production.Material6)
		prodQty = append(prodQty, production.Material1Count)
		prodQty = append(prodQty, production.Material2Count)
		prodQty = append(prodQty, production.Material3Count)
		prodQty = append(prodQty, production.Material4Count)
		prodQty = append(prodQty, production.Material5Count)
		prodQty = append(prodQty, production.Material6Count)
		probabilities = append(probabilities, production.Probability1)
		probabilities = append(probabilities, production.Probability2)
		probabilities = append(probabilities, production.Probability3)
		craftedItems = append(craftedItems, 0)
		craftedItems = append(craftedItems, production.Probability1Result)
		craftedItems = append(craftedItems, production.Probability2Result)
		craftedItems = append(craftedItems, production.Probability3Result)

		firstDigit := production.ID
		for firstDigit >= 10 {
			firstDigit /= 10
		}
		if firstDigit == 1 {
			canBreak = true
		} else if firstDigit == 2 {
			canBreak = false
		} else {
			return nil, nil
		}

		cost := uint64(production.Cost)
		c.LootGold(-cost)

		slots, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}
		index := 0
		seed := int(utils.RandInt(0, 1000))

		c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You rolled: %d of 1000. Must be lower then %d.", seed, probabilities[0])))

		for _, prob := range probabilities {
			if float64(seed) < float64(prob) {
				index++
				continue
			}
			break
		}

		reward := NewSlot()
		reward.ItemID = int64(craftedItems[index])

		for i := 0; i < len(prodMaterials); i++ {
			if int64(prodMaterials[i]) == 0 {
				continue
			} else {
				slotID, _, err := c.FindItemInInventoryForProduction(nil, int64(prodMaterials[i]))
				if err != nil {
					return nil, err
				}
				matCount := uint(prodQty[i])

				if canBreak { // Always take resources
					itemData := c.DecrementItem(slotID, matCount)
					c.Socket.Write(*itemData)
				} else {
					if reward.ItemID != 0 { // Succeeded, take resources
						itemData := c.DecrementItem(slotID, matCount)
						c.Socket.Write(*itemData)
					}
				}
			}
		}

		if reward.ItemID == 0 {
			return PRODUCTION_FAILED, nil
		}
		item := Items[reward.ItemID]
		text := "User: " + c.Socket.User.Username + "(" + c.Socket.Character.UserID + ") successfully crafted: " + item.Name + "(" + strconv.Itoa(int(reward.ItemID)) + ")"
		utils.NewLog("logs/craft_logs.txt", text)
		c.Socket.Write(messaging.InfoMessage("Success!"))
		reward.Quantity = 1
		_, new, _ := c.AddItem(reward, -1, true)
		resp.Concat(slots[new].GetData(new))
		//c.NearbyNpcCastSkill(20352, 10067)
	}
	return resp, nil
}

func (c *Character) HolyWaterUpgrade(item, holyWater *InventorySlot, itemSlot, holyWaterSlot int16) ([]byte, error) {

	itemInfo, ok := GetItemInfo(item.ItemID)
	hwInfo, ok := GetItemInfo(holyWater.ItemID)
	if !ok {
		return nil, errors.New("HolyWaterUpgrade: Item not found")
	}

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to use holy water items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	if (itemInfo.GetType() == WEAPON_TYPE && (hwInfo.HolyWaterUpg1 < 66 || hwInfo.HolyWaterUpg1 > 105)) ||
		(itemInfo.GetType() == ARMOR_TYPE && (hwInfo.HolyWaterUpg1 < 41 || hwInfo.HolyWaterUpg1 > 65)) ||
		(itemInfo.GetType() == ACC_TYPE && hwInfo.HolyWaterUpg1 > 40) { // Mismatch type

		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x10, 0x36, 0x11, 0x55, 0xAA}
		return resp, nil
	}

	if item.Plus == 0 {
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x10, 0x37, 0x11, 0x55, 0xAA}
		return resp, nil
	}

	resp := utils.Packet{}
	seed, upgrade := int(utils.RandInt(0, 60)), 0
	if seed < hwInfo.HolyWaterRate1 {
		upgrade = hwInfo.HolyWaterUpg1
	} else if seed < hwInfo.HolyWaterRate2 {
		upgrade = hwInfo.HolyWaterUpg2
	} else if seed < hwInfo.HolyWaterRate3 {
		upgrade = hwInfo.HolyWaterUpg3
	} else {
		resp = HOLYWATER_FAILED
	}

	if upgrade > 0 {
		randSlot := utils.RandInt(0, int64(item.Plus))
		preUpgrade := item.GetUpgrades()[randSlot]
		item.SetUpgrade(int(randSlot), byte(upgrade))

		if preUpgrade == byte(upgrade) {
			resp = HOLYWATER_FAILED
		} else {
			resp = HOLYWATER_SUCCESS

			r := ITEM_UPGRADED
			r.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
			r.Insert(utils.IntToBytes(uint64(itemSlot), 2, true), 17)   // slot id
			r.Insert(item.GetUpgrades(), 19)                            // item upgrades
			r[34] = byte(item.SocketCount)                              // socket count
			r.Insert(item.GetSockets(), 35)                             // item sockets
			resp.Concat(r)

			new := funk.Map(item.GetUpgrades()[:item.Plus], func(upg byte) string {
				return HaxCodes[int(upg)].Code
			}).([]string)

			old := make([]string, len(new))
			copy(old, new)
			old[randSlot] = HaxCodes[int(preUpgrade)].Code

			msg := fmt.Sprintf("[%s] has been upgraded from [%s] to [%s].", itemInfo.Name, strings.Join(old, ""), strings.Join(new, ""))
			msgData := messaging.InfoMessage(msg)
			resp.Concat(msgData)
		}
	}

	itemData, _ := c.RemoveItem(holyWaterSlot)
	resp.Concat(itemData)

	err := item.Update()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Character) RegisterItem(item *InventorySlot, price uint64, itemSlot int16) ([]byte, error) {

	items, err := FindConsignmentItemsBySellerID(c.ID)
	if err != nil {
		return nil, err
	}

	if len(items) >= 10 {
		return nil, nil
	}

	commision := uint64(math.Min(float64(price/100), 50000000))
	if c.Gold < commision {
		return nil, nil
	}

	sale := FindSale(c.PseudoID)
	if sale != nil {
		return messaging.SystemMessage(messaging.CANNOT_MOVE_ITEM_IN_TRADE), nil
	}
	if c.TradeID != "" {
		return messaging.SystemMessage(messaging.CANNOT_MOVE_ITEM_IN_TRADE), nil
	}

	info, ok := Items[item.ItemID]
	if !ok {
		return nil, nil
	}
	if !info.Tradable {
		return nil, nil
	}
	consItem := &ConsignmentItem{
		ID:       item.ID,
		SellerID: c.ID,
		ItemName: info.Name,
		Quantity: int(item.Quantity),
		IsSold:   false,
		Price:    price,
	}

	if err := consItem.Create(); err != nil {
		return nil, err
	}

	c.LootGold(-commision)
	resp := ITEM_REGISTERED
	resp.Insert(utils.IntToBytes(uint64(consItem.ID), 4, true), 9)  // consignment item id
	resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 29) // item id

	if item.Pet != nil {
		resp[34] = byte(item.SocketCount)
	}

	resp.Insert(utils.IntToBytes(uint64(item.Quantity), 2, true), 35) // item count
	resp.Insert(item.GetUpgrades(), 37)                               // item upgrades

	if item.Pet != nil {
		resp[42] = 0 // item socket count
	} else {
		resp[42] = byte(item.SocketCount) // item socket count
	}

	resp.Insert(item.GetSockets(), 43) // item sockets

	newItem := NewSlot()
	*newItem = *item
	newItem.SlotID = -1
	newItem.Consignment = true
	newItem.Update()
	InventoryItems.Add(newItem.ID, newItem)

	*item = *NewSlot()
	resp.Concat(c.GetGold())
	resp.Concat(item.GetData(itemSlot))

	claimData, err := c.ClaimMenu()
	if err != nil {
		return nil, err
	}
	resp.Concat(claimData)

	return resp, nil
}

func (c *Character) ClaimMenu() ([]byte, error) {
	items, err := FindConsignmentItemsBySellerID(c.ID)
	if err != nil {
		return nil, err
	}
	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to claim consignment items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	resp := CLAIM_MENU
	resp.SetLength(int16(len(items)*0x6B + 6))
	resp.Insert(utils.IntToBytes(uint64(len(items)), 2, true), 8) // items count

	index := 10
	for _, item := range items {

		slot, err := FindInventorySlotByID(item.ID) // FIX: Buyer can destroy the item..
		if err != nil {
			continue
		}
		if slot == nil {
			slot = NewSlot()
			slot.ItemID = 17502455
			slot.Quantity = 1
		}

		info := Items[int64(slot.ItemID)]

		if item.IsSold {
			resp.Insert([]byte{0x01}, index)
		} else {
			resp.Insert([]byte{0x00}, index)
		}
		index++

		resp.Insert(utils.IntToBytes(uint64(item.ID), 4, true), index) // consignment item id
		index += 4

		resp.Insert([]byte{0x5E, 0x15, 0x01, 0x00}, index)
		index += 4

		resp.Insert([]byte(c.Name), index) // seller name
		index += len(c.Name)

		for j := len(c.Name); j < 20; j++ {
			resp.Insert([]byte{0x00}, index)
			index++
		}

		resp.Insert(utils.IntToBytes(item.Price, 8, true), index) // item price
		index += 8

		time := item.ExpiresAt.Time.Format("2006-01-02 15:04:05") // expires at
		resp.Insert([]byte(time), index)
		index += 19

		resp.Insert([]byte{0x00, 0x09, 0x00, 0x00, 0x00, 0x99, 0x31, 0xF5, 0x00}, index)
		index += 9

		resp.Insert(utils.IntToBytes(uint64(slot.ItemID), 4, true), index) // item id
		index += 4

		resp.Insert([]byte{0x00, 0xA1}, index)
		index += 2

		if info.GetType() == PET_TYPE {
			resp[index-1] = byte(slot.SocketCount)
		}

		resp.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), index) // item count
		index += 2

		resp.Insert(slot.GetUpgrades(), index) // item upgrades
		index += 15

		resp.Insert([]byte{byte(slot.SocketCount)}, index) // socket count
		index++

		resp.Insert(slot.GetSockets(), index)
		index += 15

		resp.Insert([]byte{0x00, 0x00, 0x00}, index)
		index += 3
	}

	return resp, nil
}

func (c *Character) BuyConsignmentItem(consignmentID int) ([]byte, error) {
	consignmentItem, err := FindConsignmentItemByID(consignmentID)
	if err != nil || consignmentItem == nil || consignmentItem.IsSold {
		return nil, err
	}

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to buy consignment items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	slot, err := FindInventorySlotByID(consignmentItem.ID)
	if err != nil {
		return nil, err
	}

	if c.Gold < consignmentItem.Price {
		return nil, nil
	}

	seller, err := FindCharacterByID(int(slot.CharacterID.Int64))
	if err != nil {
		return nil, err
	}

	resp := CONSIGMENT_ITEM_BOUGHT
	resp.Insert(utils.IntToBytes(uint64(consignmentID), 4, true), 8) // consignment item id

	slotID, err := c.FindFreeSlot()
	if err != nil {
		return nil, nil
	}

	newItem := NewSlot()
	*newItem = *slot
	newItem.Consignment = false
	newItem.UserID = null.StringFrom(c.UserID)
	newItem.CharacterID = null.IntFrom(int64(c.ID))
	newItem.SlotID = slotID

	err = newItem.Update()
	if err != nil {
		return nil, err
	}

	*slots[slotID] = *newItem
	InventoryItems.Add(newItem.ID, slots[slotID])
	c.LootGold(-consignmentItem.Price)

	resp.Concat(newItem.GetData(slotID))
	resp.Concat(c.GetGold())

	s, ok := Sockets[seller.UserID]
	if ok {
		s.Write(CONSIGMENT_ITEM_SOLD)
	}

	logger.Log(logging.ACTION_BUY_CONS_ITEM, c.ID, fmt.Sprintf("Bought consignment item (%d) with %d gold from (%d)", newItem.ID, consignmentItem.Price, seller.ID), c.UserID)

	text := fmt.Sprintf("Character :(%s)(%s) Bought consignment item (%d) with %d gold from seller (%s)(%s)", c.Name, c.UserID, consignmentItem.ID, consignmentItem.Price, seller.Name, seller.UserID)
	utils.NewLog("logs/consignment_bought_logs.txt", text)

	consignmentItem.IsSold = true
	go consignmentItem.Update()
	return resp, nil
}

func (c *Character) ClaimConsignmentItem(consignmentID int, isCancel bool) ([]byte, error) {

	consignmentItem, err := FindConsignmentItemByID(consignmentID)
	if err != nil || consignmentItem == nil {
		return nil, err
	}
	if c.TradeID != "" {
		msg := "cannot do that while trading."
		info := messaging.InfoMessage(msg)

		text := "Name: " + c.Name + "(" + c.UserID + ") tried to claim consignment items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return info, nil
	}

	resp := CONSIGMENT_ITEM_CLAIMED
	resp.Insert(utils.IntToBytes(uint64(consignmentID), 4, true), 10) // consignment item id

	if isCancel {
		if consignmentItem.IsSold {
			return nil, nil
		}

		slots, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}

		slotID, err := c.FindFreeSlot()
		if err != nil {
			return nil, err
		}

		slot, err := FindInventorySlotByID(consignmentItem.ID)
		if err != nil {
			return nil, err
		}

		newItem := NewSlot()
		*newItem = *slot
		newItem.Consignment = false
		newItem.SlotID = slotID

		err = newItem.Update()
		if err != nil {
			return nil, err
		}

		*slots[slotID] = *newItem
		InventoryItems.Add(newItem.ID, slots[slotID])

		resp.Concat(slot.GetData(slotID))

	} else {
		if !consignmentItem.IsSold {
			return nil, nil
		}

		logger.Log(logging.ACTION_BUY_CONS_ITEM, c.ID, fmt.Sprintf("Claimed consignment item (consid:%d) with %d gold", consignmentID, consignmentItem.Price), c.UserID)

		c.LootGold(consignmentItem.Price)
		resp.Concat(c.GetGold())
	}

	s, _ := FindInventorySlotByID(consignmentItem.ID)
	if s != nil && !s.UserID.Valid && !s.CharacterID.Valid {
		s.Delete()
	}

	go consignmentItem.Delete()
	return resp, nil
}

func (c *Character) UseConsumable(item *InventorySlot, slotID int16) ([]byte, error) {

	c.AntiDupeMutex.Lock()
	defer c.AntiDupeMutex.Unlock()
	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to use consumable items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	boxQuantity := uint(item.Quantity)
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			log.Printf("%+v", string(dbg.Stack()))

			r := utils.Packet{}
			r.Concat(*c.DecrementItem(slotID, 0))
			c.Socket.Write(r)
		}
	}()

	stat := c.Socket.Stats
	skills := c.Socket.Skills
	if stat.HP <= 0 {
		return *c.DecrementItem(slotID, 0), nil
	}

	info := Items[item.ItemID]
	if info == nil {
		return nil, nil
	} else if info.MinLevel > c.Level || (info.MaxLevel > 0 && info.MaxLevel < c.Level) {
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xF0, 0x03, 0x55, 0xAA} // inappropriate level
		return resp, nil
	}
	usedEarlier, _ := c.FindItemInUsed([]int64{item.ItemID})
	if usedEarlier {
		return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil
	}

	resp := utils.Packet{}
	canUse := c.CanUse(info.CharacterType)
	switch info.GetType() {
	case AFFLICTION_TYPE:
		err := stat.ResetStats()
		if err != nil {
			return nil, err
		}

		statData, _ := c.GetStats()
		resp.Concat(statData)
	case TRANSFORMATION_PAPER_TYPE:
		err := skills.ResetSkills()
		if err != nil {
			return nil, err
		}
		statData, _ := c.GetStats()
		resp.Concat(statData)

		c.Class = 0
		c.Update()
		goto DARKBACK
	case CHARM_OF_RETURN_TYPE:
		d := SavePoints[uint8(c.Map)]
		coordinate := ConvertPointToLocation(d.Point)
		resp.Concat(c.Teleport(coordinate))

		slots, err := c.InventorySlots()
		if err == nil {
			pet := slots[0x0A].Pet
			if pet != nil && pet.IsOnline {
				pet.IsOnline = false
				resp.Concat(DISMISS_PET)
				showpet, _ := c.ShowItems()
				resp.Concat(showpet)
				c.IsMounting = false
			}
		}

	case DEAD_SPIRIT_INCENSE_TYPE:
		slots, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}

		pet := slots[0x0A].Pet
		if pet != nil && !pet.IsOnline && pet.HP <= 0 {
			pet.HP = pet.MaxHP / 10
			resp.Concat(c.GetPetStats())
			resp.Concat(c.TogglePet())
		} else {
			goto FALLBACK
		}

	case MOVEMENT_SCROLL_TYPE:
		mapID := int16(info.SellPrice)
		data, _ := c.ChangeMap(mapID, nil)
		resp.Concat(data)

	case BAG_EXPANSION_TYPE:
		buff, err := FindBuffByCharacter(int(item.ItemID), c.ID)
		if err != nil {
			return nil, err
		} else if buff != nil {
			return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
		}

		buff = &Buff{ID: int(item.ItemID), CharacterID: c.ID, Name: info.Name, BagExpansion: true, StartedAt: c.Epoch, Duration: int64(info.Timer) * 60}
		err = buff.Create()
		if err != nil {
			return nil, err
		}

		resp = BAG_EXPANDED

	case FIRE_SPIRIT:
		buff, err := FindBuffByCharacter(10100, c.ID)
		if err != nil {
			return nil, err
		} else if buff != nil {
			return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
		}

		buff, err = FindBuffByCharacter(10098, c.ID) // check for water spirit
		if err != nil {
			return nil, err
		} else if buff != nil {
			return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
		}

		buff, err = c.CraftBuff(info.Name, 10100, 0, 604800, true)
		if err != nil {
			return nil, err
		}
		err = buff.Create()
		if err != nil {
			return nil, err
		}

		itemData, _, _ := c.AddItem(&InventorySlot{ItemID: 17502645, Quantity: 1}, -1, false)
		resp.Concat(*itemData)

		data, _ := c.GetStats()
		resp.Concat(data)

	case WATER_SPIRIT:
		buff, err := FindBuffByCharacter(10098, c.ID)
		if err != nil {
			return nil, err
		} else if buff != nil {
			return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
		}

		buff, err = FindBuffByCharacter(10100, c.ID) // check for fire spirit
		if err != nil {
			return nil, err
		} else if buff != nil {
			return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
		}

		buff, err = c.CraftBuff(info.Name, 10098, 0, 2592000, true)
		if err != nil {
			return nil, err
		}
		err = buff.Create()
		if err != nil {
			return nil, err
		}

		itemData, _, _ := c.AddItem(&InventorySlot{ItemID: 17502646, Quantity: 1}, -1, false)
		resp.Concat(*itemData)

		data, _ := c.GetStats()
		resp.Concat(data)

	case FORTUNE_BOX_TYPE:

		c.InvMutex.Lock()
		defer c.InvMutex.Unlock()

		gambling := GamblingItems[int(item.ItemID)]
		if gambling == nil || gambling.Cost > c.Gold { // FIX Gambling null
			resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x08, 0xF9, 0x03, 0x55, 0xAA} // not enough gold
			return resp, nil
		}

		slots, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}

		c.LootGold(-gambling.Cost)
		resp.Concat(c.GetGold())

		drop, ok := Drops[gambling.DropID]
		if drop == nil || !ok {
			goto FALLBACK
		}

		var itemID int
		for ok {
			index := 0
			seed := int(utils.RandInt(0, 1000))
			items := drop.GetItems()
			probabilities := drop.GetProbabilities()

			for _, prob := range probabilities {
				if float64(seed) > float64(prob) {
					index++
					continue
				}
				break
			}

			if index >= len(items) {
				break
			}

			itemID = items[index]
			drop, ok = Drops[itemID]
		}

		plus, quantity, upgs := uint8(0), uint(1), []byte{}
		rewardInfo := Items[int64(itemID)]
		if rewardInfo != nil {
			if rewardInfo.ID == 235 || rewardInfo.ID == 242 || rewardInfo.ID == 254 || rewardInfo.ID == 255 { // Socket-PP-Ghost Dagger-Dragon Scale
				var rates []int
				if rewardInfo.ID == 235 { // Socket
					rates = []int{300, 550, 750, 900, 1000}
				} else {
					rates = []int{500, 900, 950, 975, 990, 995, 998, 100}
				}

				seed := int(utils.RandInt(0, 1000))
				for ; seed > rates[plus]; plus++ {
				}
				plus++

				upgs = utils.CreateBytes(byte(rewardInfo.ID), int(plus), 15)

			} else if rewardInfo.GetType() == MARBLE_TYPE { // Marble
				rates := []int{200, 300, 500, 750, 950, 1000}
				seed := int(utils.RandInt(0, 1000))
				for i := 0; seed > rates[i]; i++ {
					itemID++
				}

				rewardInfo = Items[int64(itemID)]

			} else if funk.Contains(haxBoxes, item.ItemID) { // Hax Box
				seed := utils.RandInt(0, 1000)
				plus = uint8(sort.SearchInts(plusRates, int(seed)) + 1)

				upgradesArray := []byte{}
				rewardType := rewardInfo.GetType()
				if rewardType == WEAPON_TYPE {
					upgradesArray = WeaponUpgrades
				} else if rewardType == ARMOR_TYPE {
					upgradesArray = ArmorUpgrades
				} else if rewardType == ACC_TYPE {
					upgradesArray = AccUpgrades
				}

				index := utils.RandInt(0, int64(len(upgradesArray)))
				code := upgradesArray[index]
				if (code-1)%5 == 3 {
					code--
				} else if (code-1)%5 == 4 {
					code -= 2
				}

				upgs = utils.CreateBytes(byte(code), int(plus), 15)
			}

			if q, ok := rewardCounts[item.ItemID]; ok {
				quantity = q
			}

			if box, ok := rewardCounts2[item.ItemID]; ok {
				if q, ok := box[rewardInfo.ID]; ok {
					quantity = q
				}
			}
			if rewardInfo.Timer != 0 {
				quantity = uint(rewardInfo.Timer)
			}
			item := &InventorySlot{ItemID: rewardInfo.ID, Plus: uint8(plus), Quantity: quantity}
			item.SetUpgrades(upgs)

			if rewardInfo.GetType() == PET_TYPE {
				petInfo := Pets[int64(rewardInfo.ID)]
				petExpInfo := PetExps[int16(petInfo.Level)]

				targetExps := []int{petExpInfo.ReqExpEvo1, petExpInfo.ReqExpEvo2, petExpInfo.ReqExpEvo3, petExpInfo.ReqExpHt, petExpInfo.ReqExpDivEvo1, petExpInfo.ReqExpDivEvo2, petExpInfo.ReqExpDivEvo3}
				item.Pet = &PetSlot{
					Fullness: 100, Loyalty: 100,
					Exp:   uint64(targetExps[petInfo.Evolution-1]),
					HP:    petInfo.BaseHP,
					Level: byte(petInfo.Level),
					Name:  "",
					CHI:   petInfo.BaseChi,
				}
			}

			_, slot, err := c.AddItem(item, -1, true)
			if err != nil {
				return nil, err
			}

			resp.Concat(slots[slot].GetData(slot))
		}

	case NPC_SUMMONER_TYPE:
		if item.ItemID == 17502966 || item.ItemID == 17100004 { // Tavern
			r := utils.Packet{0xAA, 0x55, 0x07, 0x00, 0x57, 0x03, 0x01, 0x06, 0x00, 0x00, 0x00, 0x55, 0xAA}
			resp.Concat(r)
		} else if item.ItemID == 17502967 || item.ItemID == 17100005 { // Bank
			resp.Concat(c.BankItems())
		}
	case TRANSFORMATION_SCROLL_TYPE:
		if item.ItemID == 15830000 || item.ItemID == 15830001 || item.ItemID == 17502883 {
			r := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x01, 0xB4, 0x0A, 0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xB4, 0x55, 0xAA}
			resp.Concat(r)
		}
	case PASSIVE_SKILL_BOOK_TYPE:
		if info.CharacterType > 0 && !canUse { // invalid character type
			return INVALID_CHARACTER_TYPE, nil
		}

		skills, err := FindSkillsByID(c.ID)
		if err != nil {
			return nil, err
		}

		skillSlots, err := skills.GetSkills()
		if err != nil {
			return nil, err
		}

		i := -1
		if info.Name == "Air Slide Arts" || info.Name == "Wind Drift Arts" {
			i = 7
			if skillSlots.Slots[i].BookID > 0 {
				return SKILL_BOOK_EXISTS, nil
			}

		} else {
			for j := 5; j < 7; j++ {
				if skillSlots.Slots[j].BookID == 0 {
					i = j
					break
				} else if skillSlots.Slots[j].BookID == item.ItemID { // skill book exists
					return SKILL_BOOK_EXISTS, nil
				}
			}
		}

		if i == -1 {
			return NO_SLOTS_FOR_SKILL_BOOK, nil // FIX resp
		}

		set := &SkillSet{BookID: item.ItemID}
		set.Skills = append(set.Skills, &SkillTuple{SkillID: int(info.ID), Plus: 0})
		skillSlots.Slots[i] = set
		skills.SetSkills(skillSlots)

		go skills.Update()

		skillsData, err := skills.GetSkillsData()
		if err != nil {
			return nil, err
		}

		resp.Concat(skillsData)

	case PET_POTION_TYPE:
		slots, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}

		petSlot := slots[0x0A]
		pet := petSlot.Pet

		if pet == nil || !pet.IsOnline {
			goto FALLBACK
		}

		pet.HP = int(math.Min(float64(pet.HP+info.HpRecovery), float64(pet.MaxHP)))
		pet.CHI = int(math.Min(float64(pet.CHI+info.ChiRecovery), float64(pet.MaxCHI)))
		pet.Fullness = byte(math.Min(float64(pet.Fullness+5), float64(100)))
		resp.Concat(c.GetPetStats())

	case POTION_TYPE:
		if item.ItemID == 100080008 { // Eye of Divine
			buff, err := FindBuffByCharacter(105, c.ID)
			if err != nil {
				return nil, err
			} else if buff != nil {
				return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
			}

			buff = &Buff{ID: 105, CharacterID: c.ID, Name: info.Name, StartedAt: c.Epoch, Duration: 20, CanExpire: true}
			err = buff.Create()
			if err != nil {
				return nil, err
			}

			data, _ := c.GetStats()
			resp.Concat(data)

			c.DetectionMode = true

			goto REMOVEITEM
		}
		if c.UsedPotion {
			return item.GetData(slotID), nil
		}
		hpRec := info.HpRecovery
		chiRec := info.ChiRecovery
		if hpRec == 0 && chiRec == 0 {
			hpRec = 50000
			chiRec = 50000
		}

		stat.HP = int(math.Min(float64(stat.HP+hpRec), float64(stat.MaxHP)))
		stat.CHI = int(math.Min(float64(stat.CHI+chiRec), float64(stat.MaxCHI)))
		resp.Concat(c.GetHPandChi())
		c.UsedPotion = true
		time.AfterFunc(time.Second*2, func() {
			c.UsedPotion = false
		})
		//POTIDELAY := utils.Packet{0xaa, 0x55, 0xe4, 0x00, 0x14, 0x33, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0c, 0xf2, 0x07, 0x08, 0x00, 0x00, 0x00, 0x00, 0x20, 0x1c, 0x00, 0x00, 0x00, 0xc4, 0xd3, 0xeb, 0x08, 0x00, 0x00, 0x00, 0x00, 0x23, 0x00, 0x4d, 0x00, 0x0d, 0x00, 0x37, 0x00, 0x28, 0x01, 0x48, 0x01, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x41, 0x66, 0x66, 0x46, 0x41, 0xf4, 0x03, 0x00, 0x00, 0x00, 0x40, 0xde, 0x19, 0x00, 0x00, 0x10, 0x08, 0x00, 0x00, 0x04, 0x02, 0xcf, 0x02, 0x0b, 0x04, 0x00, 0x00, 0x0b, 0x04, 0x00, 0x00, 0x0b, 0x04, 0x00, 0x00, 0x67, 0x0a, 0x00, 0x00, 0x7f, 0x0e, 0x00, 0x00, 0xa8, 0x08, 0x00, 0x00, 0x11, 0x01, 0x4f, 0x01, 0x05, 0x00, 0x00, 0x00, 0x82, 0x00, 0x00, 0x00, 0x00, 0x50, 0x00, 0x00, 0x00, 0x00, 0x8c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b, 0x04, 0x00, 0x00, 0xa8, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0x00, 0x30, 0x30, 0x31, 0x2d, 0x30, 0x31, 0x2d, 0x30, 0x31, 0x20, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x00, 0x64, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x74, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0x3f, 0x00, 0x00, 0x80, 0x3f, 0x00, 0x00, 0x55, 0xaa}
		//resp.Concat(POTIDELAY)
	case FILLER_POTION_TYPE:
		hpRecovery, chiRecovery := math.Min(float64(stat.MaxHP-stat.HP), 50000), float64(0)
		if hpRecovery > float64(item.Quantity) {
			hpRecovery = float64(item.Quantity)
		} else {
			chiRecovery = math.Min(float64(stat.MaxCHI-stat.CHI), 50000)
			if chiRecovery+hpRecovery > float64(item.Quantity) {
				chiRecovery = float64(item.Quantity) - hpRecovery
			}
		}

		stat.HP = int(math.Min(float64(stat.HP)+hpRecovery, float64(stat.MaxHP)))
		stat.CHI = int(math.Min(float64(stat.CHI)+chiRecovery, float64(stat.MaxCHI)))
		resp.Concat(c.GetHPandChi())
		resp.Concat(*c.DecrementItem(slotID, uint(hpRecovery+chiRecovery)))
		resp.Concat(item.GetData(slotID))
		return resp, nil

	case SKILL_BOOK_TYPE:
		if info.CharacterType > 0 && !canUse { // invalid character type
			return INVALID_CHARACTER_TYPE, nil
		}

		skills, err := FindSkillsByID(c.ID)
		if err != nil {
			return nil, err
		}

		skillSlots, err := skills.GetSkills()
		if err != nil {
			return nil, err
		}

		i := -1
		for j := 0; j < 5; j++ {
			if skillSlots.Slots[j].BookID == 0 {
				i = j
				break
			} else if skillSlots.Slots[j].BookID == item.ItemID { // skill book exists
				return SKILL_BOOK_EXISTS, nil
			}
		}

		if i == -1 {
			return NO_SLOTS_FOR_SKILL_BOOK, nil // FIX resp
		}

		skillInfos := SkillInfosByBook[item.ItemID]
		set := &SkillSet{BookID: item.ItemID}
		c := 0
		for i := 1; i <= 24; i++ { // there should be 24 skills with empty ones
			if len(skillInfos) <= c {
				set.Skills = append(set.Skills, &SkillTuple{SkillID: 0, Plus: 0})
			} else if si := skillInfos[c]; si.Slot == i {
				tuple := &SkillTuple{SkillID: si.ID, Plus: 0}
				set.Skills = append(set.Skills, tuple)
				c++
			} else {
				set.Skills = append(set.Skills, &SkillTuple{SkillID: 0, Plus: 0})
			}
		}
		if info.MinLevel < 100 {
			divtuple := &DivineTuple{DivineID: 0, DivinePlus: 0}
			div2tuple := &DivineTuple{DivineID: 1, DivinePlus: 0}
			div3tuple := &DivineTuple{DivineID: 2, DivinePlus: 0}
			set.DivinePoints = append(set.DivinePoints, divtuple, div2tuple, div3tuple)
		}
		skillSlots.Slots[i] = set
		skills.SetSkills(skillSlots)
		go skills.Update()

		skillsData, err := skills.GetSkillsData()
		if err != nil {
			fmt.Println("SkillError: %s", err.Error())
			return nil, err
		}
		resp.Concat(skillsData)

	case WRAPPER_BOX_TYPE:

		c.InvMutex.Lock()
		defer c.InvMutex.Unlock()
		if item.ItemID == 90000304 {
			if c.Exp >= 544951059310 && c.Level == 200 {
				c.Type += 10
				c.Update()
				c.Socket.Skills.Delete()
				c.Socket.Skills.SkillPoints = 28000
				c.Socket.Skills.Create(c)
				c.Update()
				data, levelUp := c.AddExp(10)
				if levelUp {
					skillsData, err := c.Socket.Skills.GetSkillsData()
					resp.Concat(skillsData)
					if err == nil && c.Socket != nil {
						c.Socket.Write(skillsData)
					}
					statData, err := c.GetStats()
					if err == nil && c.Socket != nil {
						c.Socket.Write(statData)
					}
				}
				if c.Socket != nil {
					c.Socket.Write(data)
				}
				goto DARKBACK
			}
			goto FALLBACK
		}
		gambling := GamblingItems[int(item.ItemID)]
		if gambling == nil {
			goto FALLBACK
		}
		d := Drops[gambling.DropID]
		items := d.GetItems()
		_, err := c.FindFreeSlots(len(items))
		if err != nil {
			goto FALLBACK
		}

		slots, err := c.InventorySlots()
		if err != nil {
			goto FALLBACK
		}
		for _, itemID := range items {
			if itemID == 0 {
				continue
			}
			info := Items[int64(itemID)]
			reward := NewSlot()
			reward.ItemID = int64(itemID)
			if boxQuantity > 5000 {
				boxQuantity = 5000
			}
			reward.Quantity = uint(boxQuantity)
			plus, upgs := uint8(0), []byte{}

			if info.ID == 235 || info.ID == 242 || info.ID == 254 || info.ID == 255 { // Socket-PP-Ghost Dagger-Dragon Scale
				reward.Quantity = 1
				var rewardCount []int
				var rates []int
				if info.ID == 235 { // Socket
					rates = []int{300, 550, 750, 900, 1000}
					rewardCount = []int{0, 0, 0, 0, 0}
				} else {
					rates = []int{500, 900, 950, 975, 990, 995, 998, 100}
					rewardCount = []int{0, 0, 0, 0, 0, 0, 0, 0}
				}

				for i := 0; i < int(boxQuantity); i++ {
					plus = 0
					seed := int(utils.RandInt(0, 1000))
					for ; seed > rates[plus]; plus++ {
					}
					rewardCount[plus]++
				}

				plus = 1
				for i := 0; i < len(rewardCount); {
					if rewardCount[i] != 0 {
						reward.Quantity = uint(rewardCount[i])
						upgs = utils.CreateBytes(byte(info.ID), int(plus), 15)
						reward.Plus = plus
						reward.SetUpgrades(upgs)
						_, slot, _ := c.AddItem(reward, -1, true)
						resp.Concat(slots[slot].GetData(slot))
					}
					plus++
					i++
				}
				goto REMOVEBOXES
			}

			itemType := info.GetType()
			if info.Timer > 0 && itemType != BAG_EXPANSION_TYPE {
				reward.Quantity = uint(info.Timer)
			} else if q, ok := rewardCounts[item.ItemID]; ok {
				reward.Quantity = q
			} else if itemType == FILLER_POTION_TYPE {
				reward.Quantity = uint(info.SellPrice)
			}

			reward.Plus = plus
			reward.SetUpgrades(upgs)

			_, slot, _ := c.AddItem(reward, -1, true)
			resp.Concat(slots[slot].GetData(slot))

		}
		goto REMOVEBOXES // Make sure to remove the correct quanitity

	case HOLY_WATER_TYPE:
		goto FALLBACK
	case MOB_SUMMONING_SCROLL:
		//goto FALLBACK
	case FORM_TYPE:

		info, ok := Items[int64(item.ItemID)]
		if !ok || item.Activated != c.Morphed || FindPlayerInArena(c) {
			goto FALLBACK
		}

		item.Activated = !item.Activated
		item.InUse = !item.InUse
		c.Morphed = item.Activated
		c.MorphedNPCID = info.NPCID
		if item.Activated {
			r := FORM_ACTIVATED
			r.Insert(utils.IntToBytes(uint64(info.NPCID), 4, true), 5) // form npc id
			resp.Concat(r)
			characters, err := c.GetNearbyCharacters()
			if err != nil {
				log.Println(err)
				//return
			}
			//test := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x21, 0x02, 0x00, 0x55, 0xAA}
			for _, chars := range characters {
				delete(chars.OnSight.Players, c.ID)
			}
		} else {
			c.MorphedNPCID = 0
			resp.Concat(FORM_DEACTIVATED)
			characters, err := c.GetNearbyCharacters()
			if err != nil {
				log.Println(err)
				//return
			}
			//test := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x21, 0x02, 0x00, 0x55, 0xAA}
			for _, chars := range characters {
				delete(chars.OnSight.Players, c.ID)
			}
		}

		go item.Update()

		statData, err := c.GetStats()
		if err != nil {
			return nil, err
		}

		resp.Concat(statData)
		goto FALLBACK
	case STORAGE_EXPANSION_TYPE:
		expbag := BANK_EXPANDED
		expiration := "UNLIMITED"
		expbag.Overwrite([]byte(expiration), 7) // bag expiration
		resp.Concat(expbag)
		goto FALLBACK
	default:
		if item.ItemID == 80006001 || item.ItemID == 80006002 {
			if c.GuildID != -1 {
				c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("First quit from your guild!")))
				goto FALLBACK
			}
			r := []byte{0xaa, 0x55, 0x05, 0x00, 0x2f, 0xff, 0x01, 0x00, 0x00, 0x55, 0xaa}
			if c.Faction == 1 {
				c.Faction = 2
				r[6] = 0x02
			} else {
				c.Faction = 1
				r[7] = 0x02
			}
			c.Update()
			characters, err := c.GetNearbyCharacters()
			if err != nil {
				log.Println(err)
			}
			//test := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x21, 0x02, 0x00, 0x55, 0xAA}
			for _, chars := range characters {
				delete(chars.OnSight.Players, c.ID)
			}
			chars, err := FindCharactersByUserID(c.UserID)
			if err == nil {
				for _, char := range chars {
					char.Faction = c.Faction
					char.GuildID = -1
					char.Update()
				}
				resp.Concat(r)
				break
			}
		}
		if info.Timer > 0 {
			item.Activated = !item.Activated
			item.InUse = !item.InUse
			resp.Concat(item.GetData(slotID))

			statsData, _ := c.GetStats()
			resp.Concat(statsData)
			goto FALLBACK
		} else {
			goto FALLBACK
		}
	}

	resp.Concat(*c.DecrementItem(slotID, 1))
	return resp, nil

REMOVEBOXES:
	resp.Concat(*c.DecrementItem(slotID, boxQuantity))
	return resp, nil

REMOVEITEM:
	resp.Concat(*c.DecrementItem(slotID, 1))
	return resp, nil

FALLBACK:
	resp.Concat(*c.DecrementItem(slotID, 0))
	return resp, nil

DARKBACK:
	resp.Concat(*c.DecrementItem(item.SlotID, 1))
	c.Socket.Write(resp)
	charmenu := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x09, 0x09, 0x00, 0x55, 0xAA} //Select Character
	return charmenu, nil
}

func (c *Character) CanUse(t int) bool {
	if c.Type == 0x32 && (t == 0x32 || t == 0x01 || t == 0x03) { // MALE BEAST
		return true
	} else if c.Type == 0x33 && (t == 0x33 || t == 0x02 || t == 0x03) { // FEMALE BEAST
		return true
	} else if c.Type == 0x34 && (t == 0x34 || t == 0x01) { // Monk
		return true
	} else if c.Type == 0x35 && (t == 0x35 || t == 0x37 || t == 0x01) { //MALE_BLADE
		return true
	} else if c.Type == 0x36 && (t == 0x36 || t == 0x37 || t == 0x02) { //FEMALE_BLADE
		return true
	} else if c.Type == 0x38 && (t == 0x38 || t == 0x3A || t == 0x01) { //AXE
		return true
	} else if c.Type == 0x39 && (t == 0x39 || t == 0x3A || t == 0x02) { //FEMALE_ROD
		return true
	} else if c.Type == 0x3B && (t == 0x3B || t == 0x02) { //DUAL_BLADE
		return true
	} else if c.Type == 0x3C && (t == 0x3C || t == 0x01 || t == 0x03 || t == 0x0A) { // DIVINE MALE BEAST
		return true
	} else if c.Type == 0x3D && (t == 0x3D || t == 0x02 || t == 0x03 || t == 0x0A) { // DIVINE FEMALE BEAST
		return true
	} else if c.Type == 0x3E && (t == 0x3E || t == 0x01 || t == 0x34 || t == 0x0A) { //DIVINE MONK
		return true
	} else if c.Type == 0x3F && (t == 0x3F || t == 0x41 || t == 0x01 || t == 0x35 || t == 0x37 || t == 0x0A) { //DIVINE MALE_BLADE
		return true
	} else if c.Type == 0x40 && (t == 0x40 || t == 0x41 || t == 0x02 || t == 0x36 || t == 0x37 || t == 0x0A) { //DIVINE FEMALE_BLADE
		return true
	} else if c.Type == 0x42 && (t == 0x42 || t == 0x44 || t == 0x01 || t == 0x38 || t == 0x3A || t == 0x0A) { //DIVINE MALE_AXE
		return true
	} else if c.Type == 0x43 && (t == 0x43 || t == 0x44 || t == 0x02 || t == 0x39 || t == 0x3A || t == 0x0A) { //DIVINE FEMALE_ROD
		return true
	} else if c.Type == 0x45 && (t == 0x45 || t == 0x02 || t == 0x3B || t == 0x0A) { //DIVINE Dual Sword
		return true
	} else if c.Type == 0x46 && (t == 0x46 || t == 0x01 || t == 0x03 || t == 0x0A) { // DARK LORD MALE BEAST
		return true
	} else if c.Type == 0x47 && (t == 0x47 || t == 0x02 || t == 0x03 || t == 0x0A) { // DARK LORD FEMALE BEAST
		return true
	} else if c.Type == 0x48 && (t == 0x48 || t == 0x01 || t == 0x3E || t == 0x34 || t == 0x14) { //DARK LORD MONK
		return true
	} else if c.Type == 0x49 && (t == 0x49 || t == 0x4B || t == 0x01 || t == 0x35 || t == 0x37 || t == 0x41 || t == 0x3F || t == 0x14) { //DARK LORD MALE_BLADE
		return true
	} else if c.Type == 0x4A && (t == 0x4A || t == 0x4B || t == 0x02 || t == 0x36 || t == 0x37 || t == 0x40 || t == 0x41 || t == 0x14) { //DARK LORD FEMALE_BLADE
		return true
	} else if c.Type == 0x4C && (t == 0x4C || t == 0x4E || t == 0x01 || t == 0x38 || t == 0x3A || t == 0x42 || t == 0x44 || t == 0x14) { //DARK LORD MALE_AXE
		return true
	} else if c.Type == 0x4D && (t == 0x4D || t == 0x4E || t == 0x02 || t == 0x39 || t == 0x3A || t == 0x43 || t == 0x44 || t == 0x14) { //DARK LORD FEMALE_ROD
		return true
	} else if c.Type == 0x4F && (t == 0x4F || t == 0x02 || t == 0x45 || t == 0x3B) { //DARK LORD Dual Sword
		return true
	} else if t == 0x00 || t == 0x20 { //All character Type
		return true
	}

	return false
}
func (c *Character) AidBuffHandle() ([]byte, error) {
	buff, err := FindBuffByCharacter(int(11152), c.ID)
	if err != nil {
		return nil, err
	}
	if !c.AidMode {
		buff.Delete()
		//return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
	} else {
		infection := BuffInfections[11152]
		buff = &Buff{ID: int(11152), CharacterID: c.ID, Name: infection.Name, StartedAt: c.Epoch, Duration: int64(99999), SkillPlus: 1}
		err = buff.Create()
		if err != nil {
			fmt.Println("Error: 3")
			return nil, err
		}
	}

	return nil, nil
}
func (c *Character) UpgradeSkill(slotIndex, skillIndex byte) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[slotIndex]
	skill := set.Skills[skillIndex]

	info := SkillInfos[skill.SkillID]
	if int8(skill.Plus) >= info.MaxPlus {
		return nil, nil
	}

	requiredSP := 1
	if info.ID >= 28000 && info.ID <= 28007 { // 2nd job passives (non-divine)
		requiredSP = SkillPTS["sjp"][skill.Plus]
	} else if info.ID >= 29000 && info.ID <= 29007 { // 2nd job passives (divine)
		requiredSP = SkillPTS["dsjp"][skill.Plus]
	} else if info.ID >= 20193 && info.ID <= 30217 { // 3nd job passives (darkness)
		requiredSP = SkillPTS["dsjp"][skill.Plus]
	}

	if skills.SkillPoints < requiredSP {
		return nil, nil
	}

	skills.SkillPoints -= requiredSP
	skill.Plus++
	resp := SKILL_UPGRADED
	resp[8] = slotIndex
	resp[9] = skillIndex
	resp.Insert(utils.IntToBytes(uint64(skill.SkillID), 4, true), 10) // skill id
	resp[14] = byte(skill.Plus)

	skills.SetSkills(skillSlots)
	skills.Update()

	if info.Passive {
		statData, err := c.GetStats()
		if err == nil {
			resp.Concat(statData)
		}
	}

	resp.Concat(c.GetExpAndSkillPts())
	return resp, nil
}

func (c *Character) DowngradeSkill(slotIndex, skillIndex byte) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[slotIndex]
	skill := set.Skills[skillIndex]

	info := SkillInfos[skill.SkillID]
	if int8(skill.Plus) <= 0 {
		return nil, nil
	}

	requiredSP := 1
	if info.ID >= 28000 && info.ID <= 28007 && skill.Plus > 0 { // 2nd job passives (non-divine)
		requiredSP = SkillPTS["sjp"][skill.Plus-1]
	} else if info.ID >= 29000 && info.ID <= 29007 && skill.Plus > 0 { // 2nd job passives (divine)
		requiredSP = SkillPTS["dsjp"][skill.Plus-1]
	} else if info.ID >= 20193 && info.ID <= 20217 { // 3nd job passives (darkness)
		requiredSP = SkillPTS["dsjp"][skill.Plus]
	}

	skills.SkillPoints += requiredSP
	skill.Plus--
	resp := SKILL_DOWNGRADED
	resp[8] = slotIndex
	resp[9] = skillIndex
	resp.Insert(utils.IntToBytes(uint64(skill.SkillID), 4, true), 10) // skill id
	resp[14] = byte(skill.Plus)
	resp.Insert([]byte{0, 0, 0}, 15) //

	skills.SetSkills(skillSlots)
	skills.Update()

	if info.Passive {
		statData, err := c.GetStats()
		if err == nil {
			resp.Concat(statData)
		}
	}

	resp.Concat(c.GetExpAndSkillPts())
	return resp, nil
}
func (c *Character) DivineUpgradeSkills(skillIndex, slot int, bookID int64) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}
	resp := utils.Packet{}
	//divineID := 0
	bonusPlus := 0
	usedPoints := 0
	for _, skill := range skillSlots.Slots {
		if skill.BookID == bookID {
			if len(skill.DivinePoints) == 0 {
				divtuple := &DivineTuple{DivineID: 0, DivinePlus: 0}
				div2tuple := &DivineTuple{DivineID: 1, DivinePlus: 0}
				div3tuple := &DivineTuple{DivineID: 2, DivinePlus: 0}
				skill.DivinePoints = append(skill.DivinePoints, divtuple, div2tuple, div3tuple)
				skills.SetSkills(skillSlots)
				skills.Update()
			}
			for _, point := range skill.DivinePoints {
				usedPoints += point.DivinePlus
				//if point.DivineID == slot {
				if usedPoints >= 10 {
					return nil, nil
				}
				//	divineID = point.DivineID
				if point.DivineID == slot {
					bonusPlus = point.DivinePlus
				}
			}
			skill.DivinePoints[slot].DivinePlus++
		}
	}
	bonusPlus++
	resp = DIVINE_SKILL_BOOk
	resp[8] = byte(skillIndex)
	index := 9
	resp.Insert([]byte{byte(slot)}, index) // divine id
	index++
	resp.Insert(utils.IntToBytes(uint64(bookID), 4, true), index) // book id
	index += 4
	resp.Insert([]byte{byte(bonusPlus)}, index) // divine plus
	index++
	skills.SetSkills(skillSlots)
	skills.Update()
	return resp, nil
}

func (c *Character) RemoveSkill(slotIndex byte, bookID int64) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[slotIndex]
	if set.BookID != bookID {
		return nil, fmt.Errorf("RemoveSkill: skill book not found")
	}

	skillSlots.Slots[slotIndex] = &SkillSet{}
	skills.SetSkills(skillSlots)
	skills.Update()

	resp := SKILL_REMOVED
	resp[8] = slotIndex
	resp.Insert(utils.IntToBytes(uint64(bookID), 4, true), 9) // book id

	return resp, nil
}

func (c *Character) UpgradePassiveSkill(slotIndex, skillIndex byte) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[skillIndex]
	if len(set.Skills) == 0 || set.Skills[0].Plus >= 12 {
		return nil, nil
	}

	if skillIndex == 5 || skillIndex == 6 { // 1st job skill
		requiredSP := SkillPTS["fjp"][set.Skills[0].Plus]
		if skills.SkillPoints < requiredSP {
			return nil, nil
		}

		skills.SkillPoints -= requiredSP

	} else if skillIndex == 7 { // running
		requiredSP := SkillPTS["wd"][set.Skills[0].Plus]
		if skills.SkillPoints < requiredSP {
			return nil, nil
		}

		skills.SkillPoints -= requiredSP
		c.RunningSpeed = 10.0 + (float64(set.Skills[0].Plus) * 0.2)
	}

	set.Skills[0].Plus++

	skills.SetSkills(skillSlots)
	skills.Update()

	resp := PASSIVE_SKILL_UGRADED
	resp[8] = slotIndex
	resp[9] = byte(set.Skills[0].Plus)

	statData, err := c.GetStats()
	if err != nil {
		return nil, err
	}

	resp.Concat(statData)
	resp.Concat(c.GetExpAndSkillPts())
	return resp, nil
}

func (c *Character) DowngradePassiveSkill(slotIndex, skillIndex byte) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[skillIndex]
	if len(set.Skills) == 0 || set.Skills[0].Plus <= 0 {
		return nil, nil
	}

	if skillIndex == 5 && set.Skills[0].Plus > 0 { // 1st job skill
		//requiredSP := SkillPTS["fjp"][set.Skills[0].Plus-1]

		//skills.SkillPoints += requiredSP

	} else if skillIndex == 7 && set.Skills[0].Plus > 0 { // running
		//requiredSP := SkillPTS["wd"][set.Skills[0].Plus]

		//skills.SkillPoints += requiredSP
		c.RunningSpeed = 10.0 + (float64(set.Skills[0].Plus-1) * 0.2)
	}

	set.Skills[0].Plus--

	skills.SetSkills(skillSlots)
	skills.Update()

	resp := PASSIVE_SKILL_UGRADED
	resp[8] = slotIndex
	resp[9] = byte(set.Skills[0].Plus)

	statData, err := c.GetStats()
	if err != nil {
		return nil, err
	}

	resp.Concat(statData)
	resp.Concat(c.GetExpAndSkillPts())
	return resp, nil
}

func (c *Character) RemovePassiveSkill(slotIndex, skillIndex byte, bookID int64) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[skillIndex]
	fmt.Println(set, bookID)
	if set.BookID != bookID {
		return nil, fmt.Errorf("RemovePassiveSkill: skill book not found")
	}

	skillSlots.Slots[skillIndex] = &SkillSet{}
	skills.SetSkills(skillSlots)
	skills.Update()

	resp := PASSIVE_SKILL_REMOVED
	resp.Insert(utils.IntToBytes(uint64(bookID), 4, true), 8) // book id
	resp[12] = slotIndex

	return resp, nil
}

func (c *Character) CastSkill(attackCounter, skillID, targetID int, cX, cY, cZ float64) ([]byte, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}
	petSlot := slots[0x0A]
	pet := petSlot.Pet
	petInfo, ok := Pets[petSlot.ItemID]
	if pet != nil && ok && pet.IsOnline && !petInfo.Combat {
		return nil, nil
	}

	stat := c.Socket.Stats
	user := c.Socket.User
	skills := c.Socket.Skills

	canCast := false
	skillInfo := SkillInfos[skillID]
	weapon := slots[c.WeaponSlot]
	if weapon.ItemID == 0 { // there are some skills which can be casted without weapon such as monk skills
		if c.Type == MONK || c.Type == DIVINE_MONK || c.Type == DARKNESS_MONK {
			canCast = true
		}
	} else if c.Type == BEAST_KING || c.Type == DIVINE_BEAST_KING || c.Type == DARKNESS_BEAST_KING || c.Type == EMPRESS || c.Type == DIVINE_EMPRESS || c.Type == DARKNESS_EMPRESS {
		canCast = true
	} else {
		weaponInfo := Items[weapon.ItemID]
		canCast = weaponInfo.CanUse(skillInfo.Type)
	}
	if !canCast {
		return nil, nil
	}
	plus, err := skills.GetPlus(skillID)
	if err != nil {
		return nil, err
	}
	skillSlots, err := c.Socket.Skills.GetSkills()
	if err != nil {
		return nil, err
	}
	plusCooldown := 0
	plusChiCost := 0
	divinePlus := 0
	for _, slot := range skillSlots.Slots {
		if slot.BookID == skillInfo.BookID {
			for _, points := range slot.DivinePoints {
				if points.DivineID == 0 && points.DivinePlus > 0 {
					divinePlus = points.DivinePlus
					plusChiCost = 50
				}
				if points.DivinePlus == 2 && points.DivinePlus > 0 {
					plusCooldown = 100
				}
			}
		}
	}
	t := c.SkillHistory.Get(skillID)
	if t != nil {
		castedAt := t.(time.Time)
		cooldown := time.Duration(skillInfo.Cooldown*100) * time.Millisecond
		cooldown -= time.Duration(plusCooldown * divinePlus) //plusCooldown * divinePlus
		if time.Now().Sub(castedAt) < cooldown {
			return nil, nil
		}
	}
	c.SkillHistory.Add(skillID, time.Now())
	if pet != nil {
		petSkill, _ := utils.Contains(petInfo.GetSkills(), skillID)
		if petSkill {
			if pet.CHI >= skillInfo.BaseChi && skillID != 0 && pet.IsOnline && skillInfo.InfectionID == 0 {
				pet.Target = c.Selection
				r := SKILL_CASTED
				index := 7
				r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // character pseudo id
				index += 2
				r[index] = byte(attackCounter)
				index++
				r.Insert(utils.IntToBytes(uint64(skillID), 4, true), index) // skill id
				index += 4
				r.Insert(utils.FloatToBytes(cX, 4, true), index) // coordinate-x
				index += 4
				r.Insert(utils.FloatToBytes(cY, 4, true), index) // coordinate-y
				index += 4
				r.Insert(utils.FloatToBytes(cZ, 4, true), index) // coordinate-z
				index += 5
				r.Insert(utils.IntToBytes(uint64(0), 2, true), index) // target id
				index += 3
				r.Insert(utils.IntToBytes(uint64(0), 2, true), index) // target id
				c.Socket.Write(r)
				return pet.CastSkill(c, skillID), nil
			} else if pet.CHI >= skillInfo.BaseChi && skillID != 0 && pet.IsOnline && skillInfo.InfectionID != 0 {
				pet.Target = c.Selection
				r := SKILL_CASTED
				index := 7
				r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // character pseudo id
				index += 2
				r[index] = byte(attackCounter)
				index++
				r.Insert(utils.IntToBytes(uint64(skillID), 4, true), index) // skill id
				index += 4
				r.Insert(utils.FloatToBytes(cX, 4, true), index) // coordinate-x
				index += 4
				r.Insert(utils.FloatToBytes(cY, 4, true), index) // coordinate-y
				index += 4
				r.Insert(utils.FloatToBytes(cZ, 4, true), index) // coordinate-z
				index += 5
				r.Insert(utils.IntToBytes(uint64(0), 2, true), index) // target id
				index += 3
				r.Insert(utils.IntToBytes(uint64(0), 2, true), index) // target id
				c.Socket.Write(r)
				return pet.CastSkillToPlayer(c, skillID), nil
			} else {
				return nil, nil
			}

		}
	}
	addChiCost := float64(skillInfo.AdditionalChi*int(plus)) * 2.2 / 3 // some bad words here
	chiCost := skillInfo.BaseChi + int(addChiCost) - (plusChiCost * divinePlus)
	if stat.CHI < chiCost {
		return nil, nil
	}

	stat.CHI -= chiCost
	resp := utils.Packet{}
	if target := skillInfo.Target; target == 0 || target == 2 { // buff skill
		character := c
		if target == 2 {
			ch := FindCharacterByPseudoID(c.Socket.User.ConnectedServer, uint16(c.Selection))
			if ch != nil {
				character = ch
			}
		}

		/*petSlot := slots[0x0A]
		pet := petSlot.Pet
		if pet == nil || petSlot.ItemID == 0 || !pet.IsOnline {

		}*/

		if skillInfo.InfectionID == 0 {
			goto COMBAT
		}
		infection := BuffInfections[skillInfo.InfectionID]
		duration := (skillInfo.BaseTime + skillInfo.AdditionalTime*int(plus)) / 10
		expire := true
		if c.Type == BEAST_KING || c.Type == DIVINE_BEAST_KING || c.Type == DARKNESS_BEAST_KING {
			ok := funk.Contains(Beast_King_Infections, int16(skillInfo.InfectionID))
			if ok {
				for _, buffid := range Beast_King_Infections {
					buff, err := FindBuffByCharacter(int(buffid), character.ID)
					if buff != nil && err == nil {
						buff.Delete()
					}
				}

			}
		}
		if c.Type == EMPRESS || c.Type == DIVINE_EMPRESS || c.Type == DARKNESS_EMPRESS {
			ok := funk.Contains(Empress_Infections, int16(skillInfo.InfectionID))
			if ok {
				for _, buffid := range Empress_Infections {
					buff, err := FindBuffByCharacter(int(buffid), character.ID)
					if buff != nil && err == nil {
						buff.Delete()
					}
				}

			}
		}
		if skillInfo.InfectionID != 0 && duration == 0 {
			expire = false
		}
		buff, err := FindBuffByCharacter(infection.ID, character.ID)
		if err != nil {
			return nil, err
		} else if buff != nil {
			buff.Delete()
			stat, err = FindStatByID(character.ID)
			if err != nil {
				return nil, err
			}

			err = stat.Calculate()
			if err != nil {
				return nil, err
			}
			c.HandleBuffs()
			if !infection.IsPercent {
				buff = &Buff{ID: infection.ID, CharacterID: character.ID, StartedAt: character.Epoch, Duration: int64(duration), Name: skillInfo.Name,
					ATK: infection.BaseATK + infection.AdditionalATK*int(plus), ArtsATK: infection.BaseArtsATK + infection.AdditionalArtsATK*int(plus),
					ArtsDEF: infection.ArtsDEF + infection.AdditionalArtsDEF*int(plus), ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef*int(plus),
					DEF: infection.BaseDef + infection.AdditionalDEF*int(plus), DEX: infection.DEX + infection.AdditionalDEX*int(plus), HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery*int(plus), INT: infection.INT + infection.AdditionalINT*int(plus),
					MaxHP: infection.MaxHP + infection.AdditionalHP*int(plus), ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef*int(plus), PoisonDEF: infection.ParalysisDef + infection.AdditionalPoisonDef*int(plus), STR: infection.STR + infection.AdditionalSTR*int(plus),
					Accuracy: infection.Accuracy + infection.AdditionalAccuracy*int(plus), Dodge: infection.DodgeRate + infection.AdditionalDodgeRate*int(plus), RunningSpeed: infection.MovSpeed + infection.AdditionalMovSpeed*float64(plus), CanExpire: expire}
				buff.Create()
			} else {
				percentArtsDEF := int(float64(character.Socket.Stats.ArtsDEF) * (float64(infection.ArtsDEF+infection.AdditionalArtsDEF*int(plus)) / 1000))
				percentDEF := int(float64(character.Socket.Stats.DEF) * (float64(infection.BaseDef+infection.AdditionalDEF*int(plus)) / 1000))
				percentATK := int(float64(character.Socket.Stats.MinATK) * (float64(infection.BaseATK+infection.AdditionalATK*int(plus)) / 1000))
				percentArtsATK := int(float64(character.Socket.Stats.MinArtsATK) * (float64(infection.BaseArtsATK+infection.AdditionalArtsATK*int(plus)) / 1000))
				buff = &Buff{ID: infection.ID, CharacterID: character.ID, StartedAt: character.Epoch, Duration: int64(duration), Name: skillInfo.Name,
					ATK: percentATK, ArtsATK: percentArtsATK,
					ArtsDEF: percentArtsDEF, ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef*int(plus),
					DEF: percentDEF, DEX: infection.DEX + infection.AdditionalDEX*int(plus), HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery*int(plus), INT: infection.INT + infection.AdditionalINT*int(plus),
					MaxHP: infection.MaxHP + infection.AdditionalHP*int(plus), ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef*int(plus), PoisonDEF: infection.ParalysisDef + infection.AdditionalPoisonDef*int(plus), STR: infection.STR + infection.AdditionalSTR*int(plus),
					Accuracy: infection.Accuracy + infection.AdditionalAccuracy*int(plus), Dodge: infection.DodgeRate + infection.AdditionalDodgeRate*int(plus), CanExpire: expire}
				buff.Create()
			}
			buff.Update()

		} else if buff == nil {
			c.HandleBuffs()
			if !infection.IsPercent {
				buff = &Buff{ID: infection.ID, CharacterID: character.ID, StartedAt: character.Epoch, Duration: int64(duration), Name: skillInfo.Name,
					ATK: infection.BaseATK + infection.AdditionalATK*int(plus), ArtsATK: infection.BaseArtsATK + infection.AdditionalArtsATK*int(plus),
					ArtsDEF: infection.ArtsDEF + infection.AdditionalArtsDEF*int(plus), ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef*int(plus),
					DEF: infection.BaseDef + infection.AdditionalDEF*int(plus), DEX: infection.DEX + infection.AdditionalDEX*int(plus), HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery*int(plus), INT: infection.INT + infection.AdditionalINT*int(plus),
					MaxHP: infection.MaxHP + infection.AdditionalHP*int(plus), ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef*int(plus), PoisonDEF: infection.ParalysisDef + infection.AdditionalPoisonDef*int(plus), STR: infection.STR + infection.AdditionalSTR*int(plus),
					Accuracy: infection.Accuracy + infection.AdditionalAccuracy*int(plus), Dodge: infection.DodgeRate + infection.AdditionalDodgeRate*int(plus), RunningSpeed: infection.MovSpeed + infection.AdditionalMovSpeed*float64(plus), CanExpire: expire}
			} else {
				percentArtsDEF := int(float64(character.Socket.Stats.ArtsDEF) * (float64(infection.ArtsDEF+infection.AdditionalArtsDEF*int(plus)) / 1000))
				percentDEF := int(float64(character.Socket.Stats.DEF) * (float64(infection.BaseDef+infection.AdditionalDEF*int(plus)) / 1000))
				percentATK := int(float64(character.Socket.Stats.MinATK) * (float64(infection.BaseATK+infection.AdditionalATK*int(plus)) / 1000))
				percentArtsATK := int(float64(character.Socket.Stats.MinArtsATK) * (float64(infection.BaseArtsATK+infection.AdditionalArtsATK*int(plus)) / 1000))
				buff = &Buff{ID: infection.ID, CharacterID: character.ID, StartedAt: character.Epoch, Duration: int64(duration), Name: skillInfo.Name,
					ATK: percentATK, ArtsATK: percentArtsATK,
					ArtsDEF: percentArtsDEF, ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef*int(plus),
					DEF: percentDEF, DEX: infection.DEX + infection.AdditionalDEX*int(plus), HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery*int(plus), INT: infection.INT + infection.AdditionalINT*int(plus),
					MaxHP: infection.MaxHP + infection.AdditionalHP*int(plus), ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef*int(plus), PoisonDEF: infection.ParalysisDef + infection.AdditionalPoisonDef*int(plus), STR: infection.STR + infection.AdditionalSTR*int(plus),
					Accuracy: infection.Accuracy + infection.AdditionalAccuracy*int(plus), Dodge: infection.DodgeRate + infection.AdditionalDodgeRate*int(plus), CanExpire: expire}
			}
			err := buff.Create()
			if err != nil {
				fmt.Println("Buff error: ", err)
			}
		}

		if buff.ID == 241 || buff.ID == 244 || buff.ID == 139 || buff.ID == 50 { // invisibility
			time.AfterFunc(time.Second*1, func() {
				if character != nil {
					character.Invisible = true
				}
			})
		} else if buff.ID == 242 || buff.ID == 245 || buff.ID == 105 || buff.ID == 53 || buff.ID == 59 || buff.ID == 142 || buff.ID == 164 || buff.ID == 214 || buff.ID == 217 { // detection arts
			character.DetectionMode = true
		}
		//if buff.ID == 257 {
		//c.Poisoned = true
		//}
		if buff.ID == 258 {
			c.Confusioned = true
		}
		if buff.ID == 259 {
			c.Paralysised = true
		}
		statData, _ := character.GetStats()
		character.Socket.Write(statData)

		p := &nats.CastPacket{CastNear: true, CharacterID: character.ID, Data: character.GetHPandChi()}
		p.Cast()
	} else { // combat skill
		goto COMBAT
	}

COMBAT:
	target := GetFromRegister(user.ConnectedServer, c.Map, uint16(targetID))
	if skillInfo.PassiveType == 34 {
		teleport := c.Teleport(ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", cX, cY)))
		c.Socket.Write(teleport)
	}
	if ai, ok := target.(*AI); ok { // attacked to ai
		if skillID == 41201 || skillID == 41301 { // howl of tame
			npcPos := NPCPos[ai.PosID]
			npc := NPCs[npcPos.NPCID]
			petInfo := Pets[int64(npc.ID)]
			if petInfo.PetCardItemID != 0 {
				slotID, _, _ := c.FindItemInInventory(nil, petInfo.PetCardItemID)
				/*if items == nil {
					return nil, nil
				}*/
				slots, err := c.InventorySlots()
				if err != nil {
					return nil, err
				}

				item := slots[slotID]
				if item.Quantity > 0 {
					c.TamingAI = ai
					itemData := c.DecrementItem(slotID, 1)
					c.Socket.Write(*itemData)
					c.SkillHistory.Delete(skillID)
					goto OUT
				} else {
					c.SkillHistory.Delete(skillID)
				}
			} else {
				c.TamingAI = ai
				goto OUT
			}
		}
		if skillInfo.PassiveType == 14 {
			st := c.Socket.Stats
			st.HP += 25 + 50*int(plus)
			c.Socket.Write(c.GetHPandChi())
			goto OUT
		}
		pos := NPCPos[ai.PosID]
		if pos.Attackable { // target is attackable
			castLocation := ConvertPointToLocation(c.Coordinate)
			if skillInfo.AreaCenter == 1 || skillInfo.AreaCenter == 2 {
				castLocation = ConvertPointToLocation(ai.Coordinate)
			}
			skillSlots, err := c.Socket.Skills.GetSkills()
			if err != nil {
				return nil, err
			}
			plusRange := 0.0
			divinePlus := 0
			plusDamage := 0
			for _, slot := range skillSlots.Slots {
				if slot.BookID == skillInfo.BookID {
					for _, points := range slot.DivinePoints {
						if points.DivineID == 2 && points.DivinePlus > 0 {
							divinePlus = points.DivinePlus
							plusRange = 0.5
						}
						if points.DivineID == 1 && points.DivinePlus > 0 {
							divinePlus = points.DivinePlus
							plusDamage = 100
						}
					}
				}
			}
			castRange := skillInfo.BaseRadius + skillInfo.AdditionalRadius*float64(plus+1) + (float64(plusRange) * float64(divinePlus))
			candidates := AIsByMap[ai.Server][ai.Map]

			candidates = funk.Filter(candidates, func(cand *AI) bool {
				nPos := NPCPos[cand.PosID]
				if nPos == nil {
					return false
				}

				aiCoordinate := ConvertPointToLocation(cand.Coordinate)
				return (cand.PseudoID == ai.PseudoID || (utils.CalculateDistance(aiCoordinate, castLocation) < castRange)) && cand.HP > 0 && nPos.Attackable
			}).([]*AI)

			if skillInfo.InfectionID != 0 && skillInfo.Target == 1 {
				//c.DealBuffInfection(ai, nil, skillID)
			}
			for _, mob := range candidates {
				dmg, _ := c.CalculateDamage(mob, true)
				dmg += plusDamage * divinePlus
				c.Targets = append(c.Targets, &Target{Damage: dmg, AI: mob, Skill: true})
			}

		} else { // target is not attackable
			if funk.Contains(miningSkills, skillID) { // mining skill
				c.Targets = []*Target{{Damage: 10, AI: ai, Skill: true}}
			}
		}

	} else { // FIX => attacked to player
		enemy := FindCharacterByPseudoID(user.ConnectedServer, uint16(targetID))
		if enemy != nil && enemy.IsActive && skillInfo.PassiveType != 14 && enemy != c {
			dmg, _ := c.CalculateDamageToPlayer(enemy, true)
			c.PlayerTargets = append(c.PlayerTargets, &PlayerTarget{Damage: dmg, Enemy: enemy, Skill: true})
		} else if skillInfo.PassiveType == 14 {
			if enemy != nil {
				partyPlayer := FindCharacterByPseudoID(user.ConnectedServer, uint16(c.Selection))
				est := partyPlayer.Socket.Stats
				est.HP += skillInfo.BaseMinHP + skillInfo.AdditionalMinHP*int(plus)
				partyPlayer.Socket.Write(partyPlayer.GetHPandChi())
			} else if enemy == nil {
				st := c.Socket.Stats
				st.HP += skillInfo.BaseMinHP + skillInfo.AdditionalMinHP*int(plus)
				c.Socket.Write(c.GetHPandChi())
			}
		}
		if skillInfo.InfectionID != 0 && skillInfo.Target == 1 && enemy != c {
			c.DealBuffInfection(nil, enemy, skillID)
		}
	}

OUT:
	r := SKILL_CASTED
	index := 7
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // character pseudo id
	index += 2
	r[index] = byte(attackCounter)
	index++
	r.Insert(utils.IntToBytes(uint64(skillID), 4, true), index) // skill id
	index += 4
	r.Insert(utils.FloatToBytes(cX, 4, true), index) // coordinate-x
	index += 4
	r.Insert(utils.FloatToBytes(cY, 4, true), index) // coordinate-y
	index += 4
	r.Insert(utils.FloatToBytes(cZ, 4, true), index) // coordinate-z
	index += 5
	r.Insert(utils.IntToBytes(uint64(targetID), 2, true), index) // target id
	index += 3
	r.Insert(utils.IntToBytes(uint64(targetID), 2, true), index) // target id
	//index += 2

	p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.CAST_SKILL, Data: r}
	if err = p.Cast(); err != nil {
		return nil, err
	}

	resp.Concat(r)
	resp.Concat(c.GetHPandChi())

	//go stat.Update()
	return resp, nil
}

func (c *Character) HealPlayer(amount int) {
	st := c.Socket.Stats
	st.HP += amount
	c.Socket.Write(c.GetHPandChi())
}
func (c *Character) CraftBuff(name string, infectionID, plus int, duration int64, IsServerEpoch bool) (*Buff, error) {
	buff := &Buff{}
	infection := BuffInfections[infectionID]
	expire := true
	var userid string
	var startedAt int64

	var drop int
	var gold int
	var exp int
	if infectionID == 21 { //Need to reservate 2 more IDs for Gold and Drop Event
		startedAt = GetServerEpoch()
		userid = "event"
		drop = 0
		gold = 0
		exp = 0
		if infectionID == 21 {
			parts := strings.Split(name, " ")
			amStr := strings.TrimSuffix(parts[len(parts)-1], "%")
			expInt, _ := strconv.Atoi(amStr)
			exp = expInt * 10
			expMultiplier := float64(1 + (float64(expInt) / 100))
			fmt.Println(expMultiplier)
			EVENT_DETAILS := utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0xE7, 0x00, 0x00, 0x00, 0x90, 0x3F, 0x30, 0x00, 0x00, 0x00, 0x55, 0xAA}
			EVENT_DETAILS.Overwrite(utils.FloatToBytes(expMultiplier, 4, true), 6)
			EVENT_DETAILS.Overwrite(utils.IntToBytes(uint64(duration), 4, true), 10)
			c.Socket.Write(EVENT_DETAILS)
		}
	} else {
		if IsServerEpoch {
			startedAt = GetServerEpoch()
			userid = c.UserID
		} else {
			startedAt = c.Epoch
			userid = "none"
		}
		drop = infection.DropRate
		gold = infection.GoldRate
		exp = infection.ExpRate
	}

	if infectionID != 0 && duration == 0 {
		expire = false
	}

	if !infection.IsPercent {
		buff = &Buff{ID: infectionID, CharacterID: c.ID, StartedAt: startedAt, Duration: int64(duration), Name: name,
			ATK:     infection.BaseATK + infection.AdditionalATK*int(plus),
			ArtsATK: infection.BaseArtsATK + infection.AdditionalArtsATK*int(plus),

			DEF:     infection.BaseDef + infection.AdditionalDEF*int(plus),
			ArtsDEF: infection.ArtsDEF + infection.AdditionalArtsDEF*int(plus),

			ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef*int(plus),
			ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef*int(plus),
			PoisonDEF:    infection.ParalysisDef + infection.AdditionalPoisonDef*int(plus),

			STR: infection.STR + infection.AdditionalSTR*int(plus),
			DEX: infection.DEX + infection.AdditionalDEX*int(plus),
			INT: infection.INT + infection.AdditionalINT*int(plus),

			Water: infection.Water + infection.AdditionalWater*int(plus),
			Fire:  infection.Fire + infection.AdditionalFire*int(plus),
			Wind:  infection.Wind + infection.AdditionalWind*int(plus),

			MaxHP:          infection.MaxHP + infection.AdditionalHP*int(plus),
			HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery*int(plus),

			Accuracy: infection.Accuracy + infection.AdditionalAccuracy*int(plus),
			Dodge:    infection.DodgeRate + infection.AdditionalDodgeRate*int(plus),

			RunningSpeed: infection.MovSpeed + infection.AdditionalMovSpeed*float64(plus),

			DropRate: drop,
			GoldRate: gold,
			ExpRate:  exp,

			Plus:          plus,
			UserID:        userid,
			IsServerEpoch: IsServerEpoch,
			IsPercent:     infection.IsPercent,
			CanExpire:     expire}
	} else {
		percentATK := int(float64(c.Socket.Stats.MinATK) * (float64(infection.BaseATK+infection.AdditionalATK*int(plus)) / 1000))
		percentArtsATK := int(float64(c.Socket.Stats.MinArtsATK) * (float64(infection.BaseArtsATK+infection.AdditionalArtsATK*int(plus)) / 1000))

		percentDEF := int(float64(c.Socket.Stats.DEF) * (float64(infection.BaseDef+infection.AdditionalDEF*int(plus)) / 1000))
		percentArtsDEF := int(float64(c.Socket.Stats.ArtsDEF) * (float64(infection.ArtsDEF+infection.AdditionalArtsDEF*int(plus)) / 1000))

		percentConfusionDEF := int(float64(c.Socket.Stats.ConfusionDEF) * (float64(infection.ConfusionDef+infection.AdditionalConfusionDef*int(plus)) / 1000))
		percentParalysisDEF := int(float64(c.Socket.Stats.ParalysisDEF) * (float64(infection.ParalysisDef+infection.ParalysisDef*int(plus)) / 1000))
		percentPoisonDEF := int(float64(c.Socket.Stats.PoisonDEF) * (float64(infection.PoisonDef+infection.AdditionalPoisonDef*int(plus)) / 1000))

		percentSTR := int(float64(c.Socket.Stats.STR) * (float64(infection.STR+infection.AdditionalSTR*int(plus)) / 1000))
		percentDEX := int(float64(c.Socket.Stats.DEX) * (float64(infection.DEX+infection.AdditionalDEX*int(plus)) / 1000))
		percentINT := int(float64(c.Socket.Stats.INT) * (float64(infection.INT+infection.AdditionalINT*int(plus)) / 1000))

		percentWater := int(float64(c.Socket.Stats.Water) * (float64(infection.Water+infection.AdditionalWater*int(plus)) / 1000))
		percentFire := int(float64(c.Socket.Stats.Fire) * (float64(infection.Fire+infection.AdditionalFire*int(plus)) / 1000))
		percentWind := int(float64(c.Socket.Stats.Wind) * (float64(infection.Wind+infection.AdditionalWind*int(plus)) / 1000))

		percentMaxHP := int(float64(c.Socket.Stats.MaxHP) * (float64(infection.MaxHP) / 1000))
		percentHPRecoveryRate := int(float64(c.Socket.Stats.HPRecoveryRate) * (float64(infection.HPRecoveryRate) / 1000))

		percentAccuracy := int(float64(c.Socket.Stats.Accuracy) * (float64(infection.Accuracy+infection.AdditionalAccuracy*int(plus)) / 1000))
		percentDodge := int(float64(c.Socket.Stats.Dodge) * (float64(infection.DodgeRate+infection.AdditionalDodgeRate*int(plus)) / 1000))

		buff = &Buff{ID: infection.ID, CharacterID: c.ID, StartedAt: startedAt, Duration: int64(duration), Name: name,
			ATK:     percentATK,
			ArtsATK: percentArtsATK,

			DEF:     percentDEF,
			ArtsDEF: percentArtsDEF,

			ConfusionDEF: percentConfusionDEF,
			ParalysisDEF: percentParalysisDEF,
			PoisonDEF:    percentPoisonDEF,

			STR: percentSTR,
			DEX: percentDEX,
			INT: percentINT,

			Water: percentWater,
			Fire:  percentFire,
			Wind:  percentWind,

			MaxHP:          percentMaxHP,
			HPRecoveryRate: percentHPRecoveryRate,

			Accuracy: percentAccuracy,
			Dodge:    percentDodge,

			RunningSpeed: infection.MovSpeed + infection.AdditionalMovSpeed*float64(plus),

			DropRate: drop,
			GoldRate: gold,
			ExpRate:  exp,

			Plus:          plus,
			UserID:        userid,
			IsServerEpoch: IsServerEpoch,
			IsPercent:     infection.IsPercent,
			CanExpire:     expire}
	}
	return buff, nil
}

func (c *Character) DealBuffInfection(ai *AI, character *Character, skillID int) {
	if ai != nil { //AI BUFF ADD
		/*skills := c.Socket.Skills
		_, err := skills.GetPlus(skillID)
		if err == nil {
			skillInfo := SkillInfos[skillID]
			if skillInfo.InfectionID != 0 {
				infection := BuffInfections[skillInfo.InfectionID]
				//duration := (skillInfo.BaseTime + skillInfo.AdditionalTime*int(plus)) / 10
				//fmt.Println(duration)
				aibuff, err := FindBuffByAIID(infection.ID, int(ai.PseudoID))
				if err == nil {
					if aibuff == nil {
						now := time.Now()
						secs := now.Unix()
						aibuff = &AiBuff{ID: infection.ID, AiID: int(ai.PseudoID), Name: infection.Name, StartedAt: secs, CharacterID: character.ID, Duration: int64(1)}
						//aibuff.Create()
					}
				}
			}

		}*/
	} else if character != nil { //PLAYER BUFF ADD
		skills := c.Socket.Skills
		plus, err := skills.GetPlus(skillID)
		if err != nil {
			//return nil, err
		}
		skillInfo := SkillInfos[skillID]
		infection := BuffInfections[skillInfo.InfectionID]
		duration := (skillInfo.BaseTime + skillInfo.AdditionalTime*int(plus)) / 10
		buff, err := FindBuffByCharacter(infection.ID, character.ID)
		if err != nil {
			//return nil, err
		} else if buff != nil {
			buff = &Buff{ID: infection.ID, CharacterID: character.ID, StartedAt: character.Epoch, Duration: int64(duration), Name: skillInfo.Name,
				ATK: infection.BaseATK + infection.AdditionalATK*int(plus), ArtsATK: infection.BaseArtsATK + infection.AdditionalArtsATK*int(plus),
				ArtsDEF: infection.ArtsDEF + infection.AdditionalArtsDEF*int(plus), ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef*int(plus), CanExpire: true,
				DEF: infection.BaseDef + infection.AdditionalDEF*int(plus), DEX: infection.DEX + infection.AdditionalDEX*int(plus), HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery*int(plus), INT: infection.INT + infection.AdditionalINT*int(plus),
				MaxHP: infection.MaxHP + infection.AdditionalHP*int(plus), ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef*int(plus), PoisonDEF: infection.ParalysisDef + infection.AdditionalPoisonDef*int(plus), STR: infection.STR + infection.AdditionalSTR*int(plus),
				Accuracy: infection.Accuracy + infection.AdditionalAccuracy*int(plus), Dodge: infection.DodgeRate + infection.AdditionalDodgeRate*int(plus), RunningSpeed: infection.MovSpeed + infection.AdditionalMovSpeed*float64(plus), SkillPlus: int(plus)}
			buff.Update()

		} else if buff == nil {
			if infection.IsPercent == false {
				buff = &Buff{ID: infection.ID, CharacterID: character.ID, StartedAt: character.Epoch, Duration: int64(duration), Name: skillInfo.Name,
					ATK: infection.BaseATK + infection.AdditionalATK*int(plus), ArtsATK: infection.BaseArtsATK + infection.AdditionalArtsATK*int(plus),
					ArtsDEF: infection.ArtsDEF + infection.AdditionalArtsDEF*int(plus), ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef*int(plus), CanExpire: true,
					DEF: infection.BaseDef + infection.AdditionalDEF*int(plus), DEX: infection.DEX + infection.AdditionalDEX*int(plus), HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery*int(plus), INT: infection.INT + infection.AdditionalINT*int(plus),
					MaxHP: infection.MaxHP + infection.AdditionalHP*int(plus), ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef*int(plus), PoisonDEF: infection.ParalysisDef + infection.AdditionalPoisonDef*int(plus), STR: infection.STR + infection.AdditionalSTR*int(plus),
					Accuracy: infection.Accuracy + infection.AdditionalAccuracy*int(plus), Dodge: infection.DodgeRate + infection.AdditionalDodgeRate*int(plus), RunningSpeed: infection.MovSpeed + infection.AdditionalMovSpeed*float64(plus), SkillPlus: int(plus)}
			} else {
				percentArtsDEF := int(float64(character.Socket.Stats.ArtsDEF) * (float64(infection.ArtsDEF+infection.AdditionalArtsDEF*int(plus)) / 1000))
				percentDEF := int(float64(character.Socket.Stats.DEF) * (float64(infection.BaseDef+infection.AdditionalDEF*int(plus)) / 1000))
				percentATK := int(float64(character.Socket.Stats.MinATK) * (float64(infection.BaseATK+infection.AdditionalATK*int(plus)) / 1000))
				percentArtsATK := int(float64(character.Socket.Stats.MinArtsATK) * (float64(infection.BaseArtsATK+infection.AdditionalArtsATK*int(plus)) / 1000))
				percentRunningSpeed := float64(character.RunningSpeed) * (float64(infection.MovSpeed+infection.AdditionalMovSpeed*float64(plus)) / 1000)
				buff = &Buff{ID: infection.ID, CharacterID: character.ID, StartedAt: character.Epoch, Duration: int64(duration), Name: skillInfo.Name,
					ATK: percentATK, ArtsATK: percentArtsATK,
					ArtsDEF: percentArtsDEF, ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef*int(plus), CanExpire: true,
					DEF: percentDEF, DEX: infection.DEX + infection.AdditionalDEX*int(plus), HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery*int(plus), INT: infection.INT + infection.AdditionalINT*int(plus),
					MaxHP: infection.MaxHP + infection.AdditionalHP*int(plus), ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef*int(plus), PoisonDEF: infection.ParalysisDef + infection.AdditionalPoisonDef*int(plus), STR: infection.STR + infection.AdditionalSTR*int(plus),
					Accuracy: infection.Accuracy + infection.AdditionalAccuracy*int(plus), Dodge: infection.DodgeRate + infection.AdditionalDodgeRate*int(plus), SkillPlus: int(plus), RunningSpeed: percentRunningSpeed}
			}

			buff.Create()
		}
		statData, _ := character.GetStats()
		character.Socket.Write(statData)

		p := &nats.CastPacket{CastNear: true, CharacterID: character.ID, Data: character.GetHPandChi()}
		p.Cast()
	}
}

func (c *Character) CalculateDamage(ai *AI, isSkill bool) (int, error) {

	st := c.Socket.Stats

	npcPos := NPCPos[ai.PosID]
	npc := NPCs[npcPos.NPCID]

	def, min, max := npc.DEF, st.MinATK, st.MaxATK
	if isSkill {
		def, min, max = npc.ArtsDEF, st.MinArtsATK, st.MaxArtsATK
	}

	dmg := int(utils.RandInt(int64(min), int64(max))) - def
	if dmg < 3 {
		dmg = 3
	} else if dmg > ai.HP {
		dmg = ai.HP
	}

	if diff := int(npc.Level) - c.Level; diff > 0 {
		reqAcc := utils.SigmaFunc(float64(diff))
		if float64(st.Accuracy) < reqAcc {
			probability := float64(st.Accuracy) * 1000 / reqAcc
			if utils.RandInt(0, 1000) > int64(probability) {
				dmg = 0
			}
		}
	}

	return dmg, nil
}
func (c *Character) CalculateSpecialAttacksToPlayer(enemy *Character, isSkill bool) (bool, error) {
	st := c.Socket.Stats
	enemySt := enemy.Socket.Stats
	if st.PoisonATK > enemySt.PoisonDEF {
		buff, err := FindBuffByCharacter(257, enemy.ID)
		if err == nil && buff == nil {
			diff := st.PoisonATK - enemySt.PoisonDEF
			reqPoisonDMG := int64((float64(diff) / float64(st.PoisonATK)) * 1000)
			lastPoison := float64(reqPoisonDMG) * 0.7
			if utils.RandInt(0, 1000) < int64(lastPoison) {
				infection := BuffInfections[257]
				enemy.SufferedPoison = st.PoisonATK - enemySt.PoisonDEF
				buff = &Buff{ID: 257, CharacterID: enemy.ID, StartedAt: enemy.Epoch, Duration: int64(st.PoisonTime), Name: infection.Name,
					ATK: infection.BaseATK, ArtsATK: infection.BaseArtsATK + infection.AdditionalArtsATK,
					ArtsDEF: infection.ArtsDEF, ConfusionDEF: infection.ConfusionDef,
					DEF: infection.BaseDef, DEX: infection.DEX, HPRecoveryRate: infection.HPRecoveryRate, INT: infection.INT,
					MaxHP: infection.MaxHP, ParalysisDEF: infection.ParalysisDef, PoisonDEF: infection.ParalysisDef, STR: infection.STR, Accuracy: infection.Accuracy, Dodge: infection.DodgeRate, CanExpire: true}
				buff.Create()
				enemy.Poisoned = true
				enemy.PoisonSource = c
			}
		}
	}
	if st.ConfusionATK > enemySt.ConfusionDEF {
		diff := st.ConfusionATK - enemySt.ConfusionDEF
		reqConfDMG := int64((float64(diff) / float64(st.ConfusionATK)) * 1000)
		lastPoison := float64(reqConfDMG) * 0.7
		if utils.RandInt(0, 1000) < int64(lastPoison) {
			buff, err := FindBuffByCharacter(258, enemy.ID)
			if err == nil && buff == nil {
				infection := BuffInfections[258]
				buff := &Buff{ID: 258, CharacterID: enemy.ID, StartedAt: enemy.Epoch, Duration: int64(st.ConfusionTime), Name: infection.Name,
					ATK: infection.BaseATK, ArtsATK: infection.BaseArtsATK + infection.AdditionalArtsATK,
					ArtsDEF: infection.ArtsDEF, ConfusionDEF: infection.ConfusionDef,
					DEF: infection.BaseDef, DEX: infection.DEX, HPRecoveryRate: infection.HPRecoveryRate, INT: infection.INT,
					MaxHP: infection.MaxHP, ParalysisDEF: infection.ParalysisDef, PoisonDEF: infection.ParalysisDef, STR: infection.STR, Accuracy: infection.Accuracy, Dodge: infection.DodgeRate, CanExpire: true}
				buff.Create()
				enemy.Confusioned = true
				enemy.ConfusionSource = c
			}
		}
	}
	if st.ParalysisATK > enemySt.ParalysisDEF {
		diff := st.ParalysisATK - enemySt.ParalysisDEF
		reqParaDMG := int64((float64(diff) / float64(st.ParalysisATK)) * 1000)
		lastPoison := float64(reqParaDMG) * 0.7
		if utils.RandInt(0, 1000) < int64(lastPoison) {
			buff, err := FindBuffByCharacter(259, enemy.ID)
			if err == nil && buff == nil {
				infection := BuffInfections[259]
				buff := &Buff{ID: 259, CharacterID: enemy.ID, StartedAt: enemy.Epoch, Duration: int64(st.Paratime), Name: infection.Name,
					ATK: infection.BaseATK, ArtsATK: infection.BaseArtsATK + infection.AdditionalArtsATK,
					ArtsDEF: infection.ArtsDEF, ConfusionDEF: infection.ConfusionDef,
					DEF: infection.BaseDef, DEX: infection.DEX, HPRecoveryRate: infection.HPRecoveryRate, INT: infection.INT,
					MaxHP: infection.MaxHP, ParalysisDEF: infection.ParalysisDef, PoisonDEF: infection.ParalysisDef, STR: infection.STR, Accuracy: infection.Accuracy, Dodge: infection.DodgeRate, CanExpire: true}
				buff.Create()
				enemy.Paralysised = true
				enemy.ParaSource = c
			}
		}
	}
	return true, nil
}

func (c *Character) CalculateDamageToPlayer(enemy *Character, isSkill bool) (int, error) {
	st := c.Socket.Stats
	enemySt := enemy.Socket.Stats

	def, min, max := enemySt.DEF, st.MinATK, st.MaxATK
	if isSkill {
		def, min, max = enemySt.ArtsDEF, st.MinArtsATK, st.MaxArtsATK
	}

	def = utils.PvPFunc(def)

	dmg := int(utils.RandInt(int64(min), int64(max))) - def
	if dmg < 0 {
		dmg = 3
	} else if dmg > enemySt.HP {
		dmg = enemySt.HP
	}

	reqAcc := (float64(enemySt.Dodge) * 0.5) - float64(st.Accuracy) + float64(c.Level-int(enemy.Level))*10
	//probability := float64(st.Accuracy) * 1000 / reqAcc
	if utils.RandInt(0, 1000) < int64(reqAcc) {
		dmg = 0
	}

	if dmg > 0 && c != enemy && c.CanAttack(enemy) {
		c.CalculateSpecialAttacksToPlayer(enemy, isSkill)
	}
	return dmg, nil
}

func (c *Character) CancelTrade() {

	trade := FindTrade(c)
	if trade == nil {
		return
	}

	receiver, sender := trade.Receiver.Character, trade.Sender.Character
	trade.Delete()

	resp := TRADE_CANCELLED
	sender.Socket.Write(resp)
	receiver.Socket.Write(resp)
}

func (c *Character) OpenSale(name string, slotIDs []int16, prices []uint64) ([]byte, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to open sale while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	sale := &Sale{ID: c.PseudoID, Seller: c, Name: name}
	for i := 0; i < len(slotIDs); i++ {
		slotID := slotIDs[i]
		price := prices[i]
		item := slots[slotID]

		info := Items[item.ItemID]

		if slotID == 0 || price == 0 || item == nil || item.ItemID == 0 || !info.Tradable {
			continue
		}
		saleItem := &SaleItem{SlotID: slotID, Price: price, IsSold: false}
		sale.Items = append(sale.Items, saleItem)
	}

	text := fmt.Sprintf("Character: " + c.Name + "(" + c.UserID + ") Opened shop with items :\n")
	for _, saleItem := range sale.Items {
		slotID := saleItem.SlotID
		item := slots[slotID]
		info, _ := GetItemInfo(item.ItemID)
		text += fmt.Sprintf("%d		%s			Price: %d 			Qty: %d\n", info.ID, info.Name, saleItem.Price, item.Quantity)
	}

	utils.NewLog("logs/shops_logs.txt", text)

	sale.Data, err = sale.SaleData()
	if err != nil {
		return nil, err
	}

	go sale.Create()

	resp := OPEN_SALE
	spawnData, err := c.SpawnCharacter()
	if err == nil {
		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.PLAYER_SPAWN, Data: spawnData}
		p.Cast()
		resp.Concat(spawnData)
	}

	return resp, nil
}

func FindSaleVisitors(saleID uint16) []*Character {

	characterMutex.RLock()
	allChars := funk.Values(characters)
	characterMutex.RUnlock()

	return funk.Filter(allChars, func(c *Character) bool {
		return c.IsOnline && c.VisitedSaleID == saleID
	}).([]*Character)
}

func (c *Character) CloseSale() ([]byte, error) {
	sale := FindSale(c.PseudoID)
	if sale != nil {
		sale.Delete()
		resp := CLOSE_SALE

		spawnData, err := c.SpawnCharacter()
		if err == nil {
			p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.PLAYER_SPAWN, Data: spawnData}
			p.Cast()
			resp.Concat(spawnData)
		}

		return resp, nil
	}

	return nil, nil
}

func (c *Character) BuySaleItem(saleID uint16, saleSlotID, inventorySlotID int16) ([]byte, error) {
	sale := FindSale(saleID)
	if sale == nil {
		return nil, nil
	}
	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to buy sale items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	mySlots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	seller := sale.Seller
	slots, err := seller.InventorySlots()
	if err != nil {
		return nil, err
	}

	saleItem := sale.Items[saleSlotID]
	if saleItem == nil || saleItem.IsSold {
		return nil, nil
	}

	item := slots[saleItem.SlotID]
	if item == nil || item.ItemID == 0 || c.Gold < saleItem.Price {
		return nil, nil
	}

	c.LootGold(-saleItem.Price)
	seller.Gold += saleItem.Price

	resp := BOUGHT_SALE_ITEM
	resp.Insert(utils.IntToBytes(c.Gold, 8, true), 8)                   // buyer gold
	resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 17)     // sale item id
	resp.Insert(utils.IntToBytes(uint64(item.Quantity), 2, true), 23)   // sale item quantity
	resp.Insert(utils.IntToBytes(uint64(inventorySlotID), 2, true), 25) // inventory slot id
	resp.Insert(item.GetUpgrades(), 27)                                 // sale item upgrades
	resp[42] = byte(item.SocketCount)                                   // item socket count
	resp.Insert(item.GetSockets(), 43)                                  // sale item sockets

	myItem := NewSlot()
	*myItem = *item
	myItem.CharacterID = null.IntFrom(int64(c.ID))
	myItem.UserID = null.StringFrom(c.UserID)
	myItem.SlotID = int16(inventorySlotID)
	mySlots[inventorySlotID] = myItem
	myItem.Update()
	InventoryItems.Add(myItem.ID, myItem)

	resp.Concat(item.GetData(inventorySlotID))
	logger.Log(logging.ACTION_BUY_SALE_ITEM, c.ID, fmt.Sprintf("Bought sale item (%d) with %d gold from seller (%d)", myItem.ID, saleItem.Price, seller.ID), c.UserID)
	saleItem.IsSold = true

	sellerResp := SOLD_SALE_ITEM
	sellerResp.Insert(utils.IntToBytes(uint64(saleSlotID), 2, true), 8)  // sale slot id
	sellerResp.Insert(utils.IntToBytes(seller.Gold, 8, true), 10)        // seller gold
	sellerResp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 18) // buyer pseudo id

	*item = *NewSlot()
	sellerResp.Concat(item.GetData(saleItem.SlotID))

	remainingCount := len(funk.Filter(sale.Items, func(i *SaleItem) bool {
		return i.IsSold == false
	}).([]*SaleItem))

	if remainingCount > 0 {
		sale.Data, _ = sale.SaleData()
		resp.Concat(sale.Data)

	} /*else {
		sale.Delete()

		spawnData, err := seller.SpawnCharacter()
		if err == nil {
			p := nats.CastPacket{CastNear: true, CharacterID: seller.ID, Type: nats.PLAYER_SPAWN, Data: spawnData}
			p.Cast()
			resp.Concat(spawnData)
		}

		visitors := FindSaleVisitors(sale.ID)
		for _, v := range visitors {
			v.Socket.Write(CLOSE_SALE)
			v.VisitedSaleID = 0
		}

		//resp.Concat(CLOSE_SALE)
		sellerResp.Concat(CLOSE_SALE)
	}*/

	seller.Socket.Write(sellerResp)
	return resp, nil
}

func (c *Character) UpdatePartyStatus() {

	user := c.Socket.User
	stat := c.Socket.Stats

	party := FindParty(c)
	if party == nil {
		return
	}

	coordinate := ConvertPointToLocation(c.Coordinate)

	resp := PARTY_STATUS
	resp.Insert(utils.IntToBytes(uint64(c.ID), 4, true), 6)             // character id
	resp.Insert(utils.IntToBytes(uint64(stat.HP), 4, true), 10)         // character hp
	resp.Insert(utils.IntToBytes(uint64(stat.MaxHP), 4, true), 14)      // character max hp
	resp.Insert(utils.FloatToBytes(float64(coordinate.X), 4, true), 19) // coordinate-x
	resp.Insert(utils.FloatToBytes(float64(coordinate.Y), 4, true), 23) // coordinate-y
	resp.Insert(utils.IntToBytes(uint64(stat.CHI), 4, true), 27)        // character chi
	resp.Insert(utils.IntToBytes(uint64(stat.MaxCHI), 4, true), 31)     // character max chi
	resp.Insert(utils.IntToBytes(uint64(c.Level), 4, true), 35)         // character level
	resp[39] = byte(c.Type)                                             // character type
	resp[41] = byte(user.ConnectedServer - 1)                           // connected server id

	members := party.GetMembers()
	members = funk.Filter(members, func(m *PartyMember) bool {
		return m.Accepted
	}).([]*PartyMember)

	party.Leader.Socket.Write(resp)
	for _, m := range members {
		m.Socket.Write(resp)
	}
}

func (c *Character) LeaveParty() {

	party := FindParty(c)
	if party == nil {
		return
	}

	//c.PartyID = ""

	members := party.GetMembers()
	members = funk.Filter(members, func(m *PartyMember) bool {
		return m.Accepted
	}).([]*PartyMember)

	resp := utils.Packet{}
	removedLobby := false
	findNewPartyLeader := false
	if c.ID == party.Leader.ID { // disband party
		arenaType := GetArenaTypeByLevelType(c.Level)
		if !FindPlayerInArena(c) || LevelTypeArenaLobby[arenaType].Parties[c.PartyID] != nil { //if arena founded or in lobby then kick all players
			RemovePartyFromLobby(c)
			removedLobby = true
		}
		if FindPlayerInArena(c) {
			findNewPartyLeader = true
		}

		if findNewPartyLeader {
			if len(party.GetMembers()) == 0 {
				c.PartyID = ""
				resp.Concat(PARTY_DISBANDED)
				party.Delete()
				party.Leader.Socket.Write(resp)
			} else {
				for k, member := range members {
					if member.ID != c.ID {
						if k == 0 {
							party.Leader = member.Character
						}
						member.PartyID = party.Leader.UserID
					}
				}
			}
			c.PartyID = ""
		} else {
			resp = PARTY_DISBANDED
			party.Leader.Socket.Write(resp)
			for _, member := range members {
				member.PartyID = ""
				member.Socket.Write(resp)
				if removedLobby {
					//message info: You have been kicked from the party, because the party leader has left the game.
					member.Socket.Write(messaging.InfoMessage("You have been kicked from the lobby, because the party leader has left the party."))
				}
			}
			c.PartyID = ""
			party.Delete()
		}

	} else { // leave party
		member := party.GetMember(c.ID)
		party.RemoveMember(member)
		arenaType := GetArenaTypeByLevelType(c.Level)
		if !FindPlayerInArena(c) || LevelTypeArenaLobby[arenaType].Parties[c.PartyID] != nil { //if arena founded or in lobby then kick all players
			RemovePartyFromLobby(c)
			removedLobby = true
		}
		c.PartyID = ""
		resp = LEFT_PARTY
		resp.Insert(utils.IntToBytes(uint64(c.ID), 4, true), 8) // character id

		leader := party.Leader
		if len(party.GetMembers()) == 0 {
			leader.PartyID = ""
			resp.Concat(PARTY_DISBANDED)
			party.Delete()

		}

		leader.Socket.Write(resp)
		for _, m := range members {
			m.Socket.Write(resp)
			if removedLobby {
				//message info: You have been kicked from the party, because the party leader has left the game.
				m.Socket.Write(messaging.InfoMessage("You have been kicked from the lobby, because somebody has left the party."))
			}
		}

	}
}

func (c *Character) GetGuildData() ([]byte, error) {

	if c.GuildID > 0 {
		guild, err := FindGuildByID(c.GuildID)
		if err != nil {
			return nil, err
		} else if guild == nil {
			return nil, nil
		}

		return guild.GetData(c)
	}

	return nil, nil
}

func (c *Character) JobPassives(stat *Stat) error {

	//stat := c.Socket.Stats
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return err
	}

	if passive := skillSlots.Slots[5]; passive.BookID > 0 {
		info := JobPassives[int8(c.Class)]
		if info != nil {
			plus := passive.Skills[0].Plus
			stat.MaxHP += info.MaxHp * plus
			stat.MaxCHI += info.MaxChi * plus
			stat.MinATK += info.ATK * plus
			stat.MaxATK += info.ATK * plus
			stat.MinArtsATK += info.ArtsATK * plus
			stat.MaxArtsATK += info.ArtsATK * plus
			stat.DEF += info.DEF * plus
			stat.ArtsDEF += info.ArtsDef * plus
			stat.Accuracy += info.Accuracy * plus
			stat.Dodge += info.Dodge * plus
			stat.ConfusionDEF += info.ConfusionDEF * plus
			stat.PoisonDEF += info.PoisonDEF * plus
			stat.ParalysisDEF += info.ParalysisDEF * plus
			stat.HPRecoveryRate += info.HPRecoveryRate * plus
		}
	}

	slots := funk.Filter(skillSlots.Slots, func(slot *SkillSet) bool { // get 2nd job passive book
		return slot.BookID == 16100200 || slot.BookID == 16100300 || slot.BookID == 100030021 || slot.BookID == 100030023 || slot.BookID == 100030025
	}).([]*SkillSet)

	for _, slot := range slots {
		for _, skill := range slot.Skills {
			info := SkillInfos[skill.SkillID]
			if info == nil {
				continue
			}

			amount := int(float64(info.BasePassive) + info.AdditionalPassive*float64(skill.Plus))
			switch info.PassiveType {
			case 1: // passive hp
				stat.MaxHP += amount
			case 2: // passive chi
				stat.MaxCHI += amount
			case 3: // passive arts defense
				stat.ArtsDEF += amount
			case 4: // passive defense
				stat.DEF += amount
			case 5: // passive accuracy
				stat.Accuracy += amount
			case 6: // passive dodge
				stat.Dodge += amount
			case 7: // passive arts atk
				stat.MinArtsATK += amount
				stat.MaxArtsATK += amount
			case 8: // passive atk
				stat.MinATK += amount
				stat.MaxATK += amount
			case 9: //HP AND CHI
				stat.MaxHP += amount
				stat.MaxCHI += amount
			case 11: //Dodge RAte AND ACCURACY
				stat.Accuracy += amount
				stat.Dodge += amount
			case 12: //EXTERNAL ATK AND INTERNAL ATK
				stat.MinArtsATK += amount
				stat.MaxArtsATK += amount
				stat.MinATK += amount
				stat.MaxATK += amount
			case 13: //INTERNAL ATTACK AND INTERNAL DEF
				stat.MinATK += amount
				stat.MaxATK += amount
				stat.DEF += amount
			case 14: //EXTERNAL ATK MINUS AND HP +
				stat.MaxHP += amount
				stat.MinArtsATK -= amount
				stat.MaxArtsATK -= amount
			case 15: //DAMAGE + HP
				stat.MaxHP += amount
				stat.MinATK += amount
				stat.MaxATK += amount
			case 16: //MINUS HP AND PLUS DEFENSE
				stat.MaxHP -= 15 //
				stat.DEF += amount
			}
		}
	}

	return nil
}

func (c *Character) BuffEffects(stat *Stat) error {

	allBuffs, err := c.FindAllRelevantBuffs()
	if err != nil {
		return err
	}

	for _, buff := range allBuffs {
		if buff.ID == 10100 || buff.ID == 10098 || buff.ID == 21 { //Skip Special buffs
			continue
		}
		if (!buff.IsServerEpoch && buff.StartedAt+buff.Duration > c.Epoch) || (buff.IsServerEpoch && buff.StartedAt+buff.Duration > GetServerEpoch()) || !buff.CanExpire {
			stat.MinATK += buff.ATK
			stat.MaxATK += buff.ATK
			stat.ATKRate += buff.ATKRate
			stat.DEF += buff.DEF
			stat.DefRate += buff.DEFRate

			stat.MinArtsATK += buff.ArtsATK
			stat.MaxArtsATK += buff.ArtsATK
			stat.ArtsATKRate += buff.ArtsATKRate
			stat.ArtsDEF += buff.ArtsDEF
			stat.ArtsDEFRate += buff.ArtsDEFRate

			stat.MaxCHI += buff.MaxCHI
			stat.MaxHP += buff.MaxHP
			stat.HPRecoveryRate += buff.HPRecoveryRate
			stat.CHIRecoveryRate += buff.CHIRecoveryRate
			stat.Dodge += buff.Dodge
			stat.Accuracy += buff.Accuracy

			stat.ParalysisDEF += buff.ParalysisDEF
			stat.PoisonDEF += buff.PoisonDEF
			stat.ConfusionDEF += buff.ConfusionDEF

			stat.DEXBuff += buff.DEX
			stat.INTBuff += buff.INT
			stat.STRBuff += buff.STR
			stat.FireBuff += buff.Fire
			stat.WaterBuff += buff.Water
			stat.WindBuff += buff.Wind
			c.RunningSpeed += buff.RunningSpeed
		}
	}

	return nil
}

func (c *Character) SpecialBuffEffects(stat *Stat) error {

	allBuffs, err := c.FindAllRelevantBuffs()
	if err != nil {
		return err
	}

	for _, buff := range allBuffs {
		if buff.ID != 10100 && buff.ID != 10098 && buff.ID != 21 { //skip non-Special buffs
			continue
		}
		if (!buff.IsServerEpoch && buff.StartedAt+buff.Duration > c.Epoch) || (buff.IsServerEpoch && buff.StartedAt+buff.Duration > GetServerEpoch()) || !buff.CanExpire {
			stat.MinATK += buff.ATK
			stat.MaxATK += buff.ATK
			stat.ATKRate += buff.ATKRate
			stat.DEF += buff.DEF
			stat.DefRate += buff.DEFRate

			stat.MinArtsATK += buff.ArtsATK
			stat.MaxArtsATK += buff.ArtsATK
			stat.ArtsATKRate += buff.ArtsATKRate
			stat.ArtsDEF += buff.ArtsDEF
			stat.ArtsDEFRate += buff.ArtsDEFRate

			stat.MaxCHI += buff.MaxCHI
			stat.MaxHP += buff.MaxHP
			stat.HPRecoveryRate += buff.HPRecoveryRate
			stat.CHIRecoveryRate += buff.CHIRecoveryRate
			stat.Dodge += buff.Dodge
			stat.Accuracy += buff.Accuracy

			stat.ParalysisDEF += buff.ParalysisDEF
			stat.PoisonDEF += buff.PoisonDEF
			stat.ConfusionDEF += buff.ConfusionDEF

			stat.DEXBuff += buff.DEX
			stat.INTBuff += buff.INT
			stat.STRBuff += buff.STR
			stat.FireBuff += buff.Fire
			stat.WaterBuff += buff.Water
			stat.WindBuff += buff.Wind

			stat.ExpRate += float64(buff.ExpRate) / 1000
			stat.GoldRate += float64(buff.GoldRate) / 1000
			stat.DropRate += float64(buff.DropRate) / 1000
			c.RunningSpeed += buff.RunningSpeed
		}
	}

	return nil
}

func (c *Character) GetLevelText() string {
	if c.Level < 10 {
		return fmt.Sprintf("%dKyu", c.Level)
	} else if c.Level <= 100 {
		return fmt.Sprintf("%dDan %dKyu", c.Level/10, c.Level%10)
	} else if c.Level < 110 {
		return fmt.Sprintf("Divine %dKyu", c.Level%100)
	} else if c.Level <= 200 {
		return fmt.Sprintf("Divine %dDan %dKyu", (c.Level-100)/10, c.Level%100)
	}

	return ""
}

func (c *Character) RelicDrop(itemID int64) []byte {

	itemName := Items[itemID].Name
	msg := fmt.Sprintf("%s has acquired [%s].", c.Name, itemName)
	length := int16(len(msg) + 3)

	now := time.Now().UTC()
	relic := &RelicLog{CharID: c.ID, ItemID: itemID, DropTime: null.TimeFrom(now)}
	RelicsLog[len(RelicsLog)] = relic
	err := relic.Create()
	if err != nil {
		fmt.Println("Error with load: ", err)
	}
	resp := RELIC_DROP
	resp.SetLength(length)
	resp[6] = byte(len(msg))
	resp.Insert([]byte(msg), 7)

	return resp
}

func (c *Character) AidStatus() []byte {

	resp := utils.Packet{}
	if c.AidMode {
		resp = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0xFA, 0x01, 0x55, 0xAA}
		resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 5) // pseudo id
		r2 := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x43, 0x01, 0x55, 0xAA}
		r2.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 5) // pseudo id

		resp.Concat(r2)

	} else {
		resp = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0xFA, 0x00, 0x55, 0xAA}
		resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 5) // pseudo id
		r2 := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x43, 0x00, 0x55, 0xAA}
		r2.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 5) // pseudo id

		resp.Concat(r2)
	}

	return resp
}

func (c *Character) PickaxeActivated() bool {

	slots, err := c.InventorySlots()
	if err != nil {
		return false
	}

	pickaxeIDs := []int64{17200219, 17300005, 17501009, 17502536, 17502537, 17502538}

	return len(funk.Filter(slots, func(slot *InventorySlot) bool {
		return slot.Activated && funk.Contains(pickaxeIDs, slot.ItemID)
	}).([]*InventorySlot)) > 0
}

func (c *Character) TogglePet() []byte {
	slots, err := c.InventorySlots()
	if err != nil {
		return nil
	}

	petSlot := slots[0x0A]
	pet := petSlot.Pet
	if pet == nil {
		return nil
	}
	spawnData, _ := c.SpawnCharacter()
	pet.PetOwner = c
	petInfo, _ := Pets[petSlot.ItemID]
	if petInfo.Combat {
		location := ConvertPointToLocation(c.Coordinate)
		pet.Coordinate = utils.Location{X: location.X + 3, Y: location.Y}
		pet.IsOnline = !pet.IsOnline

		if pet.IsOnline {
			GeneratePetID(c, pet)
			pet.PetCombatMode = 0
			pet.CombatPet = true
			c.PetHandlerCB = c.PetHandler
			go c.PetHandlerCB()

			resp := utils.Packet{
				0xAA, 0x55, 0x0B, 0x00, 0x75, 0x00, 0x01, 0x00, 0x80, 0xa1, 0x43, 0x00, 0x00, 0x3d, 0x43, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x01, 0x0A, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x06, 0x00, 0x51, 0x05, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x06, 0x0A, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x07, 0x0A, 0x00, 0x00, 0x55, 0xAA,
				0xaa, 0x55, 0x12, 0x00, 0x51, 0x08, 0x0a, 0x00, 0x03, 0x01, 0x3b, 0x00, 0x3b, 0x00, 0x26, 0x00, 0x00, 0x00, 0x8d, 0x00, 0x00, 0x00, 0x55, 0xaa,
			}
			resp.Concat(spawnData)
			return resp
		}
	} else {
		location := ConvertPointToLocation(c.Coordinate)
		pet.Coordinate = utils.Location{X: location.X, Y: location.Y}
		pet.IsOnline = !pet.IsOnline
		if pet.IsOnline {
			GeneratePetID(c, pet)
			pet.PetCombatMode = 0
			c.IsMounting = true
			pet.CombatPet = false
			c.PetHandlerCB = c.PetHandler
			go c.PetHandlerCB()

			/*resp := utils.Packet{
				0xAA, 0x55, 0x0B, 0x00, 0x75, 0x00, 0x01, 0x00, 0x80, 0xA1, 0x43, 0x00, 0x00, 0x3D, 0x43, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x01, 0x0A, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x06, 0x0A, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x06, 0x00, 0x51, 0x05, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x07, 0x0A, 0x00, 0x00, 0x55, 0xAA,
			}*/
			c.Socket.Write([]byte{0xAA, 0x55, 0x0B, 0x00, 0x75, 0x00, 0x01, 0x00, 0x80, 0xA1, 0x43, 0x00, 0x00, 0x3D, 0x43, 0x55, 0xAA})
			c.Socket.Write([]byte{0xAA, 0x55, 0x05, 0x00, 0x51, 0x01, 0x0A, 0x00, 0x00, 0x55, 0xAA})
			c.Socket.Write(spawnData)
			c.Socket.Write([]byte{0xAA, 0x55, 0x05, 0x00, 0x51, 0x06, 0x0A, 0x00, 0x00, 0x55, 0xAA})
			c.Socket.Write([]byte{0xAA, 0x55, 0x06, 0x00, 0x51, 0x05, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA})
			c.Socket.Write([]byte{0xAA, 0x55, 0x05, 0x00, 0x51, 0x07, 0x0A, 0x00, 0x00, 0x55, 0xAA})
			//resp.Concat(spawnData)
			return nil
		}
	}
	/*characters, err := c.GetNearbyCharacters()
	if err != nil {
		log.Println(err)
	}*/
	//test := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x21, 0x02, 0x00, 0x55, 0xAA}
	/*for _, chars := range characters {
		delete(chars.OnSight.Players, c.ID)
	}*/
	pet.Target = 0
	pet.Casting = false
	pet.IsMoving = false
	c.PetHandlerCB = nil
	c.IsMounting = false
	RemovePetFromRegister(c)
	showpet, _ := c.ShowItems()
	resp := DISMISS_PET
	resp.Concat(showpet)
	return resp
}

func (c *Character) CalculateInjury() []int {
	remaining := c.Injury
	divCount := []int{0, 0, 0, 0}
	divNumbers := []float64{0.1, 0.7, 1.09, 17.48}
	for i := len(divNumbers) - 1; i >= 0; i-- {
		if remaining < divNumbers[i] || remaining == 0 {
			continue
		}
		test := remaining / divNumbers[i]
		if test > 15 {
			test = 15
		}
		divCount[i] = int(test)
		test2 := test * divNumbers[i]
		remaining -= test2
	}
	return divCount
}
func (c *Character) CalculateNumbers(number float64) []int {
	remaining := number
	divCount := []int{0, 0, 0, 0}
	divNumbers := []float64{0.1, 0.7, 1.09, 17.48}
	for i := len(divNumbers) - 1; i >= 0; i-- {
		if remaining < divNumbers[i] || remaining == 0 {
			continue
		}
		test := remaining / divNumbers[i]
		if test > 15 {
			test = 15
		}
		divCount[i] = int(test)
		test2 := test * divNumbers[i]
		remaining -= test2
	}
	return divCount
}

func RemoveIndex(s []*AI, index int) []*AI {
	return append(s[:index], s[index+1:]...)
}
func (c *Character) DealPoisonDamageToPlayer(char *Character, dmg int) {
	if c == nil {
		log.Println("character is nil")
		return
	} else if c.Socket.Stats.HP <= 0 {
		return
	}
	if dmg > c.Socket.Stats.HP {
		dmg = c.Socket.Stats.HP
	}

	c.Socket.Stats.HP -= dmg
	if c.Socket.Stats.HP <= 0 {
		c.Socket.Stats.HP = 0
	}
	if !char.IsOnline {
		char = c
	}
	index := 5
	r := DEAL_POISON_DAMAGE
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(char.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(c.Socket.Stats.HP), 4, true), index) // ai current hp
	index += 4
	r.Insert(utils.IntToBytes(uint64(c.Socket.Stats.CHI), 4, true), index) // ai current chi
	char.Socket.Write(r)
	c.Socket.Write(r)
}

func (c *Character) DealPoisonDamageToMob(enemy *AI, dmg int) {
	if c == nil {
		log.Println("character is nil")
		return
	} else if enemy.HP <= 0 {
		return
	}
	if dmg > enemy.HP {
		dmg = enemy.HP
	}

	enemy.HP -= dmg
	if enemy.HP <= 0 {
		enemy.HP = 0
		CheckMobDead(enemy, c)
	}

	index := 5
	r := DEAL_POISON_DAMAGE
	r.Insert(utils.IntToBytes(uint64(enemy.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(enemy.HP), 4, true), index) // ai current hp
	index += 4
	r.Insert(utils.IntToBytes(uint64(enemy.CHI), 4, true), index) // ai current chi
	c.Socket.Write(r)

}

func (c *Character) DealPoisonDamageToAI(ai *AI) {
	/*if c == nil {
		log.Println("character is nil")
		return
	} else if c.Socket.Stats.HP <= 0 {
		return
	}
	if dmg > c.Socket.Stats.HP {
		dmg = c.Socket.Stats.HP
	}

	c.Socket.Stats.HP -= dmg
	if c.Socket.Stats.HP <= 0 {
		c.Socket.Stats.HP = 0
	}*/
	index := 5
	r := DEAL_BUFF_AI
	r.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(ai.HP), 4, true), index) // ai current hp
	index += 4
	r.Insert(utils.IntToBytes(uint64(ai.CHI), 4, true), index) // ai current chi
	r.Insert(utils.IntToBytes(uint64(257), 4, true), 22)       //BUFF ID
	c.Socket.Write(r)
}

func (c *Character) DealDamageToPlayer(char *Character, dmg int, skill bool, critical bool) {
	if c == nil {
		log.Println("character is nil")
		return
	} else if char.Socket.Stats.HP <= 0 {
		return
	}
	if dmg > char.Socket.Stats.HP {
		dmg = char.Socket.Stats.HP

	}

	char.Socket.Stats.HP -= dmg
	if char.Socket.Stats.HP <= 0 {
		char.Socket.Stats.HP = 0
	}

	/*char.Injury += 0.1
	injuryNumbers := char.CalculateInjury()
	injury1 := fmt.Sprintf("%x", injuryNumbers[1]) //0.7
	injury0 := fmt.Sprintf("%x", injuryNumbers[0]) //0.1
	injury3 := fmt.Sprintf("%x", injuryNumbers[3]) //17.48
	injury2 := fmt.Sprintf("%x", injuryNumbers[2]) //1.09
	injuryByte1 := string(injury1 + injury0)
	data, err := hex.DecodeString(injuryByte1)
	if err != nil {
		panic(err)
	}
	injuryByte2 := string(injury2 + injury3)
	data2, err := hex.DecodeString(injuryByte2)
	if err != nil {
		panic(err)
	}*/
	meditresp := utils.Packet{}
	if char.Meditating {
		meditresp = MEDITATION_MODE
		meditresp.Insert(utils.IntToBytes(uint64(char.PseudoID), 2, true), 6) // character pseudo id
		meditresp[8] = 0
		char.Meditating = false
	}
	buffs, err := FindBuffsByCharacterID(int(char.PseudoID))
	index := 5
	r := DEAL_DAMAGE
	r.Insert(utils.IntToBytes(uint64(char.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(char.Socket.Stats.HP), 4, true), index) // ai current hp
	index += 4
	r.Insert(utils.IntToBytes(uint64(char.Socket.Stats.CHI), 4, true), index) // ai current chi
	index += 4
	if skill {
		r.Overwrite([]byte{0xFF, 0xFF, 0xFF, 0x00}, index)
	}
	if critical {
		index += 4
		r.Overwrite([]byte{0x02}, index)
		index += 2
	}
	if err == nil {
		r.Overwrite(utils.IntToBytes(uint64(len(buffs)), 1, true), index) //BUFF ID
		index++
		//index = 22
		count := 0
		for _, buff := range buffs {
			r.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), index) //BUFF ID
			index += 4
			if count < len(buffs)-1 {
				r.Insert(utils.IntToBytes(uint64(0), 2, true), index) //BUFF ID
				index += 2
			}
			count++
		}
		index += 4
	} else {
		fmt.Println("Valami error: %s", err)
	}
	//index += 3
	r.Insert([]byte{0x00, 0x00, 0x00}, index) // INJURY
	index += 2
	r.SetLength(int16(binary.Size(r) - 6))
	c.Socket.Write(r)
	r.Concat(meditresp)
	char.Socket.Write(r)
}

func (c *Character) LoadQuests(questID, state int) {

	resp := utils.Packet{}
	index := 6
	resp = QUEST_HANDLER
	resp.Insert(utils.IntToBytes(uint64(questID), 4, true), index)
	index += 4
	resp.Insert(utils.IntToBytes(uint64(state), 1, true), index)
	index += 1
	npcID := 0
	var npPos *NpcPosition
	quest, _ := FindQuestByID(int64(questID))
	mquest, _ := FindPlayerQuestByID(quest.ID, c.ID)
	if mquest != nil {
		mquest.QuestState = state
		go mquest.Update()
	}
	if state == 2 || state == 4 {
		npcID, _ = c.GetQuestNPCID(int64(quest.FinishNPC))
		npPos = NPCPos[npcID]
	} else if state == 3 || state == 1 {
		npcID, _ = c.GetQuestNPCID(int64(quest.NPCID))
		npPos = NPCPos[npcID]
	}
	qindex := 6
	jelresp := utils.Packet{0xaa, 0x55, 0x09, 0x00, 0x31, 0x03, 0x55, 0xaa}
	jelresp.Insert(utils.IntToBytes(uint64(npPos.PseudoID), 2, true), qindex)
	qindex += 2
	jelresp.Insert(utils.IntToBytes(uint64(state), 1, true), qindex)
	qindex++
	jelresp.Insert(utils.IntToBytes(uint64(npcID), 4, true), qindex)
	resp.Concat(jelresp)
	c.Socket.Write(resp)
}
func (c *Character) LoadReturnQuests(questID, state int) ([]byte, error) {
	quest, err := FindQuestByID(int64(questID))
	if err != nil {
		fmt.Println(fmt.Sprintf("LoadError: %s", err.Error()))
	}
	resp := utils.Packet{}
	npcID := 0
	var npPos *NpcPosition
	/*if !quest.DropFromMobs && quest.request_items == nil{
		fmt.Println("KÉSZ IS A KÜLDI")
	}*/
	if state == 2 || state == 4 {
		npcID, _ = c.GetQuestNPCID(int64(quest.FinishNPC))
		npPos = NPCPos[npcID]
	} else if state == 3 || state == 1 {
		npcID, _ = c.GetQuestNPCID(quest.NPCID)
		npPos = NPCPos[npcID]
	}
	/*if quest.NPCID != int64(quest.FinishNPC){
		resetindex := 6
		resetresp := utils.Packet{0xaa,0x55,0x09,0x00,0x31,0x03,0x55,0xaa}
		resetnpcID,_ := c.GetQuestNPCID(quest.NPCID)
		resetPos := NPCPos[resetnpcID]
		resetresp.Insert(utils.IntToBytes(uint64(resetPos.PseudoID), 2, true),resetindex)
		resetindex+=2
		resetresp.Insert(utils.IntToBytes(uint64(0), 1, true),resetindex)
		resetindex++
		resetresp.Insert(utils.IntToBytes(uint64(quest.NPCID), 4, true),resetindex)
		resp.Concat(resetresp)
	}*/
	qindex := 6
	jelresp := utils.Packet{0xaa, 0x55, 0x09, 0x00, 0x31, 0x03, 0x55, 0xaa}
	jelresp.Insert(utils.IntToBytes(uint64(npPos.PseudoID), 2, true), qindex)
	qindex += 2
	jelresp.Insert(utils.IntToBytes(uint64(state), 1, false), qindex)
	qindex++
	jelresp.Insert(utils.IntToBytes(uint64(npcID), 4, true), qindex)
	resp.Concat(jelresp)
	return resp, nil
}

func (c *Character) DealDamage(ai *AI, dmg int, skill bool, critical bool) {

	if c == nil {
		log.Println("character is nil")
		return
	} else if ai.HP <= 0 {
		return
	}
	if critical {
		dmg = int(math.Round(float64(dmg) * 1.3))
	}
	//s := c.Socket
	if dmg > ai.HP {
		dmg = ai.HP
	}

	ai.HP -= dmg
	if ai.HP <= 0 {
		ai.HP = 0
	}
	ai.TargetPlayerID = int(c.PseudoID)
	d := ai.DamageDealers.Get(c.ID)
	if d == nil {
		ai.DamageDealers.Add(c.ID, &Damage{Damage: dmg, DealerID: c.ID})
	} else {
		d.(*Damage).Damage += dmg
		ai.DamageDealers.Add(c.ID, d)
	}

	if c.Invisible {
		buff, _ := FindBuffByCharacter(241, c.ID)
		if buff != nil {
			buff.Duration = 0
			go buff.Update()
		}

		buff, _ = FindBuffByCharacter(244, c.ID)
		if buff != nil {
			buff.Duration = 0
			go buff.Update()
		}

		if c.DuelID > 0 {
			opponent, _ := FindCharacterByID(c.DuelID)
			spawnData, _ := c.SpawnCharacter()

			r := utils.Packet{}
			r.Concat(spawnData)
			r.Overwrite(utils.IntToBytes(500, 2, true), 13) // duel state

			sock := GetSocket(opponent.UserID)
			if sock != nil {
				sock.Write(r)
			}
		}
	}
	/*	c.Injury += 0.1
		injuryNumbers := c.CalculateInjury()
		injury1 := fmt.Sprintf("%x", injuryNumbers[1]) //0.7
		injury0 := fmt.Sprintf("%x", injuryNumbers[0]) //0.1
		injury3 := fmt.Sprintf("%x", injuryNumbers[3]) //17.48
		injury2 := fmt.Sprintf("%x", injuryNumbers[2]) //1.09
		injuryByte1 := string(injury1 + injury0)
		data, err := hex.DecodeString(injuryByte1)
		if err != nil {
			panic(err)
		}
		injuryByte2 := string(injury2 + injury3)
		data2, err := hex.DecodeString(injuryByte2)
		if err != nil {
			panic(err)
		}*/
	buffs, err := FindBuffsByAiPseudoID(ai.PseudoID)
	npcID := uint64(NPCPos[ai.PosID].NPCID)
	npc := NPCs[int(npcID)]
	npcMaxHPHalf := (npc.MaxHp / 2) / 10
	index := 5
	r := DEAL_DAMAGE
	r.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(ai.HP), 4, true), index) // ai current hp
	index += 4
	r.Insert(utils.IntToBytes(uint64(npcMaxHPHalf), 4, true), index) // ai current chi
	index += 4
	if skill {
		r.Overwrite([]byte{0xFF, 0xFF, 0xFF, 0x00}, index)
	}
	if critical {
		index += 4
		r.Overwrite([]byte{0x02}, index)
		index += 2
	}
	if err == nil {
		r.Overwrite(utils.IntToBytes(uint64(len(buffs)), 1, true), index) //BUFF ID
		index++
		//index = 22
		count := 0
		for _, buff := range buffs {
			r.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), index) //BUFF ID
			index += 4
			if count < len(buffs)-1 {
				r.Insert(utils.IntToBytes(uint64(0), 2, true), index) //BUFF ID
				index += 2
			}
			count++
		}
		index += 4
	} else {
		fmt.Println("Valami error: %s", err)
	}
	//index += 3
	r.Insert([]byte{0x00, 0x00, 0x00}, index) // INJURY
	index += 2
	r.SetLength(int16(binary.Size(r) - 6))
	p := &nats.CastPacket{CastNear: true, MobID: ai.ID, Data: r, Type: nats.MOB_ATTACK}
	if err := p.Cast(); err != nil {
		log.Println("deal damage broadcast error:", err)
		return
	}
	CheckMobDead(ai, c)
	//c.Socket.Write(r)

}

func CheckMobDead(ai *AI, c *Character) {
	npcPos := NPCPos[ai.PosID]
	if npcPos == nil {
		log.Println("npc pos is nil")
		return
	}

	npc := NPCs[npcPos.NPCID]
	if npc == nil {
		log.Println("npc is nil")
		return
	}
	/*if npc.ID == 50065{
		health := ai.HP - (ai.HP /20)
		if ai.HP <= health{

		}
	}*/

	if !npcPos.Attackable {
		go ai.DropHandler(c)
	}
	if ai.HP <= 0 { // ai died
		DeleteBuffsByAiPseudoID(ai.PseudoID)
		if funk.Contains(BossMobsID, npc.ID) {
			BossEvents[ai.EventID].LastKilledAt = null.NewTime(time.Now(), true)
			itIsActive, err := isMobEventActiveYetByID(ai.EventID)
			if err == nil {
				if !itIsActive {
					MakeAnnouncement("Boss event finished, it was the last boss. See you at the next event.")
					BossEvents[ai.EventID].Activated = false
					ai.Handler = nil
					ai.PseudoID = 0
					ai.Once = true
				} else {
					MakeAnnouncement(fmt.Sprintf("%s has been annihilated. It will revive in %d hours and %d minutes", npc.Name, BossEvents[ai.EventID].RespawnTime.Time.Hour(), BossEvents[ai.EventID].RespawnTime.Time.Minute()))
				}
			}
			go BossEvents[ai.EventID].Update()
		}
		if ai.Once {
			ai.Handler = nil
		} else {
			respawnMin := int(float64(npcPos.RespawnTime) * 0.75)
			respawnMax := int(float64(npcPos.RespawnTime) * 1.25)
			dynamicRespawnTime := rand.Intn(respawnMax-respawnMin+1) + respawnMin
			time.AfterFunc(time.Duration(dynamicRespawnTime)*time.Second/2, func() { // respawn mob n secs later
				curCoordinate := ConvertPointToLocation(ai.Coordinate)
				minCoordinate := ConvertPointToLocation(npcPos.MinLocation)
				maxCoordinate := ConvertPointToLocation(npcPos.MaxLocation)

				X := utils.RandFloat(minCoordinate.X, maxCoordinate.X)
				Y := utils.RandFloat(minCoordinate.Y, maxCoordinate.Y)

				X = (X / 3) + 2*curCoordinate.X/3
				Y = (Y / 3) + 2*curCoordinate.Y/3

				coordinate := &utils.Location{X: X, Y: Y}
				ai.TargetLocation = *coordinate
				ai.SetCoordinate(coordinate)

				ai.HP = npc.MaxHp
				ai.IsDead = false
			})
		}
		if c.Selection == int(ai.PseudoID) {
			c.Selection = 0
		}
		if npc.ID == 424201 && WarStarted {
			OrderPoints -= 200
		} else if npc.ID == 424202 && WarStarted {
			ShaoPoints -= 200
		}

		exp := int64(0)
		if c.Level <= 100 {
			exp = npc.Exp
		} else if c.Level <= 200 {
			exp = npc.DivineExp
		} else {
			exp = npc.DarknessExp
		}

		// EXP gained
		r, levelUp := c.AddExp(exp)
		if levelUp {
			statData, err := c.GetStats()
			if err == nil {
				c.Socket.Write(statData)
			}
		}
		c.Socket.Write(r)

		// EXP gain for party members
		party := FindParty(c)
		if party != nil {
			members := funk.Filter(party.GetMembers(), func(m *PartyMember) bool {
				return m.Accepted || m.ID == c.ID
			}).([]*PartyMember)
			members = append(members, &PartyMember{Character: party.Leader, Accepted: true})

			//coordinate := ConvertPointToLocation(c.Coordinate) //Quoted out because EXP share is now for the whole map
			for _, m := range members {
				user, err := FindUserByID(m.UserID)
				if err != nil || user == nil || (c.Level-m.Level) > 35 {
					continue
				}

				//memberCoordinate := ConvertPointToLocation(m.Coordinate) //Quoted out because EXP share is now for the whole map

				if m.ID == c.ID && !m.Accepted {
					break
				}

				if m.ID == c.ID || m.Map != c.Map || c.Socket.User.ConnectedServer != user.ConnectedServer || m.Socket.Stats.HP <= 0 { // || utils.CalculateDistance(coordinate, memberCoordinate) > 300
					continue
				}

				exp := int64(0)
				if m.Level <= 100 {
					exp = npc.Exp
				} else if m.Level <= 200 {
					exp = npc.DivineExp
				} else {
					exp = npc.DarknessExp
				}

				exp /= int64(len(members))

				r, levelUp := m.AddExp(exp)
				if levelUp {
					statData, err := m.GetStats()
					if err == nil {
						m.Socket.Write(statData)
					}
				}
				m.Socket.Write(r)
			}
		}
		//GIVE 200+ ataraxia ITEM
		if npc.ID == 43401 {
			if c.Exp >= 544951059310 && c.Level == 200 {
				resp := utils.Packet{}
				slots, err := c.InventorySlots()
				if err != nil {
				}
				reward := NewSlot()
				reward.ItemID = int64(90000304)
				reward.Quantity = 1
				_, slot, _ := c.AddItem(reward, -1, true)
				if slot > 0 {
					resp.Concat(slots[slot].GetData(slot))
					c.Socket.Write(resp)
				} else {
					c.Socket.Write(messaging.InfoMessage("Not enough space in your inventory."))
				}
				c.Socket.Write(messaging.InfoMessage("You kill the Wyrm, now you can make the transformation."))
			}
		}
		//60006	Underground(EXP)
		//60011	Waterfall(EXP)
		//60016	Forest(EXP)
		//60021	Sky Garden(EXP)
		//FIVE ELEMENT CASTLE
		go func() {
			/*	currentTime := time.Now()
				diff := mail.ExpiresAt.Time.Sub(currentTime)
				if diff < 0 {
					MessageExpired(mail.ID)
					continue
				}*/
			exp := time.Now().UTC().Add(time.Hour * 2)
			if npc.ID == 423308 { //HWARANG GUARDIAN STATUE //SOUTHERN WOOD TEMPLE
				if c.GuildID > 0 {
					FiveClans[0].ClanID = c.GuildID
					FiveClans[0].ExpiresAt = null.TimeFrom(exp)
					FiveClans[0].Update()
					guild, err := FindGuildByID(c.GuildID)
					if err != nil {
						return
					}
					allmembers, _ := guild.GetMembers()
					for _, clanmembers := range allmembers {
						char, err := FindCharacterByID(clanmembers.ID)
						if err != nil {
							continue
						}
						infection := BuffInfections[60006]
						buff := &Buff{ID: int(60006), CharacterID: char.ID, Name: infection.Name, ExpRate: infection.ExpRate, StartedAt: char.Epoch, Duration: 7200}
						err = buff.Create()
						if err != nil {
							continue
						}
					}
				}
			} else if npc.ID == 423310 { //SUGUN GUARDIAN STATUE //LIGHTNING HILL TEMPLE

			} else if npc.ID == 423312 { //CHUNKYUNG GUARDIAN STATUE //OCEAN ARMY TEMPLE

			} else if npc.ID == 423314 { //MOKNAM GUARDIAN STATUE //FLAME WOLF TEMPLE

			} else if npc.ID == 423316 { //JISU GUARDIAN STATUE //WESTERN LAND TEMPLE

			}
		}()
		//QUEST ITEMS DROP
		if funk.Contains(c.questMobsIDs, npc.ID) {
			resp := utils.Packet{}
			item, itemcount, questID := c.GetQuestItemsDrop(npc.ID)
			itemData, slotID, err := c.AddItem(&InventorySlot{ItemID: item, Quantity: 1}, -1, true)
			if err != nil {
				fmt.Println("Quest item add error!")
			}
			qslots, _ := c.InventorySlots()
			qitem := qslots[slotID]
			if qitem.Quantity <= uint(itemcount) {
				resp.Concat(*itemData)
				itemName := Items[item].Name
				mess := messaging.InfoMessage(fmt.Sprintf("Acquired the %s", itemName))
				resp.Concat(mess)
				c.Socket.Write(resp)
			} else if qitem.Quantity >= uint(itemcount) {
				quest, _ := FindPlayerQuestByID(questID, c.ID)
				quest.QuestState = 4
				quest.Update()
				test, _ := c.LoadReturnQuests(quest.ID, quest.QuestState)
				resp.Concat(test)
				c.Socket.Write(resp)
			}
		}

		// PTS gained LOOT
		dropMaxLevel := int(npc.Level + 75)
		if c.Level <= dropMaxLevel {
			c.PTS++
			if c.PTS%50 == 0 {
				r = c.GetPTS()
				c.HasLot = true
				c.Socket.Write(r)
			}
		}
		// Gold dropped
		goldDrop := int64(float64(npc.GoldDrop) + (float64(npc.GoldDrop) * c.Socket.Stats.GoldRate))
		if goldDrop > 0 {
			amount := uint64(utils.RandInt(goldDrop/2, goldDrop))
			r = c.LootGold(amount)
			c.Socket.Write(r)
		}

		claimer, err := ai.FindClaimer()
		if funk.Contains(FiveclanMobs, npc.ID) {
			if claimer.GuildID != -1 {
				for num, id := range FiveclanMobs {
					if id == npc.ID {
						exp := time.Now().UTC().Add(time.Hour * 2)
						FiveClans[num+1].ExpiresAt = null.TimeFrom(exp)
						FiveClans[num+1].ClanID = claimer.GuildID
						go FiveClans[num+1].Update()
						CaptureFive(num+1, claimer)
					}
				}
			}
		}
		//Item dropped
		go func() {

			if err != nil || claimer == nil {
				return
			}

			dropMaxLevel := int(npc.Level + 75)
			if c.Level <= dropMaxLevel {
				ai.DropHandler(claimer)
			}
			time.AfterFunc(time.Second, func() {
				ai.DamageDealers.Clear()
			})
		}()

		time.AfterFunc(time.Second, func() { // disappear mob 1 sec later
			ai.TargetPlayerID = 0
			ai.TargetPetID = 0
			ai.IsDead = true
		})
	} else if ai.TargetPlayerID == 0 {
		ai.IsMoving = false
		ai.MovementToken = 0
		ai.TargetPlayerID = c.ID
	} else {
		ai.IsMoving = false
		ai.MovementToken = 0
	}
}
func (c *Character) GetPetStats() []byte {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil
	}

	petSlot := slots[0x0A]
	pet := petSlot.Pet
	if pet == nil {
		return nil
	}

	resp := utils.Packet{}
	resp = petSlot.GetPetStats(c)
	resp.Concat(petSlot.GetData(0x0A))
	return resp
}

func (c *Character) StartPvP(timeLeft int) {

	info, resp := "", utils.Packet{}
	if timeLeft > 0 {
		info = fmt.Sprintf("Duel will start %d seconds later.", timeLeft)
		time.AfterFunc(time.Second, func() {
			c.StartPvP(timeLeft - 1)
		})

	} else if c.DuelID > 0 {
		info = "Duel has started."
		resp.Concat(c.OnDuelStarted())
	}

	resp.Concat(messaging.InfoMessage(info))
	c.Socket.Write(resp)
}

func (c *Character) CanAttack(enemy *Character) bool {
	if c.IsinWar && enemy.IsinWar && c.Faction == enemy.Faction && !funk.Contains(ArenaZones, c.Map) {
		return false
	}
	return (c.DuelID == enemy.ID && c.DuelStarted) || (funk.Contains(PvPZones, c.Map) || funk.Contains(PVPServers, int16(c.Socket.User.ConnectedServer)) && c.PartyID != enemy.PartyID || (funk.Contains(ArenaZones, c.Map)) && c.PartyID != enemy.PartyID)
}

func (c *Character) OnDuelStarted() []byte {

	c.DuelStarted = true
	statData, _ := c.GetStats()

	opponent, err := FindCharacterByID(c.DuelID)
	if err != nil || opponent == nil {
		return nil
	}

	opData, err := opponent.SpawnCharacter()
	if err != nil || opData == nil || len(opData) < 13 {
		return nil
	}
	r := utils.Packet{}
	r.Concat(opData)
	r.Overwrite(utils.IntToBytes(500, 2, true), 13) // duel state

	resp := utils.Packet{}
	resp.Concat(opponent.GetHPandChi())
	resp.Concat(r)
	resp.Concat(statData)
	resp.Concat([]byte{0xAA, 0x55, 0x02, 0x00, 0x2A, 0x04, 0x55, 0xAA})
	return resp
}

func (c *Character) HasAidBuff() bool {
	slots, err := c.InventorySlots()
	if err != nil {
		return false
	}

	return len(funk.Filter(slots, func(s *InventorySlot) bool {
		return (s.ItemID == 13000170 || s.ItemID == 13000171) && s.Activated && s.InUse
	}).([]*InventorySlot)) > 0
}

func (c *Socket) Respawn() {
	resp := utils.Packet{}

	stat := c.Stats

	gomap, _ := c.Character.ChangeMap(1, nil)
	c.Write(gomap)

	c.Character.IsActive = false
	stat.HP = stat.MaxHP
	stat.CHI = stat.MaxCHI
	c.Character.Respawning = false
	hpData := c.Character.GetHPandChi()
	resp.Concat(hpData)

	p := nats.CastPacket{CastNear: true, CharacterID: c.Character.ID, Type: nats.PLAYER_RESPAWN}
	p.Cast()

	go c.Character.ActivityStatus(30)
}
