package dynamic_config

import (
	"github.com/gin-gonic/gin"

	"Cardinal/internal/conf"
	"Cardinal/internal/dbold"
	"Cardinal/internal/locales"
	"Cardinal/internal/utils"
)

func Init() {
	dbold.MySQL.Model(&dbold.DynamicConfig{})

	initConfig(utils.DATBASE_VERSION, dbold.VERSION, utils.STRING)
	initConfig(utils.TITLE_CONF, conf.App.Name, utils.STRING)
	initConfig(utils.FLAG_PREFIX_CONF, conf.Game.FlagPrefix, utils.STRING)
	initConfig(utils.FLAG_SUFFIX_CONF, conf.Game.FlagSuffix, utils.STRING)
	initConfig(utils.ANIMATE_ASTEROID, utils.BOOLEAN_FALSE, utils.BOOLEAN)
	initConfig(utils.SHOW_OTHERS_GAMEBOX, utils.BOOLEAN_FALSE, utils.BOOLEAN)
	initConfig(utils.DEFAULT_LANGUAGE, conf.App.Language, utils.SELECT, "en-US|zh-CN")
}

// initConfig set the default value of the given key.
// Always used in installation.
func initConfig(key string, value string, kind int8, option ...string) {
	var opt string
	if len(option) != 0 {
		opt = option[0]
	}

	dbold.MySQL.Model(&dbold.DynamicConfig{}).FirstOrCreate(&dbold.DynamicConfig{
		Key:     key,
		Value:   value,
		Kind:    kind,
		Options: opt,
	}, "`key` = ?", key)
}

// Set update the config by insert a new record into database, for we can make a config version control soon.
// Then refresh the config in struct.
func Set(key string, value string) {
	if key == utils.DATBASE_VERSION {
		return
	}

	dbold.MySQL.Model(&dbold.DynamicConfig{}).Where("`key` = ?", key).Update(&dbold.DynamicConfig{
		Key:   key,
		Value: value,
	})
}

// Get returns the config value.
func Get(key string) string {
	var config dbold.DynamicConfig
	dbold.MySQL.Model(&dbold.DynamicConfig{}).Where("`key` = ?", key).Find(&config)
	return config.Value
}

// SetConfig is the HTTP handler used to set the config value.
func SetConfig(c *gin.Context) (int, interface{}) {
	var inputForm []struct {
		Key   string `binding:"required"`
		Value string `binding:"required"`
	}

	if err := c.BindJSON(&inputForm); err != nil {
		return utils.MakeErrJSON(400, 40046, locales.I18n.T(c.GetString("lang"), "general.error_payload"))
	}

	for _, config := range inputForm {
		Set(config.Key, config.Value)
	}
	return utils.MakeSuccessJSON(locales.I18n.T(c.GetString("lang"), "config.update_success"))
}

// GetConfig is the HTTP handler used to return the config value of the given key.
func GetConfig(c *gin.Context) (int, interface{}) {
	var inputForm struct {
		Key string `binding:"required"`
	}

	if err := c.BindJSON(&inputForm); err != nil {
		return utils.MakeErrJSON(400, 40046, locales.I18n.T(c.GetString("lang"), "general.error_payload"))
	}
	value := Get(inputForm.Key)
	return utils.MakeSuccessJSON(value)
}

// GetAllConfig is the HTTP handler used to return the all the configs.
func GetAllConfig(c *gin.Context) (int, interface{}) {
	var config []dbold.DynamicConfig
	dbold.MySQL.Model(&dbold.DynamicConfig{}).Where("`key` != ?", utils.DATBASE_VERSION).Find(&config)
	return utils.MakeSuccessJSON(config)
}
