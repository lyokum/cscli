package main

import (
	"fmt"
	"github.com/lyokum/update"
	"sync"
)

type ClassList struct {
	Map  map[int]Class // maps CRNs to classes
	List []Class
}

/* ClassList Receivers */
func (classes *ClassList) Init() {
	classes.List = make([]Class, 0, 20)
	classes.Map = make(map[int]Class)
}

func (classlist *ClassList) Generate(classes []Class) {
	classlist.Init()
	classlist.List = classes

	// fill map
	for _, class := range classes {
		classlist.Map[class.CRN] = class
	}
}

func (target *ClassList) Update(source ClassList) (err error) {
	// update map (error if key not found)
	for key, class := range source.Map {
		if _, ok := target.Map[key]; !ok {
			return ErrNoClass
		}

		target.Map[key] = class
	}

	// update list
	for i, t_class := range target.List {
		if s_class, ok := source.Map[t_class.CRN]; ok {
			target.List[i] = s_class
		}
	}

	return nil
}

func (classes *ClassList) Add(class Class) {
	classes.List = append(classes.List, class)
	classes.Map[class.CRN] = class
}

func (classes ClassList) Filter(info FilterInfo) (filteredList ClassList) {
	filteredList.Init()

	// filter elements
	for _, class := range classes.Map {
		if class.Filter(info) {
			filteredList.Add(class)
		}
	}

	return filteredList
}

func (classes ClassList) Notify(info NotifInfo) {
	var notif update.Update
	wg := &sync.WaitGroup{}
	notif.Subject = "CLASSES"

	// fill in class info
	for _, class := range classes.List {
		available := "X"
		if class.Open > 0 {
			available = "O"

			// send class open notification
			wg.Add(1)
			go func(class Class) {
				class.Notify(info, wg)
				wg.Done()
			}(class)
		}

		fmt.Printf("%s: %s %s\n", available, class.Title, class.Instructor)
		notif.Body += fmt.Sprintf("%s: %s %s %s\n", available, class.Section, class.Title, class.Instructor)
	}

	if info.SendUpdate {
		wg.Add(1)
		go func(notif update.Update) {
			notif.SendRequest(info.Server)
			wg.Done()
		}(notif)
	}

	wg.Wait()
}
