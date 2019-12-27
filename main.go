package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/marzzzello/moodleAPI"
	"github.com/pborman/getopt/v2"
)

// Settings : That is how the settings file is formatted (in JSON)
type Settings struct {
	BaseURL  string `json:"baseURL"`
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

// Token : Response on requesting a new token with username and password
type Token struct {
	Token        string `json:"token"`
	PrivateToken string `json:"privatetoken"`
}

func main() {
	var (
		defaultSettings = "~/.config/moodleDownloader/settings.json"
	)

	settingsPath := getopt.StringLong("settings", 's', defaultSettings, "Path to settings file")
	coursesList := getopt.StringLong("courses", 'c', "all", "List of courses to download")
	// vp := getopt.Counter('v', "Increase verbosity for some operations")

	optHelp := getopt.BoolLong("help", 'h', "Help")
	getopt.Parse()

	if *optHelp {
		getopt.Usage()
		os.Exit(0)
	}
	
	log.Println("settingsPath", *settingsPath)
	log.Println("coursesList", *coursesList)

	api, err := login("settings_rub_test.json")
	logErr(err)
	userID, err := getUserID(api)
	logErr(err)
	log.Println("UserID:", userID)

}

func logErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func login(settingsPath string) (*moodleAPI.MoodleApi, error) {
	log.Println("Trying to login")
	// read file as a byte array.
	byteValue, _ := ioutil.ReadFile(settingsPath)

	var settings Settings
	// unmarshal byteArray which contains the settingsFile's content into 'settings' which is defined above
	json.Unmarshal(byteValue, &settings)

	if settings.Token != "" {
		log.Printf("Checking if the token \"%s\" is still valid\n", settings.Token)
		api := moodleAPI.NewMoodleApi(settings.BaseURL, settings.Token)
		_, err := getUserID(api)
		if err == nil {
			log.Println("Token is valid")
			return api, nil
		}
	}

	log.Println("Token is invalid or not set")
	newToken, err := renewToken(settingsPath)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}
	settings.Token = newToken

	api := moodleAPI.NewMoodleApi(settings.BaseURL, settings.Token)
	return api, nil
}

func getUserID(api *moodleAPI.MoodleApi) (int64, error) {
	_, _, _, userID, err := api.GetSiteInfo()
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func renewToken(settingsPath string) (string, error) {
	log.Println("Renewing Token")

	// read file as a byte array.
	byteValue, _ := ioutil.ReadFile(settingsPath)

	var settings Settings
	// unmarshal byteArray which contains the settingsFile's content into 'settings' which is defined above
	json.Unmarshal(byteValue, &settings)

	// use client with timeout of 10s
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	apiURL := settings.BaseURL + "login/token.php"

	data := url.Values{}
	data.Set("username", settings.Username)
	data.Set("password", settings.Password)
	data.Set("service", "moodle_mobile_app")

	// build request
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		log.Fatalln(err)
		return "", err
	}
	req.Header.Add("User-Agent", "Moodle Downloader/0.0.1")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// send request with custom httpclient
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
		return "", err
	}

	// read and save body
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
		return "", err
	}
	// log.Println("Body", string(body))

	// save body into Token struct
	var token Token
	err = json.Unmarshal(body, &token)
	if err != nil {
		log.Fatalln(err)
		return "", err
	}

	log.Println("Token:", token.Token)
	settings.Token = token.Token

	// if token is valid, then save it in settings file and return it as string
	log.Printf("Checking if the token \"%s\" is valid\n", settings.Token)
	api := moodleAPI.NewMoodleApi(settings.BaseURL, settings.Token)
	_, err = getUserID(api)
	if err == nil {
		log.Println("Token is valid")
		// save back in settings file
		file, _ := json.MarshalIndent(settings, "", " ")
		err = ioutil.WriteFile(settingsPath, file, 0600)
		logErr(err)
		return settings.Token, nil
	}
	return "", err
}
