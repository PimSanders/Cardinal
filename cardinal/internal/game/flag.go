package game

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"Cardinal/internal/asteroid"
	"Cardinal/internal/conf"
	"Cardinal/internal/dbold"
	"Cardinal/internal/dynamic_config"
	"Cardinal/internal/livelog"
	"Cardinal/internal/locales"
	"Cardinal/internal/logger"
	"Cardinal/internal/misc/webhook"
	"Cardinal/internal/timer"
	"Cardinal/internal/utils"
)

// SubmitFlag is submit flag handler for teams.
func SubmitFlag(c *gin.Context) (int, interface{}) {
	log.Println("SubmitFlag called")

	// Submit flag is forbidden if the competition isn't started.
	if timer.Get().Status != "on" {
		log.Println("Competition has not started")
		return utils.MakeErrJSON(403, 40304,
			locales.I18n.T(c.GetString("lang"), "general.not_begin"),
		)
	}

	secretKey := c.GetHeader("Authorization")
	if secretKey == "" {
		log.Println("Missing Authorization header")
		return utils.MakeErrJSON(403, 40305,
			locales.I18n.T(c.GetString("lang"), "general.invalid_token"),
		)
	}

	var t dbold.Team
	dbold.MySQL.Model(&dbold.Team{}).Where(&dbold.Team{SecretKey: secretKey}).Find(&t)
	teamID := t.ID
	if teamID == 0 {
		log.Printf("Invalid secret key: %s\n", secretKey)
		return utils.MakeErrJSON(403, 40306,
			locales.I18n.T(c.GetString("lang"), "general.invalid_token"),
		)
	}
	log.Printf("Team ID: %d found for secret key\n", teamID)

	type InputForm struct {
		Flag string `json:"flag" binding:"required"`
	}
	var inputForm InputForm
	err := c.BindJSON(&inputForm)
	if err != nil {
		log.Println("Error binding input form: ", err)
		return utils.MakeErrJSON(400, 40021,
			locales.I18n.T(c.GetString("lang"), "general.error_payload"),
		)
	}

	// Remove the space
	inputForm.Flag = strings.TrimSpace(inputForm.Flag)
	log.Printf("Flag submitted: %s\n", inputForm.Flag)

	var flagDataPrevRound dbold.Flag
	dbold.MySQL.Model(&dbold.Flag{}).Where(&dbold.Flag{Flag: inputForm.Flag, Round: (timer.Get().NowRound - 1)}).Find(&flagDataPrevRound)

	if flagDataPrevRound.Flag == inputForm.Flag { // Please note that you are not allowed to submit the flag of the previous round
		log.Printf("Flag for previous round detected: TeamID: %d, Flag TeamID: %d\n", teamID, flagDataPrevRound.TeamID)
		return utils.MakeErrJSON(403, 40307,
			locales.I18n.T(c.GetString("lang"), "flag.wrong_round"),
		)
	}

	var flagDataNextRound dbold.Flag
	dbold.MySQL.Model(&dbold.Flag{}).Where(&dbold.Flag{Flag: inputForm.Flag, Round: (timer.Get().NowRound + 1)}).Find(&flagDataNextRound)

	if flagDataNextRound.Flag == inputForm.Flag { // Please note that you are not allowed to submit the flag of the next round
		log.Printf("Flag for next round detected: TeamID: %d, Flag TeamID: %d\n", teamID, flagDataNextRound.TeamID)
		return utils.MakeErrJSON(403, 40307,
			locales.I18n.T(c.GetString("lang"), "flag.wrong_round"),
		)
	}

	var flagData dbold.Flag
	dbold.MySQL.Model(&dbold.Flag{}).Where(&dbold.Flag{Flag: inputForm.Flag, Round: timer.Get().NowRound}).Find(&flagData) // Pay attention to whether it is this round

	if flagData.ID == 0 { // Please note that you are not allowed to submit your own flag
		log.Printf("Invalid flag detected: TeamID: %d, Flag TeamID: %d\n", teamID, flagData.TeamID)
		return utils.MakeErrJSON(403, 40307,
			locales.I18n.T(c.GetString("lang"), "flag.wrong"),
		)
	}

	if teamID == flagData.TeamID { // Please note that you are not allowed to submit your own flag
		log.Printf("Self-submission detected: TeamID: %d, Flag TeamID: %d\n", teamID, flagData.TeamID)
		return utils.MakeErrJSON(403, 40307,
			locales.I18n.T(c.GetString("lang"), "flag.self"),
		)
	}

	// Check the challenge is visible or not.
	var gamebox dbold.GameBox
	dbold.MySQL.Model(&dbold.GameBox{}).Where(&dbold.GameBox{Model: gorm.Model{ID: flagData.GameBoxID}, Visible: true}).Find(&gamebox)
	if gamebox.ID == 0 {
		log.Printf("GameBox not visible or invalid: %d\n", flagData.GameBoxID)
		return utils.MakeErrJSON(403, 40308,
			locales.I18n.T(c.GetString("lang"), "flag.wrong"),
		)
	}

	// Check if the flag has been submitted by the team before.
	var repeatAttackCheck dbold.AttackAction
	dbold.MySQL.Model(&dbold.AttackAction{}).Where(&dbold.AttackAction{
		TeamID:         flagData.TeamID,
		GameBoxID:      flagData.GameBoxID,
		AttackerTeamID: teamID,
		Round:          flagData.Round,
	}).Find(&repeatAttackCheck)
	if repeatAttackCheck.ID != 0 {
		log.Printf("Repeat flag submission detected for TeamID: %d\n", teamID)
		// Animate Asteroid
		animateAsteroid, _ := strconv.ParseBool(dynamic_config.Get(utils.ANIMATE_ASTEROID))
		if animateAsteroid {
			log.Printf("Sending asteroid animation for attack: Attacker %d -> Victim %d\n", teamID, flagData.TeamID)
			asteroid.SendAttack(int(teamID), int(flagData.TeamID))
		}

		return utils.MakeErrJSON(403, 40309,
			locales.I18n.T(c.GetString("lang"), "flag.repeat"),
		)
	}

	// Update the victim's gamebox status to `down`.
	log.Printf("Updating gamebox status to down for GameBoxID: %d\n", flagData.GameBoxID)
	dbold.MySQL.Model(&dbold.GameBox{}).Where(&dbold.GameBox{Model: gorm.Model{ID: flagData.GameBoxID}}).Update(&dbold.GameBox{IsAttacked: true})

	// Save this attack record.
	log.Println("Saving attack record")
	tx := dbold.MySQL.Begin()
	if tx.Create(&dbold.AttackAction{
		TeamID:         flagData.TeamID,
		GameBoxID:      flagData.GameBoxID,
		AttackerTeamID: teamID,
		ChallengeID:    flagData.ChallengeID,
		Round:          flagData.Round,
	}).RowsAffected != 1 {
		log.Println("Failed to save attack record, rolling back")
		tx.Rollback()
		return utils.MakeErrJSON(500, 50013,
			locales.I18n.T(c.GetString("lang"), "flag.submit_error"),
		)
	}
	tx.Commit()
	log.Println("Attack record saved successfully")

	// Update the gamebox status in ranking list.
	log.Println("Updating rank list")
	SetRankList()
	// Webhook
	log.Println("Sending webhook for flag submission")
	go webhook.Add(webhook.SUBMIT_FLAG_HOOK, gin.H{"from": teamID, "to": gamebox.TeamID, "gamebox": gamebox.ID})
	// Send Unity3D attack message.
	log.Printf("Sending Unity3D attack message: Attacker %d -> Victim %d\n", teamID, flagData.TeamID)
	asteroid.SendAttack(int(teamID), int(flagData.TeamID))

	// Get attack team data
	var flagTeam dbold.Team
	dbold.MySQL.Model(&dbold.Team{}).Where(&dbold.Team{Model: gorm.Model{ID: flagData.TeamID}}).Find(&flagTeam)
	// Get challenge data
	var challenge dbold.Challenge
	dbold.MySQL.Model(&dbold.Challenge{}).Where(&dbold.Challenge{Model: gorm.Model{ID: flagData.ChallengeID}}).Find(&challenge)
	// Live log
	log.Printf("Writing live log: From %s -> To %s, Challenge: %s\n", t.Name, flagTeam.Name, challenge.Title)
	_ = livelog.Stream.Write(livelog.GlobalStream, livelog.NewLine("submit_flag",
		gin.H{"Round": timer.Get().NowRound, "From": t.Name, "To": flagTeam.Name, "Challenge": challenge.Title}))

	log.Println("Flag submitted successfully")
	return utils.MakeSuccessJSON(locales.I18n.T(c.GetString("lang"), "flag.submit_success"))
}

// GetFlags get flags from the database for backstage manager.
func GetFlags(c *gin.Context) (int, interface{}) {
	pageStr := c.DefaultQuery("page", "1")
	perStr := c.DefaultQuery("per", "15")

	// filter
	roundStr := c.DefaultQuery("round", "0")
	teamStr := c.DefaultQuery("team", "0")
	challengeStr := c.DefaultQuery("challenge", "0")

	round, err := strconv.Atoi(roundStr)
	if err != nil || round < 0 {
		return utils.MakeErrJSON(400, 40022,
			locales.I18n.T(c.GetString("lang"), "general.error_query"),
		)
	}
	teamID, err := strconv.Atoi(teamStr)
	if err != nil || teamID < 0 {
		return utils.MakeErrJSON(400, 40022,
			locales.I18n.T(c.GetString("lang"), "general.error_query"),
		)
	}
	challengeID, err := strconv.Atoi(challengeStr)
	if err != nil || challengeID < 0 {
		return utils.MakeErrJSON(400, 40022,
			locales.I18n.T(c.GetString("lang"), "general.error_query"),
		)
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		return utils.MakeErrJSON(400, 40022,
			locales.I18n.T(c.GetString("lang"), "general.error_query"),
		)
	}

	per, err := strconv.Atoi(perStr)
	if err != nil || per <= 0 || per >= 100 { // Limit to 100 items per page
		return utils.MakeErrJSON(400, 40023,
			locales.I18n.T(c.GetString("lang"), "general.error_query"),
		)
	}

	var total int
	dbold.MySQL.Model(&dbold.Flag{}).Where(&dbold.Flag{
		TeamID:      uint(teamID),
		ChallengeID: uint(challengeID),
		Round:       round,
	}).Count(&total)

	var flags []dbold.Flag
	dbold.MySQL.Model(&dbold.Flag{}).Where(&dbold.Flag{
		TeamID:      uint(teamID),
		ChallengeID: uint(challengeID),
		Round:       round,
	}).Offset((page - 1) * per).Limit(per).Find(&flags)

	return utils.MakeSuccessJSON(gin.H{
		"array": flags,
		"total": total,
	})
}

// ExportFlag exports the flags of a challenge.
func ExportFlag(c *gin.Context) (int, interface{}) {
	challengeIDStr := c.DefaultQuery("id", "1")

	challengeID, err := strconv.Atoi(challengeIDStr)
	if err != nil || challengeID <= 0 {
		return utils.MakeErrJSON(400, 40024,
			locales.I18n.T(c.GetString("lang"), "general.error_query"),
		)
	}

	var flags []dbold.Flag
	dbold.MySQL.Model(&dbold.Flag{}).Where(&dbold.Flag{ChallengeID: uint(challengeID)}).Find(&flags)
	return utils.MakeSuccessJSON(flags)
}

// GenerateFlag is the generate flag handler for manager.
func GenerateFlag(c *gin.Context) (int, interface{}) {
	var gameBoxes []dbold.GameBox
	dbold.MySQL.Model(&dbold.GameBox{}).Find(&gameBoxes)

	startTime := time.Now().UnixNano()
	// Delete all the flags in the table.
	dbold.MySQL.Unscoped().Delete(&dbold.Flag{})

	flagPrefix := dynamic_config.Get(utils.FLAG_PREFIX_CONF)
	flagSuffix := dynamic_config.Get(utils.FLAG_SUFFIX_CONF)

	salt := utils.Sha1Encode(conf.App.SecuritySalt)
	for round := 1; round <= timer.Get().TotalRound; round++ {
		// Flag = FlagPrefix + hmacSha1(TeamID + | + GameBoxID + | + Round, sha1(salt)) + FlagSuffix
		for _, gameBox := range gameBoxes {
			flag := flagPrefix + utils.HmacSha1Encode(fmt.Sprintf("%d|%d|%d", gameBox.TeamID, gameBox.ID, round), salt) + flagSuffix
			dbold.MySQL.Create(&dbold.Flag{
				TeamID:      gameBox.TeamID,
				GameBoxID:   gameBox.ID,
				ChallengeID: gameBox.ChallengeID,
				Round:       round,
				Flag:        flag,
			})
		}
	}

	var count int
	dbold.MySQL.Model(&dbold.Flag{}).Count(&count)
	endTime := time.Now().UnixNano()
	logger.New(logger.WARNING, "system",
		string(locales.I18n.T(c.GetString("lang"), "log.generate_flag", gin.H{"total": count, "time": float64(endTime-startTime) / float64(time.Second)})),
	)
	return utils.MakeSuccessJSON(locales.I18n.T(c.GetString("lang"), "flag.generate_success"))
}

// RefreshFlag refreshes all the flags in the current round.
func RefreshFlag() {
	var log = log.New(os.Stdout, "", log.LstdFlags) // Initialize logger to output to stdout
	log.Println("Starting RefreshFlag execution.")

	// Get the auto refresh flag challenges.
	var challenges []dbold.Challenge
	result := dbold.MySQL.Model(&dbold.Challenge{}).Where(&dbold.Challenge{AutoRefreshFlag: true}).Find(&challenges)
	if result.Error != nil {
		log.Printf("ERROR: Failed to retrieve challenges: %v\n", result.Error)
		return
	}

	log.Printf("INFO: Retrieved %d challenges with AutoRefreshFlag set to true.\n", len(challenges))

	for _, challenge := range challenges {
		var gameboxes []dbold.GameBox
		result = dbold.MySQL.Model(&dbold.GameBox{}).Where(&dbold.GameBox{ChallengeID: challenge.ID}).Find(&gameboxes)
		if result.Error != nil {
			log.Printf("ERROR: Failed to retrieve game boxes for challenge ID %d: %v\n", challenge.ID, result.Error)
			continue
		}

		log.Printf("INFO: Processing challenge ID %d with %d game boxes.\n", challenge.ID, len(gameboxes))

		for _, gamebox := range gameboxes {
			go func(gamebox dbold.GameBox, challenge dbold.Challenge) {
				var flag dbold.Flag
				result := dbold.MySQL.Model(&dbold.Flag{}).Where(&dbold.Flag{
					TeamID:    gamebox.TeamID,
					GameBoxID: gamebox.ID,
					Round:     timer.Get().NowRound,
				}).Find(&flag)

				if result.Error != nil {
					log.Printf("ERROR: Failed to retrieve flag for GameBox ID %d: %v\n", gamebox.ID, result.Error)
					return
				}

				// Replace the flag placeholder.
				command := strings.Replace(challenge.Command, "{{FLAG}}", flag.Flag, -1)
				log.Printf("INFO: Executing command for GameBox ID %d: %s\n", gamebox.ID, command)

				_, err := utils.SSHExecute(gamebox.IP, gamebox.SSHPort, gamebox.SSHUser, gamebox.SSHPassword, command)
				if err != nil {
					log.Printf("IMPORTANT: Team: %d GameBox: %d Round: %d Failed to plant new flag: %v\n", gamebox.TeamID, gamebox.ID, timer.Get().NowRound, err.Error())
					logger.New(logger.IMPORTANT, "system", string(fmt.Sprintf("Team: %d GameBox: %d Round: %d Failed to plant new flag: %v\n", gamebox.TeamID, gamebox.ID, timer.Get().NowRound, err.Error())))

				} else {
					log.Printf("INFO: Successfully executed command for GameBox ID %d.\n", gamebox.ID)
				}
			}(gamebox, challenge)
		}
	}
}

func TestAllSSH(c *gin.Context) (int, interface{}) {
	var challenges []dbold.Challenge
	dbold.MySQL.Model(&dbold.Challenge{}).Where(&dbold.Challenge{AutoRefreshFlag: true}).Find(&challenges)

	type errorMessage struct {
		TeamID      uint
		ChallengeID uint
		GameBoxID   uint
		Error       string
	}
	var errs []errorMessage

	wg := sync.WaitGroup{}
	for _, challenge := range challenges {
		var gameboxes []dbold.GameBox
		dbold.MySQL.Model(&dbold.GameBox{}).Where(&dbold.GameBox{ChallengeID: challenge.ID}).Find(&gameboxes)

		for _, gamebox := range gameboxes {
			wg.Add(1)
			go func(gamebox dbold.GameBox, challenge dbold.Challenge) {
				defer wg.Done()
				_, err := utils.SSHExecute(gamebox.IP, gamebox.SSHPort, gamebox.SSHUser, gamebox.SSHPassword, "whoami")
				if err != nil {
					errs = append(errs, errorMessage{
						TeamID:      gamebox.TeamID,
						ChallengeID: challenge.ID,
						GameBoxID:   gamebox.ID,
						Error:       err.Error(),
					})
				}
			}(gamebox, challenge)
		}
	}
	wg.Wait()
	return utils.MakeSuccessJSON(errs)
}

func TestSSH(c *gin.Context) (int, interface{}) {
	var inputForm struct {
		IP       string `binding:"required"`
		Port     string `binding:"required"`
		User     string `binding:"required"`
		Password string `binding:"required"`
		Command  string `binding:"required"`
	}
	err := c.BindJSON(&inputForm)
	if err != nil {
		return utils.MakeErrJSON(400, 40036,
			locales.I18n.T(c.GetString("lang"), "general.error_payload"),
		)
	}

	output, err := utils.SSHExecute(inputForm.IP, inputForm.Port, inputForm.User, inputForm.Password, inputForm.Command)
	if err != nil {
		return utils.MakeErrJSON(400, 40037, err)
	}
	return utils.MakeSuccessJSON(output)
}

func GetLatestScoreRound() int {
	var latestScore dbold.Score
	dbold.MySQL.Model(&dbold.Score{}).Order("`round` DESC").Limit(1).Find(&latestScore)
	return latestScore.Round
}
