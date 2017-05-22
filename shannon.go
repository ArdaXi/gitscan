package main

import (
	"io"
	"bufio"
	"strings"
	"math"
	"fmt"
)

func CheckShannon(path string, file io.Reader) (int, []*CheckResult) {
	var results []*CheckResult
	var count int
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		text := scanner.Text()
		entropy := getEntropy(text)
		if entropy > 4.5 {
			results = append(results, &CheckResult{
				Path: path,
				Caption: fmt.Sprintf("High entropy string: %f", entropy),
				Description: text,
			})
			count++
		}
	}
	return count, results
}

func getEntropy(data string) float64 {
	var entropy float64
	length := float64(len(data))
	for _, x := range "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=" {
		pX := float64(strings.Count(data, string(x))) / length
		if pX > 0 {
			entropy += - pX * math.Log2(pX)
		}
	}
	return entropy
}