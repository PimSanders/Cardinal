package auth

import (
	"Cardinal/internal/dbold"
	"Cardinal/internal/locales"
	"Cardinal/internal/utils"

	"github.com/gin-gonic/gin"
)

// TeamAuthRequired is the team permission check middleware.
func TeamAuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(utils.MakeErrJSON(403, 40300,
				locales.I18n.T(c.GetString("lang"), "general.no_auth"),
			))
			c.Abort()
			return
		}

		var tokenData dbold.Token
		dbold.MySQL.Where(&dbold.Token{Token: token}).Find(&tokenData)
		if tokenData.ID == 0 {
			c.JSON(utils.MakeErrJSON(401, 40100,
				locales.I18n.T(c.GetString("lang"), "general.no_auth"),
			))
			c.Abort()
			return
		}

		c.Set("teamID", tokenData.TeamID)
		c.Next()
	}
}

// AdminAuthRequired is the admin permission check middleware.
func AdminAuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(utils.MakeErrJSON(403, 40302,
				locales.I18n.T(c.GetString("lang"), "general.no_auth"),
			))
			c.Abort()
			return
		}

		var managerData dbold.Manager
		dbold.MySQL.Where(&dbold.Manager{Token: token}).Find(&managerData)
		if managerData.ID == 0 {
			c.JSON(utils.MakeErrJSON(401, 40101,
				locales.I18n.T(c.GetString("lang"), "general.no_auth"),
			))
			c.Abort()
			return
		}

		c.Set("managerData", managerData)
		c.Set("isCheck", managerData.IsCheck)
		c.Next()
	}
}

// ManagerRequired make sure the account is the manager.
func ManagerRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("isCheck") {
			c.JSON(utils.MakeErrJSON(401, 40102,
				locales.I18n.T(c.GetString("lang"), "manager.manager_required"),
			))
			c.Abort()
			return
		}
		c.Next()
	}
}
