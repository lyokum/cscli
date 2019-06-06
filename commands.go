package main

import (
	"fmt"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

/* init functions */
func setDirectory(ctx *cli.Context, dir string) (err error) {
	// check that cache is being used
	if ctx.Bool("no-cache") {
		return ErrNoCache
	}

	// set cache directories
	return Storage.SetDirectory(dir)
}

/* Helper Funcs */
func getAllClasses(ctx *cli.Context, CRNs []int) (classes ClassList, err error) {
	log.Println("Fetching all classes")

	// use cache if specified
	if ctx.Parent().Bool("no-cache") {
		// get generic page
		doc, err := ParseHTML("")
		if err != nil {
			return classes, err
		}

		// get options from page
		opts, err := GetOptions(doc)
		if err != nil {
			return classes, err
		}

		// setup form input for general search
		var input FormInput
		input.Init(opts)
		doc, err = ParseHTML(input.String())
		if err != nil {
			return classes, err
		}

		// get all classes
		classes, err = GetClasses(doc, nil)
		if err != nil {
			return classes, err
		}
	} else {
		if len(CRNs) > 0 {
			// update data based on CRNs
			err = Storage.FetchUpdates(CRNs)
			if err != nil {
				return classes, err
			}
		}

		// get all classes
		classes = Storage.Classes
	}

	log.Println("All classes fetched")
	return classes, nil
}

func slice2Regex(slice []string) (regs []*regexp.Regexp, err error) {
	regs = make([]*regexp.Regexp, 0, 10)

	for _, str := range slice {
		expr, err := regexp.Compile(strings.ToLower(str))
		if err != nil {
			return regs, err
		}
		regs = append(regs, expr)
	}

	return regs, nil
}

/* check command */
func checkCRNs(ctx *cli.Context) (err error) {
	log.Println("Checking CRNs")

	// get CRNs in string form
	CRNs := make([]int, 0, 10)
	strCRNs := make([]string, 0, 10)
	if ctx.NArg() > 0 {
		CRNs = make([]int, 0, 10)
		for _, arg := range ctx.Args() {
			strCRNs = append(strCRNs, arg)
		}
	} else if !isatty.IsTerminal(os.Stdin.Fd()) {
		spliter := regexp.MustCompile(`\d{5}`)

		// read in stdin
		CRNinput, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

		// create tokens
		strCRNs = spliter.FindAllString(string(CRNinput), -1) // -1 means find all matches
	}

	// convert CRNs to ints
	if len(strCRNs) > 0 {
		CRNs = make([]int, 0, 10)
		for _, strCRN := range strCRNs {
			CRN, err := strconv.Atoi(strCRN)
			if err != nil {
				return err
			}

			CRNs = append(CRNs, CRN)
		}
	} else {
		return ErrNoCRNs
	}

	// get full class repo
	fullList, err := getAllClasses(ctx, CRNs)
	if err != nil {
		return
	}

	// filter classes
	filterinfo := FilterInfo{CRNs: CRNs}
	classes := fullList.Filter(filterinfo)

	// send notifications
	notifinfo := NotifInfo{SendUpdate: ctx.Bool("update"), SendText: ctx.Bool("text")}

	// get server info
	if notifinfo.SendUpdate {
		// get server
		notifinfo.Server = ctx.String("server")

		// check no server specified
		if len(notifinfo.Server) == 0 {
			return ErrServerNotFound
		}

		// test connection
		/*
			if _, err := exec.Command("ping", "-W", "1", "-c", "1", notifinfo.Server).Output(); err != nil {
				return ErrServerNotFound
			}
		*/
	}

	// get phone info
	if notifinfo.SendText {
		notifinfo.Phone = ctx.String("cellphone")
		provider := ctx.String("provider")

		// check number of digits
		if len(notifinfo.Phone) != 10 {
			return ErrPhoneInvalid
		}

		// get provider
		var ok bool
		notifinfo.Provider, ok = Providers[strings.ToLower(provider)]

		if !ok {
			return ErrProviderNotFound
		}
	}

	// send notification
	classes.Notify(notifinfo)
	log.Println("CRNs checked and notified")
	return nil
}

/* search command */
func performSearch(ctx *cli.Context) (err error) {
	log.Println("Starting search")
	var info FilterInfo

	// check open
	if ctx.Bool("open") {
		info.Open = true
	}

	// check CRNs
	if len(ctx.IntSlice("id")) > 0 {
		info.CRNs = ctx.IntSlice("id")
		if err != nil {
			return err
		}
	}

	// check instructors
	if len(ctx.StringSlice("professor")) > 0 {
		info.Professors, err = slice2Regex(ctx.StringSlice("professor"))
		if err != nil {
			return err
		}
	}

	// check departments
	if len(ctx.StringSlice("department")) > 0 {
		info.Departments, err = slice2Regex(ctx.StringSlice("department"))
		if err != nil {
			return err
		}
	}

	// check names
	if ctx.NArg() > 0 {
		info.Names, err = slice2Regex(ctx.Args())
		if err != nil {
			return err
		}
	}

	// get full class repo
	classes, err := getAllClasses(ctx, info.CRNs)
	if err != nil {
		return
	}

	// filter classes
	results := classes.Filter(info)

	// update results if necessary
	if !ctx.Parent().Bool("no-cache") && len(results.Map) > 0 && ctx.Bool("update") {
		updateCRNs := make([]int, 0, 10)

		// fill CRNs
		for crn, _ := range results.Map {
			updateCRNs = append(updateCRNs, crn)
		}

		// update cache
		err = Storage.FetchUpdates(updateCRNs)
		if err != nil {
			return err
		}

		// make new results
		info.CRNs = updateCRNs
		results = Storage.Classes.Filter(info)

	}

	log.Println("Printing search results")

	// print info
	for CRN, class := range results.Map {
		if ctx.Bool("info") {
			//fmt.Printf("%-5s %3s %-30s %-30s %-35s %-20s\n", fmt.Sprintf("%d", class.CRN), fmt.Sprintf("%d", class.Open), class.Title, class.Instructor, class.Time, class.Location)
			fmt.Println(strings.Join([]string{fmt.Sprintf("%d", class.CRN), fmt.Sprintf("%d", class.Open), class.Title, class.Instructor, class.Time, class.Location}, "\t"))
		} else {
			fmt.Println(CRN)
		}
	}

	log.Println("Search complete")
	return nil
}

/* refresh command */
func refreshCache(ctx *cli.Context) (err error) {
	log.Println("Deleting files and forcing refresh")

	// remove cache files if present
	os.Remove(Storage.OptCache.Info.Filename)
	os.Remove(Storage.Info.Filename)

	// force refresh
	err = Storage.Restore()
	if err != nil {
		return
	}

	log.Println("Refresh complete")
	return nil
}
