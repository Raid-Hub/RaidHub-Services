package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Instance struct {
	InstanceId                 int64
	Hash                       uint32
	Completed                  bool
	PlayerCount                int
	Fresh, Flawless            uint8
	DateStarted, DateCompleted time.Time
	PlatformType               uint16
	Duration, Score            int
}

type InstancePlayer struct {
	InstanceId        int64
	MembershipId      int64
	Completed         bool
	TimePlayedSeconds int
	Sherpas           int
	IsFirstClear      bool
}

type InstanceCharacter struct {
	InstanceId        int64
	MembershipId      int64
	CharacterId       int64
	ClassHash         uint32
	EmblemHash        uint32
	Completed         bool
	Score             int
	Kills             int
	Assists           int
	Deaths            int
	PrecisionKills    int
	SuperKills        int
	GrenadeKills      int
	MeleeKills        int
	TimePlayedSeconds int
	StartSeconds      int
}

type InstanceCharacterWeapon struct {
	InstanceId     int64
	MembershipId   int64
	CharacterId    int64
	WeaponHash     uint32
	Kills          int
	PrecisionKills int
}

func InsertInstances(conn driver.Conn, instances []Instance) error {
	ctx := context.Background()
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO instance SETTINGS async_insert=1, wait_for_async_insert=1")
	if err != nil {
		return fmt.Errorf("error preparing batch for instances: %s", err)
	}

	for _, instance := range instances {
		err = batch.Append(instance.InstanceId, instance.Hash, instance.Completed, instance.PlayerCount, instance.Fresh, instance.Flawless, instance.DateStarted, instance.DateCompleted, instance.PlatformType, instance.Duration, instance.Score)
		if err != nil {
			return err
		}
	}

	return batch.Send()
}

func InsertInstancePlayers(conn driver.Conn, players []InstancePlayer) error {
	ctx := context.Background()
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO player SETTINGS async_insert=1, wait_for_async_insert=1")
	if err != nil {
		return fmt.Errorf("error preparing batch for players: %s", err)
	}

	for _, player := range players {
		err = batch.Append(player.InstanceId, player.MembershipId, player.Completed, player.TimePlayedSeconds, player.Sherpas, player.IsFirstClear)
		if err != nil {
			return err
		}

	}
	return batch.Send()
}

func InsertInstanceCharacters(conn driver.Conn, characters []InstanceCharacter) error {
	ctx := context.Background()
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO character SETTINGS async_insert=1, wait_for_async_insert=1")
	if err != nil {
		return fmt.Errorf("error preparing batch for characters: %s", err)
	}

	for _, character := range characters {
		err = batch.Append(character.InstanceId, character.MembershipId, character.CharacterId, character.ClassHash, character.EmblemHash, character.Completed, character.Score, character.Kills, character.Assists, character.Deaths, character.PrecisionKills, character.SuperKills, character.GrenadeKills, character.MeleeKills, character.TimePlayedSeconds, character.StartSeconds)
		if err != nil {
			return err
		}

	}
	return batch.Send()
}

func InsertCharacterWeapons(conn driver.Conn, weapons []InstanceCharacterWeapon) error {
	ctx := context.Background()
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO weapon SETTINGS async_insert=1, wait_for_async_insert=1")
	if err != nil {
		return fmt.Errorf("error preparing batch for weapons: %s", err)
	}
	for _, weapon := range weapons {
		err = batch.Append(weapon.InstanceId, weapon.MembershipId, weapon.CharacterId, weapon.WeaponHash, weapon.Kills, weapon.PrecisionKills)
		if err != nil {
			return err
		}
	}
	return batch.Send()
}
