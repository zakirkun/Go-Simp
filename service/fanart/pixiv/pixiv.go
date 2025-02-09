package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/JustHumanz/Go-Simp/pkg/config"
	"github.com/JustHumanz/Go-Simp/pkg/database"
	"github.com/JustHumanz/Go-Simp/pkg/engine"
	"github.com/JustHumanz/Go-Simp/pkg/network"
	pilot "github.com/JustHumanz/Go-Simp/service/pilot/grpc"
	"github.com/JustHumanz/Go-Simp/service/utility/runfunc"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

//Public variable
var (
	Bot          *discordgo.Session
	lewd         = flag.Bool("LewdFanart", false, "Enable lewd fanart module")
	torTransport = flag.Bool("Tor", false, "Enable multiTor for bot transport")
	gRCPconn     pilot.PilotServiceClient
	ServiceUUID  = uuid.New().String()
)

const (
	BaseURL     = "https://www.pixiv.net/en/artworks/"
	Limit       = 10
	ServiceName = config.PixivService
)

func init() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, DisableColors: true})
	flag.Parse()
	gRCPconn = pilot.NewPilotServiceClient(network.InitgRPC(config.Pilot))
}

//Start start pixiv module
func main() {
	var (
		configfile config.ConfigFile
	)

	res, err := gRCPconn.GetBotPayload(context.Background(), &pilot.ServiceMessage{
		Message:     "Init " + ServiceName + " service",
		ServiceUUID: ServiceUUID,
		Service:     ServiceName,
	})
	if err != nil {
		if configfile.Discord != "" {
			pilot.ReportDeadService(err.Error(), ServiceName)
		}
		log.Error("Error when request payload: %s", err)
	}
	err = json.Unmarshal(res.ConfigFile, &configfile)
	if err != nil {
		log.Fatalln(err)
	}

	configfile.InitConf()
	Bot = engine.StartBot(*torTransport)

	database.Start(configfile)

	c := cron.New()
	c.Start()

	if *lewd {
		log.Info("Enable lewd " + ServiceName)

	} else {
		log.Info("Enable " + ServiceName)

	}

	go pilot.RunHeartBeat(gRCPconn, ServiceName, ServiceUUID)
	go ReqRunningJob(gRCPconn)
	runfunc.Run(Bot)
}

func Pixiv(p string, FixFanArt *database.DataFanart, l bool) error {
	var Art engine.PixivArtworks
	req, err := http.NewRequest("GET", p, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", network.RandomAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Dnt", "1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Connection", "keep-alive")
	if l {
		req.Header.Set("Cookie", "PHPSESSID="+config.GoSimpConf.PixivSession)
	}
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Te", "Trailers")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		if l && response.StatusCode == http.StatusUnauthorized {
			pilot.ReportDeadService("Pixiv Session outdate", ServiceUUID)
		}
		log.WithFields(log.Fields{
			"Status":  response.StatusCode,
			"Reason":  response.Status,
			"Payload": p,
		}).Error("Status code not daijobu")
		return errors.New(response.Status)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &Art)
	if err != nil {
		return err
	}

	if Art.Body.Illustmanga.Data != nil {
		for i, v := range Art.Body.Illustmanga.Data {
			v2 := v.(map[string]interface{})
			IsVtuber := false
			IsNotLoli := true

			for _, tag := range v2["tags"].([]interface{}) {
				Tag := strings.ToLower(tag.(string))
				GrpName := strings.ToLower(FixFanArt.Group.GroupName)
				HashTag := FixFanArt.Member.EnName
				if FixFanArt.Member.TwitterHashtag != "" {
					HashTag = FixFanArt.Member.TwitterHashtag[1:]
				} else if FixFanArt.Member.BiliBiliHashtag != "" {
					HashTag = FixFanArt.Member.BiliBiliHashtag[1 : len(FixFanArt.Member.BiliBiliHashtag)-1]
				}

				match, _ := regexp.MatchString("("+strings.ToLower(HashTag)+"|"+GrpName+")", Tag)
				if match {
					IsVtuber = true
				}

				if l {
					for _, black := range config.BlackList {
						if strings.ToLower(black) == Tag {
							log.WithFields(log.Fields{
								"URL": BaseURL + v2["id"].(string),
							}).Info("Lol,it's loli")
							IsNotLoli = false
						}
					}
				}
			}

			var (
				Illusts map[string]interface{}
				User    map[string]interface{}
				TextFix string
			)

			if IsVtuber {
				if v2["xRestrict"].(float64) == 0 && !l {
					illusbyte, err := network.Curl(config.PixivIllustsEnd+v2["id"].(string), nil)
					if err != nil {
						return err
					}

					err = json.Unmarshal(illusbyte, &Illusts)
					if err != nil {
						return err
					}

					Body := Illusts["body"].(map[string]interface{})
					Tags := Body["tags"].(map[string]interface{})
					Img := Body["urls"].(map[string]interface{})
					FixImg := Img["regular"].(string)

					usrbyte, err := network.Curl(config.PixivUserEnd+Tags["authorId"].(string), nil)
					if err != nil {
						return err
					}

					err = json.Unmarshal(usrbyte, &User)
					if err != nil {
						return err
					}

					UserBody := User["body"].(map[string]interface{})

					Desc := RemoveHtmlTag(Body["description"].(string))
					if match, _ := regexp.MatchString("http://twitter.com", Desc); match {
						TextFix = ClearTwitterURL(Desc)
					} else {
						TextFix = "**" + Body["title"].(string) + "**\n" + Desc
					}

					FixFanArt.AddAuthor(v2["userName"].(string)).AddPermanentURL(BaseURL + v2["id"].(string)).
						AddAuthorAvatar(config.PixivProxy + UserBody["imageBig"].(string)).AddPhotos([]string{FixImg}).
						AddText(TextFix).AddPixivID(v2["id"].(string)).SetState(config.PixivArt)

					new, err := FixFanArt.CheckPixivFanArt()
					if err != nil {
						log.WithFields(log.Fields{
							"Agency": FixFanArt.Group.GroupName,
							"Vtuber": FixFanArt.Member.Name,
						}).Warn(err)
					}

					if new {
						path, err := DownloadImg(Img["mini"].(string))
						if err != nil {
							log.WithFields(log.Fields{
								"Agency": FixFanArt.Group.GroupName,
								"Vtuber": FixFanArt.Member.Name,
							}).Error(err)
						}

						FixFanArt.Photos[0] = config.PixivProxy + FixImg
						FixFanArt.FilePath = path

						if config.GoSimpConf.Metric {
							gRCPconn.MetricReport(context.Background(), &pilot.Metric{
								MetricData: FixFanArt.MarshallBin(),
								State:      config.FanartState,
							})
						}
						engine.SendFanArtNude(*FixFanArt, Bot)
					}
				} else if l && v2["xRestrict"].(float64) == 1 && IsNotLoli {
					illusbyte, err := network.Curl(config.PixivIllustsEnd+v2["id"].(string), nil)
					if err != nil {
						return err
					}

					err = json.Unmarshal(illusbyte, &Illusts)
					if err != nil {
						return err
					}

					Body := Illusts["body"].(map[string]interface{})
					Tags := Body["tags"].(map[string]interface{})
					Img := Body["urls"].(map[string]interface{})
					FixImg := Img["regular"].(string)

					usrbyte, err := network.Curl(config.PixivUserEnd+Tags["authorId"].(string), nil)
					if err != nil {
						return err
					}

					err = json.Unmarshal(usrbyte, &User)
					if err != nil {
						return err
					}

					UserBody := User["body"].(map[string]interface{})

					Desc := RemoveHtmlTag(Body["description"].(string))
					if match, _ := regexp.MatchString("http://twitter.com", Desc); match {
						TextFix = ClearTwitterURL(Desc)
					} else {
						TextFix = "**" + Body["title"].(string) + "**\n" + Desc
					}

					FixFanArt.AddAuthor(v2["userName"].(string)).AddPermanentURL(BaseURL + v2["id"].(string)).
						AddAuthorAvatar(config.PixivProxy + UserBody["imageBig"].(string)).AddPhotos([]string{FixImg}).
						AddText(TextFix).AddPixivID(v2["id"].(string)).SetState(config.PixivArt)

					new, err := FixFanArt.CheckPixivFanArt()
					if err != nil {
						log.WithFields(log.Fields{
							"Agency": FixFanArt.Group.GroupName,
							"Vtuber": FixFanArt.Member.Name,
						}).Error(err)
					}

					if new {
						path, err := DownloadImg(Img["mini"].(string))
						if err != nil {
							log.WithFields(log.Fields{
								"Agency": FixFanArt.Group.GroupName,
								"Vtuber": FixFanArt.Member.Name,
							}).Error(err)
						}

						if config.GoSimpConf.Metric {
							gRCPconn.MetricReport(context.Background(), &pilot.Metric{
								MetricData: FixFanArt.MarshallBin(),
								State:      config.FanartState,
							})
						}

						FixFanArt.Photos[0] = config.PixivProxy + FixImg
						FixFanArt.FilePath = path
						engine.SendFanArtNude(*FixFanArt, Bot)
					}
				}
				if i == Limit {
					break
				}
			}
		}
	}
	return nil
}

func GetPixivURL(str string) string {
	return "https://www.pixiv.net/ajax/search/artworks/" + str + "?word=" + str + "&order=date_d&mode=all&p=1&s_mode=s_tag&type=all&lang=en"
}

func GetPixivLewdURL(str string) string {
	return "https://www.pixiv.net/ajax/search/artworks/" + str + "?word=" + str + "&order=date_d&mode=r18&p=1&s_mode=s_tag&type=all&lang=en"
}

func ClearTwitterURL(str1 string) string {
	re := regexp.MustCompile(`(https\:\/\/twitter\.com\/.*)\<`)
	submatchall := re.FindStringSubmatch(str1)
	fix := ""
	for _, element := range submatchall {
		fix = element
	}
	return fix
}

func RemoveHtmlTag(in string) string {
	// regex to match html tag
	const pattern = `(<\/?[a-zA-A]+?[^>]*\/?>)*`
	r := regexp.MustCompile(pattern)
	groups := r.FindAllString(in, -1)
	// should replace long string first
	sort.Slice(groups, func(i, j int) bool {
		return len(groups[i]) > len(groups[j])
	})
	for _, group := range groups {
		if strings.TrimSpace(group) != "" {
			in = strings.ReplaceAll(in, group, "")
		}
	}
	return in
}

func DownloadImg(u string) (string, error) {
	dir := config.TmpDir + engine.RanString()
	out, err := os.Create(dir)
	if err != nil {
		return "", err
	}

	defer out.Close()
	request, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	request.Header.Set("cache-control", "no-cache")
	request.Header.Set("User-Agent", network.RandomAgent())
	request.Header.Set("Referer", "https://www.pixiv.net")

	spaceClient := http.Client{}
	resp, err := spaceClient.Do(request.WithContext(ctx))
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}
	return dir, nil
}

type checkPxJob struct {
	wg      sync.WaitGroup
	Agency  []database.Group
	Reverse bool
}

func ReqRunningJob(client pilot.PilotServiceClient) {
	Pix := &checkPxJob{}

	for {
		res, err := client.RequestRunJobsOfService(context.Background(), &pilot.ServiceMessage{
			Service:     ServiceName,
			Message:     "Request",
			ServiceUUID: ServiceUUID,
		})
		if err != nil {
			log.Error(err)
		}

		if res.Run {
			log.WithFields(log.Fields{
				"Running":        true,
				"UUID":           ServiceUUID,
				"Agency Payload": res.VtuberMetadata,
			}).Info(res.Message)

			Pix.Agency = engine.UnMarshalPayload(res.VtuberPayload)
			if len(Pix.Agency) == 0 {
				msg := "vtuber agency was nill,force close the unit"
				pilot.ReportDeadService(msg, ServiceName)
				log.Fatalln(msg)
			}
			Pix.Run()

			_, _ = client.RequestRunJobsOfService(context.Background(), &pilot.ServiceMessage{
				Service:     ServiceName,
				Message:     "Done",
				ServiceUUID: ServiceUUID,
			})
			log.WithFields(log.Fields{
				"Running": false,
				"UUID":    ServiceUUID,
			}).Info("reporting job was done")
		} else {
			log.WithFields(log.Fields{
				"Running": false,
				"UUID":    ServiceUUID,
			}).Info(res.Message)
		}
		time.Sleep(1 * time.Minute)
	}
}

func (k *checkPxJob) Run() {

	//make request to pixiv randomly
	if rand.Float32() < 0.5 {
		return
	}

	Cek := func(wg *sync.WaitGroup, Member database.Member, Group database.Group, l bool) {
		defer wg.Done()
		FixFanArt := &database.DataFanart{
			Member: Member,
			Group:  Group,
			Lewd:   l,
		}

		if Member.JpName != "" && Member.Region == "JP" {
			log.WithFields(log.Fields{
				"Vtuber": Member.JpName,
				"Agency": Group.GroupName,
				"Lewd":   l,
			}).Info("Start curl pixiv")
			URLJP := GetPixivURL(url.QueryEscape(Member.JpName))
			err := Pixiv(URLJP, FixFanArt, l)
			if err != nil {
				log.Error(err)
				gRCPconn.ReportError(context.Background(), &pilot.ServiceMessage{
					Message:     err.Error(),
					Service:     ServiceName,
					ServiceUUID: ServiceUUID,
				})
			}
		} else if Member.EnName != "" && Member.Region != "JP" {
			log.WithFields(log.Fields{
				"Vtuber": Member.EnName,
				"Agency": Group.GroupName,
				"Lewd":   l,
			}).Info("Start curl pixiv")
			URLEN := GetPixivURL(engine.UnderScoreName(Member.EnName))
			err := Pixiv(URLEN, FixFanArt, l)
			if err != nil {
				log.Error(err)
				gRCPconn.ReportError(context.Background(), &pilot.ServiceMessage{
					Message:     err.Error(),
					Service:     ServiceName,
					ServiceUUID: ServiceUUID,
				})
			}
		} else {
			log.WithFields(log.Fields{
				"Vtuber": Member.EnName,
				"Agency": Group.GroupName,
				"Lewd":   l,
			}).Info("Start curl pixiv")
			URLEN := GetPixivURL(engine.UnderScoreName(Member.EnName))
			err := Pixiv(URLEN, FixFanArt, l)
			if err != nil {
				log.Error(err)
				gRCPconn.ReportError(context.Background(), &pilot.ServiceMessage{
					Message:     err.Error(),
					Service:     ServiceName,
					ServiceUUID: ServiceUUID,
				})
			}
		}
	}

	if k.Reverse {
		for j := len(k.Agency) - 1; j >= 0; j-- {
			Group := k.Agency
			for _, Member := range Group[j].Members {
				k.wg.Add(1)

				go Cek(&k.wg, Member, Group[j], false)
				if *lewd {
					k.wg.Add(1)
					go Cek(&k.wg, Member, Group[j], true)
				}
				if j%4 == 0 {
					k.wg.Wait()
				}
			}
		}
		k.Reverse = false
	} else {
		for _, Group := range k.Agency {
			for i, Member := range Group.Members {
				k.wg.Add(1)

				go Cek(&k.wg, Member, Group, false)
				if *lewd {
					k.wg.Add(1)
					go Cek(&k.wg, Member, Group, true)
				}
				if i%4 == 0 {
					k.wg.Wait()
				}
			}
		}
		k.Reverse = true
	}
	k.wg.Wait()
}
