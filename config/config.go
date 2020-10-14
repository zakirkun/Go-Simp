package config

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

var (
	// Public variables
	Token   string
	YtToken []string
	//helmp me to pick emoji
	EmojiFanart    []string
	EmojiStream    []string
	PFanart        string
	PYoutube       string
	PBilibili      string
	PGeneral       string
	TwitterToken   string
	ImgurClient    string
	BiliBiliSes    string
	SauceAPI       string
	Logging        string
	OwnerDiscordID string
	// Private variables
	config ConfigFile
)

const (
	NotFound    = "https://raw.githubusercontent.com/JustHumanz/Go-simp/master/Img/404.jpg"
	YoutubeIMG  = "https://raw.githubusercontent.com/JustHumanz/Go-simp/master/Img/youtube.png"
	BiliBiliIMG = "https://raw.githubusercontent.com/JustHumanz/Go-simp/master/Img/bilibili.png"
	TwitterIMG  = "https://raw.githubusercontent.com/JustHumanz/Go-simp/master/Img/twitter.png"
	WorryIMG    = "https://raw.githubusercontent.com/JustHumanz/Go-simp/master/Img/parerunworry.png"
	Go_Simp     = "https://raw.githubusercontent.com/JustHumanz/Vtube_bot/master/Img/go-simp.png"
)

type ConfigFile struct {
	Discord       string `toml:"Discord"`
	TwitterBearer string `toml:"TwitterBearer"`
	ImgurClinet   string `toml:"ImgurClinet"`
	BiliSess      string `toml:"BiliSess"`
	SauceAPI      string `toml:"SauceAPI"`
	SQL           struct {
		User string `toml:"User"`
		Pass string `toml:"Pass"`
		Host string `toml:"Host"`
	} `toml:"Sql"`
	BotPrefix struct {
		Fanart   string `toml:"Fanart"`
		Youtube  string `toml:"Youtube"`
		Bilibili string `toml:"Bilibili"`
		General  string `toml:"General"`
	} `toml:"BotPrefix"`
	Emoji struct {
		Fanart     []string `toml:"Fanart"`
		Livestream []string `toml:"Livestream"`
	} `toml:"Emoji"`
	YtToken []string `toml:"YtToken"`
}

//read from config file
func ReadConfig() (*sql.DB, error) {
	fmt.Println("Reading config file...")
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})

	file, err := ioutil.ReadFile("../config/config.toml")

	if err != nil {
		return nil, err
	}

	fmt.Println(string(file))

	_, err = toml.Decode(string(file), &config)
	if err != nil {
		return nil, err
	}
	log.Info("Open DB")

	db, err := sql.Open("mysql", ""+config.SQL.User+":"+config.SQL.Pass+"@tcp("+config.SQL.Host+":3306)/Vtuber?parseTime=true")
	if err != nil {
		log.Error(err, " Something worng with database,make sure you create Vtuber database first")
		os.Exit(1)
	}

	//make sure can access database
	_, err = db.Exec(`SELECT NOW()`)
	if err != nil {
		log.Error(err, " Something worng with database,make sure you create Vtuber database first")
		os.Exit(1)
	}

	TwitterToken = config.TwitterBearer
	ImgurClient = config.ImgurClinet
	BiliBiliSes = config.BiliSess
	//NotFound =
	SauceAPI = config.SauceAPI
	Logging = os.Getenv("LOG")
	OwnerDiscordID = os.Getenv("OWNER")

	Token = config.Discord
	YtToken = config.YtToken
	EmojiFanart = config.Emoji.Fanart

	PGeneral = config.BotPrefix.General
	PFanart = config.BotPrefix.Fanart
	PYoutube = config.BotPrefix.Youtube
	PBilibili = config.BotPrefix.Bilibili

	return db, nil
}
