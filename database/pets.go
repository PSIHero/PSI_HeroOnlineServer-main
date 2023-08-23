package database

import (
	"PsiHero/utils"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	gorp "gopkg.in/gorp.v1"
)

var (
	Pets        = make(map[int64]*Pet)
	DPSPets     = []int64{40024, 40026, 40027, 40028}
	TankPets    = []int64{40029, 40032, 40033, 40034}
	SupportPets = []int64{40096, 40099}
)

type Pet struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Evolution     int16   `json:"evolution"`
	Level         int16   `json:"level"`
	TargetLevel   int16   `json:"target_level"`
	EvolvedID     int64   `json:"evolved_id"`
	BaseSTR       int     `json:"base_str"`
	AdditionalSTR int     `json:"additional_str"`
	BaseDEX       int     `json:"base_dex"`
	AdditionalDEX int     `json:"additional_dex"`
	BaseINT       int     `json:"base_int"`
	AdditionalINT int     `json:"additional_int"`
	BaseHP        int     `json:"base_hp"`
	AdditionalHP  int     `json:"additional_hp"`
	BaseChi       int     `json:"base_chi"`
	AdditionalChi float64 `json:"additional_chi"`
	SkillID       string  `json:"skill_ids"`
	Combat        bool    `json:"combat"`
	RunningSpeed  float64 `json:"running_speed"`
	PetCardItemID int64   `json:"petcard_id"`
}

func (e *Pet) Create() error {
	return db.Insert(e)
}

func (e *Pet) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *Pet) Update() error {
	_, err := db.Update(e)
	return err
}

func (e *Pet) Delete() error {
	_, err := db.Delete(e)
	return err
}

func GetAllPets() error {

	var unlockedColums = []int16{1, 2, 20, 21, 22, 23, 27, 28, 29, 30, 31, 32, 39, 40, 41, 42, 115}
	f, err := excelize.OpenFile("data\\tb_PetTable.xlsx")
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer func() {
		// Close the spreadsheet.
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		fmt.Println(err)
		return err
	}

	for rowIndex, row := range rows {
		if rowIndex == 0 { //SKIP HEADER
			continue
		}
		isCombat, _ := strconv.Atoi(row[unlockedColums[16]])
		isCombatBool := isCombat == 2
		var petJson = `{
			"id":` + row[unlockedColums[0]] + `,
			"evolution":` + row[unlockedColums[2]] + `,
			"level":` + row[unlockedColums[3]] + `,
			"target_level":` + row[unlockedColums[4]] + `,
			"evolved_id":` + row[unlockedColums[5]] + `,
			"base_str":` + row[unlockedColums[6]] + `,
			"additional_str":` + row[unlockedColums[7]] + `,
			"base_dex":` + row[unlockedColums[8]] + `,
			"additional_dex":` + row[unlockedColums[9]] + `,
			"base_int":` + row[unlockedColums[10]] + `,
			"additional_int":` + row[unlockedColums[11]] + `,
			"base_hp":` + row[unlockedColums[12]] + `,
			"additional_hp":` + row[unlockedColums[13]] + `,
			"base_chi":` + row[unlockedColums[14]] + `,
			"additional_chi":` + row[unlockedColums[15]] + `,
			"running_speed":` + row[71] + `,
			"combat":` + strconv.FormatBool(isCombatBool) + `
			}`
		var skillIDsColumns [3]string
		for i := 0; i < 3; i++ {
			skillIDsColumns[i] = row[75+i]
		}
		skillIDJson := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(skillIDsColumns)), ","), "[]")
		skillIDJson = fmt.Sprintf("{%s}", skillIDJson)
		id, _ := strconv.ParseInt(row[unlockedColums[0]], 10, 64)
		var detailsEntity *Pet
		in := []byte(petJson)
		err := json.Unmarshal(in, &detailsEntity)
		if err != nil {
			log.Printf("error decoding response: %v", err)
			if e, ok := err.(*json.SyntaxError); ok {
				log.Printf("syntax error at byte offset %d", e.Offset)
			}
			log.Printf("response: %q", in)
			return err
		}
		detailsEntity.Name = row[unlockedColums[1]]
		detailsEntity.SkillID = skillIDJson
		Pets[id] = detailsEntity
	}
	return nil
}

func (e *Pet) GetSkills() []int {
	probs := strings.Trim(e.SkillID, "{}")
	sProbs := strings.Split(probs, ",")

	var arr []int
	for _, sProb := range sProbs {
		d, _ := strconv.Atoi(sProb)
		arr = append(arr, d)
	}
	return arr
}
func (e *Pet) LoadPetsSkills() []byte {
	resp := utils.Packet([]byte{0xaa, 0x55, 0x09, 0x00, 0x81, 0x07, 0x0a, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa})
	r := COMBAT_SKILL_BOOK

	r[6] = byte(5)                                       // book index
	r.Insert(utils.IntToBytes(uint64(e.ID), 4, true), 7) // book id
	c, index, length := 0, 14, int16(10)
	for _, skill := range e.GetSkills() {
		if skill == 0 {
			continue
		}
		//info := SkillInfos[skill]

		r.Insert(utils.IntToBytes(uint64(skill), 4, true), index) // skill id
		index += 4

		r.Insert([]byte{byte(5)}, index) // skill plus
		index++

		c++
		length += 5
	}
	r.SetLength(int16(binary.Size(r) - 6))
	resp.Concat(r)
	return resp
}
func RemovePetsSkills(id int64) []byte {
	resp := SKILL_REMOVED
	resp[8] = 5
	resp.Insert(utils.IntToBytes(uint64(id), 4, true), 9) // book id

	resp.SetLength(int16(binary.Size(resp) - 6))
	return resp
}
