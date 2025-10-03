package main

import (
	"net/http"
	"os"
	"text/template"
)

var ws_prefix = Cyan + "[net/http] " + Reset
var glob_appConfig *AppConfig

func StartServer(appConfig *AppConfig) {
	glob_appConfig = appConfig
	infoLog.Println(ws_prefix + "Setting up net/http routes...")
	//Static file route handler
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("templates/static/"))))

	http.HandleFunc("/", helloHandler)

	infoLog.Println(ws_prefix + Green + "Successfully " + Reset + "Configured routes!")

	infoLog.Println(ws_prefix + Green + "Serving " + Reset + "web service on port : " + Blue + appConfig.WebPort + Reset)
	http.ListenAndServe(":"+appConfig.WebPort, nil)
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	fileInfo, _ := os.Stat("./sqlite.db")
	fileSize := fileInfo.Size()
	fileSizeMB := float64(fileSize) / float64(1024)
	fileSizeMB = fileSizeMB / 1024

	rows, _ := db.Query(`select count(*) from UPLINK_QUEUE`)
	var aaa int64
	for rows.Next() {
		err := rows.Scan(&aaa)
		if err != nil {
			println(err)
		}
	}
	hostname, _ := os.Hostname()
	data := struct {
		Hostname          string
		DatabaseUrl       string
		MqttBrokerAddress string
		MqttBrokerPort    string
		MqttBrokerUser    string
		UplinkEndpoint    string
		WebUiPort         string
		CacheSize         float64
		CacheRowCount     int64
	}{
		Hostname:          hostname,
		DatabaseUrl:       glob_appConfig.DatabaseUrl,
		MqttBrokerAddress: glob_appConfig.MqttBrokerAddress,
		MqttBrokerPort:    glob_appConfig.MqttBrokerPort,
		MqttBrokerUser:    glob_appConfig.MqttBrokerUser,
		UplinkEndpoint:    glob_appConfig.UplinkEndpoint,
		WebUiPort:         glob_appConfig.WebPort,
		CacheSize:         fileSizeMB,
		CacheRowCount:     aaa}

	tmpl.Execute(w, data)
}
