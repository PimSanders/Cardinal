package cardinal_test

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	log "unknwon.dev/clog/v2"

	"Cardinal/internal/asteroid"
	"Cardinal/internal/bootstrap"
	"Cardinal/internal/conf"
	"Cardinal/internal/dbold"
	"Cardinal/internal/dynamic_config"
	"Cardinal/internal/game"
	"Cardinal/internal/livelog"
	"Cardinal/internal/misc/webhook"
	"Cardinal/internal/route"
	"Cardinal/internal/store"
	"Cardinal/internal/timer"
	"Cardinal/internal/utils"
)

var managerToken = utils.GenerateToken()

var checkToken string

var team = make([]struct {
	Name      string `json:"Name"`
	Password  string `json:"Password"`
	Token     string `json:"token"`
	AccessKey string `json:"access_key"`
}, 0)

var router *gin.Engine

func TestMain(m *testing.M) {
	prepare()
	log.Trace("Cardinal Test ready...")
	m.Run()

	os.Exit(0)
}

func prepare() {
	_ = log.NewConsole(100)

	log.Trace("Prepare for Cardinal test environment...")

	gin.SetMode(gin.ReleaseMode)

	err := conf.TestInit()
	if err != nil {
		panic(err)
	}

	// Init MySQL database.
	dbold.InitMySQL()

	// Test manager account e99:qwe1qwe2qwe3
	dbold.MySQL.Create(&dbold.Manager{
		Name:     "e99",
		Password: utils.AddSalt("qwe1qwe2qwe3"),
		Token:    managerToken,
		IsCheck:  false,
	})

	// Refresh the dynamic config from the database.
	dynamic_config.Init()

	bootstrap.GameToTimerBridge()
	timer.Init()

	asteroid.Init(game.AsteroidGreetData)

	// Cache
	store.Init()
	webhook.RefreshWebHookStore()

	// Live log
	livelog.Init()

	// Web router.
	router = route.Init()
}
