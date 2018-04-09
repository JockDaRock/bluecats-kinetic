package main

import (
        "crypto/tls"
        "fmt"
	"log"
        "os"
	"os/signal"
        "strconv"
        "time"
	"syscall"
	"gopkg.in/ini.v1"
	"strings"
	"net/http"
	"io/ioutil"
	"encoding/json"

        MQTT "github.com/eclipse/paho.mqtt.golang"
)

type bluecatsData struct {
	Mac    string    `json:"mac"`
	Ts     time.Time `json:"ts"`
	Event  string    `json:"event"`
	EnZone string    `json:"enZone"`
}

func sparkMessage(url string, sparkToken string, email string, status string ){

	payload := strings.NewReader(fmt.Sprintf("{\r\n  \"toPersonEmail\" : \"%s\",\r\n  \"markdown\" : \"# %s \"\r\n}", email, status))

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("authorization", fmt.Sprintf("Bearer %s", sparkToken))
	//fmt.Printf("Bearer %s", sparkToken)
	req.Header.Add("Content-type", "application/json; charset=utf-8")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error: %s\n", err)
		return
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	fmt.Println(res)
	fmt.Println(string(body))
}

func onMessageReceived(client MQTT.Client, message MQTT.Message) {
	cfg, _ := configDefine()

	url := "https://api.ciscospark.com/v1/messages"

	sparkToken := cfg.Section("SparkParams").Key("bot_token").String()
	email := cfg.Section("SparkParams").Key("email").String()

        fmt.Printf("Received message on topic: %s\nMessage: %s\n", message.Topic(), message.Payload())

	s := message.Payload()
	rawS := json.RawMessage(s)
	bc_data, err := json.Marshal(rawS)
	if err != nil {
		panic(err)
    	}

	var bcD bluecatsData
	err = json.Unmarshal(bc_data, &bcD)
    	if err != nil {
		panic(err)
    	}

	status := (bcD.Event)
	if status == "exZone" {
		status = "Beacon Exited"
	} else {
		status = "Beacon Entered"
	}

	sparkMessage(url, sparkToken, email, status)
}

func configDefine() (*ini.File, error) {
	val, ok := os.LookupEnv("CAF_APP_CONFIG_FILE")
	if ok {
		cfgFile := val
		cfg, err := ini.Load(cfgFile)
		return cfg, err
	} else {
		cfgFile := "package_config.ini"
		cfg, err := ini.Load(cfgFile)
		return cfg, err
	}
}

func main() {
	//MQTT.DEBUG = log.New(os.Stdout, "", 0)
	//MQTT.ERROR = log.New(os.Stdout, "", 0)

	time.Sleep(10000 * time.Millisecond)

	cfg, err := configDefine()

	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	hostname, _ := os.Hostname()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	mq := cfg.Section("DataMQTTBroker").Key("ip_or_url").String()
	port := cfg.Section("DataMQTTBroker").Key("port").String()

	server := fmt.Sprintf("tcp://%s:%s", mq, port)
	topic := cfg.Section("DataMQTTBroker").Key("topic").String()
	qos := 0
	clientid := hostname + strconv.Itoa(time.Now().Second())

	username := cfg.Section("DataMQTTBroker").Key("username").String()
	password := cfg.Section("DataMQTTBroker").Key("password").String()

	connOpts := MQTT.NewClientOptions().AddBroker(server).SetClientID(clientid).SetCleanSession(true)
	if *username != "" {
	      connOpts.SetUsername(*username)
	      if *password != "" {
	              connOpts.SetPassword(*password)
	      }
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	connOpts.SetTLSConfig(tlsConfig)

	connOpts.OnConnect = func(c MQTT.Client) {
		if token := c.Subscribe(topic, byte(qos), onMessageReceived); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}
	}

	client := MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		for {
			fmt.Println("Not working")
		}
		panic(token.Error())
	} else {
		fmt.Printf("Connected to %s\n", server)
	}
	<-c
}

