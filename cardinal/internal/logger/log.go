package logger

import (
	"Cardinal/internal/dbold"
	"Cardinal/internal/utils"

	"github.com/gin-gonic/gin"
)

// Log levels
const (
	NORMAL = iota
	WARNING
	IMPORTANT
)

// New create a new log record in database.
func New(level int, kind string, content string) {
	dbold.MySQL.Create(&dbold.Log{
		Level:   level,
		Kind:    kind,
		Content: content,
	})
}

// GetLogs returns the latest 50 logs.
func GetLogs(c *gin.Context) (int, interface{}) {
	var logs []dbold.Log
	dbold.MySQL.Model(&dbold.Log{}).Order("`id` DESC").Limit(50).Find(&logs)
	return utils.MakeSuccessJSON(logs)
}
