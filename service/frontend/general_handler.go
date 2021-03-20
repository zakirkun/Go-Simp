package main

import (
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"

	config "github.com/JustHumanz/Go-Simp/pkg/config"
	database "github.com/JustHumanz/Go-Simp/pkg/database"
	engine "github.com/JustHumanz/Go-Simp/pkg/engine"
	network "github.com/JustHumanz/Go-Simp/pkg/network"
	"github.com/bwmarrin/discordgo"
	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
)

//Fanart discord message handler
func Fanart(s *discordgo.Session, m *discordgo.MessageCreate) {
	m.Content = strings.ToLower(m.Content)
	Prefix := configfile.BotPrefix.Fanart
	var (
		Member      bool
		Group       bool
		Pic         = config.NotFound
		Msg         string
		embed       *discordgo.MessageEmbed
		DynamicData DynamicSvr
	)

	if strings.HasPrefix(m.Content, Prefix) {
		SendNude := func(Data *database.DataFanart) bool {
			Color, err := engine.GetColor(config.TmpDir, m.Author.AvatarURL("128"))
			if err != nil {
				log.Error(err)
			}

			if Data.State == "TBiliBili" {
				body, errcurl := network.CoolerCurl("https://api.vc.bilibili.com/dynamic_svr/v1/dynamic_svr/get_dynamic_detail?dynamic_id="+Data.Dynamic_id, nil)
				if errcurl != nil {
					log.Error(errcurl)
				}
				json.Unmarshal(body, &DynamicData)
				embed = engine.NewEmbed().
					SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
					SetTitle(Data.Author).
					SetThumbnail(DynamicData.GetUserAvatar()).
					SetDescription(Data.Text).
					SetURL(Data.PermanentURL).
					SetImage(Pic).
					SetColor(Color).
					InlineAllFields().
					SetFooter(Msg, config.BiliBiliIMG).MessageEmbed
			} else {
				embed = engine.NewEmbed().
					SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
					SetTitle(Data.Author).
					SetThumbnail(engine.GetAuthorAvatar(Data.Author)).
					SetDescription(RemovePic(Data.Text)).
					SetURL(Data.PermanentURL).
					SetImage(Pic).
					SetColor(Color).
					InlineAllFields().
					SetFooter(Msg, config.TwitterIMG).MessageEmbed
			}
			msg, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
			if err != nil {
				log.Error(err, msg)
			}
			err = engine.Reacting(map[string]string{
				"ChannelID": m.ChannelID,
				"Content":   m.Content,
				"Prefix":    Prefix,
			}, s)
			if err != nil {
				log.Error(err)
			}
			return true
		}
		for _, GroupData := range Payload.VtuberData {
			if m.Content == strings.ToLower(Prefix+GroupData.GroupName) {
				FanArtData, err := database.GetFanart(GroupData.ID, 0)
				if err != nil {
					log.Error(err)
					s.ChannelMessageSend(m.ChannelID, "Opps,something goes worng,like dev life\n"+err.Error())
				}
				if FanArtData.Videos != "" {
					Msg = "Video type,check original post"
					Pic = config.NotFound
				} else if len(FanArtData.Photos) > 0 {
					Pic = FanArtData.Photos[0]
					Msg = "1/" + strconv.Itoa(len(FanArtData.Photos)) + " Photos"
				}
				for _, v := range GroupData.Members {
					if v.ID == FanArtData.Member.ID {
						FanArtData.AddMember(v)
						break
					}
				}
				Group = SendNude(FanArtData)
				break
			}
			for _, MemberData := range GroupData.Members {
				if m.Content == strings.ToLower(Prefix+MemberData.Name) || m.Content == strings.ToLower(Prefix+MemberData.JpName) {
					FanArtData, err := database.GetFanart(0, MemberData.ID)
					if err != nil {
						log.Error(err)
						s.ChannelMessageSend(m.ChannelID, "Opps,something goes worng,like dev life\n"+err.Error())
					}

					if FanArtData.Videos != "" {
						Msg = "Video type,check original post"
						Pic = config.NotFound
					} else if len(FanArtData.Photos) > 0 {
						Msg = "1/" + strconv.Itoa(len(FanArtData.Photos)) + " Photos"
						Pic = FanArtData.Photos[0]
					}
					FanArtData.AddMember(MemberData)
					Member = SendNude(FanArtData)
					break
				}
			}
		}
		if Member || Group {
			return
		}
		if !Group && !Member {
			s.ChannelMessageSend(m.ChannelID, "`"+m.Content[len(Prefix):]+"` was invalid name")
		}
	}
}

//Tags command message handler
func Tags(s *discordgo.Session, m *discordgo.MessageCreate) {
	Prefix := configfile.BotPrefix.General
	m.Content = strings.ToLower(m.Content)
	if strings.HasPrefix(m.Content, Prefix) {
		var (
			Already     []string
			Done        []string
			MemberTag   []NameStruct
			ReminderInt = 0
		)
		User := &database.UserStruct{
			DiscordID:       m.Author.ID,
			DiscordUserName: m.Author.Username,
			Channel_ID:      m.ChannelID,
			Human:           true,
			Reminder:        ReminderInt,
		}
		Color, err := engine.GetColor(config.TmpDir, m.Author.AvatarURL("128"))
		if err != nil {
			log.Error(err)
		}
		if strings.HasPrefix(m.Content, Prefix+TagMe) {
			Already = nil
			Done = nil
			UserInput := strings.Replace(m.Content, Prefix+TagMe, "", -1)
			var (
				VtuberName   string
				ReminderUser int
				re           = regexp.MustCompile(`(?m)-setreminder\s[0-9]`)
			)

			if len(re.FindAllString(UserInput, -1)) > 0 {
				tmpvar := re.FindAllString(UserInput, -1)[0]
				tmpvar2, err := strconv.Atoi(strings.TrimSpace(tmpvar[len(tmpvar)-2:]))
				if err != nil {
					s.ChannelMessageSend(m.ChannelID, "Invalid number, "+err.Error())
					return
				} else {
					if tmpvar2 < 10 && tmpvar2 != 0 {
						s.ChannelMessageSend(m.ChannelID, "10 was minimum number")
						return
					} else if tmpvar2 > 65 {
						s.ChannelMessageSend(m.ChannelID, "Can't set Reminder over than 65 Minutes")
						return
					} else if tmpvar2 == 0 {
						ReminderUser = 0
						s.ChannelMessageSend(m.ChannelID, "You disable reminder time")
					} else {
						if tmpvar2%5 != 0 && tmpvar2%10 != 0 {
							s.ChannelMessageSend(m.ChannelID, "I do not recommend this number, set number with modulus 5 or 10")
						}
						ReminderUser = tmpvar2
					}
				}
				VtuberName = strings.TrimSpace(strings.Replace(UserInput, tmpvar, "", -1))
			} else {
				VtuberName = strings.TrimSpace(strings.Replace(UserInput, "-setreminder", "", -1))
			}
			if VtuberName != "" {
				tmp := strings.Split(VtuberName, ",")
				for _, Name := range tmp {
					Data := FindName(Name)
					if Data.IsNull() {
						VTuberGroup, err := FindGropName(Name)
						if err != nil {
							s.ChannelMessageSend(m.ChannelID, "`"+Name+"` was invalid,use `"+VtuberData+"` command to see vtuber groups and names.\nFor more information visit:\n "+config.VtubersData)
							return
						}
						if database.CheckChannelEnable(m.ChannelID, Name, VTuberGroup.ID) {
							User.SetGroup(VTuberGroup).
								SetReminder(ReminderUser)

							for _, Member := range VTuberGroup.Members {
								err := User.SetMember(Member).Adduser()
								if err != nil {
									Already = append(Already, "`"+Member.Name+"`")
								} else {
									Done = append(Done, "`"+Member.Name+"`")
								}
							}
							if Already != nil {
								_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
									SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
									SetDescription(m.Author.ID+" Already Added\n"+strings.Join(Already, " ")).
									AddField("Group Name", "**"+VTuberGroup.GroupName+"**").
									SetThumbnail(config.GoSimpIMG).
									SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
									InlineAllFields().
									SetColor(Color).MessageEmbed)
								if err != nil {
									log.Error(err)
								}
							}
							if Done != nil {
								_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
									SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
									SetDescription(m.Author.ID+" notifications have been added to these members\n"+strings.Join(Done, " ")).
									AddField("Group Name", "**"+VTuberGroup.GroupName+"**").
									SetThumbnail(config.GoSimpIMG).
									SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
									InlineAllFields().
									SetColor(Color).MessageEmbed)
								if err != nil {
									log.Error(err)
								}
							}
						} else {
							_, err := s.ChannelMessageSend(m.ChannelID, "look like this channel not enable `"+VTuberGroup.GroupName+"`")
							if err != nil {
								log.Error(err)
							}
							return
						}
					} else {
						MemberTag = append(MemberTag, Data)
					}
					Already = nil
					Done = nil
				}
				for i, Member := range MemberTag {
					if database.CheckChannelEnable(m.ChannelID, tmp[i], Member.Group.ID) {
						User.SetGroup(Member.Group).
							SetReminder(ReminderUser)

						err := User.SetMember(Member.Member).Adduser()
						if err != nil {
							Already = append(Already, "`"+tmp[i]+"`")
						} else {
							Done = append(Done, "`"+tmp[i]+"`")
						}
					} else {
						_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
							SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
							SetDescription("look like this channel not enable `"+Member.Group.GroupName+"`").
							SetThumbnail(config.GoSimpIMG).
							SetColor(Color).MessageEmbed)
						if err != nil {
							log.Error(err)
						}
						return
					}
				}
				if Already != nil {
					_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
						SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
						SetDescription("You Already Added\n"+strings.Join(Already, " ")+" from your list").
						SetThumbnail(config.GoSimpIMG).
						SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
						SetColor(Color).MessageEmbed)
					if err != nil {
						log.Error(err)
					}

				}
				if Done != nil {
					_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
						SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
						SetDescription("You Add\n"+strings.Join(Done, " ")+" to your list").
						SetThumbnail(config.GoSimpIMG).
						SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
						SetColor(Color).MessageEmbed)
					if err != nil {
						log.Error(err)
					}
				}
			} else {
				_, err := s.ChannelMessageSend(m.ChannelID, "Incomplete `"+TagMe+"` command")
				if err != nil {
					log.Error(err)
				}
			}
		} else if strings.HasPrefix(m.Content, Prefix+SetReminder) {
			var (
				ReminderUser int
				UserInput    = strings.Replace(m.Content, Prefix+SetReminder, "", -1)
				FindInt      = strings.Split(UserInput, " ")
			)

			if UserInput != "" {
				if len(FindInt) > 2 {
					tmpvar := FindInt[2]
					tmpvar2, err := strconv.Atoi(tmpvar)
					if err != nil {
						s.ChannelMessageSend(m.ChannelID, "Invalid number, "+err.Error())
						return
					} else {
						if tmpvar2 < 10 && tmpvar2 != 0 {
							s.ChannelMessageSend(m.ChannelID, "10 was minimum number")
							return
						} else if tmpvar2 > 65 {
							s.ChannelMessageSend(m.ChannelID, "Can't set Reminder over than 65 Minutes")
							return
						} else {
							if tmpvar2%5 != 0 && tmpvar2%10 != 0 {
								s.ChannelMessageSend(m.ChannelID, "I do not recommend this number, set number with modulus 5 or 10")
							}
							ReminderUser = tmpvar2
						}
					}
				} else {
					_, err := s.ChannelMessageSend(m.ChannelID, "Invalid `"+SetReminder+"` command")
					if err != nil {
						log.Error(err)
					}
					return
				}

				tmp := strings.Split(FindInt[1], ",")
				for _, Name := range tmp {
					Data := FindName(Name)
					if Data.IsNull() {
						VTuberGroup, err := FindGropName(Name)
						if err != nil {
							_, err := s.ChannelMessageSend(m.ChannelID, "`"+Name+"` was invalid,use `"+VtuberData+"` command to see vtubers name or see at web site \n "+config.VtubersData)
							if err != nil {
								log.Error(err)
							}
							return
						}
						if database.CheckChannelEnable(m.ChannelID, Name, VTuberGroup.ID) {
							User.SetGroup(VTuberGroup).
								SetReminder(ReminderUser)
							for _, Member := range VTuberGroup.Members {
								User.SetMember(Member)
								err = User.UpdateReminder()
								if err != nil {
									log.Error(err)
									Already = append(Already, "`"+Member.Name+"`")
								} else {
									Done = append(Done, "`"+Member.Name+"`")
								}
							}
							if Done != nil {
								if ReminderUser == 0 {
									_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
										SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
										SetDescription("You Disable reminder time\n"+strings.Join(Done, " ")).
										SetThumbnail(config.GoSimpIMG).
										SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
										SetColor(Color).MessageEmbed)
									if err != nil {
										log.Error(err)
									}
								} else {
									_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
										SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
										SetDescription("You Update reminder time\n"+strings.Join(Done, " ")).
										SetThumbnail(config.GoSimpIMG).
										SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
										SetColor(Color).MessageEmbed)
									if err != nil {
										log.Error(err)
									}
								}
								Done = nil
							} else if Already != nil {
								_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
									SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
									SetDescription("You not tag \n"+strings.Join(Already, " ")).
									SetThumbnail(config.GoSimpIMG).
									SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
									SetColor(Color).MessageEmbed)
								if err != nil {
									log.Error(err)
								}
								Already = nil
							}
						} else {
							_, err := s.ChannelMessageSend(m.ChannelID, "look like this channel not enable `"+VTuberGroup.GroupName+"`")
							if err != nil {
								log.Error(err)
							}
							return
						}
					} else {
						MemberTag = append(MemberTag, Data)
					}
				}
				for i, Member := range MemberTag {
					if database.CheckChannelEnable(m.ChannelID, tmp[i], Member.Group.ID) {
						User.SetGroup(Member.Group).
							SetReminder(ReminderUser).
							SetMember(Member.Member)
						err := User.UpdateReminder()
						if err != nil {
							log.Error(err)
							Already = append(Already, "`"+tmp[i]+"`")
						} else {
							Done = append(Done, "`"+tmp[i]+"`")
						}

					} else {
						_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
							SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
							SetDescription("look like this channel not enable `"+Member.Group.GroupName+"`").
							SetThumbnail(config.GoSimpIMG).
							SetColor(Color).MessageEmbed)
						if err != nil {
							log.Error(err)
						}
						return
					}
				}
				if Done != nil {
					if ReminderUser == 0 {
						_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
							SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
							SetDescription("You Disable reminder time\n"+strings.Join(Done, " ")).
							SetThumbnail(config.GoSimpIMG).
							SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
							SetColor(Color).MessageEmbed)
						if err != nil {
							log.Error(err)
						}
					} else {
						_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
							SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
							SetDescription("You Update reminder time\n"+strings.Join(Done, " ")).
							SetThumbnail(config.GoSimpIMG).
							SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
							SetColor(Color).MessageEmbed)
						if err != nil {
							log.Error(err)
						}
					}
				} else if Already != nil {
					_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
						SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
						SetDescription("You not tag \n"+strings.Join(Already, " ")).
						SetThumbnail(config.GoSimpIMG).
						SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
						SetColor(Color).MessageEmbed)
					if err != nil {
						log.Error(err)
					}

				}
			} else {
				_, err := s.ChannelMessageSend(m.ChannelID, "Incomplete `"+SetReminder+"` command")
				if err != nil {
					log.Error(err)
				}
				return
			}
		} else if strings.HasPrefix(m.Content, Prefix+DelTag) {
			Already = nil
			Done = nil
			VtuberName := strings.TrimSpace(strings.Replace(m.Content, Prefix+DelTag, "", -1))
			if VtuberName != "" {
				tmp := strings.Split(VtuberName, ",")
				for _, Name := range tmp {
					Data := FindName(Name)
					if Data.IsNull() {
						VTuberGroup, err := FindGropName(Name)
						if err != nil {
							_, err := s.ChannelMessageSend(m.ChannelID, "`"+Name+"` was invalid,use `"+VtuberData+"` command to see vtubers name or see at web site \n "+config.VtubersData)
							if err != nil {
								log.Error(err)
							}
							return
						}
						if database.CheckChannelEnable(m.ChannelID, Name, VTuberGroup.ID) {
							User.SetGroup(VTuberGroup)
							for _, Member := range VTuberGroup.Members {
								err := User.SetMember(Member).Deluser()
								if err != nil {
									Already = append(Already, "`"+Member.Name+"`")
								} else {
									Done = append(Done, "`"+Member.Name+"`")
								}
							}
							if Already != nil {
								_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
									SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
									SetDescription("You already removed this Group/Member from your list, or you never added them.\n"+strings.Join(Already, " ")).
									AddField("Group Name", "**"+VTuberGroup.GroupName+"**").
									SetThumbnail(config.GoSimpIMG).
									SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
									InlineAllFields().
									SetColor(Color).MessageEmbed)
								if err != nil {
									log.Error(err)
								}
							}
							if Done != nil {
								_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
									SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
									SetDescription("You removed these Members from your list. "+strings.Join(Done, " ")).
									AddField("Group Name", "**"+VTuberGroup.GroupName+"**").
									SetThumbnail(config.GoSimpIMG).
									SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
									InlineAllFields().
									SetColor(Color).MessageEmbed)
								if err != nil {
									log.Error(err)
								}
							}
						} else {
							_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
								SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
								SetDescription("look like this channel not enable `"+VTuberGroup.GroupName+"`").
								SetImage(VTuberGroup.IconURL).
								SetThumbnail(config.GoSimpIMG).
								SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
								SetColor(Color).MessageEmbed)
							if err != nil {
								log.Error(err)
							}
							return
						}

					} else {
						MemberTag = append(MemberTag, Data)
					}
				}
				Already = nil
				Done = nil
				for i, Member := range MemberTag {
					if database.CheckChannelEnable(m.ChannelID, tmp[i], Member.Group.ID) {
						err := User.SetGroup(Member.Group).SetMember(Member.Member).Deluser()
						if err != nil {
							Already = append(Already, "`"+tmp[i]+"`")
						} else {
							Done = append(Done, "`"+tmp[i]+"`")
						}
					} else {
						_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
							SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
							SetDescription("look like this channel not enable `"+Member.Group.GroupName+"`").
							SetThumbnail(config.GoSimpIMG).
							SetColor(Color).MessageEmbed)
						if err != nil {
							log.Error(err)
						}
						return
					}
				}

				if Already != nil {
					_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
						SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
						SetDescription("Already Removed from your tags or You never tag them\n"+strings.Join(Already, " ")).
						SetThumbnail(config.GoSimpIMG).
						SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
						SetColor(Color).MessageEmbed)
					if err != nil {
						log.Error(err)
					}
				}

				if Done != nil {
					//return
					_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
						SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
						SetDescription("You remove "+strings.Join(Done, " ")+" from your tag list").
						SetThumbnail(config.GoSimpIMG).
						SetFooter("Use \""+Prefix+MyTags+"\" to show you tags list").
						SetColor(Color).MessageEmbed)
					if err != nil {
						log.Error(err)
					}
				}
			} else {
				_, err := s.ChannelMessageSend(m.ChannelID, "Incomplete `"+DelTag+"` command")
				if err != nil {
					log.Error(err)
				}
			}
		} else if strings.HasPrefix(m.Content, Prefix+TagRoles) {
			Admin, err := MemberHasPermission(m.GuildID, m.Author.ID)
			if err != nil {
				log.Error(err)
			}
			if Admin {
				Already = nil
				Done = nil
				UserInput := strings.Replace(m.Content, Prefix+TagRoles, "", -1)
				ReminderUser := 0
				re := regexp.MustCompile(`(?m)-setreminder\s[1-9]`)

				if len(re.FindAllString(UserInput, -1)) > 0 {
					tmpvar := re.FindAllString(UserInput, -1)[0]
					tmpvar2, err := strconv.Atoi(strings.TrimSpace(tmpvar[len(tmpvar)-2:]))
					if err != nil {
						s.ChannelMessageSend(m.ChannelID, "Invalid number, "+err.Error())
						return
					} else {
						if tmpvar2 < 10 && tmpvar2 != 0 {
							s.ChannelMessageSend(m.ChannelID, "10 was minimum number")
							return
						} else if tmpvar2 > 65 {
							s.ChannelMessageSend(m.ChannelID, "Can't set Reminder over than 65 Minutes")
							return
						} else {
							if tmpvar2%5 != 0 && tmpvar2%10 != 0 {
								s.ChannelMessageSend(m.ChannelID, "I do not recommend this number, set number with modulus 5 or 10")
							}
							ReminderUser = tmpvar2
						}
					}

					UserInput = strings.TrimSpace(strings.Replace(UserInput, tmpvar, "", -1))

				} else {
					UserInput = strings.TrimSpace(strings.Replace(UserInput, "-setreminder", "", -1))
				}

				VtuberName := strings.Split(strings.TrimSpace(UserInput), " ")
				guild, err := s.Guild(m.GuildID)
				if err != nil {
					log.Error(err)
				}

				if len(VtuberName[len(VtuberName)-1:]) > 0 {
					tmp := strings.Split(VtuberName[len(VtuberName)-1:][0], ",")
					for _, Name := range tmp {
						Data := FindName(Name)
						if Data.IsNull() {
							VTuberGroup, err := FindGropName(Name)
							if err != nil {
								log.Error(err)
								_, err := s.ChannelMessageSend(m.ChannelID, "`"+Name+"` was invalid,use `"+VtuberData+"` command to see vtubers name or see at web site \n "+config.VtubersData)
								if err != nil {
									log.Error(err)
								}
								return
							}

							if database.CheckChannelEnable(m.ChannelID, Name, VTuberGroup.ID) {
								for _, Role := range guild.Roles {
									for _, UserRole := range VtuberName {
										if UserRole == Role.Mention() {
											for _, Member := range VTuberGroup.Members {
												User := database.UserStruct{
													DiscordID:       Role.ID,
													DiscordUserName: Role.Name,
													Channel_ID:      m.ChannelID,
													Group:           VTuberGroup,
													Human:           false,
													Reminder:        ReminderUser,
												}
												err := User.SetMember(Member).Adduser()
												if err != nil {
													Already = append(Already, "`"+Member.Name+"`")
												} else {
													Done = append(Done, "`"+Member.Name+"`")

												}
											}
											if Already != nil {
												_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
													SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
													SetDescription(Role.Mention()+" Already Added\n"+strings.Join(Already, " ")).
													AddField("Group Name", "**"+VTuberGroup.GroupName+"**").
													SetThumbnail(config.GoSimpIMG).
													SetFooter("Use \""+Prefix+RolesTags+" @"+Role.Name+"\" to role tags list").
													InlineAllFields().
													SetColor(Color).MessageEmbed)
												if err != nil {
													log.Error(err)
												}
												Already = nil
											}
											if Done != nil {
												_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
													SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
													SetDescription(Role.Mention()+" notifications have been added to these members \n"+strings.Join(Done, " ")).
													AddField("Group Name", "**"+VTuberGroup.GroupName+"**").
													SetThumbnail(config.GoSimpIMG).
													SetFooter("Use \""+Prefix+RolesTags+" @"+Role.Name+"\" to show role tags list").
													InlineAllFields().
													SetColor(Color).MessageEmbed)
												if err != nil {
													log.Error(err)
												}
												Done = nil
											}
										}
									}
								}
							} else {
								_, err := s.ChannelMessageSend(m.ChannelID, "look like this channel not enable `"+VTuberGroup.GroupName+"`")
								if err != nil {
									log.Error(err)
								}
								return
							}
						} else {
							MemberTag = append(MemberTag, Data)
						}
						Already = nil
						Done = nil
					}
					for i, Member := range MemberTag {
						if database.CheckChannelEnable(m.ChannelID, tmp[i], Member.Group.ID) {
							for _, Role := range guild.Roles {
								for _, UserRole := range VtuberName {
									if UserRole == Role.Mention() {
										User := database.UserStruct{
											DiscordID:       Role.ID,
											DiscordUserName: Role.Name,
											Channel_ID:      m.ChannelID,
											Group:           Member.Group,
											Human:           false,
											Reminder:        ReminderUser,
										}
										err := User.SetMember(Member.Member).Adduser()
										if err != nil {
											Already = append(Already, "`"+tmp[i]+"`")
										} else {
											Done = append(Done, "`"+tmp[i]+"`")
										}

										if Already != nil {
											_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
												SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
												SetDescription(Role.Mention()+" Already Added\n"+strings.Join(Already, " ")).
												SetThumbnail(config.GoSimpIMG).
												SetFooter("Use \""+Prefix+RolesTags+" @"+Role.Name+"\" to show role tags list").
												SetColor(Color).MessageEmbed)
											if err != nil {
												log.Error(err)
											}
											Already = nil
										}
										if Done != nil {
											_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
												SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
												SetDescription(Role.Mention()+" Add\n"+strings.Join(Done, " ")).
												SetThumbnail(config.GoSimpIMG).
												SetFooter("Use \""+Prefix+RolesTags+" @"+Role.Name+"\" to show you tags list").
												SetColor(Color).MessageEmbed)
											if err != nil {
												log.Error(err)
											}
											Done = nil
										}
									}
								}
							}

						} else {
							_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
								SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
								SetDescription("look like this channel not enable `"+Member.Group.GroupName+"`").
								SetThumbnail(config.GoSimpIMG).
								SetColor(Color).MessageEmbed)
							if err != nil {
								log.Error(err)
							}
							return
						}
					}
				} else {
					_, err := s.ChannelMessageSend(m.ChannelID, "Incomplete `"+TagRoles+"` command")
					if err != nil {
						log.Error(err)
					}
				}
			} else {
				_, err := s.ChannelMessageSend(m.ChannelID, "Your roles don't have permission to enable/disable/update,make sure your roles have `Manage Channels` permission")
				if err != nil {
					log.Error(err)
				}
			}
		} else if strings.HasPrefix(m.Content, Prefix+DelRoles) {
			Admin, err := MemberHasPermission(m.GuildID, m.Author.ID)
			if err != nil {
				log.Error(err)
			}
			if Admin {
				Already = nil
				Done = nil
				VtuberName := strings.Split(strings.TrimSpace(strings.Replace(m.Content, Prefix+TagRoles, "", -1)), " ")

				guild, err := s.Guild(m.GuildID)
				if err != nil {
					log.Error(err)
				}

				if len(VtuberName[len(VtuberName)-1:]) > 0 {
					tmp := strings.Split(VtuberName[len(VtuberName)-1:][0], ",")

					for _, Name := range tmp {
						Data := FindName(Name)
						if Data.IsNull() {
							VTuberGroup, err := FindGropName(Name)
							if err != nil {
								log.Error(err)
								_, err := s.ChannelMessageSend(m.ChannelID, "`"+Name+"` was invalid,use `"+VtuberData+"` command to see vtubers name or see at web site \n "+config.VtubersData)
								if err != nil {
									log.Error(err)
								}
								return
							}

							if database.CheckChannelEnable(m.ChannelID, Name, VTuberGroup.ID) {
								for _, Role := range guild.Roles {
									for _, UserRole := range VtuberName {
										if UserRole == Role.Mention() {
											for _, Member := range VTuberGroup.Members {
												User := database.UserStruct{
													DiscordID:       Role.ID,
													DiscordUserName: Role.Name,
													Channel_ID:      m.ChannelID,
													Group:           VTuberGroup,
													Human:           false,
												}
												err := User.SetMember(Member).Deluser()
												if err != nil {
													Already = append(Already, "`"+Member.Name+"`")
												} else {
													Done = append(Done, "`"+Member.Name+"`")
												}
											}

											if Already != nil {
												_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
													SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
													SetDescription(Role.Mention()+" Already Remove "+strings.Join(Already, " ")+" from tags list or "+Role.Mention()+" never add them \n").
													AddField("Group Name", "**"+VTuberGroup.GroupName+"**").
													SetThumbnail(config.GoSimpIMG).
													SetFooter("Use \""+Prefix+RolesTags+" @"+Role.Name+"\" to role tags list").
													InlineAllFields().
													SetColor(Color).MessageEmbed)
												if err != nil {
													log.Error(err)
												}
												Already = nil
											} else if Done != nil {
												_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
													SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
													SetDescription(Role.Mention()+"notifications have been removed for these members\n"+strings.Join(Done, " ")).
													AddField("Group Name", "**"+VTuberGroup.GroupName+"**").
													SetThumbnail(config.GoSimpIMG).
													SetFooter("Use \""+Prefix+RolesTags+" @"+Role.Name+"\" to show role tags list").
													InlineAllFields().
													SetColor(Color).MessageEmbed)
												if err != nil {
													log.Error(err)
												}
												Done = nil
											}
										}
									}
								}
							} else {
								_, err := s.ChannelMessageSend(m.ChannelID, "look like this channel not enable `"+VTuberGroup.GroupName+"`")
								if err != nil {
									log.Error(err)
								}
								return
							}
						} else {
							MemberTag = append(MemberTag, Data)
						}
					}
					Already = nil
					Done = nil
					for i, Member := range MemberTag {
						if database.CheckChannelEnable(m.ChannelID, tmp[i], Member.Group.ID) {
							for _, Role := range guild.Roles {
								for _, UserRole := range VtuberName {
									if UserRole == Role.Mention() {
										User := database.UserStruct{
											DiscordID:       Role.ID,
											DiscordUserName: Role.Name,
											Channel_ID:      m.ChannelID,
											Group:           Member.Group,
											Human:           false,
										}
										err := User.SetMember(Member.Member).Deluser()
										if err != nil {
											Already = append(Already, "`"+tmp[i]+"`")
										} else {
											Done = append(Done, "`"+tmp[i]+"`")
										}

										if Already != nil {
											_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
												SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
												SetDescription(Role.Mention()+" Already Remove "+strings.Join(Already, " ")+" from tags list or "+Role.Mention()+" never add them \n").
												SetThumbnail(config.GoSimpIMG).
												SetFooter("Use \""+Prefix+RolesTags+" @"+Role.Name+"\" to show role tags list").
												SetColor(Color).MessageEmbed)
											if err != nil {
												log.Error(err)
											}
											Already = nil
										}
										if Done != nil {
											_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
												SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
												SetDescription(Role.Mention()+" Remove\n"+strings.Join(Done, " ")+"\n from tag list").
												SetThumbnail(config.GoSimpIMG).
												SetFooter("Use \""+Prefix+RolesTags+" @"+Role.Name+"\" to show you tags list").
												SetColor(Color).MessageEmbed)
											if err != nil {
												log.Error(err)
											}
											Done = nil
										}
									}
								}
							}
						} else {
							_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
								SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
								SetDescription("look like this channel not enable `"+Member.Group.GroupName+"`").
								SetThumbnail(config.GoSimpIMG).
								SetColor(Color).MessageEmbed)
							if err != nil {
								log.Error(err)
							}
							return
						}
					}
				} else {
					_, err := s.ChannelMessageSend(m.ChannelID, "Incomplete `"+DelRoles+"` command")
					if err != nil {
						log.Error(err)
					}
				}
			} else {
				_, err := s.ChannelMessageSend(m.ChannelID, "Your roles don't have permission to enable/disable/update,make sure your roles have `Manage Channels` permission")
				if err != nil {
					log.Error(err)
				}
			}
		} else if strings.HasPrefix(m.Content, Prefix+RolesReminder) {
			Admin, err := MemberHasPermission(m.GuildID, m.Author.ID)
			if err != nil {
				log.Error(err)
			}
			if Admin {
				Already = nil
				Done = nil
				UserInput := strings.Replace(m.Content, Prefix+RolesReminder, "", -1)
				VtuberName := strings.Split(strings.TrimSpace(UserInput), " ")
				tmpvar := VtuberName[len(VtuberName)-1]
				ReminderUser := 0
				tmpvar2, err := strconv.Atoi(strings.TrimSpace(tmpvar))
				if err != nil {
					s.ChannelMessageSend(m.ChannelID, "Invalid number, "+err.Error()+"\nExample: `"+Prefix+RolesReminder+"` @somerole Hololive 30")
					return
				} else {
					if tmpvar2 < 10 && tmpvar2 != 0 {
						s.ChannelMessageSend(m.ChannelID, "10 was minimum number")
						return
					} else if tmpvar2 > 65 {
						s.ChannelMessageSend(m.ChannelID, "Can't set Reminder over than 65 Minutes")
						return
					} else {
						if tmpvar2%5 != 0 && tmpvar2%10 != 0 {
							s.ChannelMessageSend(m.ChannelID, "I do not recommend this number, set number with modulus 5 or 10")
						}
						ReminderUser = tmpvar2
					}
				}

				guild, err := s.Guild(m.GuildID)
				if err != nil {
					log.Error(err)
				}

				if len(VtuberName[len(VtuberName)-2:]) > 0 {
					tmp := strings.Split(VtuberName[len(VtuberName)-2:][0], ",")
					for _, Name := range tmp {
						Data := FindName(Name)
						if Data.IsNull() {
							VTuberGroup, err := FindGropName(Name)
							if err != nil {
								log.Error(err)
								_, err := s.ChannelMessageSend(m.ChannelID, "`"+Name+"` was invalid,use `"+VtuberData+"` command to see vtubers name or see at web site \n "+config.VtubersData)
								if err != nil {
									log.Error(err)
								}
								return
							}

							if database.CheckChannelEnable(m.ChannelID, Name, VTuberGroup.ID) {
								for _, Role := range guild.Roles {
									for _, UserRole := range VtuberName {
										if UserRole == Role.Mention() {
											for _, Member := range VTuberGroup.Members {
												User := database.UserStruct{
													DiscordID:       Role.ID,
													DiscordUserName: Role.Name,
													Channel_ID:      m.ChannelID,
													Group:           VTuberGroup,
													Human:           false,
													Reminder:        ReminderUser,
												}
												err := User.SetMember(Member).UpdateReminder()
												if err != nil {
													log.Error(err)
													_, err := s.ChannelMessageSend(m.ChannelID, Role.Mention()+" hasn't been added to `"+VTuberGroup.GroupName+"`. If you want to add a reminder to `"+VTuberGroup.GroupName+"`, first add "+Role.Mention()+" to `"+VTuberGroup.GroupName+"` with: `"+Prefix+RolesReminder+"`[roles] [Group/Member].")
													if err != nil {
														log.Error(err)
													}
													return
												} else {
													Done = append(Done, "`"+Member.Name+"`")
												}
											}
											if Done != nil {
												if ReminderUser == 0 {
													_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
														SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
														SetDescription(Role.Mention()+" Disable reminder time\n"+strings.Join(Done, " ")).
														AddField("Group Name", "**"+VTuberGroup.GroupName+"**").
														SetThumbnail(config.GoSimpIMG).
														SetFooter("Use \""+Prefix+RolesTags+" @"+Role.Name+"\" to show role tags list").
														InlineAllFields().
														SetColor(Color).MessageEmbed)
													if err != nil {
														log.Error(err)
													}
												} else {
													_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
														SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
														SetDescription(Role.Mention()+" Change reminder to "+strconv.Itoa(ReminderUser)+"\n"+strings.Join(Done, " ")).
														AddField("Group Name", "**"+VTuberGroup.GroupName+"**").
														SetThumbnail(config.GoSimpIMG).
														SetFooter("Use \""+Prefix+RolesTags+" @"+Role.Name+"\" to show role tags list").
														InlineAllFields().
														SetColor(Color).MessageEmbed)
													if err != nil {
														log.Error(err)
													}

												}
												Done = nil
											}
										}
									}
								}
							} else {
								_, err := s.ChannelMessageSend(m.ChannelID, "look like this channel not enable `"+VTuberGroup.GroupName+"`")
								if err != nil {
									log.Error(err)
								}
								return
							}
						} else {
							MemberTag = append(MemberTag, Data)
						}
						Done = nil
					}
					for i, Member := range MemberTag {
						if database.CheckChannelEnable(m.ChannelID, tmp[i], Member.Group.ID) {
							for _, Role := range guild.Roles {
								for _, UserRole := range VtuberName {
									if UserRole == Role.Mention() {
										User := database.UserStruct{
											DiscordID:       Role.ID,
											DiscordUserName: Role.Name,
											Channel_ID:      m.ChannelID,
											Group:           Member.Group,
											Human:           false,
											Reminder:        ReminderUser,
										}
										err := User.SetMember(Member.Member).UpdateReminder()
										if err != nil {
											log.Error(err)
											_, err := s.ChannelMessageSend(m.ChannelID, Role.Mention()+" not tag `"+Member.Member.Name+"`")
											if err != nil {
												log.Error(err)
											}
											return
										} else {
											Done = append(Done, "`"+tmp[i]+"`")
										}
									}

									if Done != nil {
										_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
											SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
											SetDescription(Role.Mention()+"Change reminder to "+strconv.Itoa(ReminderUser)+"\n"+strings.Join(Done, " ")).
											SetThumbnail(config.GoSimpIMG).
											SetFooter("Use \""+Prefix+RolesTags+" @"+Role.Name+"\" to show you tags list").
											SetColor(Color).MessageEmbed)
										if err != nil {
											log.Error(err)
										}
										Done = nil
									}
								}
							}

						} else {
							_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
								SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
								SetDescription("look like this channel not enable `"+Member.Group.GroupName+"`").
								SetThumbnail(config.GoSimpIMG).
								SetColor(Color).MessageEmbed)
							if err != nil {
								log.Error(err)
							}
							return
						}
					}
				} else {
					_, err := s.ChannelMessageSend(m.ChannelID, "Incomplete `"+RolesReminder+"` command")
					if err != nil {
						log.Error(err)
					}
				}
			} else {
				_, err := s.ChannelMessageSend(m.ChannelID, "Your roles don't have permission to enable/disable/update,make sure your roles have `Manage Channels` permission")
				if err != nil {
					log.Error(err)
				}
			}
		}
	}
}

//EnableState Enable command message handler
func EnableState(s *discordgo.Session, m *discordgo.MessageCreate) {
	m.Content = strings.ToLower(m.Content)
	Prefix := configfile.BotPrefix.General
	if strings.HasPrefix(m.Content, Prefix) {

		var (
			ChannelState = database.DiscordChannel{
				TypeTag:     0,
				LiveOnly:    false,
				NewUpcoming: false,
				Dynamic:     false,
				ChannelID:   m.ChannelID,
			}
			CommandArray = strings.Split(m.Content, " ")
			already      []string
			done         []string
		)
		/*
			Color, err := engine.GetColor(config.TmpDir, m.Author.AvatarURL("128"))
			if err != nil {
				log.Error(err)
			}
				msg1    = engine.NewEmbed().
					SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
					SetDescription("Remember boys,always respect the author,**do not save the fanart without permission from the author**").
					SetThumbnail(config.GoSimpIMG).
					SetColor(Color)

				msg2 = engine.NewEmbed().
					SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
					SetDescription("Every livestream have some **rule**,follow the **rule** and don't be asshole").
					SetThumbnail(config.GoSimpIMG).
					SetColor(Color)

				msg3 = engine.NewEmbed().
					SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
					SetDescription("Remember boys,always respect the author,**do not save the fanart without permission from the author**\nEvery livestream have some **rule**,follow the **rule** and don't be asshole").
					SetThumbnail(config.GoSimpIMG).
					SetColor(Color)

				msg4 = engine.NewEmbed().
					SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
					SetTitle("Donate").
					SetDescription("Buy dev a coffie to improve bot performance and make dev happy or if you a broke gang you can upvote this bot").
					SetURL(configfile.DonationLink).
					SetThumbnail(config.GoSimpIMG).
					SetColor(Color)
				if CommandArray[0] == Prefix+Enable {
					if len(CommandArray) > 2 {
						if CommandArray[1] == "art" {
							ChannelState.SetTypeTag(1)
						} else if CommandArray[1] == "live" {
							ChannelState.SetTypeTag(2)
						} else if CommandArray[1] == "all" {
							ChannelState.SetTypeTag(3)
						} else {
							_, err := s.ChannelMessageSend(m.ChannelID, "`"+CommandArray[1]+"`,Invalid type")
							if err != nil {
								log.Error(err)
							}
							return
						}

						if (ChannelState.TypeTag == 2 || ChannelState.TypeTag == 3) && len(CommandArray) >= 4 {
							if CommandArray[3] == "-liveonly" {
								ChannelState.SetLiveOnly(true)
							} else if CommandArray[3] == "-newupcoming" {
								ChannelState.SetNewUpcoming(true)
							} else if CommandArray[3] == "-dynamic" {
								ChannelState.SetDynamic(true)
							} else {
								_, err := s.ChannelMessageSend(m.ChannelID, "`"+CommandArray[3]+"`,Invalid options")
								if err != nil {
									log.Error(err)
								}
								return
							}
						} else if ChannelState.TypeTag == 1 && len(CommandArray) >= 4 {
							_, err := s.ChannelMessageSend(m.ChannelID, "You enabled `Art` state,Ignoring `"+CommandArray[3]+"` options")
							if err != nil {
								log.Error(err)
							}
							ChannelState.SetLiveOnly(false)
							ChannelState.SetNewUpcoming(false)
							ChannelState.SetDynamic(false)
						}

						if (ChannelState.TypeTag == 2 || ChannelState.TypeTag == 3) && len(CommandArray) >= 5 {
							if CommandArray[4] == "-newupcoming" {
								ChannelState.SetNewUpcoming(true)
							} else if CommandArray[4] == "-liveonly" {
								ChannelState.SetLiveOnly(true)
							} else if CommandArray[4] == "-dynamic" {
								ChannelState.SetDynamic(true)
							} else {
								_, err := s.ChannelMessageSend(m.ChannelID, "`"+CommandArray[4]+"`,Invalid options")
								if err != nil {
									log.Error(err)
								}
								return
							}
						}

						FindGroupArry := strings.Split(strings.TrimSpace(CommandArray[2]), ",")
						for i := 0; i < len(FindGroupArry); i++ {
							VTuberGroup, err := FindGropName(FindGroupArry[i])
							if err != nil {
								_, err := s.ChannelMessageSend(m.ChannelID, "`"+FindGroupArry[i]+"`,Name of Vtuber Group was invalid")
								if err != nil {
									log.Error(err)
								}
								return
							}
							Admin, err := MemberHasPermission(m.GuildID, m.Author.ID)
							if err != nil {
								log.Error(err)
							}
							if Admin {
								ChannelState.SetVtuberGroupID(VTuberGroup.ID)
								if ChannelState.ChannelCheck() {
									already = append(already, "`"+VTuberGroup.GroupName+"`")
								} else {
									err := ChannelState.AddChannel()
									if err != nil {
										if err.Error() == "force to set Dynamic" {
											_, err := s.ChannelMessageSend(m.ChannelID, "You set `dynamic` mode,channel has been forcing to set `live only`")
											if err != nil {
												log.Error(err)
											}
										} else {
											log.Error(err)
											_, err := s.ChannelMessageSend(m.ChannelID, "Something error XD "+err.Error())
											if err != nil {
												log.Error(err)
											}
											return
										}
									}
									done = append(done, "`"+VTuberGroup.GroupName+"`")

								}
							} else {
								_, err := s.ChannelMessageSend(m.ChannelID, "Your roles don't have permission to enable/disable/update,make sure your roles have `Manage Channels` permission")
								if err != nil {
									log.Error(err)
								}
								return
							}
						}
						if done != nil {
							_, err := s.ChannelMessageSend(m.ChannelID, "done, <@"+m.Author.ID+"> is enable "+strings.Join(done, ",")+" on this channel")
							if err != nil {
								log.Error(err)
							}
							msgID := ""
							if ChannelState.TypeTag == 1 {
								tmp, err := s.ChannelMessageSendEmbed(m.ChannelID, msg1.MessageEmbed)
								if err != nil {
									log.Error(err)
								}
								msgID = tmp.ID
							} else if ChannelState.TypeTag == 2 {
								tmp, err := s.ChannelMessageSendEmbed(m.ChannelID, msg2.MessageEmbed)
								if err != nil {
									log.Error(err)
								}
								msgID = tmp.ID
							} else {
								tmp, err := s.ChannelMessageSendEmbed(m.ChannelID, msg3.MessageEmbed)
								if err != nil {
									log.Error(err)
								}
								msgID = tmp.ID
							}
							MessagePinned, err := s.ChannelMessagesPinned(m.ChannelID)
							if err != nil {
								log.Error(err)
							}
							for _, Message := range MessagePinned {
								if Message.Author.ID == BotInfo.ID {
									err := s.ChannelMessageUnpin(m.ChannelID, Message.ID)
									if err != nil {
										log.Error(err)
									}
								}
							}
							err = s.ChannelMessagePin(m.ChannelID, msgID)
							if err != nil {
								log.Error(err)
							}
							if rand.Float32() < 0.5 && configfile.DonationLink != "" {
								_, err := s.ChannelMessageSendEmbed(m.ChannelID, msg4.MessageEmbed)
								if err != nil {
									log.Error(err)
								}
							}
						} else {
							_, err := s.ChannelMessageSend(m.ChannelID, strings.Join(already, ",")+", already enabled")
							if err != nil {
								log.Error(err)
							}
						}
					} else {
						_, err := s.ChannelMessageSend(m.ChannelID, "Incomplete `"+Enable+"` command")
						if err != nil {
							log.Error(err)
						}
					}
				} else
		*/
		if CommandArray[0] == Prefix+Disable {
			if len(CommandArray) > 1 {
				FindGroupArry := strings.Split(strings.TrimSpace(CommandArray[1]), ",")

				for _, cmd := range FindGroupArry {
					VTuberGroup, err := FindGropName(cmd)
					if err != nil {
						_, err := s.ChannelMessageSend(m.ChannelID, "`"+cmd+"`,Name of Vtuber Group was not valid")
						if err != nil {
							log.Error(err)
						}
						return
					}
					Admin, err := MemberHasPermission(m.GuildID, m.Author.ID)
					if err != nil {
						log.Error(err)
					}
					if Admin {
						ChannelState.SetVtuberGroupID(VTuberGroup.ID)
						if ChannelState.ChannelCheck() {
							err := ChannelState.DelChannel("Delete")
							if err != nil {
								log.Error(err)
								_, err := s.ChannelMessageSend(m.ChannelID, "Something error XD")
								if err != nil {
									log.Error(err)
								}
								return
							}
							done = append(done, "`"+VTuberGroup.GroupName+"`")
						} else {
							already = append(already, "`"+VTuberGroup.GroupName+"`")
						}
					} else {
						_, err := s.ChannelMessageSend(m.ChannelID, "Your roles don't have permission to enable/disable/update,make sure your roles have `Manage Channels` permission")
						if err != nil {
							log.Error(err)
						}
						return
					}
				}

				if done != nil {
					_, err := s.ChannelMessageSend(m.ChannelID, "done, <@"+m.Author.ID+"> is disabled "+strings.Join(done, ",")+" from this channel")
					if err != nil {
						log.Error(err)
					}
				} else {
					_, err := s.ChannelMessageSend(m.ChannelID, strings.Join(already, ",")+", already removed or never enable on this channel")
					if err != nil {
						log.Error(err)
					}
				}
			} else {
				_, err := s.ChannelMessageSend(m.ChannelID, "Incomplete `"+Disable+"` command")
				if err != nil {
					log.Error(err)
				}
			}
		}
		/*
			CommandArray[0] == Prefix+Update {
				if len(CommandArray) > 2 {
					if CommandArray[1] == "art" {
						ChannelState.SetTypeTag(1)
					} else if CommandArray[1] == "live" {
						ChannelState.SetTypeTag(2)
					} else if CommandArray[1] == "all" {
						ChannelState.SetTypeTag(3)
					} else {
						_, err := s.ChannelMessageSend(m.ChannelID, "`"+CommandArray[1]+"`,Invalid type")
						if err != nil {
							log.Error(err)
						}
						return
					}

					FindGroupArry := strings.Split(strings.TrimSpace(CommandArray[2]), ",")
					for i := 0; i < len(FindGroupArry); i++ {
						VTuberGroup, err := FindGropName(FindGroupArry[i])
						if err != nil {
							_, err := s.ChannelMessageSend(m.ChannelID, "`"+FindGroupArry[i]+"`,Name of Vtuber Group was invalid")
							if err != nil {
								log.Error(err)
							}
							return
						}
						Admin, err := MemberHasPermission(m.GuildID, m.Author.ID)
						if err != nil {
							log.Error(err)
						}
						if Admin {
							ChannelState.SetVtuberGroupID(VTuberGroup.ID)
							if ChannelState.ChannelCheck() {
								if (ChannelState.TypeTag == 2 || ChannelState.TypeTag == 3) && len(CommandArray) >= 4 {
									if CommandArray[3] == "-liveonly" {
										err := ChannelState.SetLiveOnly(true).UpdateChannel("LiveOnly")
										if err != nil {
											log.Error(err)
											already = append(already, "`"+VTuberGroup.GroupName+"`")
										} else {
											done = append(done, "`"+VTuberGroup.GroupName+"`")
										}
									} else if CommandArray[3] == "-newupcoming" {
										err := ChannelState.SetNewUpcoming(true).UpdateChannel("NewUpcoming")
										if err != nil {
											log.Error(err)
											already = append(already, "`"+VTuberGroup.GroupName+"`")
										} else {
											done = append(done, "`"+VTuberGroup.GroupName+"`")
										}
									} else if CommandArray[3] == "-dynamic" {
										err := ChannelState.SetDynamic(true).UpdateChannel("Dynamic")
										if err != nil {
											log.Error(err)
											already = append(already, "`"+VTuberGroup.GroupName+"`")
										} else {
											done = append(done, "`"+VTuberGroup.GroupName+"`")
										}
									} else if CommandArray[3] == "-rm_liveonly" {
										err := ChannelState.SetLiveOnly(false).UpdateChannel("LiveOnly")
										if err != nil {
											log.Error(err)
											already = append(already, "`"+VTuberGroup.GroupName+"`")
										} else {
											done = append(done, "`"+VTuberGroup.GroupName+"`")
										}
									} else if CommandArray[3] == "-rm_newupcoming" {
										err := ChannelState.SetNewUpcoming(false).UpdateChannel("NewUpcoming")
										if err != nil {
											log.Error(err)
											already = append(already, "`"+VTuberGroup.GroupName+"`")
										} else {
											done = append(done, "`"+VTuberGroup.GroupName+"`")
										}
									} else if CommandArray[3] == "-rm_dynamic" {
										err := ChannelState.SetDynamic(false).UpdateChannel("Dynamic")
										if err != nil {
											log.Error(err)
											already = append(already, "`"+VTuberGroup.GroupName+"`")
										} else {
											done = append(done, "`"+VTuberGroup.GroupName+"`")
										}
									} else {
										_, err := s.ChannelMessageSend(m.ChannelID, "`"+CommandArray[3]+"`,Invalid options")
										if err != nil {
											log.Error(err)
										}
										return
									}
								}

								if (ChannelState.TypeTag == 2 || ChannelState.TypeTag == 3) && len(CommandArray) >= 5 {
									if CommandArray[4] == "-newupcoming" {
										err := ChannelState.SetNewUpcoming(true).UpdateChannel("NewUpcoming")
										if err != nil {
											log.Error(err)
											already = append(already, "`"+VTuberGroup.GroupName+"`")
										} else {
											done = append(done, "`"+VTuberGroup.GroupName+"`")
										}
									} else if CommandArray[4] == "-liveonly" {
										err := ChannelState.SetLiveOnly(true).UpdateChannel("LiveOnly")
										if err != nil {
											log.Error(err)
											already = append(already, "`"+VTuberGroup.GroupName+"`")
										} else {
											done = append(done, "`"+VTuberGroup.GroupName+"`")
										}
									} else if CommandArray[4] == "-dynamic" {
										err := ChannelState.SetDynamic(true).UpdateChannel("Dynamic")
										if err != nil {
											log.Error(err)
											already = append(already, "`"+VTuberGroup.GroupName+"`")
										} else {
											done = append(done, "`"+VTuberGroup.GroupName+"`")
										}
									} else if CommandArray[4] == "-rm_liveonly" {
										err := ChannelState.SetLiveOnly(false).UpdateChannel("LiveOnly")
										if err != nil {
											log.Error(err)
											already = append(already, "`"+VTuberGroup.GroupName+"`")
										} else {
											done = append(done, "`"+VTuberGroup.GroupName+"`")
										}
									} else if CommandArray[4] == "-rm_newupcoming" {
										err := ChannelState.SetNewUpcoming(false).UpdateChannel("NewUpcoming")
										if err != nil {
											log.Error(err)
											already = append(already, "`"+VTuberGroup.GroupName+"`")
										} else {
											done = append(done, "`"+VTuberGroup.GroupName+"`")
										}
									} else if CommandArray[3] == "-rm_dynamic" {
										err := ChannelState.SetDynamic(false).UpdateChannel("Dynamic")
										if err != nil {
											log.Error(err)
											already = append(already, "`"+VTuberGroup.GroupName+"`")
										} else {
											done = append(done, "`"+VTuberGroup.GroupName+"`")
										}
									} else {
										_, err := s.ChannelMessageSend(m.ChannelID, "`"+CommandArray[4]+"`,Invalid options")
										if err != nil {
											log.Error(err)
										}
										return
									}
								}

								err := ChannelState.UpdateChannel("Type")
								if err != nil {
									already = append(already, "`"+VTuberGroup.GroupName+"`")
								} else {
									done = append(done, "`"+VTuberGroup.GroupName+"`")
								}
							} else {
								_, err := s.ChannelMessageSend(m.ChannelID, "this channel not enable `"+VTuberGroup.GroupName+"`")
								if err != nil {
									log.Error(err)
								}
								return
							}
						} else {
							_, err := s.ChannelMessageSend(m.ChannelID, "Your roles don't have permission to enable/disable/update,make sure your roles have `Manage Channels` permission")
							if err != nil {
								log.Error(err)
							}
							return
						}
					}
					if done != nil {
						_, err := s.ChannelMessageSend(m.ChannelID, "done,<@"+m.Author.ID+"> update channel state "+strings.Join(done, ","))
						if err != nil {
							log.Error(err)
						}

						msgID := ""
						if ChannelState.TypeTag == 1 {
							tmp, err := s.ChannelMessageSendEmbed(m.ChannelID, msg1.MessageEmbed)
							if err != nil {
								log.Error(err)
							}
							msgID = tmp.ID
						} else if ChannelState.TypeTag == 2 {
							tmp, err := s.ChannelMessageSendEmbed(m.ChannelID, msg2.MessageEmbed)
							if err != nil {
								log.Error(err)
							}
							msgID = tmp.ID
						} else {
							tmp, err := s.ChannelMessageSendEmbed(m.ChannelID, msg3.MessageEmbed)
							if err != nil {
								log.Error(err)
							}
							msgID = tmp.ID
						}
						MessagePinned, err := s.ChannelMessagesPinned(m.ChannelID)
						if err != nil {
							log.Error(err)
						}
						for _, Message := range MessagePinned {
							if Message.Author.ID == BotInfo.ID {
								err := s.ChannelMessageUnpin(m.ChannelID, Message.ID)
								if err != nil {
									log.Error(err)
								}
							}
						}
						err = s.ChannelMessagePin(m.ChannelID, msgID)
						if err != nil {
							log.Error(err)
						}
						if rand.Float32() < 0.5 && configfile.DonationLink != "" {
							_, err := s.ChannelMessageSendEmbed(m.ChannelID, msg4.MessageEmbed)
							if err != nil {
								log.Error(err)
							}
						}

					} else {
						_, err := s.ChannelMessageSend(m.ChannelID, strings.Join(already, ",")+" Same state")
						if err != nil {
							log.Error(err)
						}
					}

				} else {
					_, err := s.ChannelMessageSend(m.ChannelID, "Incomplete `"+Update+"` command")
					if err != nil {
						log.Error(err)
					}
				}
			}
		*/
	}
}

//Help helmp command message handler
func Help(s *discordgo.Session, m *discordgo.MessageCreate) {
	m.Content = strings.ToLower(m.Content)
	Prefix := configfile.BotPrefix.General
	if strings.HasPrefix(m.Content, Prefix) {
		Color, err := engine.GetColor(config.TmpDir, m.Author.AvatarURL("128"))
		if err != nil {
			log.Error(err)
		}
		if m.Content == Prefix+"help en" || m.Content == Prefix+"help" {
			_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
				SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
				SetTitle("Help").
				SetURL(config.CommandURL).
				SetDescription("A simple VTuber bot which pings you or your roles if any new Videos, Fanarts, or Livestreams and Upcoming streams of VTubers are posted!").
				AddField("Command list", "[Exec]("+config.CommandURL+")").
				AddField("Guide", "[Guide]("+config.GuideURL+")").
				AddField("Vtuber list", "[Vtubers]("+config.VtubersData+")").
				AddField("Made by Golang", "[Go-Simp](https://github.com/JustHumanz/Go-Simp)").
				AddField("Server count", strconv.Itoa(len(GuildList))).
				AddField("Member count", strconv.Itoa(database.GetMemberCount())).
				InlineAllFields().
				SetThumbnail(config.BSD).
				SetFooter(os.Getenv("VERSION")).
				SetColor(Color).MessageEmbed)
			if err != nil {
				log.Error(err)
			}
			return
		} else if m.Content == Prefix+"help jp" { //i'm just joking lol
			_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
				SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
				SetTitle("Help").
				SetDescription("日本語が話せるようになってヘルプメニューを作りたい\n~Dev").
				SetImage("https://i.imgur.com/f0no1r7.png").
				SetFooter("More like,help me").
				SetColor(Color).MessageEmbed)
			if err != nil {
				log.Error(err)
			}
			return
		} else if m.Content == Prefix+Kings {
			s.ChannelMessageSend(m.ChannelID, "https://github.com/JustHumanz/Go-Simp/blob/master/King.md")
		} else if m.Content == Prefix+Upvote {
			s.ChannelMessageSend(m.ChannelID, config.GoSimpConf.TopGG)
		}
	}
}

//Status command message handler
func Status(s *discordgo.Session, m *discordgo.MessageCreate) {
	m.Content = strings.ToLower(m.Content)
	Prefix := configfile.BotPrefix.General

	if strings.HasPrefix(m.Content, Prefix) {
		Color, err := engine.GetColor(config.TmpDir, m.Author.AvatarURL("128"))
		if err != nil {
			log.Error(err)
		}

		tableString := &strings.Builder{}
		table := tablewriter.NewWriter(tableString)
		table.SetAutoWrapText(false)
		table.SetAutoFormatHeaders(true)
		table.SetCenterSeparator("")
		table.SetColumnSeparator("")
		table.SetRowSeparator("")
		table.SetHeaderLine(true)
		table.SetBorder(false)
		table.SetTablePadding("\t")
		table.SetNoWhiteSpace(true)
		if strings.HasPrefix(m.Content, Prefix+RolesTags) {
			guild, err := s.Guild(m.GuildID)
			if err != nil {
				log.Error(err)
			}
			RolesInput := strings.Split(strings.TrimSpace(strings.Replace(m.Content, Prefix+RolesTags, "", -1)), " ")
			if len(RolesInput) > 0 {
				for _, UserRoles := range RolesInput {
					for _, Role := range guild.Roles {
						if UserRoles == Role.Mention() {
							list := database.UserStatus(Role.ID, m.ChannelID)
							if list != nil {
								tableString := &strings.Builder{}
								table := tablewriter.NewWriter(tableString)
								table.SetHeader([]string{"Vtuber Group", "Vtuber Name", "Reminder"})
								table.SetAutoWrapText(false)
								table.SetAutoFormatHeaders(true)
								table.SetCenterSeparator("")
								table.SetColumnSeparator("")
								table.SetRowSeparator("")
								table.SetHeaderLine(true)
								table.SetBorder(false)
								table.SetTablePadding("\t")
								table.SetNoWhiteSpace(true)
								table.AppendBulk(list)
								table.Render()

								_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
									SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
									SetThumbnail(m.Author.AvatarURL("128")).
									SetDescription("Role "+Role.Mention()+"\n```"+tableString.String()+"```").
									SetColor(Color).MessageEmbed)
								if err != nil {
									log.Error(err)
								}

							} else {
								_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
									SetTitle("404 Not found").
									SetImage(config.NotFound).
									SetColor(Color).MessageEmbed)
								if err != nil {
									log.Error(err)
								}
							}
						}
					}
				}
			} else {
				_, err := s.ChannelMessageSend(m.ChannelID, "Incomplete `"+RolesTags+"` command")
				if err != nil {
					log.Error(err)
				}
			}

		} else if m.Content == Prefix+MyTags {
			list := database.UserStatus(m.Author.ID, m.ChannelID)

			if list != nil {
				table.SetHeader([]string{"Vtuber Group", "Vtuber Name", "Reminder"})
				table.AppendBulk(list)
				table.Render()

				_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
					SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
					SetThumbnail(m.Author.AvatarURL("128")).
					SetDescription("```"+tableString.String()+"```").
					SetColor(Color).MessageEmbed)
				if err != nil {
					log.Error(err)
				}
			} else {
				_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
					SetDescription("Your tag list is empty.").
					SetTitle("404 Not found").
					SetImage(config.NotFound).
					SetColor(Color).MessageEmbed)
				if err != nil {
					log.Error(err)
				}
			}
		} else if m.Content == Prefix+ChannelState {
			var (
				Typestr     string
				LiveOnly    = config.No
				NewUpcoming = config.No
				Dynamic     = config.No
				LiteMode    = config.No
				Indie       = ""
				Region      = "All"
			)
			ChannelData := database.ChannelStatus(m.ChannelID)
			if len(ChannelData) > 0 {
				for _, Channel := range ChannelData {
					ChannelRaw, err := s.Channel(m.ChannelID)
					if err != nil {
						log.Error(err)
					}

					if Channel.Region != "" {
						Region = Channel.Region
					}
					if Channel.IndieNotif && Channel.Group.GroupName == config.Indie {
						Indie = config.Ok
					} else if Channel.Group.GroupName != config.Indie {
						Indie = "-"
					} else {
						Indie = config.No
					}

					if Channel.IsFanart() && !Channel.IsLewd() && !Channel.IsLive() {
						Typestr = "Art"
					} else if !Channel.IsFanart() && !Channel.IsLewd() && Channel.IsLive() {
						Typestr = "Live"
					} else if Channel.IsFanart() && !Channel.IsLewd() && Channel.IsLive() {
						Typestr = "Fanart & Livestream"
					} else if Channel.IsLewd() && !Channel.IsFanart() {
						Typestr = "Lewd"
					} else if Channel.IsLewd() && Channel.IsFanart() {
						Typestr = "Fanart & Lewd"
					}

					if Channel.LiveOnly {
						LiveOnly = config.Ok
					}

					if Channel.NewUpcoming {
						NewUpcoming = config.Ok
					}

					if Channel.Dynamic {
						Dynamic = config.Ok
					}

					if Channel.LiteMode {
						LiteMode = config.Ok
					}

					if Channel.IsFanart() && !Channel.IsLewd() && !Channel.IsLive() {
						if Channel.Group.GroupName == config.Indie {
							_, err = s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
								SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
								SetThumbnail(config.GoSimpIMG).
								SetDescription("Channel States of "+Channel.Group.GroupName).
								SetTitle(ChannelRaw.Name).
								AddField("Type", Typestr).
								AddField("Regions", Region).
								AddField("Independent notif", Indie).
								InlineAllFields().
								SetColor(Color).MessageEmbed)
							if err != nil {
								log.Error(err)
							}
						} else {
							_, err = s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
								SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
								SetThumbnail(config.GoSimpIMG).
								SetDescription("Channel States of "+Channel.Group.GroupName).
								SetTitle(ChannelRaw.Name).
								AddField("Type", Typestr).
								AddField("Regions", Region).
								InlineAllFields().
								SetColor(Color).MessageEmbed)
							if err != nil {
								log.Error(err)
							}
						}

					} else if Channel.IsLewd() && !Channel.IsFanart() {
						if Channel.Group.GroupName == config.Indie {
							_, err = s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
								SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
								SetThumbnail(config.GoSimpIMG).
								SetDescription("Channel States of "+Channel.Group.GroupName).
								SetTitle(ChannelRaw.Name).
								AddField("Type", Typestr).
								AddField("Regions", Region).
								AddField("Independent notif", Indie).
								InlineAllFields().
								SetColor(Color).MessageEmbed)
							if err != nil {
								log.Error(err)
							}
						} else {
							_, err = s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
								SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
								SetThumbnail(config.GoSimpIMG).
								SetDescription("Channel States of "+Channel.Group.GroupName).
								SetTitle(ChannelRaw.Name).
								AddField("Type", Typestr).
								AddField("Regions", Region).
								InlineAllFields().
								SetColor(Color).MessageEmbed)
							if err != nil {
								log.Error(err)
							}
						}

					} else {
						if Channel.Group.GroupName == config.Indie {
							_, err = s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
								SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
								SetThumbnail(config.GoSimpIMG).
								SetDescription("Channel States of "+Channel.Group.GroupName).
								SetTitle(ChannelRaw.Name).
								AddField("Type", Typestr).
								AddField("LiveOnly", LiveOnly).
								AddField("Dynamic", Dynamic).
								AddField("Upcoming", NewUpcoming).
								AddField("Lite", LiteMode).
								AddField("Regions", Region).
								AddField("Independent notif", Indie).
								InlineAllFields().
								SetColor(Color).MessageEmbed)
							if err != nil {
								log.Error(err)
							}
						} else {
							_, err = s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
								SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
								SetThumbnail(config.GoSimpIMG).
								SetDescription("Channel States of "+Channel.Group.GroupName).
								SetTitle(ChannelRaw.Name).
								AddField("Type", Typestr).
								AddField("LiveOnly", LiveOnly).
								AddField("Dynamic", Dynamic).
								AddField("Upcoming", NewUpcoming).
								AddField("Lite", LiteMode).
								AddField("Regions", Region).
								InlineAllFields().
								SetColor(Color).MessageEmbed)
							if err != nil {
								log.Error(err)
							}
						}
					}
				}
			} else {
				_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
					SetTitle("404 Not found").
					SetThumbnail(config.GoSimpIMG).
					SetImage(config.NotFound).
					SetColor(Color).MessageEmbed)
				if err != nil {
					log.Error(err)
				}
				return
			}
		} else if strings.HasPrefix(m.Content, Prefix+VtuberData) {
			var (
				re        = regexp.MustCompile(`(?m)-region\s.+`)
				tmpvar    = re.FindAllString(m.Content, -1)
				UserInput = strings.Replace(m.Content, Prefix+VtuberData, "", -1)
				RegInput  []string
			)

			if len(tmpvar) > 0 {
				vartmp2 := strings.TrimSpace(strings.Replace(tmpvar[0], "-region", "", -1))
				RegInput = strings.Split(vartmp2, ",")

				UserInput = strings.Replace(UserInput, tmpvar[0], "", -1)
			} else {
				UserInput = strings.Replace(UserInput, "-region", "", -1)
			}
			GroupInput := strings.Split(strings.TrimSpace(UserInput), ",")
			if len(GroupInput) > 0 {
				var (
					GroupsByReg = RegInput
					NiggList    = make(map[string]string)
				)
				if len(RegInput) > 0 {
					for _, Group := range GroupInput {
						var (
							black []string
						)
						for _, Reg := range GroupsByReg {
							Counter := CheckReg(Group, Reg)
							if !Counter {
								black = append(black, Reg)
							}
						}
						if black != nil {
							NiggList[Group] = strings.Join(black, ",")
						}
					}
				}
				for _, Group := range Payload.VtuberData {
					for _, Grp := range GroupInput {
						if Grp == strings.ToLower(Group.GroupName) {
							for _, Member := range Group.Members {
								yt := ""
								bl := ""
								if Member.YoutubeID != "" {
									yt = "✓"
								} else {
									yt = "✗"
								}

								if Member.BiliBiliID != 0 {
									bl = "✓"
								} else {
									bl = "✗"
								}

								if GroupsByReg != nil {
									table.SetHeader([]string{"Nickname", "Regions", "Youtube", "BiliBili", "Group"})
									for _, Reg := range GroupsByReg {
										if Reg == strings.ToLower(Member.Region) {
											table.Append([]string{Member.Name, Member.Region, yt, bl, Group.GroupName})
										}
									}
								} else {
									table.SetHeader([]string{"Nickname", "Regions", "Youtube", "BiliBili"})
									table.Append([]string{Member.Name, Member.Region, yt, bl})
								}
							}
						}
					}
				}
				table.Render()
				if len(tableString.String()) > engine.EmbedLimitDescription {
					_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
						SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
						SetThumbnail(config.GoSimpIMG).
						SetURL(config.VtubersData).
						SetDescription("Data too longgggggg\nsee Vtubers Data at web site\n"+config.VtubersData).
						SetImage(config.Longcatttt).
						SetColor(Color).MessageEmbed)
					if err != nil {
						log.Error(err)
					}
				} else if len(tableString.String()) > 1500 {
					_, err := s.ChannelMessageSend(m.ChannelID, "```"+tableString.String()+"```")
					if err != nil {
						log.Error(err)
					}
				} else if tableString.String() == "" {
					_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
						SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
						SetThumbnail(config.GoSimpIMG).
						SetTitle("List of Vtuber Groups").
						SetURL(config.VtubersData).
						SetDescription("```"+strings.Join(GroupsName, "\t")+"```For more detail see at "+config.VtubersData).
						SetColor(Color).
						SetFooter("Use Name of group to show vtuber members").MessageEmbed)
					if err != nil {
						log.Error(err)
					}

				} else if len(tableString.String()) > 42 {
					_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
						SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
						SetDescription("```"+tableString.String()+"```").
						SetColor(Color).
						SetFooter("Use \"Nickname\" as parameter").MessageEmbed)
					if err != nil {
						log.Error(err)
					}
				}

				if NiggList != nil {
					for key, val := range NiggList {
						_, err := s.ChannelMessageSend(m.ChannelID, "`"+strings.Title(key)+"` don't have member in `"+strings.ToUpper(val)+"`")
						if err != nil {
							log.Error(err)
						}
					}
				}
			} else {
				_, err := s.ChannelMessageSendEmbed(m.ChannelID, engine.NewEmbed().
					SetAuthor(m.Author.Username, m.Author.AvatarURL("128")).
					SetThumbnail(config.GoSimpIMG).
					SetURL(config.CommandURL).
					SetDescription("Invalid command,see command at my github\n"+config.CommandURL).
					SetColor(Color).MessageEmbed)
				if err != nil {
					log.Error(err)
				}
				return
			}
		}
	}
}
