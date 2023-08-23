package auth

import (
	"crypto/subtle"
	"fmt"
	"time"

	"PsiHero/database"
	"PsiHero/logging"
	"PsiHero/utils"

	"gopkg.in/guregu/null.v3"
)

type LoginHandler struct {
	password string
	username string
}

var (
	USER_NOT_FOUND = utils.Packet{0xAA, 0x55, 0x23, 0x00, 0x00, 0x01, 0x00, 0x1F, 0x4D, 0x69, 0x73, 0x6D, 0x61, 0x74, 0x63, 0x68, 0x20, 0x41, 0x63, 0x63, 0x6F, 0x75, 0x6E, 0x74, 0x20, 0x49, 0x44, 0x20, 0x6F, 0x72, 0x20, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6F, 0x72, 0x64, 0x55, 0xAA}
	LOGGED_IN      = utils.Packet{0xaa, 0x55, 0x57, 0x00, 0x00, 0x01, 0x01, 0x40, 0x30, 0x42, 0x41, 0x45, 0x35, 0x32, 0x46, 0x45, 0x34, 0x32, 0x30, 0x44, 0x41, 0x35, 0x39, 0x32, 0x30, 0x41, 0x30, 0x33, 0x39, 0x46, 0x31, 0x41, 0x39, 0x30, 0x38, 0x34, 0x46, 0x31, 0x38, 0x38, 0x34, 0x41, 0x39, 0x36, 0x33, 0x44, 0x34, 0x30, 0x45, 0x38, 0x41, 0x39, 0x45, 0x44, 0x37, 0x35, 0x44, 0x35, 0x43, 0x41, 0x45, 0x43, 0x31, 0x46, 0x43, 0x44, 0x39, 0x45, 0x44, 0x33, 0x31, 0x38, 0x00, 0x00, 0xdb, 0x89, 0x2d, 0x06, 0x55, 0xaa}
	USER_BANNED    = utils.Packet{0xAA, 0x55, 0x36, 0x00, 0x00, 0x01, 0x00, 0x32, 0x59, 0x6F, 0x75, 0x72, 0x20, 0x61, 0x63, 0x63, 0x6F, 0x75, 0x6E, 0x74, 0x20, 0x68, 0x61, 0x73, 0x20, 0x62, 0x65, 0x65, 0x6E, 0x20, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6C, 0x65, 0x64, 0x20, 0x75, 0x6E, 0x74, 0x69, 0x6C, 0x20, 0x5B, 0x5D, 0x2E, 0x55, 0xAA}

	logger = logging.Logger
)

func (lh *LoginHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	index := 9
	uNameLen := int(utils.BytesToInt(data[index:index+1], false))
	lh.username = string(data[index+1 : index+uNameLen+1])
	lh.password = string(data[index+uNameLen+2 : index+uNameLen+66])

	return lh.login(s)
}

func (lh *LoginHandler) login(s *database.Socket) ([]byte, error) {
	var user *database.User
	var err error
	user, err = database.FindUserByName(lh.username)
	if err != nil {
		return nil, err
	}

	if user == nil {
		time.Sleep(time.Second / 2)
		return USER_NOT_FOUND, nil
	}

	var resp utils.Packet
	if subtle.ConstantTimeCompare([]byte(lh.password), []byte(user.Password)) == 1 { // login succeeded

		if user.UserType == 0 { // Banned
			resp = USER_BANNED
			resp.Insert([]byte(parseDate(user.DisabledUntil)), 0x2E) // ban duration
			return resp, nil
		}

		if user.ConnectedIP != "" { // user already online
			logger.Log(logging.ACTION_LOGIN, 0, "Multiple login", user.ID)
			s.Conn.Close()
			user.Logout()
			if sock := database.GetSocket(user.ID); sock != nil {
				if c := sock.Character; c != nil {
					c.Logout()
				}
				sock.Conn.Close()
			}

			return nil, nil
		}
		logger.Log(logging.ACTION_LOGIN, 0, "Login successful", user.ID)
		resp = LOGGED_IN
		s.User = user
		s.User.ConnectedIP = s.ClientAddr
		s.User.IsLoginedFromPanel = true
		length := int16(len(lh.username) + 75)
		namelength := len(lh.username)
		resp.SetLength(length)
		resp.Insert([]byte(utils.IntToBytes(uint64(namelength), 1, false)), 7)
		resp.Insert([]byte(lh.username), 8)
		text := "ID: " + s.User.Username + "(" + s.User.ID + ") logged in with ip: " + s.User.ConnectedIP
		utils.NewLog("logs/ip_logs.txt", text)
		go s.User.Update()
	} else { // login failed
		logger.Log(logging.ACTION_LOGIN, 0, "Login failed.", user.ID)
		time.Sleep(time.Second / 2)
		resp = USER_NOT_FOUND
	}

	return resp, nil
}

func parseDate(date null.Time) string {
	if date.Valid {
		year, month, day := date.Time.Date()
		return fmt.Sprintf("%02d.%02d.%d", day, month, year)
	}

	return ""
}