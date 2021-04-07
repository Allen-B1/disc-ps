package main

import (
	"strconv"
	"strings"
	"sync"
	"fmt"
)

var partiesCache struct {
	v map[string]bool
	sync.Mutex
}

type ParliamentPartyCount struct {
	Count int `json:"count"`
	Role string `json:"role"`
}
func (s Server) FetchParliamentCount(body string) (map[string]*ParliamentPartyCount, error) {
	guildID := string(s)

	channels, err := dg.GuildChannels(guildID)
	if err != nil {
		return nil, err
	}

	infoChannel := ""
	for _, channel := range channels {
		if channel.Name == "maps-and-info" {
			infoChannel = channel.ID
		}
	}

	msgs, err := dg.ChannelMessages(infoChannel, 1, "", "", "")
	if err != nil {
		return nil, err
	}

	parties := make(map[string]*ParliamentPartyCount)
	lines := strings.Split(msgs[0].Content, "\n")
	partyType := ""
	hasOfficialOpp := false
	for _, line := range lines {
		if strings.HasPrefix(line, "**") || strings.HasPrefix(line, "__") {
			if strings.Contains(line, "Government") {
				partyType = "gov"
			} else if strings.Contains(line, "Most Loyal Opposition") {
				partyType = "opp"
				hasOfficialOpp = true
			} else {
				partyType = "cross"
			}
		}

		parts := strings.Split(line, "-")
		if len(parts) < 2 {
			continue
		}

		name := strings.TrimSpace(strings.Join(parts[:len(parts)-1], "-"))
		count, err := strconv.Atoi(strings.TrimSpace(parts[len(parts)-1]))
		if err != nil {
			continue
		}

		role := partyType
		if name == "Speaker" {
			role = "speaker"
		}

		parties[name] = &ParliamentPartyCount{
			Count: count,
			Role: role,
		}
	}

	if !hasOfficialOpp {
		for _, party := range parties {
			if party.Role == "cross" {
				party.Role = "opp"
			}
		}
	}

	partiesCache.Lock()
	partiesCache.v = make(map[string]bool)
	for party := range parties {
		partiesCache.v[party] = true
	}
	partiesCache.Unlock()

	if body != "Parliament" {
		return nil, fmt.Errorf("body not found")
	}

	return parties, nil
}

type ParliamentPartyInfo struct {
	Color string `json:"color"`
	RoleID string `json:"role_id"`
}

func (s Server) FetchParliamentParties() (map[string]*ParliamentPartyInfo, error) {
	s._forceCacheParliament()

	guildID := string(s)
	roles, err := dg.GuildRoles(guildID)
	if err != nil {
		return nil, err
	}

	info := make(map[string]*ParliamentPartyInfo)
	for _, role := range roles {
		if partiesCache.v[role.Name] || partiesCache.v[role.Name + "s"] {
			name := role.Name
			if partiesCache.v[role.Name + "s"] {
				name = role.Name + "s"
			}

			info[name] = &ParliamentPartyInfo{
				RoleID: role.ID,
				Color: fmt.Sprintf("#%06X", role.Color),
			}
		}
	}
	return info, nil
}

type ParliamentMember struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Party string `json:"party"`
	Body string `json:"body"`
	Role string `json:"role"`
	Constituency string `json:"constituency"`
}

func (s Server) _forceCacheParliament() {
	partiesCache.Lock()
	requiresParties := partiesCache.v == nil
	partiesCache.Unlock()
	if requiresParties {
		s.FetchParliamentCount("")
	}
}

func (s Server) FetchParliamentMembers() (map[string][]*ParliamentMember, error) {
	/* Hacky code, for now */
	s._forceCacheParliament()

	guildID := string(s)
	roles, err := dg.GuildRoles(guildID)
	if err != nil {
		return nil, err
	}

	constituencyRoles := make(map[string]string)
	partyRoles := make(map[string]string)
	leaderRoles := make(map[string]string)
	memberRoles := make(map[string]string)

	partiesCache.Lock()
	for _, role := range roles {
		if strings.HasPrefix(role.Name, "Member for") {
			constituencyRoles[role.ID] = strings.TrimSpace(role.Name[len("Member for"):])
		}
		if partiesCache.v[role.Name] {
			partyRoles[role.ID] = role.Name
		}
		if partiesCache.v[role.Name + "s"] {
			partyRoles[role.ID] = role.Name + "s"
		}
		if strings.HasPrefix(role.Name, "Leader of") || strings.Contains(role.Name, "Minister") {
			leaderRoles[role.ID] = role.Name
		}
		if strings.HasPrefix(role.Name, "Member of") {
			memberRoles[role.ID] = strings.TrimSpace(role.Name[len("Member of"):])
		}
	}
	partiesCache.Unlock()

	members, err := dg.GuildMembers(guildID, "", 1000)
	if err != nil {
		return nil, err
	}

	memberList := make(map[string][]*ParliamentMember)
	for _, member := range members {
		isMember := ""
		pMember := new(ParliamentMember)
		for _, role := range member.Roles {
			if memberRoles[role] != "" {
				isMember = memberRoles[role]
				pMember.Body = isMember
			}
			if partyRoles[role] != "" {
				pMember.Party = partyRoles[role]
			}
			if leaderRoles[role] != "" {
				pMember.Role = leaderRoles[role]
			}
			if constituencyRoles[role] != "" {
				pMember.Constituency = constituencyRoles[role]
			}
		}
		if isMember == "" {
			continue
		}

		pMember.Name = member.Nick
		pMember.ID = member.User.ID
		
		memberList[isMember] = append(memberList[isMember], pMember)
	}
	return memberList, nil
}