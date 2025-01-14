package game

import (
	"Cardinal/internal/asteroid"
	"Cardinal/internal/dbold"
	"Cardinal/internal/dynamic_config"
	"Cardinal/internal/timer"
	"Cardinal/internal/utils"
)

func AsteroidGreetData() (result asteroid.Greet) {
	var asteroidTeam []asteroid.Team
	var teams []dbold.Team
	dbold.MySQL.Model(&dbold.Team{}).Order("score DESC").Find(&teams)
	for rank, team := range teams {
		asteroidTeam = append(asteroidTeam, asteroid.Team{
			Id:    int(team.ID),
			Name:  team.Name,
			Rank:  rank + 1,
			Image: team.Logo,
			Score: int(team.Score),
		})
	}

	result.Title = dynamic_config.Get(utils.TITLE_CONF)
	result.Team = asteroidTeam
	result.Time = timer.Get().RoundRemainTime
	result.Round = timer.Get().NowRound
	return
}
