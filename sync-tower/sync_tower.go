package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"gopkg.in/yaml.v3"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Magenta = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"

var infoLog = log.New(os.Stdout, Green+"[INFO] "+Reset, log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
var warnLog = log.New(os.Stdout, Yellow+"[WARN] "+Reset, log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

var db *sql.DB
var influx_client influxdb2.Client

type (
	AppConfig struct {
		ListenAddress       string `taml:"listen_address"`
		ListenPort          string `yaml:"listen_port"`
		UplinkPath          string `yaml:"uplink_path"`
		DatabaseUrl         string `yaml:"database_url"`
		InfluxdbEnable      string `yaml:"influxdb_enable"`
		InfluxdbVersion     string `yaml:"influxdb_version"`
		InfluxdbUrl         string `yaml:"influxdb_url"`
		InfluxdbToken       string `yaml:"influxdb_token"`
		InfluxdbOrg         string `yaml:"influxdb_org"`
		InfluxdbBucket      string `yaml:"influxdb_bucket"`
		InfluxdbMeasurement string `yaml:"influxdb_measurement"`
	}

	ConfigFile map[string]*AppConfig
)

func LoadConfig(env string) (*AppConfig, error) {
	configFile := ConfigFile{}
	file, _ := os.Open("config.yaml")
	defer file.Close()
	decoder := yaml.NewDecoder(file)

	// Always check for errors!
	if err := decoder.Decode(&configFile); err != nil {
		return nil, err
	}

	appConfig, ok := configFile[env]
	if !ok {
		return nil, fmt.Errorf("no such environment: %s", env)
	}

	return appConfig, nil
}

func main() {
	mode := "dev"

	infoLog.Println("Sync Tower application is starting in " + Blue + mode + Reset + " mode")

	infoLog.Println("Loading " + Blue + "./config.yaml" + Reset + " configuration file...")
	appConfig, err := LoadConfig(mode)
	if err != nil {
		panic(err)
	}
	infoLog.Println(Green + "Successfully " + Reset + "loaded " + Blue + "config.yaml" + Reset + " configuration file!")

	infoLog.Println("Connecting to postgres database...")
	db, err = sql.Open("pgx", appConfig.DatabaseUrl)
	if err != nil {
		panic(err)
	}
	infoLog.Println(Green + "Successfully " + Reset + "connected to postgres database!")

	if appConfig.InfluxdbEnable == "y" {
		infoLog.Println("Creating InfluxDB Client...")
		influx_client = influxdb2.NewClient(appConfig.InfluxdbUrl, appConfig.InfluxdbToken)
		infoLog.Print(Green + "Successfully " + Reset + "created InfluxDB Client Object!")
	} else {
		infoLog.Println("InfluxDB integration disabled")
	}

	infoLog.Println("Starting http server configuration with " + Blue + "port:" + appConfig.ListenPort + " path:" + appConfig.UplinkPath + Reset + "...")

	infoLog.Println("Configuring routes...")
	http.HandleFunc(appConfig.UplinkPath, func(w http.ResponseWriter, r *http.Request) {
		uplinkHandler(w, r, appConfig)
	})
	infoLog.Println(Green + "Successfully " + Reset + "configured routes!")

	server := &http.Server{
		Addr:         appConfig.ListenAddress + ":" + appConfig.ListenPort,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown setup
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine so it doesn't block
	go func() {
		infoLog.Printf(Green+"Server starting on %s"+Reset, server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			infoLog.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	// Reading from channel is a blocking operation
	// Thread will stall here until a value because available in the channel
	<-done
	infoLog.Println("Server is shutting down...")

	// Create a deadline to wait for existing connections to finish
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		infoLog.Printf("Graceful shutdown failed: %v\n", err)
	} else {
		infoLog.Println("Server stopped")
	}

}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")
}

func uplinkHandler(w http.ResponseWriter, r *http.Request, appConfig *AppConfig) {

	enableCors(&w)

	//Early return is not POST request
	if r.Method != "POST" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed) // 405 status code
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Only POST method is supported",
		})
		return // Important: return to stop further execution
	}

	infoLog.Println(Magenta + "INBOUND : " + Reset + Blue + r.Method + " " + r.RequestURI + Reset + Magenta + " Source : " + Reset + Blue + r.RemoteAddr + Reset)

	var parsed map[string]any
	json.NewDecoder(r.Body).Decode(&parsed)

	devEui, _ := getNestedString(parsed, "deviceInfo", "devEui")
	payload_time, _ := getNestedString(parsed, "time")
	print(payload_time)
	parsedTime, _ := time.Parse("2006-01-02T15:04:05.999Z07:00", payload_time)

	//Inbound processing Postgres here
	sqlStatement := ` INSERT INTO chirpstack_ingest (dev_eui, tenant_name, application_name, raw_payload)
							VALUES ($1, $2, $3, $4);`
	_, err := db.Exec(sqlStatement, devEui, "cache-sync", "CSB-DEMO", parsed)

	if err != nil {
		warnLog.Println(err)
	}

	//Inbound processinf InfluxDB here.
	if appConfig.InfluxdbEnable == "y" {
		writeAPI := influx_client.WriteAPIBlocking(appConfig.InfluxdbOrg, appConfig.InfluxdbBucket)

		tags := map[string]string{
			"dev_eui": devEui,
		}
		fields := map[string]any{}
		for key, value := range parsed {
			if key == "object" {
				if devEui == "009569060003e9be" {
					temp_val := value.(map[string]any)
					fields = temp_val["data"].(map[string]any)
				} else {
					fields = value.(map[string]any)
				}

			}
		}

		point := write.NewPoint(appConfig.InfluxdbMeasurement, tags, fields, parsedTime)
		if err := writeAPI.WritePoint(context.Background(), point); err != nil {
			log.Fatal(err)
		}
	}

	//Response prep
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	payload := struct {
		Status string
	}{
		Status: "OK",
	}

	json.NewEncoder(w).Encode(payload)
}

func getNestedString(m map[string]interface{}, keys ...string) (string, error) {
	var val interface{} = m
	for _, key := range keys {
		if m, ok := val.(map[string]interface{}); ok {
			val = m[key]
		} else {
			return "", fmt.Errorf("invalid path")
		}
	}
	if s, ok := val.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("not a string")
}
