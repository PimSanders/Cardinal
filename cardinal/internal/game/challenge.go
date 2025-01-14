package game

import (
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"Cardinal/internal/dbold"
	"Cardinal/internal/locales"
	"Cardinal/internal/logger"
	"Cardinal/internal/utils"
)

// SetVisible is setting challenge visible status handler.
// When a challenge's visible status changed, all the teams' challenge scores and their total scores will be calculated immediately.
// The ranking list will also be updated.
func SetVisible(c *gin.Context) (int, interface{}) {
	type InputForm struct {
		ID      uint `binding:"required"`
		Visible bool
	}

	var inputForm InputForm
	err := c.BindJSON(&inputForm)
	if err != nil {
		log.Printf("Error binding JSON: %v", err)
		return utils.MakeErrJSON(400, 40027,
			locales.I18n.T(c.GetString("lang"), "general.error_payload"),
		)
	}

	var checkChallenge dbold.Challenge
	dbold.MySQL.Where(&dbold.Challenge{Model: gorm.Model{ID: inputForm.ID}}).Find(&checkChallenge)
	if checkChallenge.Title == "" {
		log.Printf("Challenge not found: ID %d", inputForm.ID)
		return utils.MakeErrJSON(404, 40402,
			locales.I18n.T(c.GetString("lang"), "challenge.not_found"),
		)
	}

	dbold.MySQL.Model(&dbold.GameBox{}).Where("challenge_id = ?", inputForm.ID).Update(map[string]interface{}{"visible": inputForm.Visible})

	// Calculate all the teams' score. (Only visible challenges)
	calculateTeamScore()
	// Refresh the ranking list table's header.
	SetRankListTitle()
	// Refresh the ranking list teams' scores.
	SetRankList()

	status := "invisible"
	if inputForm.Visible {
		status = "visible"
	}
	logger.New(logger.NORMAL, "manager_operate",
		string(locales.I18n.T(c.GetString("lang"), "log.set_challenge_"+status, gin.H{"challenge": checkChallenge.Title})),
	)
	log.Printf("Challenge visibility set to %s: %s", status, checkChallenge.Title)
	return utils.MakeSuccessJSON(locales.I18n.T(c.GetString("lang"), "gamebox.visibility_success"))
}

// GetAllChallenges get all challenges from the database.
func GetAllChallenges(c *gin.Context) (int, interface{}) {
	var challenges []dbold.Challenge
	dbold.MySQL.Model(&dbold.Challenge{}).Find(&challenges)
	type resultStruct struct {
		ID               uint
		CreatedAt        time.Time
		Title            string
		Visible          bool
		BaseScore        int
		AutoRefreshFlag  bool
		Command          string
		CheckdownCommand string
	}

	var res []resultStruct
	for _, v := range challenges {
		// For the challenge model doesn't have the `visible` field,
		// We can only get the challenge's visible status by one of its gamebox.
		// TODO: Need to find a better way to get the challenge's visible status.
		var gameBox dbold.GameBox
		dbold.MySQL.Where(&dbold.GameBox{ChallengeID: v.ID}).Limit(1).Find(&gameBox)

		res = append(res, resultStruct{
			ID:               v.ID,
			CreatedAt:        v.CreatedAt,
			Title:            v.Title,
			Visible:          gameBox.Visible,
			BaseScore:        v.BaseScore,
			AutoRefreshFlag:  v.AutoRefreshFlag,
			Command:          v.Command,
			CheckdownCommand: v.CheckdownCommand,
		})
	}
	log.Printf("Retrieved %d challenges", len(challenges))
	return utils.MakeSuccessJSON(res)
}

// NewChallenge is new challenge handler for manager.
func NewChallenge(c *gin.Context) (int, interface{}) {
	type InputForm struct {
		Title            string `binding:"required"`
		BaseScore        int    `binding:"required"`
		AutoRefreshFlag  bool
		Command          string
		CheckdownCommand string
	}

	var inputForm InputForm
	err := c.BindJSON(&inputForm)
	if err != nil {
		log.Printf("Error binding JSON: %v", err)
		return utils.MakeErrJSON(400, 40028,
			locales.I18n.T(c.GetString("lang"), "general.error_payload"),
		)
	}

	if inputForm.AutoRefreshFlag && inputForm.Command == "" {
		log.Printf("Empty command for auto-refresh challenge: %s", inputForm.Title)
		return utils.MakeErrJSON(400, 40029,
			locales.I18n.T(c.GetString("lang"), "challenge.empty_command"))
	}

	if !inputForm.AutoRefreshFlag {
		inputForm.Command = ""
	}

	newChallenge := &dbold.Challenge{
		Title:            inputForm.Title,
		BaseScore:        inputForm.BaseScore,
		AutoRefreshFlag:  inputForm.AutoRefreshFlag,
		Command:          inputForm.Command,
		CheckdownCommand: inputForm.CheckdownCommand,
	}
	var checkChallenge dbold.Challenge

	dbold.MySQL.Model(&dbold.Challenge{}).Where(&dbold.Challenge{Title: newChallenge.Title}).Find(&checkChallenge)
	if checkChallenge.Title != "" {
		log.Printf("Challenge already exists: %s", newChallenge.Title)
		return utils.MakeErrJSON(403, 40313,
			locales.I18n.T(c.GetString("lang"), "general.post_repeat"),
		)
	}

	tx := dbold.MySQL.Begin()
	if tx.Create(newChallenge).RowsAffected != 1 {
		tx.Rollback()
		log.Printf("Error creating new challenge: %s", newChallenge.Title)
		return utils.MakeErrJSON(500, 50016,
			locales.I18n.T(c.GetString("lang"), "challenge.post_error"),
		)
	}
	tx.Commit()

	logger.New(logger.NORMAL, "manager_operate",
		string(locales.I18n.T(c.GetString("lang"), "log.new_challenge", gin.H{"title": newChallenge.Title})),
	)
	log.Printf("New challenge created: %s", newChallenge.Title)
	return utils.MakeSuccessJSON(locales.I18n.T(c.GetString("lang"), "challenge.post_success"))
}

// EditChallenge is edit challenge handler for manager.
func EditChallenge(c *gin.Context) (int, interface{}) {
	type InputForm struct {
		ID               uint   `binding:"required"`
		Title            string `binding:"required"`
		BaseScore        int    `binding:"required"`
		AutoRefreshFlag  bool
		Command          string
		CheckdownCommand string
	}

	var inputForm InputForm
	err := c.BindJSON(&inputForm)
	if err != nil {
		log.Printf("Error binding JSON: %v", err)
		return utils.MakeErrJSON(400, 40028,
			locales.I18n.T(c.GetString("lang"), "general.error_payload"),
		)
	}

	if inputForm.AutoRefreshFlag && inputForm.Command == "" {
		log.Printf("Empty command for auto-refresh challenge: %s", inputForm.Title)
		return utils.MakeErrJSON(400, 40029,
			locales.I18n.T(c.GetString("lang"), "challenge.empty_command"))
	}

	// True off auto refresh flag, clean the command.
	if !inputForm.AutoRefreshFlag {
		inputForm.Command = ""
	}

	var checkChallenge dbold.Challenge
	dbold.MySQL.Where(&dbold.Challenge{Model: gorm.Model{ID: inputForm.ID}}).Find(&checkChallenge)
	if checkChallenge.Title == "" {
		log.Printf("Challenge not found: ID %d", inputForm.ID)
		return utils.MakeErrJSON(404, 40403,
			locales.I18n.T(c.GetString("lang"), "challenge.not_found"),
		)
	}

	// For the `AutoRefreshFlag` is a boolean value, use map here.
	editChallenge := map[string]interface{}{
		"Title":            inputForm.Title,
		"BaseScore":        inputForm.BaseScore,
		"AutoRefreshFlag":  inputForm.AutoRefreshFlag,
		"Command":          inputForm.Command,
		"CheckdownCommand": inputForm.CheckdownCommand,
	}
	tx := dbold.MySQL.Begin()
	if tx.Model(&dbold.Challenge{}).Where(&dbold.Challenge{Model: gorm.Model{ID: inputForm.ID}}).Updates(editChallenge).RowsAffected != 1 {
		tx.Rollback()
		log.Printf("Error updating challenge: %s", inputForm.Title)
		return utils.MakeErrJSON(500, 50017,
			locales.I18n.T(c.GetString("lang"), "challenge.put_error"),
		)
	}
	tx.Commit()

	// If the challenge's score is updated, we need to calculate the gameboxes' scores and the teams' scores.
	if inputForm.BaseScore != checkChallenge.BaseScore {
		// Calculate all the teams' score. (Only visible challenges)
		calculateTeamScore()
		// Refresh the ranking list table's header.
		SetRankListTitle()
		// Refresh the ranking list teams' scores.
		SetRankList()
	}

	// If the challenge's title is updated, we just need to update the ranking list table's header.
	if inputForm.Title != checkChallenge.Title {
		SetRankListTitle()
	}

	log.Printf("Challenge updated: %s", inputForm.Title)
	return utils.MakeSuccessJSON(locales.I18n.T(c.GetString("lang"), "challenge.put_success"))
}

// DeleteChallenge is delete challenge handler for manager.
func DeleteChallenge(c *gin.Context) (int, interface{}) {
	idStr, ok := c.GetQuery("id")
	if !ok {
		log.Printf("Error getting query parameter: id")
		return utils.MakeErrJSON(400, 40030,
			locales.I18n.T(c.GetString("lang"), "general.error_query"),
		)
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("Error converting id to integer: %v", err)
		return utils.MakeErrJSON(400, 40030,
			locales.I18n.T(c.GetString("lang"), "general.must_be_number", gin.H{"key": "id"}),
		)
	}

	var challenge dbold.Challenge
	dbold.MySQL.Where(&dbold.Challenge{Model: gorm.Model{ID: uint(id)}}).Find(&challenge)
	if challenge.Title == "" {
		log.Printf("Challenge not found: ID %d", id)
		return utils.MakeErrJSON(404, 40403,
			locales.I18n.T(c.GetString("lang"), "challenge.not_found"),
		)
	}

	tx := dbold.MySQL.Begin()
	// Also delete GameBox
	tx.Where("challenge_id = ?", uint(id)).Delete(&dbold.GameBox{})
	if tx.Where("id = ?", uint(id)).Delete(&dbold.Challenge{}).RowsAffected != 1 {
		tx.Rollback()
		log.Printf("Error deleting challenge: %s", challenge.Title)
		return utils.MakeErrJSON(500, 50018,
			locales.I18n.T(c.GetString("lang"), "challenge.delete_error"),
		)
	}
	tx.Commit()

	logger.New(logger.NORMAL, "manager_operate",
		string(locales.I18n.T(c.GetString("lang"), "log.delete_challenge", gin.H{"title": challenge.Title})),
	)
	log.Printf("Challenge deleted: %s", challenge.Title)
	return utils.MakeSuccessJSON(locales.I18n.T(c.GetString("lang"), "challenge.delete_success"))
}
