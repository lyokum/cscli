package main

import (
	"errors"
	"fmt"
	"github.com/lyokum/update"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

var (
	ErrNoClass = errors.New("Class not found")
)

type Class struct {
	Section    string
	Title      string
	Credits    string
	Max        int
	Open       int
	CRN        int
	Instructor string
	Time       string
	Location   string
}

type NotifInfo struct {
	SendUpdate bool
	SendText   bool
	Server     string
	Phone      string
	Provider   string
}

type FilterInfo struct {
	Open        bool
	CRNs        []int
	Names       []*regexp.Regexp
	Professors  []*regexp.Regexp
	Departments []*regexp.Regexp
}

/* Class Receivers */
func (class Class) GetSubject() (subject string) {
	re := regexp.MustCompile("^[A-Z]+")
	return re.FindString(class.Section)
}

func (class Class) Filter(info FilterInfo) (isValid bool) {
	isValid = true

	// check open
	if info.Open && class.Open == 0 {
		isValid = false
	}

	// check CRNs
	if isValid && len(info.CRNs) > 0 {
		isValid = false
		for _, CRN := range info.CRNs {
			if class.CRN == CRN {
				isValid = true
				break
			}
		}
	}

	// check Names
	if isValid && len(info.Names) > 0 {
		isValid = false
		for _, expr := range info.Names {
			if expr.MatchString(strings.ToLower(class.Title)) {
				isValid = true
				break
			}
		}
	}

	// check Professors
	if isValid && len(info.Professors) > 0 {
		isValid = false
		for _, expr := range info.Professors {
			if expr.MatchString(strings.ToLower(class.Instructor)) {
				isValid = true
				break
			}
		}
	}

	// check Departments
	if isValid && len(info.Departments) > 0 {
		isValid = false
		for _, expr := range info.Departments {
			if expr.MatchString(strings.ToLower(class.GetSubject())) {
				isValid = true
				break
			}
		}
	}

	return isValid
}

func (class Class) Notify(info NotifInfo, wg *sync.WaitGroup) {
	// create update
	var notif update.Update
	notif.Subject = "<CLASS OPENING>"
	notif.Body = fmt.Sprintf("%s (CRN %d) has %d open slot(s)!", class.Title, class.CRN, class.Open)

	// send update
	if info.SendUpdate {
		fmt.Println(notif.Body)

		wg.Add(1)
		go func(notif update.Update, info NotifInfo) {
			notif.SendRequest(info.Server)
			wg.Done()
		}(notif, info)
	}

	// send text
	if info.SendText {
		wg.Add(1)
		go func(class Class, notif update.Update, info NotifInfo) {
			class.sendText(notif, info)
			wg.Done()
		}(class, notif, info)
	}
}

func (class Class) sendText(update update.Update, info NotifInfo) {
	cmd := exec.Command("mail-send", "-r", info.Phone+"@"+info.Provider, update.Subject, update.Body)
	cmd.Run()
}
