package main

import (
	"fmt"
	"html/template"
	"os"
	"strings"

	"math"

	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
)

var dg *discordgo.Session

type Server string

func main() {
	var err error
	dg, err = discordgo.New("Bot " + os.Getenv("PS_TOKEN"))
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.SetFuncMap(template.FuncMap {
		"repeat": func (times int) []struct{} {
			return make([]struct{},times)
		},
		"dict": func (values ...interface{}) map[string]interface{} {
			m := make(map[string]interface{})
			for i := 0; i < len(values)-1; i += 2{
				m[fmt.Sprint(values[i])] = values[i+1]
			}
			return m
		},
		"transform": func (s string) string {
			return strings.NewReplacer("\t", " ", " ", "-").Replace(s)
		},
		"sqrt": func (s int) int {
			return int(math.Sqrt(float64(s)))
		},
		"mul": func (a, b int) int {
			return a*b
		},
		"min": func (a, b int) int {
			if a < b {
				return a
			}
			return b
		},
		"div": func (a, b int) int {
			return a / b
		},
	})
	r.GET("/api/:server/body/:body", func (c *gin.Context) {
		server := Server(c.Param("server"))

		count, err := server.FetchParliamentCount(c.Param("body"))
		if err != nil {
			c.String(500, "error counting: %v", err)
			return
		}

		parties, err := server.FetchParliamentParties()
		if err != nil {
			c.String(500, "error counting: %v", err)
			return
		}

		members, err := server.FetchParliamentMembers()
		if err != nil {
			c.String(500, "error finding members: %v", err)
			return
		}

		c.JSON(200, map[string]interface{} {
			"body": c.Param("body"),
			"count": count,
			"parties": parties,
			"members": members[c.Param("body")],
		})
	})

	r.GET("/api/:server/vote/:vote", func (c *gin.Context) {
		server := Server(c.Param("server"))

		votes, err := server.FetchVotes(c.Param("vote"))
		if err != nil {
			c.String(500, "%v", err)
			return
		}

		c.JSON(200, votes)
	})

	r.StaticFile("/style.css", "style.css")

	r.GET("/html/:server/vote/:vote", func (c *gin.Context) {
		server := Server(c.Param("server"))
		votes, err := server.FetchVotes(c.Param("vote"))
		if err != nil {
			c.String(500, "%v", err)
			return
		}

		fmt.Println(votes)

		r.LoadHTMLFiles("templates/vote.tmpl")

		c.HTML(200, "vote.tmpl", map[string]interface{} {
			"Votes": votes,
		})
	})

	r.GET("/html/:server/body/:body", func (c *gin.Context) {
		server := Server(c.Param("server"))
		body := c.Param("body")

		count, err := server.FetchParliamentCount(body)
		if err != nil {
			c.String(500, "error counting: %v", err)
			return
		}

		parties, err := server.FetchParliamentParties()
		if err != nil {
			c.String(500, "error counting: %v", err)
			return
		}

		members, err := server.FetchParliamentMembers()
		if err != nil {
			c.String(500, "error finding members: %v", err)
			return
		}

		membersParsed := make(map[string][]*ParliamentMember)
		for _, member := range members[body] {
			membersParsed[member.Party] = append(membersParsed[member.Party], member)
		}

		r.LoadHTMLFiles("templates/parliament.tmpl")

		c.HTML(200, "parliament.tmpl", map[string]interface{}{
			"Count": count,
			"Parties": parties,
			"Members": membersParsed,
		})
	})

	r.Run()
}