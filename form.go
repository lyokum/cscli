package main

import (
	"fmt"
)

type FormInput struct {
	Term      string
	Division  string
	Campus    string
	Subjects  []string
	Attribute string
	Credit    string
}

func (input *FormInput) Init(opt SearchOptions) {
	input.Term = "201910"
	input.Division = "A"
	input.Campus = "M"
	input.Attribute = "0ANY"
	input.Credit = "A"

	// fill all subjects
	if len(opt.Subjects) > 0 {
		input.Subjects = []string{}
		for subj, _ := range opt.Subjects {
			input.Subjects = append(input.Subjects, subj)
		}
	}
}

func (input *FormInput) String() (formStr string) {
	formTemplate := "TERM=%s&DIVS=%s&CAMPUS=%s&%s&ATTR=%s&CREDIT=%s"

	// create extended subject string
	subj_str := ""
	for i, subj := range input.Subjects {
		if i != 0 {
			subj_str += "&"
		}

		subj_str += "SUBJ=" + subj
	}

	formStr = fmt.Sprintf(formTemplate, input.Term, input.Division, input.Campus, subj_str, input.Attribute, input.Credit)
	return
}
