package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/lyokum/attr"
	"golang.org/x/net/html"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// FIXME: remove these after testing
var _ = fmt.Printf

const (
	URL = "https://class-search.nd.edu/reg/srch/ClassSearchServlet"
)

var (
	ErrNodeNotFound  = errors.New("Node not found")
	ErrFieldNotFound = errors.New("Field not found")
)

/* Doc Creation */
func ParseFull() (doc *html.Node, err error) {
	// get options from html
	opts, err := GetOptions(doc)
	if err != nil {
		return nil, err
	}

	// set default form input
	var input FormInput
	input.Init(opts)

	return ParseHTML(input.String())
}

func ParseHTML(formStr string) (doc *html.Node, err error) {
	log.Println("Sending request to site")

	// get html from form request
	cmd := exec.Command("curl", "--retry", "3", "--data", formStr, URL)
	page, err := cmd.Output()
	if err != nil {
		log.Println("Error with curl for form", formStr)
		return nil, err
	}

	log.Println("Parsing HTML of response")

	// parse html into tree
	doc, err = html.Parse(bytes.NewReader(page))
	if err != nil {
		log.Println("Error with html.Parse for form", formStr)
		return nil, err
	}

	return doc, nil
}

func ParseParallel(input FormInput) (classes ClassList, err error) {
	classes.Init()

	// concurrency vars
	wg := &sync.WaitGroup{}
	errChan := make(chan error, len(input.Subjects))
	classChan := make(chan Class, len(input.Subjects))
	done := make(chan bool)

	// accumulate classes
	go func() {
		for class := range classChan {
			classes.Add(class)
		}
		done <- true
	}()

	// run requests concurrently
	for _, subject := range input.Subjects {
		wg.Add(1)

		go func(input FormInput, subject string) {
			defer wg.Done()

			// create subset of full input
			subinput := input
			subinput.Subjects = []string{subject}

			// make request
			doc, err := ParseHTML(subinput.String())
			if err != nil {
				log.Println("Parse error with subject", subject)
				errChan <- err
				return
			}

			// parse request info
			subclasses, err := GetClasses(doc, nil)
			if err != nil {
				log.Println("Error with subject", subject)
				return
			}

			// add classes to chan
			for _, class := range subclasses.Map {
				classChan <- class
			}
		}(input, subject)
	}
	wg.Wait()
	close(errChan)
	close(classChan)
	<-done

	// fill returned error if available
	select {
	case err = <-errChan:
	default:
	}

	return classes, err
}

/* Doc Parsers */
func GetClasses(doc *html.Node, CRNs []int) (classes ClassList, err error) {
	// get table node
	tableNode := attr.GetElementById(doc, "resulttable")
	if tableNode == nil {
		return ClassList{}, ErrNodeNotFound
	}

	// find tbody
	startNode := tableNode.FirstChild
	for startNode.Data != "tbody" {
		startNode = startNode.NextSibling
	}
	startNode = startNode.FirstChild

	// loop through table rows and add classes
	classes.Init()
	for row := startNode; row != nil; row = row.NextSibling {
		// skip text elements
		if row.Type != html.ElementNode {
			continue
		}

		// loop through columns and add class info
		var class Class
		addClass := false
		for column, i := row.FirstChild, 0; column != nil; column, i = column.NextSibling, i+1 {
			// skip text elements
			if column.Type != html.ElementNode {
				i--
				continue
			}

			// extract data based on table column index
			switch i {
			case 0:
				class.Section = strings.TrimSpace(column.FirstChild.FirstChild.Data) // navigate tr -> td (index 1) -> a -> text elem
			case 1:
				class.Title = strings.TrimSpace(column.FirstChild.Data)
			case 2:
				class.Credits = strings.TrimSpace(column.FirstChild.Data)
			case 4:
				class.Max, err = strconv.Atoi(strings.TrimSpace(column.FirstChild.Data))
			case 5:
				class.Open, err = strconv.Atoi(strings.TrimSpace(column.FirstChild.Data))
			case 7:
				class.CRN, err = strconv.Atoi(strings.TrimSpace(column.FirstChild.Data))

				// check if need to store
				for _, CRN := range CRNs {
					if class.CRN == CRN {
						addClass = true
						break
					}
				}

				// quit if not
				if CRNs != nil && !addClass {
					break
				}
			case 9:
				// check if irregular format
				if column.FirstChild.NextSibling == nil {
					class.Instructor = strings.TrimSpace(column.FirstChild.Data) // just a text field for TBA
				} else {
					class.Instructor = strings.TrimSpace(column.FirstChild.NextSibling.FirstChild.Data) // must navigate anchor text
				}
			case 10:
				class.Time = strings.TrimSpace(column.FirstChild.Data)
			case 13:
				class.Location = strings.TrimSpace(column.FirstChild.Data)
			}

			// check if int conversion problem
			if err != nil {
				return classes, err
			}
		}

		// add created and filled class
		if CRNs == nil || addClass {
			classes.Add(class)
		}
	}

	return classes, nil
}

func GetFields(doc *html.Node, name string) (fields map[string]string, err error) {
	// get select node with appropriate name
	selectNode := attr.GetElementByName(doc, name)

	if selectNode == nil {
		return nil, ErrNodeNotFound
	}

	for curr := selectNode.FirstChild; curr != nil; curr = curr.NextSibling {
		// skip text nodes
		if curr.Type != html.ElementNode {
			continue
		}

		// loop through attributes to find value
		for _, attribute := range curr.Attr {
			if attribute.Key == "value" {
				// only init if there are elements to add
				if fields == nil {
					fields = make(map[string]string)
				}

				// add element
				fields[attribute.Val] = strings.TrimSpace(curr.FirstChild.Data)
				break
			}
		}

	}

	if fields != nil {
		return fields, nil
	} else {
		return nil, ErrFieldNotFound
	}
}

func GetOptions(doc *html.Node) (opts SearchOptions, err error) {
	// maps container names to corresponding struct field
	categories := map[string]*map[string]string{"TERM": &opts.Terms,
		"DIVS":   &opts.Divisions,
		"CAMPUS": &opts.Campuses,
		"SUBJ":   &opts.Subjects,
		"ATTR":   &opts.Attributes,
		"CREDIT": &opts.Credits,
	}

	// concurrency vars
	errChan := make(chan error, len(categories))
	wg := &sync.WaitGroup{}
	lock := &sync.Mutex{}

	// run searches in parallel
	for cat, field := range categories {
		wg.Add(1)
		errChan <- parallelFill(doc, cat, field, lock, wg)
	}
	wg.Wait()
	close(errChan)

	// fill returned error if available
	select {
	case err = <-errChan:
	default:
	}

	return
}

func parallelFill(doc *html.Node, name string, field *map[string]string, lock *sync.Mutex, wg *sync.WaitGroup) error {
	defer wg.Done()

	// store copy of location rather than waiting with lock
	mapping, err := GetFields(doc, name)

	if err != nil {
		return err
	} else {
		// fill in values from mapping
		lock.Lock()
		*field = make(map[string]string)
		for key, val := range mapping {
			(*field)[key] = val
		}
		lock.Unlock()
	}

	return nil
}
