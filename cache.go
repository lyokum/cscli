package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

/* FIXME: remove these after testing */
var _ = fmt.Printf

const (
	DefaultClassRate   = time.Hour * 24
	DefaultOptionsRate = time.Hour * 24 * 7
)

var (
	ErrNoCache = errors.New("Cache was not initialied")
)

type Cache interface {
	// cache-specific info
	GetInfo() (timer CacheInfo)

	// json
	MakeJSON() (blob []byte, err error)
	ExtractJSON(blob []byte) (err error)

	// api
	FetchData() (err error)
}

type CacheInfo struct {
	Timestamp time.Time
	Rate      time.Duration
	Filename  string
	Directory string
}

type ClassCache struct {
	Info     CacheInfo
	Classes  ClassList
	OptCache *OptionsCache
}

type OptionsCache struct {
	Info    CacheInfo
	Options SearchOptions
}

/* CacheInfo Functions */
func (info *CacheInfo) Init(filename string, rate time.Duration) {
	info.Timestamp = time.Now()
	info.Rate = rate
	info.Filename = filename
	info.Directory = "/tmp/"
}

func (info CacheInfo) IsStale() bool {
	return time.Now().Sub(info.Timestamp) >= info.Rate
}

func (info CacheInfo) Filepath() string {
	return info.Directory + info.Filename
}

/* ClassCache Functions */
func (cache *ClassCache) Init() {
	// init fields
	cache.Info.Init("class_cache.json", DefaultClassRate)
	cache.OptCache = &OptionsCache{}
	cache.OptCache.Init()
}

func (cache *ClassCache) Restore() (err error) {
	log.Println("Reading/Restoring cache from file")

	err = Restore(cache.OptCache)
	if err != nil {
		return
	}

	err = Restore(cache)
	if err != nil {
		return
	}

	log.Println("Restoration complete")
	return nil
}

func (cache *ClassCache) SetDirectory(dir string) (err error) {
	// check that directory file exists
	file, err := os.Open(dir)
	defer file.Close()
	if err != nil {
		return
	}

	// check that file is directory
	info, err := file.Stat()
	if err != nil {
		return
	}

	if !info.IsDir() {
		return ErrInvalidDirectory
	}

	// set cache directories
	if !(dir[len(dir)-1] == '/') {
		dir += "/"
	}

	cache.OptCache.Info.Directory = dir
	cache.Info.Directory = dir

	return nil
}

func (cache *ClassCache) FetchUpdates(CRNs []int) (err error) {
	log.Println("Performing data update")
	subjects := make(map[string]bool)

	// get subjects
	for _, CRN := range CRNs {
		class, ok := cache.Classes.Map[CRN]

		// check to make sure CRN is valid
		if !ok {
			return ErrNoClass
		}

		subjects[class.GetSubject()] = true
	}

	// create form
	var input FormInput
	input.Init(cache.OptCache.Options)
	input.Subjects = make([]string, 0, 10)

	for subject := range subjects {
		input.Subjects = append(input.Subjects, subject)
	}

	// retrieve updated classes from data fetch
	updates, err := ParseParallel(input)
	if err != nil {
		return
	}

	// add updates to cache
	err = cache.Classes.Update(updates)
	if err != nil {
		return
	}

	// store results
	err = Store(cache)
	if err != nil {
		return err
	}

	log.Println("Update complete")
	return nil
}

func (cache ClassCache) GetInfo() (info CacheInfo) {
	return cache.Info
}

func (cache ClassCache) MakeJSON() (blob []byte, err error) {
	return json.Marshal(cache)
}

func (cache *ClassCache) ExtractJSON(blob []byte) (err error) {
	return json.Unmarshal(blob, &cache)
}

func (cache *ClassCache) FetchData() (err error) {
	cache.Info.Timestamp = time.Now()

	// get html for current options
	var input FormInput
	input.Init(cache.OptCache.Options)
	cache.Classes, err = ParseParallel(input)
	if err != nil {
		return
	}

	return nil
}

/* OptionsCache Functions */
func (cache *OptionsCache) Init() {
	cache.Info.Init("options_cache.json", DefaultOptionsRate)
}

func (cache OptionsCache) GetInfo() (info CacheInfo) {
	return cache.Info
}

func (cache OptionsCache) MakeJSON() (blob []byte, err error) {
	return json.Marshal(cache)
}

func (cache *OptionsCache) ExtractJSON(blob []byte) (err error) {
	return json.Unmarshal(blob, &cache)
}

func (cache *OptionsCache) FetchData() (err error) {
	cache.Info.Timestamp = time.Now()

	doc, err := ParseHTML("")
	if err != nil {
		return
	}

	cache.Options, err = GetOptions(doc)
	return
}

/* Cache Functions */
func Store(cache Cache) (err error) {
	if cache == nil {
		return ErrNoCache
	}

	log.Println("Encoding cache data")

	// create json string
	blob, err := cache.MakeJSON()
	if err != nil {
		return
	}

	log.Println("Writing encoded data to file")

	// write to file
	err = ioutil.WriteFile(cache.GetInfo().Filepath(), blob, 0644)
	if err != nil {
		return
	}

	return nil
}

func Restore(cache Cache) (err error) {
	if cache == nil {
		return ErrNoCache
	}

	fileExists := false
	fileValid := true

	// check if file exists
	file, err := os.Open(cache.GetInfo().Filepath())
	file.Close()
	if err == nil {
		fileExists = true
	}

	if fileExists {
		log.Println("Restoring cache from file")

		// read in from file
		blob, err := ioutil.ReadFile(cache.GetInfo().Filepath())
		if err != nil {
			return err
		}

		log.Println("Extracting file data to cache")

		// parse JSON into fields
		err = cache.ExtractJSON(blob)
		if err != nil {
			fileValid = false
		}
	}

	// perform refresh
	if !fileExists || !fileValid || cache.GetInfo().IsStale() {
		log.Println("Fetching cache data for refresh")

		// get new data
		err = cache.FetchData()
		if err != nil {
			return err
		}

		log.Println("Storing new data in file")

		// store new data in file
		err = Store(cache)
		if err != nil {
			return
		}
	}

	return nil
}
