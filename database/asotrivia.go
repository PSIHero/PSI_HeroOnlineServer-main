package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"PsiHero/messaging"
	"PsiHero/utils"

	"github.com/thoas/go-funk"
	"gopkg.in/gorp.v1"
)

var (
	TriviaEventStarted = false
	TriviaCanJoin      = false
	ActiveEventTrivia  = make(map[int]*ActiveTrivia)
	NPC_MENU           = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x57, 0x02, 0x55, 0xAA}
	QuestionsList      = make(map[int]*QuestionsItem)
)

type QuestionsItem struct {
	ID       int    `db:"id"`
	Question string `db:"question"`
	Answers  bool   `db:"answer"`
}

func (e *QuestionsItem) Create() error {
	return db.Insert(e)
}

func (e *QuestionsItem) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *QuestionsItem) Delete() error {
	_, err := db.Delete(e)
	return err
}

func (e *QuestionsItem) Update() error {
	_, err := db.Update(e)
	return err
}

func getQuestionsItem() error {
	var triviaquestionItems []*QuestionsItem
	query := `select * from data.aso_trivia`

	if _, err := db.Select(&triviaquestionItems, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("QuestionsItem: %s", err.Error())
	}

	for _, b := range triviaquestionItems {
		QuestionsList[b.ID] = b
	}

	return nil
}

func RefreshQuestionsItem() error {
	var triviaquestionItems []*QuestionsItem
	query := `select * from data.aso_trivia`

	if _, err := db.Select(&triviaquestionItems, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("QuestionsItem: %s", err.Error())
	}

	for _, b := range triviaquestionItems {
		QuestionsList[b.ID] = b
	}

	return nil
}

type ActiveTrivia struct {
	ActiveTriviaID    int          `db:"-" json:"-"`
	AllPlayers        []*Character `db:"-" json:"-"`
	StayedPlayers     []*Character `db:"-" json:"-"`
	UsedQuestions     []int        `db:"-" json:"-"`
	GeneratedQuestion int          `db:"-" json:"-"`
}

func StartInTriviaTimer() {
	timein := time.Now().Add(time.Minute * 5)
	deadtime := timein.Format(time.RFC3339)

	v, err := time.Parse(time.RFC3339, deadtime)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	min, sec := secondsToMinutes(300)
	msg := fmt.Sprintf("%d minutes %d second after the ASO Trivia will start.", min, sec)
	MakeAnnouncement(msg)
	for range time.Tick(1 * time.Second) {
		timeRemaining := getTriviaTimeRemaining(v)
		if timeRemaining.t <= 0 {
			TriviaCanJoin = false
			ActiveEventTrivia[int(1)].StayedPlayers = ActiveEventTrivia[int(1)].AllPlayers
			//startTriviaEvent()
			break
		}
		if timeRemaining.t%10 == 0 {
			min, sec := secondsToMinutes(timeRemaining.t)
			msg := fmt.Sprintf("%d minutes %d second after the ASO Trivia will start.", min, sec)
			MakeAnnouncement(msg)
		}
	}
}

func StartEventTriviaTimer() {
	timein := time.Now().Add(time.Minute * 20)
	deadtime := timein.Format(time.RFC3339)

	v, err := time.Parse(time.RFC3339, deadtime)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for range time.Tick(1 * time.Second) {
		timeRemaining := getTriviaTimeRemaining(v)
		if timeRemaining.t <= 0 {
			TriviaCanJoin = false
			ActiveEventTrivia[int(1)].StayedPlayers = ActiveEventTrivia[int(1)].AllPlayers
			break
		}

	}
}

func (self *ActiveTrivia) RemoveStayedPlayers(char *Character) {
	for i, other := range self.StayedPlayers {
		if other == char {
			self.StayedPlayers = append(self.StayedPlayers[:i], self.StayedPlayers[i+1:]...)
			break
		}
	}
}
func StartTriviaEvent(reward int64, amount int64) {
	ok := generateRandomQuestion()
	if ok {
		//time.AfterFunc(time.Second * 10, func() {
		//allChars := funk.Values(characters)
		chars := funk.Filter(ActiveEventTrivia[int(1)].StayedPlayers, func(c *Character) bool {
			return c.IsOnline || c.IsInTraviaEvent
		}).([]*Character)
		resp := utils.Packet{}
		for _, char := range chars {
			mesresp := messaging.InfoMessage(QuestionsList[ActiveEventTrivia[int(1)].GeneratedQuestion].Question)
			char.TriviaSelected = 3 // THIS MEAN NON SELECTED
			resp = GetTriviaMenu(24202, 1015040093, 0, []int{3342, 3343})
			resp.Concat(mesresp)
			char.Socket.Write(resp)
		}
		//})
		time.AfterFunc(time.Second*10, func() {
			//StayedPlayers := funk.Values(characters)
			chars := funk.Filter(ActiveEventTrivia[int(1)].StayedPlayers, func(c *Character) bool {
				return c.IsOnline || c.IsInTraviaEvent
			}).([]*Character)
			answerInt := 0
			if QuestionsList[ActiveEventTrivia[int(1)].GeneratedQuestion].Answers {
				answerInt = 1
			} else {
				answerInt = 0
			}
			resp := utils.Packet{}
			for _, char := range chars {
				if !(char.TriviaSelected == answerInt) {
					char.IsInTraviaEvent = false
					ActiveEventTrivia[int(1)].RemoveStayedPlayers(char)
				}
				mesresp := messaging.InfoMessage(fmt.Sprintf("Time is up! The correct answer is: %t", QuestionsList[ActiveEventTrivia[int(1)].GeneratedQuestion].Answers))
				resp = GetTriviaMenu(24202, 0, 0, []int{0})
				resp.Concat(mesresp)
				if char.TriviaSelected == answerInt {
					itemData, _, err := char.AddItem(&InventorySlot{ItemID: reward, Quantity: uint(amount)}, -1, false)
					if err != nil {
						//return nil, false, err
					} else if itemData == nil {
						//return nil, false, nil
					}
					resp.Concat(*itemData)
				}
				char.Socket.Write(resp)
			}
		})
	}
}
func EndTriviaEvent(reward int64, amount int64) {
	resp := utils.Packet{}
	chars := funk.Filter(ActiveEventTrivia[int(1)].StayedPlayers, func(c *Character) bool {
		return c.IsOnline || c.IsInTraviaEvent
	}).([]*Character)
	for _, char := range chars {
		resp = GetTriviaMenu(24202, 0, 0, []int{0})
		//if char.TriviaSelected == answerInt{
		char.IsInTraviaEvent = false
		itemData, _, err := char.AddItem(&InventorySlot{ItemID: reward, Quantity: uint(amount)}, -1, false)
		if err != nil {
			//return nil, false, err
		} else if itemData == nil {
			//return nil, false, nil
		}
		resp.Concat(*itemData)
		//}
		MakeAnnouncement(fmt.Sprintf("The winner is: %s", char.Name))
		char.Socket.Write(resp)
	}
	allchars := funk.Filter(ActiveEventTrivia[int(1)].AllPlayers, func(c *Character) bool {
		return c.IsOnline || c.Map == 42
	}).([]*Character)
	for _, char := range allchars {
		resp, _ := char.ChangeMap(1, nil)
		char.IsInTraviaEvent = false
		char.Socket.Write(resp)
	}
	ActiveEventTrivia[int(1)].AllPlayers = nil
	ActiveEventTrivia[int(1)].StayedPlayers = nil
}
func generateRandomQuestion() bool {
	randomID := utils.RandInt(1, int64(len(QuestionsList)))
	ok := true
	for ok {
		fmt.Println("BEJÖN IDE")
		if !funk.Contains(ActiveEventTrivia[int(1)].UsedQuestions, int(randomID)) {
			ok = false
			ActiveEventTrivia[int(1)].GeneratedQuestion = int(randomID)
			ActiveEventTrivia[int(1)].UsedQuestions = append(ActiveEventTrivia[int(1)].UsedQuestions, int(randomID))
			return true
			//return int(randomID)
		}

	}
	return false
}

func GetTriviaMenu(npcID, textID, index int, actions []int) []byte {
	resp := NPC_MENU
	resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6)         // npc id
	resp.Insert(utils.IntToBytes(uint64(textID), 4, true), 10)       // text id
	resp.Insert(utils.IntToBytes(uint64(len(actions)), 1, true), 14) // action length

	counter, length := 15, int16(11)
	for i, action := range actions {
		resp.Insert(utils.IntToBytes(uint64(action), 4, true), counter) // action
		counter += 4
		resp.Insert(utils.IntToBytes(uint64(npcID), 2, true), counter) // npc id
		counter += 2

		actIndex := int(index) + (i+1)<<(len(actions)*3)
		resp.Insert(utils.IntToBytes(uint64(actIndex), 2, true), counter) // action index
		counter += 2

		length += 8
	}

	resp.SetLength(length)
	return resp
}

func getTriviaTimeRemaining(t time.Time) countdown {
	currentTime := time.Now()
	difference := t.Sub(currentTime)

	total := int(difference.Seconds())
	days := int(total / (60 * 60 * 24))
	hours := int(total / (60 * 60) % 24)
	minutes := int(total/60) % 60
	seconds := int(total % 60)
	return countdown{
		t: total,
		d: days,
		h: hours,
		m: minutes,
		s: seconds,
	}
}
