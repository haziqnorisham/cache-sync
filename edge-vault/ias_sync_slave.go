package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
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

var db, err = sql.Open("sqlite", "sqlite.db")

type (
	AppConfig struct {
		DatabaseUrl        string `yaml:"database_url"`
		MqttBrokerAddress  string `yaml:"mqtt_broker_address"`
		MqttBrokerPort     string `yaml:"mqtt_broker_port"`
		MqttBrokerUser     string `yaml:"mqtt_broker_user"`
		MqttBrokerPassword string `yaml:"mqtt_broker_password"`
		UplinkEndpoint     string `yaml:"uplink_endpoint"`
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

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	var parsed map[string]interface{}
	topic := msg.Topic()
	msgId := uuid.New().String()
	parts := strings.Split(topic, "/")
	infoLog.Println(Cyan + msgId + Reset + Magenta + " Received message!")
	appId := parts[1]
	deviceId := parts[3]
	eventType := parts[5]
	infoLog.Println(Cyan + msgId + Reset + Blue + " App_ID=" + appId + Reset)
	infoLog.Println(Cyan + msgId + Reset + Blue + " Device_ID=" + deviceId + Reset)
	infoLog.Println(Cyan + msgId + Reset + Blue + " Event_Type=" + eventType + Reset)
	if eventType == "up" {
		infoLog.Println(Cyan + msgId + Reset + " Supported event type, processing payload...")
		payload := msg.Payload()
		json.Unmarshal(payload, &parsed)
		var dedupeId string
		if rawVal, exists := parsed["deduplicationId"]; exists {
			dedupeId = rawVal.(string)
		}
		infoLog.Println(Cyan + msgId + Reset + Blue + " Deduplication_ID=" + dedupeId + Reset)
		infoLog.Println(Cyan + msgId + Reset + " Inserting data into uplink queue...")
		sqlStatement := ` INSERT INTO UPLINK_QUEUE (msg_id, deduplication_id, payload) 
							VALUES ($1, $2, $3);`
		_, err = db.Exec(sqlStatement, msgId, dedupeId, payload)
		if err != nil {
			panic(err)
		}
		infoLog.Println(Cyan + msgId + Reset + Green + " Successfully " + Reset + "queue data for uplink!")
		infoLog.Println(Cyan + msgId + Reset + Green + " Successfully " + Reset + "proccessed payload")
	} else {
		infoLog.Println(Cyan + msgId + Reset + Blue + " Unsupported event type, skipped payload processing")
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	infoLog.Println(Green + "Successfully " + Reset + "connected to MQTT Broker")
	subscribed_topic := "application/#"
	token := client.Subscribe(subscribed_topic, 1, nil)
	token.Wait()
	infoLog.Println(Green + "Subscribed to " + Reset + Blue + "Topic=" + subscribed_topic + Reset)
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	warnLog.Printf("Connection Lost, reconnecting...")
}

func main() {
	if err != nil {
		panic(err)
	}
	mode := "dev"
	infoLog.Println("Application is starting in " + Blue + mode + Reset + " mode")

	infoLog.Println("Loading " + Blue + "./config.yaml" + Reset + " configuration file...")

	appConfig, err := LoadConfig(mode)
	if err != nil {
		panic(err)
	}
	infoLog.Println(Green + "Successfully " + Reset + "loaded " + Blue + "config.yaml" + Reset + " configuration file!")

	db.SetMaxOpenConns(1)
	infoLog.Println("Initializing SQLite DB...")

	defer db.Close()
	if mode == "dev2" {
		infoLog.Println(Magenta + "DEV MODE : " + Reset + "Clearing Data...")
		_, err := db.Exec(`DROP TABLE IF EXISTS UPLINK_QUEUE;`)
		if err != nil {
			warnLog.Println(Yellow + err.Error() + Reset)
		}
		infoLog.Println(Magenta + "DEV MODE : " + Reset + Green + "Successfully " + Reset + "cleared data!")
	}
	_, err = db.Exec(`CREATE TABLE "UPLINK_QUEUE" (
						"msg_id"	TEXT NOT NULL UNIQUE,
						"id"	INTEGER,
						"deduplication_id"	TEXT NOT NULL UNIQUE,
						"payload"	TEXT NOT NULL,
						PRIMARY KEY("id" AUTOINCREMENT)
					);`)
	if err != nil {
		warnLog.Println(Yellow + err.Error() + Reset)
	}

	infoLog.Println(Green + "Successfully " + Reset + "initialized SQLite DB!")

	broker := appConfig.MqttBrokerAddress
	port := appConfig.MqttBrokerPort
	username := appConfig.MqttBrokerUser
	password := appConfig.MqttBrokerPassword
	infoLog.Println(
		"Connecting to MQTT Broker with " +
			Blue + "Address=" + broker + " Port=" + port + Reset + " ...")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%s", broker, port))
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(30 * time.Second)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	spawn_uplink_worker(appConfig)
	// Keep the program running
	select {}
}

type Uplink_Queue struct {
	id               int
	msg_id           string
	deduplication_id string
	payload          string
}

func spawn_uplink_worker(appConfig *AppConfig) {
	infoLog.Println("Spawning uplink worker...")
	c_arr := make(chan bool)
	ticker := time.NewTicker(1000 * time.Millisecond)
	go uplink_worker(ticker, c_arr, appConfig)
	infoLog.Println(Green + " Successfully spawned uplink worker!" + Reset)
}

func uplink_worker(t *time.Ticker, ch <-chan bool, appConfig *AppConfig) {
	infoLog.Println(Magenta + "UPLINK WORKER : " + Reset + "Entering event loop!")
	for {
		select {
		// Exit the loop & kill goroutines when received channel.
		case <-ch:
			return
		// Do work when ticker ticks
		case <-t.C:

			rows, err := db.Query("select * from UPLINK_QUEUE ORDER BY id DESC LIMIT 20;")
			if err != nil {
				warnLog.Println("WARNING!")
				log.Fatal(err)
			}

			uplink_queue := Uplink_Queue{
				id:               0,
				msg_id:           "",
				deduplication_id: "",
				payload:          "",
			}

			print_once := false
			var msgIdArr []string
			loop_entered := false
			for rows.Next() {
				loop_entered = true
				if !print_once {
					infoLog.Println(Magenta + "UPLINK WORKER : " + Reset + "Work started...")
					print_once = true
				}
				if err := rows.Scan(&uplink_queue.msg_id, &uplink_queue.id, &uplink_queue.deduplication_id,
					&uplink_queue.payload); err != nil {

				}

				infoLog.Println(Magenta + "UPLINK WORKER : " + Reset + "Uploading " + Blue + "Message_ID=" + uplink_queue.msg_id + Reset)
				err := send_uplink(appConfig, uplink_queue.payload)
				if err != nil {
					warnLog.Println(Magenta + "UPLINK WORKER : " + Reset + err.Error())
				} else {
					msgIdArr = append(msgIdArr, uplink_queue.msg_id)
					infoLog.Println(Magenta + "UPLINK WORKER : " + Reset + Green + "Successfully " + Reset + "uploaded!")
				}
			}
			rows.Close()

			for _, value := range msgIdArr {
				infoLog.Println(Magenta + "UPLINK WORKER : " + Reset + "Removing " + Blue + "Message_ID=" + value + Reset + " from upload queue...")
				_, err := db.Exec("DELETE FROM UPLINK_QUEUE WHERE msg_id=$1", value)
				if err != nil {
					panic(err)
				}
			}

			if loop_entered {
				infoLog.Println(Magenta + "UPLINK WORKER : " + Reset + Green + "Successfully " + Reset + "completed the work!")
			}
		}
	}
}

func send_uplink(appConfig *AppConfig, payload string) (err error) {

	resp, err := http.Post(appConfig.UplinkEndpoint, "application/json", bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return err
	} else {
		sc := resp.StatusCode

		if sc == 200 {
			return nil
		} else {
			return errors.New("!200 STATUS CODE")
		}
	}
}
