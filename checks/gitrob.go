package checks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strings"

	"github.com/ardaxi/gitscan/providers"
)

var signatures []*Signature

type part int

const (
	unknownPart part = iota
	pathPart
	filenamePart
	extensionPart
)

type GitrobSignature struct {
	Part        string
	Type        string
	Pattern     string
	Caption     string
	Description string
}

type Signature struct {
	Part        part
	Pattern     string
	Regex       *regexp.Regexp
	Caption     string
	Description string
}

func ParseSignatures(path string) error {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	var sigs []GitrobSignature
	err = json.Unmarshal(file, &sigs)
	if err != nil {
		return err
	}

	for _, sig := range sigs {
		signature := &Signature{
			Part:        parsePart(sig.Part),
			Caption:     sig.Caption,
			Description: sig.Description,
		}
		if signature.Part == unknownPart {
			return fmt.Errorf("Unknown part: %v", sig.Part)
		}
		switch sig.Type {
		case "regex":
			re, err := regexp.Compile(sig.Pattern)
			if err != nil {
				return err
			}
			signature.Regex = re
		case "match":
			signature.Pattern = sig.Pattern
		default:
			return fmt.Errorf("Unknown type: %v", sig.Type)
		}
		signatures = append(signatures, signature)
	}
	return nil
}

func parsePart(part string) part {
	switch part {
	case "path":
		return pathPart
	case "filename":
		return filenamePart
	case "extension":
		return extensionPart
	default:
		return unknownPart
	}
}

func GitrobCheck(file providers.File, c chan<- *Result, done func()) {
	filePath := file.Path()
	for _, sig := range signatures {
		var check string
		switch sig.Part {
		case extensionPart:
			check = path.Ext(filePath)
		case filenamePart:
			check = path.Base(filePath)
		default:
			check = filePath
		}
		if sig.Regex != nil {
			if sig.Regex.MatchString(check) {
				c <- &Result{File: file, Caption: sig.Caption, Description: sig.Description}
			}
			continue
		}
		if strings.Contains(check, sig.Pattern) {
			c <- &Result{File: file, Caption: sig.Caption, Description: sig.Description}
			continue
		}
	}
	done()
}

func init() {
	Checks = append(Checks, GitrobCheck)
}
