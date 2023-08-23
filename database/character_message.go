package database

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	null "gopkg.in/guregu/null.v3"
)

var (
	MailMessages = make(map[int]*MailMessage)
)

type MailMessage struct {
	ID         int       `db:"id" json:"id"`
	SenderID   int       `db:"sender_id" json:"sender_id"`
	ReceiverID int       `db:"receiver_id" json:"receiver_id"`
	Title      string    `db:"title" json:"title"`
	Content    string    `db:"content" json:"content"`
	Gold       int64     `db:"gold" json:"gold"`
	ItemsArr   string    `db:"items"`
	ExpiresAt  null.Time `db:"expires_at" json:"expires_at"`
	IsOpened   bool      `db:"is_opened" json:"is_opened"`
	IsReceived bool      `db:"is_received" json:"is_received"`
}

func (b *MailMessage) Delete() error {
	_, err := db.Delete(b)
	return err
}

func (e *MailMessage) Create() error {
	return db.Insert(e)
}

func (b *MailMessage) Update() error {
	_, err := db.Update(b)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error: %s", err.Error()))
	}
	return err
}
func (e *MailMessage) GetItems() []int {
	items := strings.Trim(e.ItemsArr, "{}")
	sItems := strings.Split(items, ",")

	var arr []int
	for _, sItem := range sItems {
		d, _ := strconv.Atoi(sItem)
		arr = append(arr, d)
	}
	return arr
}

func FindMailsByCharacterID(characterID int) ([]*MailMessage, error) {

	var arr []*MailMessage
	query := `select * from hops.characters_mails where receiver_id = $1`
	if _, err := db.Select(&arr, query, characterID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindMailsByCharacterID: %s", err.Error())
	}

	return arr, nil
}

func FindMailByID(id int) (*MailMessage, error) {

	mail := &MailMessage{}
	query := `select * from "hops".characters_mails where "id" = $1`

	if err := db.SelectOne(&mail, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindMailByID: %s", err.Error())
	}

	return mail, nil
}

func (slot *MailMessage) SetMessageItems(i int, code int) {
	upgs := slot.GetItems()
	upgs[i] = code
	slot.SetUpgrades(upgs)
}

func (slot *MailMessage) SetUpgrades(upgs []int) {
	slot.ItemsArr = fmt.Sprintf("{%s}", strings.Trim(strings.Join(strings.Fields(fmt.Sprint(upgs)), ","), "[]"))
}
