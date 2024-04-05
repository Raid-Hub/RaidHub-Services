package main

import (
	"database/sql"
	"log"
)

type PlayerRequest struct {
	MembershipId string `json:"membershipId"`
}

func processRequest(request *PlayerRequest, db *sql.DB) {
	// TODO
	log.Println("Processed request for membershipId:", request.MembershipId)
}
