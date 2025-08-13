package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"
)

var Done = make(chan bool)

type Application_Status struct {
	id                   int
	application_name     string
	application_addresss string
	response_code        int
	last_seen            string
}

func application_status_daemon(db *sql.DB) {

	rows, err := db.Query("SELECT * FROM APPLICATION_STATUS")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	application_status := Application_Status{
		id:                   0,
		application_name:     "",
		application_addresss: "",
		response_code:        0,
		last_seen:            "",
	}
	c_arr := []chan bool{}

	for rows.Next() {
		if err := rows.Scan(&application_status.id, &application_status.application_name, &application_status.application_addresss,
			&application_status.response_code, &application_status.last_seen); err != nil {

		}
		// Create a seperate go routine here to ping each server independently.
		infoLog.Println(application_status)
		temp_ch := make(chan bool)
		c_arr = append(c_arr, temp_ch)
		ticker := time.NewTicker(1000 * time.Millisecond)
		go application_ping_worker(ticker, application_status, temp_ch)
	}
	infoLog.Println(c_arr)
}

func application_ping_worker(t *time.Ticker, as Application_Status, ch <-chan bool) {
	for {
		select {
		// Exit the loop & kill goroutines when received channel.
		case <-ch:
			return
		// Do work when ticker ticks
		case <-t.C:
			rs, err := http.Get(as.application_addresss)
			if err != nil {
				log.Fatal(err)
			}
			// update DB here.
			stmt, err := db.Prepare(`UPDATE APPLICATION_STATUS SET 
									last_seen = ?
									where application_id = ?;`)

			defer stmt.Close()
			if err != nil {
				log.Fatal(err)
			}
			infoLog.Println("here")
			ct := time.Now()
			fct := ct.Format("2006-01-02 15:04:05")
			_, err = stmt.Exec(fct, as.id)
			if err != nil {
				log.Fatal(err)
			}
			infoLog.Println(rs.StatusCode)
		}
	}
}
