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
	RaidHash         uint32
	Completed        bool
	Flawless         *bool
	Fresh            *bool
	PlayerCount      int
	DateStarted      time.Time
	DateCompleted    time.Time
	DurationSeconds  int
	MembershipType   int
	PlayerActivities []ProcessedPlayerActivity
}

type ProcessedPlayerActivity struct {
	DidFinish         bool
	Kills             int
	Deaths            int
	Assists           int
	TimePlayedSeconds int
	ClassHash         *uint32
	Player            postgres.Player
}

func ProcessDestinyReport(report *DestinyPostGameCarnageReport) (*ProcessedActivity, error) {
	startDate, err := time.Parse(time.RFC3339, report.Period)
	if err != nil {
		return nil, err
	}

	expectedEntryCount := int(report.Entries[0].Values["playerCount"].Basic.Value)
	actualEntryCount := len(report.Entries)
	if expectedEntryCount >= 0 && actualEntryCount != expectedEntryCount {
		return nil, fmt.Errorf("malformed pgcr: invalid entry length: %d != %d", actualEntryCount, expectedEntryCount)
	}

	noOnePlayed := true
	for _, e := range report.Entries {
		if int(e.Values["activityDurationSeconds"].Basic.Value) != 0 {
			noOnePlayed = false
			break
		}
	}
	if noOnePlayed {
		return nil, errors.New("malformed pgcr: no one had any duration_seconds")
	}

	players := make(map[string][]DestinyPostGameCarnageReportEntry)

	for _, e := range report.Entries {
		if val, ok := players[e.Player.DestinyUserInfo.MembershipId]; ok {
			players[e.Player.DestinyUserInfo.MembershipId] = append(val, e)
		} else {
			players[e.Player.DestinyUserInfo.MembershipId] = []DestinyPostGameCarnageReportEntry{e}
		}
	}

	var processedPlayerActivities []ProcessedPlayerActivity
	for _, entries := range players {
		processedPlayerActivity := ProcessedPlayerActivity{
			Kills:             0,
			Deaths:            0,
			Assists:           0,
			TimePlayedSeconds: 0,
		}
		activityDurationSecondsValue, activityDurationSecondsExists := entries[0].Values["activityDurationSeconds"]
		activityDuration := 0
		if activityDurationSecondsExists {
			activityDuration = int(activityDurationSecondsValue.Basic.Value)
		}
		maxActivityDuration := 0
		if activityDuration == 32767 {
			maxActivityDuration = -1
		} else {
			maxActivityDuration = activityDuration
		}

		for _, entry := range entries {
			completedValue, completedExists := entry.Values["completed"]
			completionReasonValue, completionReasonExists := entry.Values["completionReason"]
			killsValue, killsExists := entry.Values["kills"]
			deathsValue, deathsExists := entry.Values["deaths"]
			assistsValue, assistsExists := entry.Values["assists"]
			timePlayedSecondsValue, timePlayedSecondsExists := entry.Values["timePlayedSeconds"]

			if !processedPlayerActivity.DidFinish && completedExists && completionReasonExists && completedValue.Basic.Value == 1 && completionReasonValue.Basic.Value == 0 {
				processedPlayerActivity.DidFinish = true
			}

			if killsExists {
				processedPlayerActivity.Kills += int(killsValue.Basic.Value)
			}

			if deathsExists {
				processedPlayerActivity.Deaths += int(deathsValue.Basic.Value)
			}

			if assistsExists {
				processedPlayerActivity.Assists += int(assistsValue.Basic.Value)
			}

			if timePlayedSecondsExists {
				processedPlayerActivity.TimePlayedSeconds += int(timePlayedSecondsValue.Basic.Value)
			}

		}

		if maxActivityDuration != -1 && processedPlayerActivity.TimePlayedSeconds > maxActivityDuration {
			processedPlayerActivity.TimePlayedSeconds = maxActivityDuration
		}

		destinyUserInfo := entries[0].Player.DestinyUserInfo
		membershipId, err := strconv.ParseInt(destinyUserInfo.MembershipId, 10, 64)
		if err != nil {
			return nil, err
		}
		classHash := entries[0].Player.ClassHash

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
			if classHash != 0 {
				processedPlayerActivity.ClassHash = new(uint32)
				*processedPlayerActivity.ClassHash = classHash
			}
		}

		processedPlayerActivities = append(processedPlayerActivities, processedPlayerActivity)
	}

	complete := false
	for _, e := range processedPlayerActivities {
		if e.DidFinish {
			complete = true
			break
		}
	}

	deathless := true
	for _, e := range processedPlayerActivities {
		if e.Deaths > 0 {
			deathless = false
			break
		}
	}

	fresh, err := isFresh(report)
	if err != nil {
		return nil, err
	}

	var flawless *bool
	if complete && deathless {
		flawless = fresh
	} else {
		flawless = new(bool) // false
	}

	instanceId, err := strconv.ParseInt(report.ActivityDetails.InstanceId, 10, 64)
	if err != nil {
		return nil, err
	}

	result := ProcessedActivity{
		InstanceId:       instanceId,
		RaidHash:         report.ActivityDetails.DirectorActivityHash,
		Completed:        complete,
		Flawless:         flawless,
		Fresh:            fresh,
		PlayerCount:      len(players),
		DateStarted:      startDate,
		DateCompleted:    CalculateDateCompleted(startDate, report.Entries[0]),
		DurationSeconds:  CalculateDurationSeconds(startDate, report.Entries[0]),
		MembershipType:   report.ActivityDetails.MembershipType,
		PlayerActivities: processedPlayerActivities,
	}

	return &result, nil
}

func CalculateDurationSeconds(startDate time.Time, entry DestinyPostGameCarnageReportEntry) int {

	durationValue, durationExists := entry.Values["activityDurationSeconds"]
	if durationExists {
		return int(durationValue.Basic.Value)
	}
	return 0
}

func CalculateDateCompleted(startDate time.Time, entry DestinyPostGameCarnageReportEntry) time.Time {

	durationValue, durationExists := entry.Values["activityDurationSeconds"]
	if durationExists {
		duration := time.Duration(durationValue.Basic.Value) * time.Second
		return startDate.Add(duration)
	}
	return startDate
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
func isFresh(pgcr *DestinyPostGameCarnageReport) (*bool, error) {
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
