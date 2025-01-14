package game

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"Cardinal/internal/asteroid"
	"Cardinal/internal/conf"
	"Cardinal/internal/dbold"
	"Cardinal/internal/livelog"
	"Cardinal/internal/locales"
	"Cardinal/internal/logger"
	"Cardinal/internal/misc/webhook"
	"Cardinal/internal/timer"
	"Cardinal/internal/utils"
)

type InputForm struct {
	GameBoxID uint `binding:"required"`
}

// PerformCheckDown performs a check down operation on a specified game box.
// It verifies if the competition has started, checks for repeated check downs within the same round,
// ensures the game box exists and is visible, fetches the checkdown command from the challenge table,
// executes the command, and processes the output to determine the status of the service.
//
// Parameters:
//   - gameBoxID: The ID of the game box to perform the check down on.
//
// Returns:
//   - error: An error if any of the checks fail or if the service is down, otherwise nil.
func PerformCheckDown(gameBoxID uint) error {
	// Check down is forbidden if the competition hasn't started yet.
	if timer.Get().Status != "on" {
		return errors.New("competition hasn't started yet")
	}

	// Does it check down one gamebox repeatedly in one round?
	var repeatCheck dbold.DownAction
	dbold.MySQL.Model(&dbold.DownAction{}).Where(&dbold.DownAction{
		GameBoxID: gameBoxID,
		Round:     timer.Get().NowRound,
	}).Find(&repeatCheck)
	if repeatCheck.ID != 0 {
		return errors.New(fmt.Sprintf("repeated check down for gamebox ID: %d", gameBoxID))
	}

	// Check the gamebox is existed or not.
	var gameBox dbold.GameBox
	dbold.MySQL.Model(&dbold.GameBox{}).Where(&dbold.GameBox{Model: gorm.Model{ID: gameBoxID}}).Find(&gameBox)
	if gameBox.ID == 0 {
		return errors.New(fmt.Sprintf("gamebox not found, ID: %d", gameBoxID))
	}
	if !gameBox.Visible {
		return errors.New(fmt.Sprintf("gamebox not visible, ID: %d", gameBoxID))
	}

	// Fetch the checkdown command from the challenge table.
	var challenge dbold.Challenge
	if err := dbold.MySQL.Model(&dbold.Challenge{}).Where(&dbold.Challenge{Model: gorm.Model{ID: gameBox.ChallengeID}}).Find(&challenge).Error; err != nil {
		return errors.New(fmt.Sprintf("error finding challenge: %v", err))
	}

	// Run the checkdown command.
	command := strings.Replace(challenge.CheckdownCommand, "{{IP}}", gameBox.IP, -1)
	command = strings.Replace(command, "{{PORT}}", gameBox.Port, -1)

	args := strings.Fields(command)

	log.Printf("Executing Checkdown command for GameBox ID %d: %s\n", gameBox.ID, command)

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(fmt.Sprintf("error running checker script: %v. Output: %v", err, string(output)))
	}

	// Check the output of the checker script.
	status := strings.TrimSpace(string(output))
	log.Printf("Checker script output: %s", status)

	if status != "UP" && status != "DOWN" {
		return errors.New(fmt.Sprintf("invalid status from checker script: %s", status))
	}

	isDown := false
	if status == "DOWN" {
		isDown = true
	}

	// Save the check down.
	if err := SaveCheckDown(gameBox, gameBoxID, isDown); err != nil {
		return errors.New(fmt.Sprintf("error saving check down: %v", err))
	}

	if status == "UP" {
		log.Printf("Service is up for gamebox ID: %d", gameBoxID)
		return nil
	}

	log.Printf("Check down successful for gamebox ID: %d", gameBoxID)

	return errors.New(fmt.Sprintf("service down for gamebox ID: %d", gameBoxID))
}

// CheckDown handles the HTTP request to check the status of a game service.
// It expects a JSON payload containing the GameBoxID and returns a JSON response
// indicating whether the service is up or down.
//
// Parameters:
//   - c (*gin.Context): The Gin context which provides request and response handling.
//
// Returns:
//   - (int, interface{}): A tuple containing the HTTP status code and a JSON response.
//     The JSON response includes an error message if the service is down or an error occurs,
//     or a success message if the service is up.
func CheckDown(c *gin.Context) (int, interface{}) {
	var inputForm InputForm
	err := c.BindJSON(&inputForm)
	if err != nil {
		return utils.MakeErrJSON(400, 40026,
			locales.I18n.T(c.GetString("lang"), "general.error_payload"),
		)
	}

	err = PerformCheckDown(inputForm.GameBoxID)
	if err != nil {
		if strings.Contains(err.Error(), "service down for gamebox ID: ") {
			return utils.MakeErrJSON(500, 50001,
				locales.I18n.T(c.GetString("lang"), "general.service_down"),
			)
		} else {
			return utils.MakeErrJSON(500, 50002,
				locales.I18n.T(c.GetString("lang"), err.Error()),
			)
		}
	}

	return utils.MakeSuccessJSON(locales.I18n.T(c.GetString("lang"), "general.service_up"))
}

// SaveCheckDown saves the check down status of a game box to the database.
func SaveCheckDown(gameBox dbold.GameBox, gameBoxID uint, isDown bool) error {
	status := "UP"
	if isDown {
		status = "DOWN"
	}
	log.Printf("Saving check down for gamebox ID: %d with status: %s", gameBox.ID, status)

	// Update the gamebox status.
	if err := dbold.MySQL.Model(&dbold.GameBox{}).Where("id = ?", gameBoxID).Update("is_down", isDown).Error; err != nil {
		log.Printf("Error updating gamebox status to %s: %v", status, err)
		return err
	}

	if isDown {
		tx := dbold.MySQL.Begin()
		if err := tx.Create(&dbold.DownAction{
			TeamID:      gameBox.TeamID,
			ChallengeID: gameBox.ChallengeID,
			GameBoxID:   gameBoxID,
			Round:       timer.Get().NowRound,
		}).Error; err != nil {
			log.Printf("Error creating down action: %v", err)
			tx.Rollback()
			return err
		}
		tx.Commit()

		// Check down hook
		go webhook.Add(webhook.CHECK_DOWN_HOOK, gin.H{"team": gameBox.TeamID, "gamebox": gameBox.ID})

		// Update the gamebox status in ranking list.
		SetRankList()

		// Asteroid Unity3D
		asteroid.SendStatus(int(gameBox.TeamID), "down")

		var t dbold.Team
		if err := dbold.MySQL.Model(&dbold.Team{}).Where(&dbold.Team{Model: gorm.Model{ID: gameBox.TeamID}}).Find(&t).Error; err != nil {
			log.Printf("Error finding team: %v", err)
			return err
		}

		var challenge dbold.Challenge
		if err := dbold.MySQL.Model(&dbold.Challenge{}).Where(&dbold.Challenge{Model: gorm.Model{ID: gameBox.ChallengeID}}).Find(&challenge).Error; err != nil {
			log.Printf("Error finding challenge: %v", err)
			return err
		}

		// Live log
		if err := livelog.Stream.Write(livelog.GlobalStream, livelog.NewLine("check_down",
			gin.H{"Team": t.Name, "Challenge": challenge.Title})); err != nil {
			log.Printf("Error writing live log: %v", err)
			return err
		}
	}

	SetRankList()

	log.Printf("Check down saved successfully for gamebox ID: %d with status: %s", gameBox.ID, status)
	return nil
}

// ScheduleCheckDowns schedules PerformCheckDown for every gamebox at a random interval for each round/tick.
func ScheduleCheckDowns() {
	var gameBoxes []dbold.GameBox
	dbold.MySQL.Model(&dbold.GameBox{}).Find(&gameBoxes)

	for _, gameBox := range gameBoxes {
		go func(gameBox dbold.GameBox) {
			// Generate a random interval between 1 and round duration - 10 seconds.
			randomInterval := time.Duration(rand.Intn((int(conf.Game.RoundDuration)*60)-10)+1) * time.Second
			log.Printf("Scheduling check down for gamebox ID %d in %v", gameBox.ID, randomInterval)
			time.Sleep(randomInterval)

			if err := PerformCheckDown(gameBox.ID); err != nil {
				log.Printf("Error performing check down for gamebox ID %d: %v", gameBox.ID, err)
				logger.New(logger.IMPORTANT, "system", string(fmt.Sprintf("%v", err)))
			} else {
				log.Printf("Successfully performed check down for gamebox ID %d", gameBox.ID)
				// logger.New(logger.NORMAL, "system", string(fmt.Sprintf("Successfully performed check down for gamebox ID %d", gameBox.ID)))

			}
		}(gameBox)
	}
}
