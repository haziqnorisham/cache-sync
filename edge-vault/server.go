package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/template"
	"time"

	_ "github.com/lib/pq"
)

var db_psql = db_mngr.db_pgsql

var ws_prefix = Cyan + "[net/http] " + Reset
var glob_appConfig *AppConfig

func StartServer(appConfig *AppConfig) {
	glob_appConfig = appConfig
	infoLog.Println(ws_prefix + "Setting up net/http routes...")
	//Static file route handler
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("templates/static/"))))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/data", dataTracerHandler)
	http.HandleFunc("/data_result", dataResultHandler)
	http.HandleFunc("/maintenance", maintenanceHandler)

	infoLog.Println(ws_prefix + Green + "Successfully " + Reset + "Configured routes!")

	infoLog.Println(ws_prefix + Green + "Serving " + Reset + "web service on port : " + Blue + appConfig.WebPort + Reset)
	http.ListenAndServe(":"+appConfig.WebPort, nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
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

	uplink_count := db_get_data_count_chart()
	uplink_count_json, _ := json.Marshal(uplink_count)

	data := struct {
		Hostname          string
		DatabaseHost      string
		MqttBrokerAddress string
		MqttBrokerPort    string
		MqttBrokerUser    string
		UplinkEndpoint    string
		WebUiPort         string
		CacheSize         float64
		CacheRowCount     int64
		UplinkCount       any
	}{
		Hostname:          hostname,
		DatabaseHost:      glob_appConfig.DatabaseHost,
		MqttBrokerAddress: glob_appConfig.MqttBrokerAddress,
		MqttBrokerPort:    glob_appConfig.MqttBrokerPort,
		MqttBrokerUser:    glob_appConfig.MqttBrokerUser,
		UplinkEndpoint:    glob_appConfig.UplinkEndpoint,
		WebUiPort:         glob_appConfig.WebPort,
		CacheSize:         fileSizeMB,
		CacheRowCount:     aaa,
		UplinkCount:       string(uplink_count_json),
	}

	tmpl.Execute(w, data)
}

func dataTracerHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/data_tracer.html"))
	device_names := db_get_device_names()
	tmpl.Execute(w, device_names)
}

func dataResultHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/data_result.html"))

	// Get GET form data from URL parameters
	query := r.URL.Query()
	startDatetime := query.Get("datetime_start")
	endDatetime := query.Get("datetime_end")
	deviceName := query.Get("device_name")

	query_results := db_get_data_trace_result(startDatetime, endDatetime, deviceName)

	query_form := struct {
		Datetime_Start string
		Datetime_End   string
		Device_Name    string
	}{
		Datetime_Start: startDatetime,
		Datetime_End:   endDatetime,
		Device_Name:    deviceName,
	}
	data := struct {
		Query_Form    any
		Query_Results any
	}{
		Query_Form:    query_form,
		Query_Results: query_results,
	}

	tmpl.Execute(w, data)
}

func maintenanceHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/maintenance.html"))
	tmpl.Execute(w, nil)
}

func db_get_device_names() []string {
	rows, err := db_psql.Query("select distinct eu.device_name from event_up eu;")
	if err != nil {
		warnLog.Println("ERROR!" + err.Error())
	}
	device_names := []string{}
	for rows.Next() {
		device_name := ""
		if err := rows.Scan(&device_name); err != nil {
		} else {
			device_names = append(device_names, device_name)
		}
	}
	return device_names
}

func db_get_data_trace_result(datetime_start string, datetime_end string, device_name string) (result any) {
	query_string := fmt.Sprintf("select time, device_name, object from event_up where time > '%s'::TIMESTAMP AT TIME ZONE 'Asia/Kuala_Lumpur' and time < '%s'::TIMESTAMP AT TIME ZONE 'Asia/Kuala_Lumpur' and device_name = '%s' and object != '{}' order by time desc;",
		datetime_start, datetime_end, device_name)

	rows, err := db_psql.Query(query_string)
	if err != nil {
		warnLog.Println("ERROR!" + err.Error())
	}

	query_row := struct {
		Time        time.Time
		Device_Name string
		Data        string
	}{
		Time:        time.Now(),
		Device_Name: "",
		Data:        "",
	}
	Data := []any{}
	for rows.Next() {
		if err := rows.Scan(&query_row.Time, &query_row.Device_Name, &query_row.Data); err != nil {
		} else {
			loc, _ := time.LoadLocation("Local")
			query_row.Time = query_row.Time.In(loc)
			Data = append(Data, query_row)
		}
	}
	return Data
}

func db_get_data_count_chart() any {
	query_string := "SELECT DATE_TRUNC('hour', time) AS hour_start, COUNT(*) AS row_count FROM event_up WHERE time >= NOW() - INTERVAL '24 hours' GROUP BY hour_start ORDER BY hour_start;"
	rows, _ := db_psql.Query(query_string)
	row := struct {
		Hour_Start   time.Time
		Uplink_Count int64
	}{
		Hour_Start:   time.Now(),
		Uplink_Count: 0,
	}

	data := []any{}

	for rows.Next() {
		if err := rows.Scan(&row.Hour_Start, &row.Uplink_Count); err != nil {
		} else {
			data = append(data, row)
		}
	}
	return data
}
