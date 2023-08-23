package database

import (
	"log"

	"PsiHero/utils"

	"github.com/xuri/excelize/v2"
)

type CraftItem struct {
	ID                 int
	Material1          int
	Material2          int
	Material3          int
	Material4          int
	Material5          int
	Material6          int
	Material1Count     int
	Material2Count     int
	Material3Count     int
	Material4Count     int
	Material5Count     int
	Material6Count     int
	Probability1       int
	Probability2       int
	Probability3       int
	Probability1Result int
	Probability2Result int
	Probability3Result int
	Cost               int64
}

var (
	CraftItems = make(map[int]*CraftItem)
)

func getCraftItem() error {
	log.Print("Reading Production table...")
	f, err := excelize.OpenFile("data/tb_production.xlsx")
	if err != nil {
		return err
	}
	defer f.Close()
	// Get all the rows in the Sheet1.
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return err
	}
	for index, row := range rows {
		if index == 0 {
			continue
		}
		CraftItems[utils.StringToInt(row[1])] = &CraftItem{
			ID:                 utils.StringToInt(row[1]),
			Material1:          utils.StringToInt(row[4]),
			Material2:          utils.StringToInt(row[5]),
			Material3:          utils.StringToInt(row[6]),
			Material4:          utils.StringToInt(row[7]),
			Material5:          utils.StringToInt(row[8]),
			Material6:          utils.StringToInt(row[9]),
			Material1Count:     utils.StringToInt(row[10]),
			Material2Count:     utils.StringToInt(row[11]),
			Material3Count:     utils.StringToInt(row[12]),
			Material4Count:     utils.StringToInt(row[13]),
			Material5Count:     utils.StringToInt(row[14]),
			Material6Count:     utils.StringToInt(row[15]),
			Probability1Result: utils.StringToInt(row[16]),
			Probability2Result: utils.StringToInt(row[17]),
			Probability3Result: utils.StringToInt(row[18]),
			Probability1:       utils.StringToInt(row[19]),
			Probability2:       utils.StringToInt(row[20]),
			Probability3:       utils.StringToInt(row[21]),
			Cost:               int64(utils.StringToInt(row[25])),
		}
	}
	return nil
}
