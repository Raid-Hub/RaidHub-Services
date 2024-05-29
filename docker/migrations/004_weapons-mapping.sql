CREATE TYPE "WeaponElement" AS ENUM ('Kinetic', 'Arc', 'Solar', 'Void', 'Stasis', 'Strand');
CREATE TYPE "WeaponSlot" AS ENUM ('Kinetic', 'Energy', 'Power');
CREATE TYPE "WeaponAmmoType" AS ENUM ('Primary', 'Special', 'Heavy');
CREATE TYPE "WeaponRarity" AS ENUM ('Common', 'Uncommon', 'Rare', 'Legendary', 'Exotic');

CREATE TABLE "weapon_definition" (
    "hash" BIGINT NOT NULL PRIMARY KEY,
    "name" TEXT NOT NULL,
    "icon_path" TEXT NOT NULL,
    "element" "WeaponElement" NOT NULL,
    "slot" "WeaponSlot" NOT NULL,
    "ammo_type" "WeaponAmmoType"  NOT NULL,
    "rarity" "WeaponRarity" NOT NULL
);


CREATE OR REPLACE FUNCTION get_element(defaultDamageType SMALLINT)
RETURNS "WeaponElement" AS $$
DECLARE
    element "WeaponElement";
BEGIN
    CASE defaultDamageType
        WHEN 1 THEN element := 'Kinetic';
        WHEN 2 THEN element := 'Arc';
        WHEN 3 THEN element := 'Solar';
        WHEN 4 THEN element := 'Void';
        WHEN 6 THEN element := 'Stasis';
        WHEN 7 THEN element := 'Strand';
        ELSE RAISE EXCEPTION 'Invalid defaultDamageType';
    END CASE;
    
    RETURN element;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION get_ammo_type(ammoType SMALLINT)
RETURNS "WeaponAmmoType" AS $$
DECLARE
    ammo "WeaponAmmoType";
BEGIN
    CASE ammoType
        WHEN 1 THEN ammo := 'Primary';
        WHEN 2 THEN ammo := 'Special';
        WHEN 3 THEN ammo := 'Heavy';
        ELSE RAISE EXCEPTION 'Invalid ammoType';
    END CASE;
    
    RETURN ammo;
END;
$$ LANGUAGE plpgsql IMMUTABLE;


CREATE OR REPLACE FUNCTION get_slot(equipmentSlotTypeHash BIGINT)
RETURNS "WeaponSlot" AS $$
DECLARE
    slot "WeaponSlot";
BEGIN
    CASE equipmentSlotTypeHash
        WHEN 1498876634 THEN slot := 'Kinetic';
        WHEN 2465295065 THEN slot := 'Energy';
        WHEN 953998645 THEN slot := 'Power';
        ELSE RAISE EXCEPTION 'Invalid equipmentSlotTypeHash';
    END CASE;
    
    RETURN slot;
END;
$$ LANGUAGE plpgsql IMMUTABLE;