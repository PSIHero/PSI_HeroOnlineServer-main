package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"syscall"

	"time"

	"PsiHero/ai"
	"PsiHero/api"
	"PsiHero/config"
	"PsiHero/database"
	"PsiHero/event"
	_ "PsiHero/factory"
	"PsiHero/logging"
	"PsiHero/nats"
	"PsiHero/utils"

	_ "github.com/go-sql-driver/mysql"
	"github.com/robfig/cron"
)

var (
	logger = logging.Logger
	path   = "payments.txt"
	// CacheFile to store state to be transferred
	CacheFile        = "cache.json"
	kernel32         = syscall.MustLoadDLL("kernel32.dll")
	procSetStdHandle = kernel32.MustFindProc("SetStdHandle")
)

func initDatabase() {
	for {
		err := database.InitDB()
		if err == nil {
			log.Printf("Connected to database...")
			return
		}
		log.Printf("Database connection error: %+v, waiting 2 sec...", err)
		time.Sleep(time.Duration(2) * time.Second)
	}
}

func StartLogging() {
	fi, err := os.OpenFile("Log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666) //log file
	if err != nil {
		recover()
		//log.Fatalf("error opening file: %v", err)
	}
	crash, err := os.OpenFile("ServerCrash.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666) //log file
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(fi)
	redirectStderr(crash)
}

func startServer() {
	cfg := config.Default
	port := cfg.Server.Port
	listen, err := net.Listen("tcp4", ":"+strconv.Itoa(port))
	defer listen.Close()
	if err != nil {
		fmt.Println("Socket listen port %d failed,%s", port, err)
		//os.Exit(1)
	}
	log.Printf("Begin listen port: %d", port)
	StartLogging()
	LoadBossEvent()
	//initQuest()
	//connections = make(map[string]net.Conn)
	//remoteAddrs = make(map[string]int)
	/*SERVER DUNGEON EXPLORE*/
	ticker := time.NewTicker(1 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				database.ServerExploreWorld()

			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalln(err)
			continue
		}

		ws := database.Socket{Conn: conn}
		go ws.Read()

	}
}

func cronHandler() {
	c := cron.New()
	c.AddFunc("0 0 0 * * *", func() {
		database.RefreshAIDs()
	})
	c.Start()
	database.RefreshAIDs()
}

func main() {
	initDatabase()
	cronHandler()
	ai.Init()
	database.RefreshOnline()
	nats.RunServer(nil)
	nats.ConnectSelf(nil)
	go api.InitGRPC()

	go database.WarAutomation()
	go database.CleanUp()
	go database.UnbanUsers()
	go database.EpochHandler()

	database.ServerStart = time.Now()
	startServer()
}

func setStdHandle(stdhandle int32, handle syscall.Handle) error {
	r0, _, e1 := syscall.Syscall(procSetStdHandle.Addr(), 2, uintptr(stdhandle), uintptr(handle), 0)
	if r0 == 0 {
		if e1 != 0 {
			return error(e1)
		}
		return syscall.EINVAL
	}
	return nil
}

// redirectStderr to the file passed in
func redirectStderr(f *os.File) {
	err := setStdHandle(syscall.STD_ERROR_HANDLE, syscall.Handle(f.Fd()))
	if err != nil {
		log.Fatalf("Failed to log into file: %v", err)
	}
	// SetStdHandle does not affect prior references to stderr
	os.Stderr = f
}

func timetosec(respawntime time.Time) int64 {
	timesec := int64(0)
	timesec += int64(3600 * respawntime.Hour())
	timesec += int64(60 * respawntime.Minute())
	return timesec
}

func LoadBossEvent() {
	for _, bossevent := range database.BossEvents {
		if bossevent.Activated {
			bossRespawnTime := bossevent.RespawnTime.Time
			bossLastKilled := bossevent.LastKilledAt.Time
			eventtime := bossLastKilled.Add(time.Hour*time.Duration(bossRespawnTime.Hour()) + time.Minute*time.Duration(bossRespawnTime.Minute()))
			remainingtime := time.Since(eventtime)
			respawnsec := timetosec(bossRespawnTime)
			if remainingtime >= 0 { //IF THE EVENT IS NOT FINISHED YET{
				bosses := bossevent.GetBosses()
				//fmt.Println("Need respawn")
				if len(bosses) > 1 {
					index := utils.RandInt(0, int64(len(bosses)))
					event.MobsCreate([]int{bosses[index]}, 1, bossevent.MapID, bossevent.MinLocation, bossevent.MaxLocation, respawnsec, bossevent.EventID)
				} else {
					event.MobsCreate(bosses, 1, bossevent.MapID, bossevent.MinLocation, bossevent.MaxLocation, respawnsec, bossevent.EventID)
				}
			} else {
				aftertime := time.Duration(time.Hour*time.Duration(math.Abs(float64(remainingtime.Hours()))) + time.Minute*time.Duration(math.Abs(float64(remainingtime.Minutes()))) + time.Second*time.Duration(math.Abs(float64(remainingtime.Seconds()))))
				fmt.Println(aftertime)
				time.AfterFunc(time.Duration(aftertime), func() { // respawn boss n secs later
					bosses := bossevent.GetBosses()
					//fmt.Println("Respawn after")
					if len(bosses) > 1 {
						index := utils.RandInt(0, int64(len(bosses)))
						event.MobsCreate([]int{bosses[index]}, 1, bossevent.MapID, bossevent.MinLocation, bossevent.MaxLocation, respawnsec, bossevent.EventID)
					} else {
						event.MobsCreate(bosses, 1, bossevent.MapID, bossevent.MinLocation, bossevent.MaxLocation, respawnsec, bossevent.EventID)
					}

				})
			}
		}
	}
}
