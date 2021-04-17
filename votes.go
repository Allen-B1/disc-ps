package main

import (
	"fmt"
	"strings"
	"regexp"
)

type Vote string

const (
	Yes Vote = "yes"
	No Vote = "no"
	Abstain Vote = "abstain"
)

type VoteSubject struct {
	Bill string `json:"bill"`
	Amendment string `json:"amendment"`
}

type Votes struct {
	Subject VoteSubject `json:"subject"`

	Members map[string]*ParliamentMember `json:"members"`
	MemberVotes map[string]Vote `json:"member_votes"`
	GenericVotes map[string]Vote `json:"generic_votes"`
}

func (s Server) FetchVotes(voteID string) (*Votes, error) {
	membersRaw, err := s.FetchParliamentMembers()
	if err != nil {
		return nil, err
	}
	members := make(map[string]*ParliamentMember)
	for _, memberList := range membersRaw {
		for _, member := range memberList {
			members[member.ID] = member
		}
	}

	v := new(Votes)
	v.Members = members

	parts := strings.Split(voteID, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid vote ID")
	}

	msg, err := dg.ChannelMessage(parts[0], parts[1])
	if err != nil {
		return nil, err
	}

	r := regexp.MustCompile("(?i)Members\\s+may\\s+vote\\s+.*\\s+on\\s+(.*)\n")
	loc := r.FindSubmatchIndex([]byte(msg.Content))
	if loc != nil {
		fullName := strings.Title(msg.Content[loc[2]:loc[3]])
		fullName = strings.Trim(fullName, ". \t")
		if strings.HasPrefix(fullName, "Amendment") {
			i := strings.Index(fullName, "To")
			if i > 0 {
				amendmentName := strings.TrimSpace(fullName[:i])
				v.Subject.Amendment = amendmentName
				fullName = strings.TrimSpace(fullName[i+2:])
			}
		}
		v.Subject.Bill = fullName
	}

	msgs, err := dg.ChannelMessages(parts[0], 100, "", parts[1], "")
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(msgs) / 2; i++ {
		msgs[i], msgs[len(msgs)-1-i] = msgs[len(msgs)-1-i], msgs[i]
	}

	v.MemberVotes = make(map[string]Vote)
	v.GenericVotes = make(map[string]Vote)
	for _, msg := range msgs {
		if r.Match([]byte(msg.Content)) {
			break
		}

		transContent := strings.Trim(strings.TrimSpace(strings.ToLower(msg.Content)), "*_")

		if !msg.Author.Bot {
			if v.Members[msg.Author.ID] == nil {
				continue
			}
			if strings.HasPrefix(transContent, "aye") || strings.HasPrefix(transContent, "yea") {
				v.MemberVotes[msg.Author.ID] = Yes
			}
			if strings.HasPrefix(transContent, "nay") {
				v.MemberVotes[msg.Author.ID] = No
			}
			if strings.HasPrefix(transContent, "abst") {
				v.MemberVotes[msg.Author.ID] = Abstain
			}
		} else {
			if strings.HasPrefix(transContent, "aye") || strings.HasPrefix(transContent, "yea") {
				v.GenericVotes[msg.Author.Username] = Yes
			}
			if strings.HasPrefix(transContent, "nay") {
				v.GenericVotes[msg.Author.Username] = No
			}
			if strings.HasPrefix(transContent, "abst") {
				v.GenericVotes[msg.Author.Username] = Abstain
			}
		}
	}

	_ = fmt.Println

	return v, nil
}