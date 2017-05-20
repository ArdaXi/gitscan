package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strings"
)

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

func ParseSignatures(path string) ([]*Signature, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var sigs []GitrobSignature
	err = json.Unmarshal(file, &sigs)
	if err != nil {
		return nil, err
	}

	var result []*Signature
	for _, sig := range sigs {
		signature := &Signature{
			Part:        parsePart(sig.Part),
			Caption:     sig.Caption,
			Description: sig.Description,
		}
		if signature.Part == unknownPart {
			return nil, fmt.Errorf("Unknown part: %v", sig.Part)
		}
		switch sig.Type {
		case "regex":
			re, err := regexp.Compile(sig.Pattern)
			if err != nil {
				return nil, err
			}
			signature.Regex = re
		case "match":
			signature.Pattern = sig.Pattern
		default:
			return nil, fmt.Errorf("Unknown type: %v", sig.Type)
		}
		result = append(result, signature)
	}
	return result, nil
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

type CheckResult struct {
	Path      string
	Signature *Signature
}

func CheckPath(sigs []*Signature, filePath string) (int, []*CheckResult) {
	var results []*CheckResult
	count := 0
	for _, sig := range sigs {
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
				results = append(results, &CheckResult{Path: filePath, Signature: sig})
				count++
			}
			continue
		}
		if strings.Contains(check, sig.Pattern) {
			results = append(results, &CheckResult{Path: filePath, Signature: sig})
			count++
			continue
		}
	}
	return count, results
}
