package route

import (
	"Cardinal/internal/asteroid"
	"Cardinal/internal/auth"
	"Cardinal/internal/auth/manager"
	"Cardinal/internal/auth/team"
	"Cardinal/internal/bulletin"
	"Cardinal/internal/conf"
	"Cardinal/internal/dynamic_config"
	"Cardinal/internal/game"
	"Cardinal/internal/healthy"
	"Cardinal/internal/livelog"
	"Cardinal/internal/locales"
	"Cardinal/internal/logger"
	"Cardinal/internal/misc/webhook"
	"Cardinal/internal/timer"
	"Cardinal/internal/upload"
	"Cardinal/internal/utils"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Init() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()
	// CORS Header
	r.Use(cors.New(cors.Config{
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
		AllowHeaders: []string{"Authorization", "Content-type", "User-Agent"},
		AllowOrigins: []string{"*"},
	}))

	api := r.Group("/api")
	api.Use(locales.Middleware()) // i18n
	// Sentry
	if conf.App.EnableSentry {
		api.Use(sentrygin.New(sentrygin.Options{
			Repanic: true,
		}))
	}

	// Cardinal basic info
	api.Any("/", func(c *gin.Context) {
		c.JSON(utils.MakeSuccessJSON("Cardinal"))
	})

	api.GET("/base", func(c *gin.Context) {
		c.JSON(utils.MakeSuccessJSON(gin.H{
			"Title":    dynamic_config.Get(utils.TITLE_CONF),
			"Language": dynamic_config.Get(utils.DEFAULT_LANGUAGE),
		}))
	})

	api.GET("/time", __(timer.GetTime))

	api.GET("/rank", func(c *gin.Context) {
		c.JSON(utils.MakeSuccessJSON(gin.H{"Title": game.GetRankListTitle(), "Rank": game.GetRankList()}))
	})

	// Static files
	api.Static("/uploads", "./uploads")

	// Team login
	api.POST("/login", __(team.TeamLogin))

	// Team logout
	api.GET("/logout", __(team.TeamLogout))

	// Live log
	api.GET("/livelog", livelog.GlobalStreamHandler)

	// Submit flag
	api.POST("/flag", __(game.SubmitFlag))

	// Asteroid websocket
	api.GET("/asteroid", func(c *gin.Context) {
		asteroid.ServeWebSocket(c)
	})

	// Team Routes
	teamRouter := api.Group("/team")
	teamRouter.Use(auth.TeamAuthRequired())
	{
		teamRouter.GET("/info", __(team.GetTeamInfo))
		teamRouter.GET("/gameboxes", __(game.GetSelfGameBoxes))
		teamRouter.GET("/gameboxes/all", __(game.GetOthersGameBox))
		teamRouter.GET("/rank", func(c *gin.Context) {
			c.JSON(utils.MakeSuccessJSON(gin.H{"Title": game.GetRankListTitle(), "Rank": game.GetRankList()}))
		})
		teamRouter.GET("/bulletins", __(bulletin.GetAllBulletins))
	}

	// Manager Routes
	managerRouter := api.Group("/manager")
	managerRouter.POST("/login", __(manager.ManagerLogin))
	managerRouter.GET("/logout", __(manager.ManagerLogout))
	managerRouter.Use(auth.AdminAuthRequired(), auth.ManagerRequired())
	{
		// Challenges
		managerRouter.GET("/challenges", __(game.GetAllChallenges))
		managerRouter.POST("/challenge", __(game.NewChallenge))
		managerRouter.PUT("/challenge", __(game.EditChallenge))
		managerRouter.DELETE("/challenge", __(game.DeleteChallenge))
		managerRouter.POST("/challenge/visible", __(game.SetVisible))

		// Teams
		managerRouter.GET("/teams", __(team.GetAllTeams))
		managerRouter.POST("/teams", __(team.NewTeams))
		managerRouter.PUT("/team", __(team.EditTeam))
		managerRouter.DELETE("/team", __(team.DeleteTeam))
		managerRouter.POST("/team/resetPassword", __(team.ResetTeamPassword))

		// GameBoxes
		managerRouter.GET("/gameboxes", __(game.GetGameBoxes))
		managerRouter.POST("/gameboxes", __(game.NewGameBoxes))
		managerRouter.PUT("/gamebox", __(game.EditGameBox))
		managerRouter.DELETE("/gamebox", __(game.DeleteGameBox))
		managerRouter.GET("/gameboxes/sshTest", __(game.TestAllSSH))
		managerRouter.POST("/gameboxes/sshTest", __(game.TestSSH))
		managerRouter.GET("/gameboxes/refreshFlag", func(c *gin.Context) {
			game.RefreshFlag()
			// TODO: i18n
			c.JSON(utils.MakeSuccessJSON("Flags updated, check the logs"))
		})
		managerRouter.GET("/gameboxes/reset", __(game.ResetAllGameBoxes))

		// Flags
		managerRouter.GET("/flags", __(game.GetFlags))
		managerRouter.POST("/flag/generate", __(game.GenerateFlag))
		managerRouter.GET("/flag/export", __(game.ExportFlag))

		// Asteroid
		managerRouter.GET("/asteroid/status", __(asteroid.GetAsteroidStatus))
		managerRouter.POST("/asteroid/attack", __(asteroid.Attack))
		managerRouter.POST("/asteroid/rank", __(asteroid.Rank))
		managerRouter.POST("/asteroid/status", __(asteroid.Status))
		managerRouter.POST("/asteroid/round", __(asteroid.Round))
		managerRouter.POST("/asteroid/easterEgg", __(asteroid.EasterEgg))
		managerRouter.POST("/asteroid/time", __(asteroid.Time))
		managerRouter.POST("/asteroid/clear", __(asteroid.Clear))
		managerRouter.POST("/asteroid/clearAll", __(asteroid.ClearAll))

		// Log
		managerRouter.GET("/logs", __(logger.GetLogs))
		managerRouter.GET("/rank", func(c *gin.Context) {
			c.JSON(utils.MakeSuccessJSON(gin.H{"Title": game.GetRankListTitle(), "Rank": game.GetManagerRankList()}))
		})
		managerRouter.GET("/panel", __(healthy.Panel))

		// WebHook
		managerRouter.GET("/webhooks", __(webhook.GetWebHook))
		managerRouter.POST("/webhook", __(webhook.NewWebHook))
		managerRouter.PUT("/webhook", __(webhook.EditWebHook))
		managerRouter.DELETE("/webhook", __(webhook.DeleteWebHook))

		// Config
		managerRouter.GET("/configs", __(dynamic_config.GetAllConfig))
		managerRouter.GET("/config", __(dynamic_config.GetConfig))
		managerRouter.PUT("/config", __(dynamic_config.SetConfig))

		// Bulletin
		managerRouter.GET("/bulletins", __(bulletin.GetAllBulletins))
		managerRouter.POST("/bulletin", __(bulletin.NewBulletin))
		managerRouter.PUT("/bulletin", __(bulletin.EditBulletin))
		managerRouter.DELETE("/bulletin", __(bulletin.DeleteBulletin))

		// File
		managerRouter.POST("/uploadPicture", __(upload.UploadPicture))
		managerRouter.GET("/dir", __(upload.GetDir))

		// Docker
		// managerRouter.POST("/docker/findImage", __(container.GetImageData))

		// Check
		managerRouter.POST("/checkDown", __(game.CheckDown))

		// Manager
		managerRouter.GET("/managers", __(manager.GetAllManager))
		managerRouter.POST("/manager", __(manager.NewManager))
		managerRouter.GET("/manager/token", __(manager.RefreshManagerToken))
		managerRouter.GET("/manager/changePassword", __(manager.ChangeManagerPassword))
		managerRouter.DELETE("/manager", __(manager.DeleteManager))
	}

	// 404 and 405 Handlers
	r.NoRoute(func(c *gin.Context) {
		c.JSON(utils.MakeErrJSON(404, 40400,
			locales.I18n.T(c.GetString("lang"), "general.not_found"),
		))
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(utils.MakeErrJSON(405, 40500,
			locales.I18n.T(c.GetString("lang"), "general.method_not_allow"),
		))
	})

	return r
}
