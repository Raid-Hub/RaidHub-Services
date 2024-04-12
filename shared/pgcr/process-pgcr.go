package pgcr

import (
	"errors"
	"fmt"
	"log"
	"raidhub/shared/bungie"
	"raidhub/shared/postgres"
	"strconv"
	"time"
)

type ProcessedActivity struct {
	InstanceId       int64
	Hash             uint32
	Completed        bool
	Flawless         *bool
	Fresh            *bool
	PlayerCount      int
	DateStarted      time.Time
	DateCompleted    time.Time
	DurationSeconds  int
	MembershipType   int
	Score            int
	PlayerActivities []ProcessedPlayerActivity
}

type ProcessedPlayerActivity struct {
	Finished          bool
	TimePlayedSeconds int
	Player            postgres.Player
	Characters        []ProcessedPlayerActivityCharacter
}

type ProcessedPlayerActivityCharacter struct {
	CharacterId       int64
	ClassHash         *uint32
	EmblemHash        *uint32
	Completed         bool
	Score             int
	Kills             int
	Deaths            int
	Assists           int
	PrecisionKills    int
	SuperKills        int
	GrenadeKills      int
	MeleeKills        int
	StartSeconds      int
	TimePlayedSeconds int
	Weapons           []ProcessedCharacterActivityWeapon
}

type ProcessedCharacterActivityWeapon struct {
	WeaponHash     uint32
	Kills          int
	PrecisionKills int
}

func ProcessDestinyReport(report *bungie.DestinyPostGameCarnageReport) (*ProcessedActivity, error) {
	startDate, err := time.Parse(time.RFC3339, report.Period)
	if err != nil {
		return nil, err
	}

	expectedEntryCount := getStat(report.Entries[0].Values, "playerCount")
	actualEntryCount := len(report.Entries)
	if expectedEntryCount >= 0 && actualEntryCount != expectedEntryCount {
		return nil, fmt.Errorf("malformed pgcr: invalid entry length: %d != %d", actualEntryCount, expectedEntryCount)
	}

	noOnePlayed := true
	for _, e := range report.Entries {
		if getStat(e.Values, "activityDurationSeconds") != 0 {
			noOnePlayed = false
			break
		}
	}
	if noOnePlayed {
		return nil, errors.New("malformed pgcr: no one had any duration_seconds")
	}

	instanceId, err := strconv.ParseInt(report.ActivityDetails.InstanceId, 10, 64)
	if err != nil {
		return nil, err
	}

	completionReason := getStat(report.Entries[0].Values, "completionReason")

	fresh, err := isFresh(report)
	if err != nil {
		return nil, err
	}

	result := ProcessedActivity{
		InstanceId:      instanceId,
		Hash:            report.ActivityDetails.DirectorActivityHash,
		Fresh:           fresh,
		DateStarted:     startDate,
		DateCompleted:   CalculateDateCompleted(startDate, report.Entries[0]),
		DurationSeconds: CalculateDurationSeconds(startDate, report.Entries[0]),
		MembershipType:  report.ActivityDetails.MembershipType,
		Score:           getStat(report.Entries[0].Values, "teamScore"),
	}

	players := make(map[string][]bungie.DestinyPostGameCarnageReportEntry)

	for _, e := range report.Entries {
		if val, ok := players[e.Player.DestinyUserInfo.MembershipId]; ok {
			players[e.Player.DestinyUserInfo.MembershipId] = append(val, e)
		} else {
			players[e.Player.DestinyUserInfo.MembershipId] = []bungie.DestinyPostGameCarnageReportEntry{e}
		}
	}

	var processedPlayerActivities []ProcessedPlayerActivity
	for _, entries := range players {
		processedPlayerActivity := ProcessedPlayerActivity{
			Characters: []ProcessedPlayerActivityCharacter{},
			Player:     postgres.Player{},
		}

		for _, entry := range entries {
			characterId, err := strconv.ParseInt(entry.CharacterId, 10, 64)
			if err != nil {
				return nil, err
			}
			character := ProcessedPlayerActivityCharacter{
				CharacterId: characterId,
				Completed:   getStat(entry.Values, "completed") == 1,
				Weapons:     []ProcessedCharacterActivityWeapon{},
			}
			if entry.Player.ClassHash != 0 {
				character.ClassHash = new(uint32)
				*character.ClassHash = entry.Player.ClassHash
			}
			if entry.Player.EmblemHash != 0 {
				character.EmblemHash = new(uint32)
				*character.EmblemHash = entry.Player.EmblemHash
			}

			character.Score = getStat(entry.Values, "score")
			character.Score = getStat(entry.Values, "completionReason")
			character.Kills = getStat(entry.Values, "kills")
			character.Deaths = getStat(entry.Values, "deaths")
			character.Assists = getStat(entry.Values, "assists")
			character.TimePlayedSeconds = getStat(entry.Values, "timePlayedSeconds")
			character.StartSeconds = getStat(entry.Values, "startSeconds")
			if entry.Extended != nil {
				character.PrecisionKills = getStat(entry.Extended.Values, "precisionKills")
				character.SuperKills = getStat(entry.Extended.Values, "weaponKillsSuper")
				character.GrenadeKills = getStat(entry.Extended.Values, "weaponKillsGrenade")
				character.MeleeKills = getStat(entry.Extended.Values, "weaponKillsMelee")

				for _, weapon := range entry.Extended.Weapons {
					processedWeapon := ProcessedCharacterActivityWeapon{
						WeaponHash: weapon.ReferenceId,
					}
					processedWeapon.Kills = getStat(weapon.Values, "uniqueWeaponKills")
					processedWeapon.PrecisionKills = getStat(weapon.Values, "uniqueWeaponPrecisionKills")
					character.Weapons = append(character.Weapons, processedWeapon)
				}
			}

			processedPlayerActivity.Characters = append(processedPlayerActivity.Characters, character)

			processedPlayerActivity.Finished = processedPlayerActivity.Finished || (character.Completed && completionReason == 0)
		}

		processedPlayerActivity.TimePlayedSeconds = calculatePlayerTimePlayedSeconds(entries)

		destinyUserInfo := entries[0].Player.DestinyUserInfo
		membershipId, err := strconv.ParseInt(destinyUserInfo.MembershipId, 10, 64)
		if err != nil {
			return nil, err
		}

		processedPlayerActivity.Player.LastSeen = startDate
		processedPlayerActivity.Player.MembershipId = membershipId
		if destinyUserInfo.MembershipType != 0 {
			processedPlayerActivity.Player.MembershipType = new(int)
			*processedPlayerActivity.Player.MembershipType = destinyUserInfo.MembershipType
			processedPlayerActivity.Player.IconPath = new(string)
			*processedPlayerActivity.Player.IconPath = *destinyUserInfo.IconPath
			processedPlayerActivity.Player.DisplayName = new(string)
			*processedPlayerActivity.Player.DisplayName = *destinyUserInfo.DisplayName
			if destinyUserInfo.BungieGlobalDisplayNameCode != nil {
				processedPlayerActivity.Player.BungieGlobalDisplayNameCode = new(string)
				*processedPlayerActivity.Player.BungieGlobalDisplayNameCode = *bungie.FixBungieGlobalDisplayNameCode(destinyUserInfo.BungieGlobalDisplayNameCode)
				if destinyUserInfo.BungieGlobalDisplayName != nil && *destinyUserInfo.BungieGlobalDisplayName != "" {
					processedPlayerActivity.Player.BungieGlobalDisplayName = new(string)
					*processedPlayerActivity.Player.BungieGlobalDisplayName = *destinyUserInfo.BungieGlobalDisplayName
				}
			}
		}

		processedPlayerActivities = append(processedPlayerActivities, processedPlayerActivity)
	}

	result.PlayerActivities = processedPlayerActivities
	result.PlayerCount = len(players)

	result.Completed = false
	for _, e := range processedPlayerActivities {
		if e.Finished {
			result.Completed = true
			break
		}
	}

	deathless := true
	for _, e := range processedPlayerActivities {
		for _, c := range e.Characters {
			if c.Deaths > 0 {
				deathless = false
				break
			}
		}
		if !deathless {
			break
		}
	}

	if result.Completed && deathless {
		result.Flawless = fresh
	} else {
		result.Flawless = new(bool) // false
	}

	return &result, nil
}

func getStat(values map[string]bungie.DestinyHistoricalStatsValue, key string) int {
	if stat, ok := values[key]; ok {
		return int(stat.Basic.Value)
	} else {
		return 0
	}
}

func calculatePlayerTimePlayedSeconds(characters []bungie.DestinyPostGameCarnageReportEntry) int {
	timeline := make([]int, getStat(characters[0].Values, "activityDurationSeconds")+1)
	for _, character := range characters {
		startSecond := getStat(character.Values, "startSeconds")
		timePlayedSeconds := getStat(character.Values, "timePlayedSeconds")
		endSecond := startSecond + timePlayedSeconds

		timeline[startSecond]++
		timeline[endSecond]--
	}

	durationSeconds := 0
	currentCharacters := 0
	for _, val := range timeline {
		currentCharacters += val
		if currentCharacters > 0 {
			durationSeconds++
		}
	}

	return durationSeconds
}

func CalculateDurationSeconds(startDate time.Time, entry bungie.DestinyPostGameCarnageReportEntry) int {
	return getStat(entry.Values, "activityDurationSeconds")
}

func CalculateDateCompleted(startDate time.Time, entry bungie.DestinyPostGameCarnageReportEntry) time.Time {
	seconds := getStat(entry.Values, "activityDurationSeconds")
	return startDate.Add(time.Duration(seconds) * time.Second)
}

var (
	beyondLightStart = time.Date(2020, time.November, 10, 9, 0, 0, 0, time.FixedZone("PST", -8*60*60)).Unix()
	witchQueenStart  = time.Date(2022, time.February, 22, 9, 0, 0, 0, time.FixedZone("PST", -8*60*60)).Unix()
	hauntedStart     = time.Date(2022, time.May, 24, 10, 0, 0, 0, time.FixedZone("PDT", -7*60*60)).Unix()
)

var leviHashes = map[uint32]bool{
	2693136600: true, 2693136601: true, 2693136602: true,
	2693136603: true, 2693136604: true, 2693136605: true,
	89727599: true, 287649202: true, 1699948563: true, 1875726950: true,
	3916343513: true, 4039317196: true, 417231112: true, 508802457: true,
	757116822: true, 771164842: true, 1685065161: true, 1800508819: true,
	2449714930: true, 3446541099: true, 4206123728: true, 3912437239: true,
	3879860661: true, 3857338478: true,
}

// isFresh checks if a DestinyPostGameCarnageReportData is considered fresh based on the period start time.
func isFresh(pgcr *bungie.DestinyPostGameCarnageReport) (*bool, error) {
	var result *bool = nil

	start, err := time.Parse(time.RFC3339, pgcr.Period)
	if err != nil {
		log.Printf("Error parsing 'period' for %s: %s", pgcr.ActivityDetails.InstanceId, err)
		return nil, err
	}

	startUnix := start.Unix()

	if startUnix < witchQueenStart {
		if startUnix < beyondLightStart {
			result = new(bool)
			if pgcr.ActivityDetails.DirectorActivityHash == 548750096 || pgcr.ActivityDetails.DirectorActivityHash == 2812525063 {
				// sotp
				*result = (pgcr.StartingPhaseIndex <= 1)
			} else if leviHashes[pgcr.ActivityDetails.DirectorActivityHash] {
				*result = (pgcr.StartingPhaseIndex == 0 || pgcr.StartingPhaseIndex == 2)
			} else {
				*result = (pgcr.StartingPhaseIndex == 0)
			}
		}
	} else {
		if pgcr.ActivityWasStartedFromBeginning {
			result = new(bool)
			*result = true
		} else if startUnix > hauntedStart {
			result = new(bool)
			*result = pgcr.ActivityWasStartedFromBeginning
		}
	}

	return result, nil
}
